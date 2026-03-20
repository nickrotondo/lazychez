package ui

import "testing"

func TestBuildTree_FlatFiles(t *testing.T) {
	items := []FileItem{
		{Path: ".zshrc", AddCol: ' ', ApplyCol: ' '},
		{Path: ".gitconfig", AddCol: ' ', ApplyCol: ' '},
	}
	root := buildTree(items)
	if len(root.children) != 2 {
		t.Fatalf("root has %d children, want 2", len(root.children))
	}
	// Files sorted alphabetically
	if root.children[0].name != ".gitconfig" {
		t.Errorf("children[0].name = %q, want .gitconfig", root.children[0].name)
	}
	if root.children[1].name != ".zshrc" {
		t.Errorf("children[1].name = %q, want .zshrc", root.children[1].name)
	}
}

func TestBuildTree_NestedDirs(t *testing.T) {
	items := []FileItem{
		{Path: ".config/nvim/init.lua"},
		{Path: ".config/nvim/plugins.lua"},
		{Path: ".config/alacritty/alacritty.toml"},
	}
	root := buildTree(items)
	// Should have one child: .config dir
	if len(root.children) != 1 {
		t.Fatalf("root has %d children, want 1", len(root.children))
	}
	config := root.children[0]
	if config.name != ".config" {
		t.Errorf("config.name = %q, want .config", config.name)
	}
	if !config.isDir() {
		t.Error(".config should be a dir node")
	}
	// .config has 2 children: alacritty and nvim (alphabetical)
	if len(config.children) != 2 {
		t.Fatalf(".config has %d children, want 2", len(config.children))
	}
}

func TestBuildTree_AlphabeticalMix(t *testing.T) {
	items := []FileItem{
		{Path: ".zshrc"},
		{Path: ".config/nvim/init.lua"},
	}
	root := buildTree(items)
	// Purely alphabetical: .config before .zshrc
	if len(root.children) != 2 {
		t.Fatalf("root has %d children, want 2", len(root.children))
	}
	if root.children[0].name != ".config" {
		t.Errorf("first child name = %q, want .config", root.children[0].name)
	}
	if root.children[1].name != ".zshrc" {
		t.Errorf("second child name = %q, want .zshrc", root.children[1].name)
	}
}

func TestBuildTree_PurelyAlphabetical(t *testing.T) {
	items := []FileItem{
		{Path: ".afile", AddCol: ' ', ApplyCol: ' '},
		{Path: ".bfile", AddCol: 'M', ApplyCol: ' '},
	}
	root := buildTree(items)
	// Purely alphabetical regardless of status
	if root.children[0].name != ".afile" {
		t.Errorf("first child = %q, want .afile", root.children[0].name)
	}
	if root.children[1].name != ".bfile" {
		t.Errorf("second child = %q, want .bfile", root.children[1].name)
	}
}

func TestFlattenTree_Basic(t *testing.T) {
	items := []FileItem{
		{Path: ".config/nvim/init.lua"},
		{Path: ".zshrc"},
	}
	root := buildTree(items)
	flat := flattenTree(root, nil)

	// Expect: .config/nvim (compressed dir), init.lua, .zshrc
	if len(flat) != 3 {
		t.Fatalf("flat has %d items, want 3", len(flat))
	}
	if !flat[0].IsDir || flat[0].TreeName != ".config/nvim" {
		t.Errorf("flat[0]: IsDir=%v, TreeName=%q, want dir '.config/nvim'", flat[0].IsDir, flat[0].TreeName)
	}
	if flat[0].TreeDepth != 0 {
		t.Errorf("flat[0].TreeDepth = %d, want 0", flat[0].TreeDepth)
	}
	if flat[1].TreeName != "init.lua" || flat[1].TreeDepth != 1 {
		t.Errorf("flat[1]: TreeName=%q depth=%d, want 'init.lua' depth=1", flat[1].TreeName, flat[1].TreeDepth)
	}
	if flat[2].TreeName != ".zshrc" || flat[2].TreeDepth != 0 {
		t.Errorf("flat[2]: TreeName=%q depth=%d, want '.zshrc' depth=0", flat[2].TreeName, flat[2].TreeDepth)
	}
}

