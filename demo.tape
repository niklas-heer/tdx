# tdx demo - Your todos, in markdown, done fast
# Run with: vhs demo.tape

Output assets/demo-0.10.0.gif

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

# Setup: Create a demo todo file with sections, tags, and priorities
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
Type "- [ ] Fix security vulnerability !p1 #backend #urgent @due(2025-11-28)"
Enter
Type "- [ ] Set up CI/CD pipeline !p2 #devops @due(2025-12-05)"
Enter
Type "- [ ] Write unit tests #testing"
Enter
Type "- [x] Initialize repository #setup"
Enter
Enter
Type "## Frontend"
Enter
Type "- [ ] Update documentation !p3 #docs @due(2025-12-15)"
Enter
Type "- [ ] Add error handling !p1 #bugfix #urgent @due(2025-11-29)"
Enter
Type "- [x] Design mockups #design"
Enter
Enter
Type "## DevOps"
Enter
Type "- [ ] Configure monitoring !p2 #devops @due(2025-12-01)"
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

# Navigate with j/k - notice priority colors
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

# Add new todo with priority
Type "n"
Sleep 600ms
Type@50ms "Implement caching layer !p2 #backend"
Sleep 400ms
Enter
Sleep 1s

# Priority filtering - show only p1 tasks
Type "p"
Sleep 800ms
Space
Sleep 1s

# Show p1 tasks are filtered - navigate
Type "j"
Sleep 500ms
Type "j"
Sleep 800ms

# Clear priority filter
Type "p"
Sleep 500ms
Type "c"
Sleep 500ms
Escape
Sleep 1s

# Sort by priority
Type ":"
Sleep 500ms
Type@60ms "sort-priority"
Sleep 600ms
Enter
Sleep 1.2s

# Undo the sort
Type "u"
Sleep 1s

# Tag filtering - show #urgent tasks
Type "t"
Sleep 800ms
Type "j"
Sleep 400ms
Space
Sleep 1s

# Clear tag filter
Type "t"
Sleep 500ms
Type "c"
Sleep 500ms
Escape
Sleep 1s

# Due date filtering - show overdue/upcoming tasks
Type "D"
Sleep 800ms
Type "j"
Sleep 400ms
Enter
Sleep 1s

# Sort by due date
Type ":"
Sleep 500ms
Type@60ms "sort-due"
Sleep 600ms
Enter
Sleep 1.2s

# Clear due filter
Type "D"
Sleep 500ms
Escape
Sleep 1s

# Fuzzy search
Type "/"
Sleep 600ms
Type@60ms "error"
Sleep 1s
Enter
Sleep 1s

# Nested tasks - indent with Tab
Type "gg"
Sleep 400ms
Type "jjj"
Sleep 400ms
Type "n"
Sleep 500ms
Type@50ms "Configure nginx"
Enter
Sleep 600ms
Tab
Sleep 800ms

# Outdent with Shift+Tab
Shift+Tab
Sleep 1s

# Move mode - move task
Type "m"
Sleep 600ms
Type "jj"
Sleep 600ms
Enter
Sleep 1s

# Theme picker - live preview
Type ":"
Sleep 500ms
Type@60ms "theme"
Sleep 600ms
Enter
Sleep 1s

# Navigate themes with live preview
Type "j"
Sleep 800ms
Type "j"
Sleep 800ms
Type "j"
Sleep 800ms

# Cancel and restore original theme
Escape
Sleep 1s

# Copy to clipboard
Type "c"
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

Type "tdx demo_todo.md add 'Review pull requests !p2 #review'"
Enter
Sleep 1s

Type "tdx demo_todo.md toggle 1"
Enter
Sleep 1s

# Final message
Type@60ms "# Fast. Vim-style. Priorities. Due dates. Themes."
Enter
Sleep 2.5s

# Cleanup
Hide
Type "rm -f demo_todo.md"
Enter
Show
