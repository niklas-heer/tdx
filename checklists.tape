# tdx checklists demo - Front matter and checklist features
# Run with: vhs checklists.tape

Output assets/checklists-0.10.0.gif

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

# Setup: Create a checklist file with front matter
Hide
Type "rm -f checklist.md"
Enter
Type "cat > checklist.md << 'EOF'"
Enter
Type "---"
Enter
Type "show-headings: true"
Enter
Type "filter-done: false"
Enter
Type "---"
Enter
Enter
Type "# Weekly Sprint Checklist"
Enter
Enter
Type "## Monday"
Enter
Type "- [ ] Review pending PRs #review #urgent"
Enter
Type "- [ ] Team standup notes #meeting"
Enter
Type "- [x] Check CI pipeline #devops"
Enter
Enter
Type "## Tuesday"
Enter
Type "- [ ] Write unit tests #testing"
Enter
Type "- [ ] Code review session #review"
Enter
Type "- [x] Update documentation #docs"
Enter
Enter
Type "## Wednesday"
Enter
Type "- [ ] Deploy to staging #devops #urgent"
Enter
Type "- [ ] Performance testing #testing"
Enter
Enter
Type "## Commands"
Enter
Type "- [ ] echo 'Deploy complete!' && date"
Enter
Type "EOF"
Enter
Sleep 300ms
Type "clear"
Enter
Sleep 300ms
Show

# Start the demo
Type@60ms "# tdx - Checklists with Front Matter"
Sleep 1.5s
Enter
Sleep 800ms

# Show the file content first
Type "head -n 10 checklist.md"
Sleep 400ms
Enter
Sleep 2.5s

Type@60ms "# Front matter configures per-file settings"
Enter
Sleep 1.5s

# Launch TUI
Type "tdx checklist.md"
Sleep 400ms
Enter
Sleep 1.5s

# Show headings are visible (from show-headings: true)
Type "gg"
Sleep 800ms

# Navigate and toggle some items
Type "jjj"
Sleep 400ms
Space
Sleep 600ms
Type "jj"
Sleep 300ms
Space
Sleep 800ms

# Filter by #urgent tag
Type "t"
Sleep 600ms
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms
Space
Sleep 1s

# Clear filter to show all
Type "t"
Sleep 500ms
Type "c"
Sleep 500ms
Escape
Sleep 1s

# Navigate to top and toggle first item
Type "gg"
Sleep 400ms
Space
Sleep 600ms

# Toggle filter-done via command palette
Type ":"
Sleep 500ms
Type@60ms "filter-done"
Sleep 500ms
Enter
Sleep 1.2s

# Completed items are now hidden - toggle it back
Type ":"
Sleep 400ms
Type@60ms "filter-done"
Sleep 500ms
Enter
Sleep 1s

# Check all items at once
Type ":"
Sleep 500ms
Type@60ms "check-all"
Sleep 500ms
Enter
Sleep 1.5s

# Uncheck all to reset the checklist
Type ":"
Sleep 500ms
Type@60ms "uncheck-all"
Sleep 500ms
Enter
Sleep 1.5s

# Copy a command to clipboard

# Navigate to the command task
Type "G"
Sleep 600ms
Type "c"
Sleep 1s

# Exit TUI
Escape
Sleep 800ms

# Paste and execute the command
Type@60ms "# Paste and run the copied command"
Enter
Sleep 800ms

Paste
Sleep 600ms
Enter
Sleep 1.5s

# Show CLI usage
Type@60ms "# CLI commands for quick operations"
Enter
Sleep 800ms

Type "tdx checklist.md list"
Enter
Sleep 1.5s

Type "tdx checklist.md add 'Friday review #review'"
Enter
Sleep 1s

Type "tdx checklist.md toggle 1"
Enter
Sleep 1s

# Show max-visible flag
Type@60ms "# Limit visible items with --max-visible"
Enter
Sleep 800ms

Type "tdx -m 3 checklist.md"
Enter
Sleep 1.5s

# Exit to show more CLI
Escape
Sleep 800ms

# Show read-only mode via CLI flag
Type@60ms "# Read-only mode prevents changes"
Enter
Sleep 800ms

Type "tdx -r checklist.md"
Enter
Sleep 1.5s

# Try to toggle (won't work)
Space
Sleep 800ms

# Exit
Escape
Sleep 1s

# Show it didn't change anything
Type "tdx checklist.md list"
Enter
Sleep 1.5s

# Final message
Type@60ms "# Organized checklists. Per-file config. Fast."
Enter
Sleep 2.5s

# Cleanup
Hide
Type "rm -f checklist.md"
Enter
Show