func TestFlattenTree_Collapsed(t *testing.T) {
	items := []FileItem{
		{Path: ".config/nvim/init.lua"},
		{Path: ".config/nvim/plugins.lua"},
		{Path: ".zshrc"},
	}
	root := buildTree(items)
	collapsed := map[string]bool{".config/nvim": true}
	flat := flattenTree(root, collapsed)

	// .config/nvim compressed and collapsed → just the dir + .zshrc
	if len(flat) != 2 {
		t.Fatalf("flat has %d items, want 2", len(flat))
	}
	if !flat[0].IsDir || !flat[0].DirCollapsed {
		t.Error("flat[0] should be a collapsed dir")
	}
	if flat[1].Path != ".zshrc" {
		t.Errorf("flat[1].Path = %q, want .zshrc", flat[1].Path)
	}
}

func TestFlattenTree_DirCompression(t *testing.T) {
	items := []FileItem{
		{Path: "a/b/c/file.txt"},
	}
	root := buildTree(items)
	flat := flattenTree(root, nil)

	// a/b/c should be compressed into one dir node
	if len(flat) != 2 {
		t.Fatalf("flat has %d items, want 2", len(flat))
	}
	if flat[0].TreeName != "a/b/c" {
		t.Errorf("flat[0].TreeName = %q, want 'a/b/c'", flat[0].TreeName)
	}
}

func TestFlattenTree_NoDirCompressionWithMultipleChildren(t *testing.T) {
	items := []FileItem{
		{Path: ".config/nvim/init.lua"},
		{Path: ".config/alacritty/alacritty.toml"},
	}
	root := buildTree(items)
	flat := flattenTree(root, nil)

	// .config has 2 children so should NOT be compressed
	// Expect: .config, alacritty (compressed with alacritty.toml? no, alacritty only has 1 child)
	// Actually: .config dir, then alacritty/alacritty.toml compressed, then nvim/init.lua compressed
	if len(flat) != 5 {
		t.Fatalf("flat has %d items, want 5", len(flat))
	}
	if flat[0].TreeName != ".config" || !flat[0].IsDir {
		t.Errorf("flat[0] = %q IsDir=%v, want '.config' dir", flat[0].TreeName, flat[0].IsDir)
	}
}

func TestFileListModel_ToggleCollapse(t *testing.T) {
	m := NewFileListModel()
	m.SetDimensions(80, 20)
	m.SetFiles([]FileItem{
		{Path: ".config/nvim/init.lua"},
		{Path: ".config/nvim/plugins.lua"},
		{Path: ".zshrc"},
	})

	// First item should be a dir
	if !m.files[0].IsDir {
		t.Fatal("first item should be a directory")
	}

	countBefore := len(m.files)
	m.ToggleCollapse() // collapse the dir

	if len(m.files) >= countBefore {
		t.Errorf("after collapse: %d items, should be less than %d", len(m.files), countBefore)
	}

	m.ToggleCollapse() // expand again
	if len(m.files) != countBefore {
		t.Errorf("after expand: %d items, want %d", len(m.files), countBefore)
	}
}

func TestFileListModel_SelectedPathReturnsEmptyForDir(t *testing.T) {
	m := NewFileListModel()
	m.SetDimensions(80, 20)
	m.SetFiles([]FileItem{
		{Path: ".config/nvim/init.lua"},
	})

	// First item is the dir node
	if m.files[0].IsDir && m.SelectedPath() != "" {
		t.Error("SelectedPath() should return empty for dir nodes")
	}
}

func TestFlattenTree_PreservesFileData(t *testing.T) {
	items := []FileItem{
		{Path: ".zshrc", SourceRelPath: "dot_zshrc", AddCol: 'M', ApplyCol: ' '},
	}
	root := buildTree(items)
	flat := flattenTree(root, nil)

	if len(flat) != 1 {
		t.Fatalf("flat has %d items, want 1", len(flat))
	}
	f := flat[0]
	if f.Path != ".zshrc" {
		t.Errorf("Path = %q, want .zshrc", f.Path)
	}
	if f.SourceRelPath != "dot_zshrc" {
		t.Errorf("SourceRelPath = %q, want dot_zshrc", f.SourceRelPath)
	}
	if f.AddCol != 'M' {
		t.Errorf("AddCol = %c, want M", f.AddCol)
	}
	if f.TreeName != ".zshrc" {
		t.Errorf("TreeName = %q, want .zshrc", f.TreeName)
	}
}
