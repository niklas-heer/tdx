# tdx demo - Your todos, in markdown, done fast
# Run with: vhs demo.tape

Output assets/demo-0.9.0.gif

Require tdx

Set Shell "bash"
Set FontSize 26
Set Width 1200
Set Height 700
Set Theme "TokyoNight"
Set TypingSpeed 40ms
Set Padding 20
Set WindowBar Colorful
Set BorderRadius 8
Set Margin 10
Set MarginFill "#1a1b26"
Set CursorBlink false

# Setup: Create a demo todo file with sections and tags
Hide
Type "rm -f demo_todo.md"
Enter
Type "cat > demo_todo.md << 'EOF'"
Enter
Type "# Project Tasks"
Enter
Enter
Type "## Backend"
Enter
Type "- [ ] Set up CI/CD pipeline #devops #urgent"
Enter
Type "- [ ] Write unit tests #testing"
Enter
Type "- [x] Initialize repository #setup"
Enter
Enter
Type "## Frontend"
Enter
Type "- [ ] Update documentation #docs"
Enter
Type "- [ ] Add error handling #bugfix #urgent"
Enter
Type "- [x] Design mockups #design"
Enter
Enter
Type "## DevOps"
Enter
Type "- [ ] Configure monitoring #devops"
Enter
Type "- [ ] Set up staging env #devops"
Enter
Type "EOF"
Enter
Sleep 300ms
Type "clear"
Enter
Sleep 300ms
Show

# Start the demo with a title
Type@60ms "# tdx - Your todos, in markdown, done fast"
Sleep 1.5s
Enter
Sleep 800ms

# Launch TUI
Type "tdx demo_todo.md"
Sleep 400ms
Enter
Sleep 1.5s

# Navigate with j/k
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms
Type "j"
Sleep 400ms

# Vim-style jump with number
Type "2k"
Sleep 1s

# Jump to last item with G
Type "G"
Sleep 1s

# Jump to first item with gg
Type "gg"
Sleep 1s

# Toggle completion with space
Space
Sleep 1s

# Navigate and toggle another
Type "jj"
Sleep 400ms
Space
Sleep 1s

# Add new todo with n (after cursor)
Type "n"
Sleep 600ms
Type@50ms "Implement caching layer #backend"
Sleep 400ms
Enter
Sleep 1s

# Jump to end and add todo with N (at end)
Type "N"
Sleep 600ms
Type@50ms "Deploy to production #release"
Sleep 400ms
Enter
Sleep 1s

# Nested tasks - indent with Tab
Type "kkk"
Sleep 400ms
Type "n"
Sleep 500ms
Type@50ms "Configure nginx"
Enter
Sleep 600ms
Tab
Sleep 800ms
Type "n"
Sleep 500ms
Type@50ms "Set up SSL certificates"
Enter
Sleep 600ms
Tab
Sleep 800ms

# Outdent with Shift+Tab
Shift+Tab
Sleep 1s

# Edit a todo
Type "kkkk"
Sleep 500ms
Type "e"
Sleep 600ms
Ctrl+E
Sleep 200ms
Type@50ms " with examples"
Sleep 400ms
Enter
Sleep 1s

# Fuzzy search
Type "/"
Sleep 600ms
Type@60ms "error"
Sleep 1s
Enter
Sleep 1s

# Tag filtering - show #urgent tasks
Type "f"
Sleep 800ms
Type "j"
Sleep 400ms
Space
Sleep 1s

# Clear tag filter and show all
Type "f"
Sleep 500ms
Type "c"
Sleep 500ms
Escape
Sleep 1s

# Move mode - move task across sections
Type "kkk"
Sleep 400ms
Type "m"
Sleep 600ms
Type "jjjj"
Sleep 800ms
Enter
Sleep 1s

# Command palette - sort tasks
Type ":"
Sleep 500ms
Type@60ms "sort"
Sleep 600ms
Enter
Sleep 1.2s

# Undo the sort
Type "u"
Sleep 1s

# Toggle line numbers
Type ":"
Sleep 400ms
Type@60ms "line-numbers"
Sleep 500ms
Enter
Sleep 1.2s

# Copy to clipboard
Type "c"
Sleep 1s

# Delete a todo
Type "jj"
Sleep 400ms
Type "d"
Sleep 1s

# Undo the delete
Type "u"
Sleep 1s

# Show help menu
Type "?"
Sleep 2.5s
Escape
Sleep 600ms

# Exit TUI
Escape
Sleep 1s

# Show CLI commands
Type@60ms "# CLI commands work too!"
Enter
Sleep 800ms

Type "tdx demo_todo.md list"
Enter
Sleep 1.5s

Type "tdx demo_todo.md add 'Review pull requests #review'"
Enter
Sleep 1s

Type "tdx demo_todo.md toggle 1"
Enter
Sleep 1s

# Recent files feature
Type@60ms "# Recent files with cursor restoration"
Enter
Sleep 800ms

Type "tdx recent"
Enter
Sleep 1.5s

# Final message
Type@60ms "# Fast. Vim-style. Markdown-native."
Enter
Sleep 2.5s

# Cleanup
Hide
Type "rm -f demo_todo.md"
Enter
Show
