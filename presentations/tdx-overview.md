---
title: "⚡ tdx"
sub_title: "Your Todos in Markdown, Done Fast"
author: Niklas Heer
theme:
  name: tokyonight-storm
  overrides:
    code:
      theme_name: "TwoDark"
    d2:
      background: transparent
---

<!--
Prerequisites for presenting:
  brew install d2
  brew install presenterm

Run with:
  presenterm -x presentations/tdx-overview.md
-->

🚀 What is tdx?
===

A **fast**, **single-binary** CLI todo manager

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

- ⚡ Native binary, no application runtime
- 📝 Markdown-native (`todo.md`)
- ⌨️ Vim-style navigation

<!-- column: 1 -->

- 🖥️ Interactive TUI + scriptable CLI
- 🔄 Version control friendly
- 🎨 Beautiful themes

<!-- reset_layout -->

<!-- end_slide -->

😩 The Problem
===

**I couldn't find a todo tool that was...**

> 🔧 **Actively maintained**

> 📝 **Markdown-native** — not a proprietary format

> 🔄 **Git-friendly** — local files, not central storage

> ⚡ **Fast** — no Electron, no bloat

---

So I built **tdx** — a Markdown workflow with keyboard-first editing 🏎️

<!-- end_slide -->

👀 Let's See It
===

```bash +exec +acquire_terminal
clear && tdx example.todo.md
```

<!-- end_slide -->

⌨️ Navigation
===

**Vim-style movement** — feels like home!

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

| Key | Action |
|-----|--------|
| `j` / `k` | ⬇️ Move down / ⬆️ up |
| `gg` | ⏫ Jump to first |
| `G` | ⏬ Jump to last |

<!-- column: 1 -->

| Key | Action |
|-----|--------|
| `5j` | Move down 5 lines |
| `3k` | Move up 3 lines |
| `/` | 🔍 Fuzzy search |

<!-- reset_layout -->

<!-- end_slide -->

✨ Core Actions
===

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

| Key | Action |
|-----|--------|
| `Space` | ✅ Toggle completion |
| `n` | ➕ New todo after cursor |
| `N` | ➕ New todo at end |

<!-- column: 1 -->

| Key | Action |
|-----|--------|
| `e` | ✏️ Edit todo |
| `d` | 🗑️ Delete todo |
| `u` | ↩️ Undo |

<!-- reset_layout -->

<!-- end_slide -->

💻 Let's Try the CLI
===

```bash +exec
# Add some todos
tdx example.todo.md add "Review the pull request"
tdx example.todo.md add "Update documentation"
tdx example.todo.md add "Deploy to staging"
```

<!-- end_slide -->

📋 List and Toggle
===

```bash +exec
# See what we have
tdx example.todo.md list
```

```bash +exec
# Mark one as done ✅
tdx example.todo.md toggle 1
tdx example.todo.md list
```

<!-- end_slide -->

📝 Checklists: Read-Only Mode
===

Perfect for **checklists** you don't want to accidentally modify! 🛡️

```bash +exec
cat > /tmp/release-checklist.md << 'EOF'
# Release Checklist

- [ ] Run full test suite
- [ ] Update CHANGELOG.md
- [ ] Bump version in tdx.toml
- [ ] Create git tag
- [ ] Push to trigger release
- [ ] Verify GitHub release
- [ ] Test brew install
EOF
```

<!-- end_slide -->

🔒 Checklists: Read-Only Demo
===

Start with `-r` flag for read-only mode:

```bash +exec +acquire_terminal
clear && tdx -r /tmp/release-checklist.md
```

> 💡 Try checking some items, then exit!

<!-- end_slide -->

✨ Checklists: Nothing Changed!
===

Even after checking items, the file is **unchanged**:

```bash +exec
cat /tmp/release-checklist.md
```

> 🎉 Changes won't be saved unless you `:save`

<!-- end_slide -->

🎮 Checklist Commands
===

Open command palette with `:` and use:

| Command | Description |
|---------|-------------|
| `check-all` | ✅ Mark all todos complete |
| `uncheck-all` | ⬜ Mark all todos incomplete |
| `clear-done` | 🧹 Delete all completed todos |
| `filter-done` | 👁️ Toggle hiding completed |
| `save` | 💾 Manually save (in read-only) |

<!-- end_slide -->

🏷️ Organization: Tags
===

Add **hashtags** for filtering:

```markdown
- [ ] Fix auth bug #urgent #backend
- [ ] Update docs #docs
- [ ] Add dark mode #feature #frontend
```

Press `t` to filter by tags 🔍

<!-- end_slide -->

🚨 Organization: Priorities
===

Add **priority markers**:

```markdown
- [ ] Security fix !p1
- [ ] Update deps !p2
- [ ] Refactor code !p3
```

<!-- column_layout: [1, 1, 1] -->

<!-- column: 0 -->

🔴 `!p1` = Critical

<!-- column: 1 -->

🟠 `!p2` = High

<!-- column: 2 -->

🟡 `!p3` = Medium

<!-- reset_layout -->

Press `p` to filter by priority

<!-- end_slide -->

📅 Organization: Due Dates
===

Add **due dates**:

```markdown
- [ ] Submit report @due(2025-12-01)
- [ ] Review PR @due(2025-11-30)
```

