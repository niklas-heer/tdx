# tdx demo - Your todos, in markdown, done fast
# Run with: vhs demo.tape

Output assets/demo.gif

Require tdx

Set Shell "bash"
Set FontSize 28
Set Width 1200
Set Height 700
Set Theme "TokyoNight"
Set TypingSpeed 50ms
Set Padding 20

# Setup: Create a fresh demo todo file
Hide
Type "rm -f demo_todo.md"
Enter
Type "cat > demo_todo.md << 'EOF'"
Enter
Type "# Project Tasks"
Enter
Enter
Type "- [ ] Set up CI/CD pipeline"
Enter
Type "- [ ] Write unit tests"
Enter
Type "- [ ] Update documentation"
Enter
Type "- [x] Initialize repository"
Enter
Type "- [ ] Add error handling"
Enter
Type "- [ ] Refactor database layer"
Enter
Type "EOF"
Enter
Sleep 500ms
Show

# Start the demo
Type "# tdx - Your todos, in markdown, done fast"
Sleep 1.5s
Enter
Sleep 1s

# Launch TUI
Type "tdx demo_todo.md"
Sleep 500ms
Enter
Sleep 2s

# Navigate with j/k
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Vim-style jump
Type "3k"
Sleep 1.5s

# Toggle completion with space
Space
Sleep 1.5s

# Navigate and toggle another
Type "jj"
Sleep 500ms
Space
Sleep 1.5s

# Add new todo
Type "n"
Sleep 800ms
Type "Implement caching layer"
Sleep 500ms
Enter
Sleep 1.5s

# Edit a todo - navigate to "Update documentation"
Type "kkk"
Sleep 800ms
Type "e"
Sleep 800ms
# Go to end and append
Ctrl+E
Sleep 300ms
Type " with examples"
Sleep 500ms
Enter
Sleep 1.5s

# Fuzzy search
Type "/"
Sleep 800ms
Type "error"
Sleep 1.5s
Enter
Sleep 1.5s

# Move mode - reorder task
Type "m"
Sleep 800ms
Type "kk"
Sleep 800ms
Enter
Sleep 1.5s

# Copy to clipboard
Type "c"
Sleep 1.5s

# Delete a todo
Type "jj"
Sleep 500ms
Type "d"
Sleep 1.5s

# Show help menu
Type "?"
Sleep 3s
Escape
Sleep 800ms

# Undo the delete
Type "u"
Sleep 1.5s

# Exit
Escape
Sleep 1.5s

# Show CLI commands
Type "# CLI commands"
Enter
Sleep 1s

Type "tdx demo_todo.md list"
Enter
Sleep 2s

Type "tdx demo_todo.md add 'Deploy to production'"
Enter
Sleep 1.5s

Type "tdx demo_todo.md toggle 1"
Enter
Sleep 1.5s

# Mention themes
Sleep 500ms
Type "# 10 builtin themes: tokyo-night, dracula, nord, gruvbox..."
Enter
Sleep 2s

# Final message
Type "# Fast, simple, markdown-native todos"
Enter
Sleep 3s

# Cleanup
Hide
Type "rm -f demo_todo.md"
Enter
Show
