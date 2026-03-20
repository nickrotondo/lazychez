# Hero GIF — Overview of lazychez
# Shows: launch, file navigation, status codes, template preview (t), re-add, pane switching, help overlay
#
# Run:  vhs demo/hero.tape
# Cleanup is automatic (hidden blocks revert ~/.zshrc changes)

Output demo/hero.gif

Require chezmoi

Set Shell "zsh"
Set FontFamily "Monaspace Argon"
Set FontSize 17
Set Width 1200
Set Height 600
Set Padding 12
Set BorderRadius 8
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 60ms
Set Framerate 24
Set Theme {"name":"Mavericks","black":"#090F11","red":"#FF6866","green":"#98E481","yellow":"#E4CD81","blue":"#81CAE4","purple":"#E481CA","cyan":"#81E2E4","white":"#E9F6FB","brightBlack":"#808080","brightRed":"#E74340","brightGreen":"#8AFF66","brightYellow":"#FFDB66","brightBlue":"#66D6FF","brightPurple":"#FF66D6","brightCyan":"#66FDFF","brightWhite":"#ffffff","background":"#172B32","foreground":"#E9F6FB","selection":"#0B5D7A","cursor":"#70CBEB"}

# --- Hidden setup: build local binary, create dest changes ---
Hide
Type 'cd ~/dev/nick/lazychez && go build -o /tmp/lazychez . && echo "# lazychez demo" >> ~/.zshrc && echo "alias lc=lazychez" >> ~/.zsh-aliases'
Enter
Sleep 8s
Type 'clear'
Enter
Sleep 500ms
Type "/tmp/lazychez"
Enter
Wait+Screen /Chezmoi Status/
Type "3"
Sleep 500ms
Show
Sleep 2s

# Switch to managed files pane
Type "1"
Sleep 1s

# Navigate to .zshrc.tmpl (second dirty file — template)
Type "j"
Sleep 700ms

# Preview the rendered template with chezmoi cat
Type "t"
Sleep 2.5s

# Back to diff view
Type "t"
Sleep 600ms

# Re-add the file
Type "s"
Sleep 700ms

# Browse a few more files
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 800ms

# Show help overlay
Type "?"
Sleep 2.5s

# Close help
Type "?"
Sleep 1s

# Switch to git status pane
Type "2"
Sleep 2s

# Show Commit
Space
Sleep 700ms
Type 'c'
Sleep 400ms
Type 'Commit to my dotfiles repo'
Sleep 500ms
Escape
Sleep 500ms

# --- Hidden cleanup: revert ~/.zshrc and ~/.zsh-aliases, remove temp binary ---
Hide

# Back to managed files
Type "1"
Sleep 1s

# Quit
Type "q"
Sleep 500ms

# Cleanup
Type 'sed -i "" "/^# lazychez demo$/d" ~/.zshrc'
Enter
Sleep 500ms
Type 'sed -i "" "/^alias lc=lazychez$/d" ~/.zsh-aliases'
Enter
Sleep 500ms
Type 'rm -f /tmp/lazychez'
Enter
Sleep 200ms