**Colors by urgency:**

- 🔴 **Overdue** = Red
- 🟠 **Today** = Orange  
- 🟡 **Soon** = Yellow (within 3 days)

Press `D` to filter by due date

<!-- end_slide -->

🪆 Nested Tasks
===

Organize **hierarchically** with `Tab` / `Shift+Tab`:

```markdown
- [ ] Main project
  - [ ] Subtask 1
  - [ ] Subtask 2
    - [ ] Sub-subtask
- [ ] Another task
```

> 💡 Great for breaking down complex tasks!

<!-- end_slide -->

🎯 Command Palette
===

Press `:` for **fuzzy-searchable** commands:

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

| Command | Action |
|---------|--------|
| `:sort-priority` | Sort by priority |
| `:sort-due` | Sort by due date |
| `:sort-done` | Sort by completion |

<!-- column: 1 -->

| Command | Action |
|---------|--------|
| `:filter-done` | Toggle completed |
| `:filter-overdue` | Show overdue only |
| `:theme` | 🎨 Change theme |

<!-- reset_layout -->

<!-- end_slide -->

🔍 Fuzzy Search
===

Press `/` to search todos:

- ⚡ **Live filtering** as you type
- 🎯 **Fuzzy matching** (e.g., "upd doc" → "Update documentation")
- `Enter` to select
- `Esc` to cancel

<!-- end_slide -->

📂 Recent Files
===

Press `r` to see recently opened files:

- 📊 Sorted by **frequency** and **recency**
- 📍 **Cursor position** restored
- 🔍 Fuzzy search to filter

```bash +exec
tdx recent
```

<!-- end_slide -->

⚙️ Configuration
===

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

**Per-file** (YAML frontmatter):

```markdown
---
read-only: true
max-visible: 10
show-headings: true
---
# My Todos
```

<!-- column: 1 -->

**Global behavior** (`config.yaml`):

```yaml
filter-done: false
word-wrap: true
```

**Theme** (`config.toml`):

```toml
[theme]
name = "tokyo-night"
```

<!-- reset_layout -->

<!-- end_slide -->

🎨 Themes
===

Press `:theme` to pick a theme:

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

- 🌃 tokyo-night *(default)*
- 🐱 catppuccin-mocha
- 🧛 dracula
- 🪴 gruvbox-dark

<!-- column: 1 -->

- 🧊 nord
- 🌹 rose-pine
- ☀️ solarized-dark
- ... and more!

<!-- reset_layout -->

> 💡 Create custom themes in `~/.config/tdx/themes/`

<!-- end_slide -->

🧠 Why It's Fast
===

**AST-based Markdown Engine**

```d2 +render
direction: right

file: 📄 File {
  style.fill: "#7aa2f7"
  style.font-color: "#1a1b26"
}
parser: 🔍 Goldmark Parser
ast: 🌳 AST {
  style.fill: "#9ece6a"
  style.font-color: "#1a1b26"
}
manipulate: ✏️ Manipulate
serialize: 📝 Serialize
output: 💾 File {
  style.fill: "#7aa2f7"
  style.font-color: "#1a1b26"
}

file -> parser -> ast -> manipulate -> serialize -> output
```

- ⚡ Parse once, manipulate in memory
- 🚫 No regex, no full-file rewrites
- 🎯 Zero-allocation navigation (~8ns)

<!-- end_slide -->

📊 Benchmarks
===

**Apple M4 results:** 🍎

| Operation | Time | Allocations |
|-----------|------|-------------|
| FuzzyScore (exact) | ⚡ 5.6ns | 0 |
| FuzzyScore (fuzzy) | ⚡ 33.4ns | 0 |
| Navigation | ⚡ 8.0ns | 0 |
| Search 100 todos | ⚡ 9.8µs | 114 |

> 🏎️ **Zero allocations** for core operations!

<!-- end_slide -->

📦 Installation
===

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

**🍺 Homebrew:**
```bash
brew install niklas-heer/tap/tdx
```

**⚡ Quick install:**
```bash
curl -fsSL https://niklas-heer.github.io/tdx/install.sh | bash
```

<!-- column: 1 -->

**❄️ Nix:**
```bash
nix run github:niklas-heer/tdx
```

**📥 Binary:**

Download from [GitHub Releases](https://github.com/niklas-heer/tdx/releases)

<!-- reset_layout -->

<!-- end_slide -->

🎉 Summary
===

**tdx gives you:**

- ⚡ Fast, single-binary todo manager
- 📝 Markdown files you can version control
- ⌨️ Vim-style navigation
- 🏷️ Powerful filtering (tags, priorities, due dates)
- 🔒 Read-only mode for checklists
- 💻 Scriptable CLI for automation
- 🎨 Beautiful themes

<!-- end_slide -->

❓ Questions?
===

<!-- column_layout: [1, 1] -->

<!-- column: 0 -->

**🔗 Links:**

- 🐙 GitHub: `github.com/niklas-heer/tdx`
- 🍺 Install: `brew install niklas-heer/tap/tdx`

<!-- column: 1 -->

**📬 Contact:**

- Twitter/X: `@niklas_heer`
- GitHub Issues welcome!

<!-- reset_layout -->

---

**Try it now:** 👇

```bash +exec +acquire_terminal
clear && tdx
```
