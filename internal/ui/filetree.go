package ui

import (
	"sort"
	"strings"
)

type treeNode struct {
	name     string      // segment name, e.g. "nvim" or "init.lua"
	dirPath  string      // full path for dirs, e.g. ".config/nvim"
	children []*treeNode // sorted: dirs first, then files
	file     *FileItem   // non-nil for leaf file nodes
}

func (n *treeNode) isDir() bool {
	return n.file == nil
}

// buildTree constructs a tree from a flat list of FileItems by splitting paths on "/".
func buildTree(items []FileItem) *treeNode {
	root := &treeNode{name: ""}
	for i := range items {
		parts := strings.Split(items[i].Path, "/")
		cur := root
		for j, part := range parts {
			if j == len(parts)-1 {
				// Leaf file node
				cur.children = append(cur.children, &treeNode{
					name: part,
					file: &items[i],
				})
			} else {
				// Directory node — find or create
				dirPath := strings.Join(parts[:j+1], "/")
				found := false
				for _, child := range cur.children {
					if child.isDir() && child.name == part {
						cur = child
						found = true
						break
					}
				}
				if !found {
					dir := &treeNode{name: part, dirPath: dirPath}
					cur.children = append(cur.children, dir)
					cur = dir
				}
			}
		}
	}
	sortTree(root)
	return root
}

// sortTree recursively sorts children: directories first (alphabetical),
// then files (alphabetical).
func sortTree(node *treeNode) {
	for _, child := range node.children {
		if child.isDir() {
			sortTree(child)
		}
	}
	sort.SliceStable(node.children, func(i, j int) bool {
		return node.children[i].name < node.children[j].name
	})
}

// flattenTree walks the tree and produces a flat list of FileItems
// suitable for the file list model. Collapsed directories' children are skipped.
// Directories with only one child directory are compressed into a single node
// (e.g. ".config/nvim" instead of separate ".config" and "nvim" nodes).
func flattenTree(root *treeNode, collapsed map[string]bool) []FileItem {
	var items []FileItem
	for _, child := range root.children {
		flattenNode(child, 0, collapsed, &items)
	}
	return items
}

func flattenNode(node *treeNode, depth int, collapsed map[string]bool, items *[]FileItem) {
	if !node.isDir() {
		// Leaf file
		*items = append(*items, FileItem{
			Path:          node.file.Path,
			SourceRelPath: node.file.SourceRelPath,
			AddCol:        node.file.AddCol,
			ApplyCol:      node.file.ApplyCol,
			TreeDepth:     depth,
			TreeName:      node.name,
		})
		return
	}

	// Compress single-child directory chains: .config/nvim → ".config/nvim"
	compressed := node
	compressedName := node.name
	for len(compressed.children) == 1 && compressed.children[0].isDir() {
		compressed = compressed.children[0]
		compressedName += "/" + compressed.name
	}

	isCollapsed := collapsed[compressed.dirPath]
	*items = append(*items, FileItem{
		IsDir:        true,
		DirPath:      compressed.dirPath,
		TreeDepth:    depth,
		TreeName:     compressedName,
		DirCollapsed: isCollapsed,
	})

	if !isCollapsed {
		for _, child := range compressed.children {
			flattenNode(child, depth+1, collapsed, items)
		}
	}
}
