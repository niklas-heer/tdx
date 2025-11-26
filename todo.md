---
filter-done: true
max-visible: 0
show-headings: true
---
# tdx - features

## Refactor

- [x] split project up into multiple files
- [x] refactor and use AST for markdown
- [x] add 3-way-merge and conflict detection

## Release

- [x] CI -> release notes should be generated through an llm (openrouter?)
- [x] CI run tests for merge requests
- [x] make sure CI still works

## Priority requests

- [x] add command to toggle heading on (add flag for it as well or markdown metadata)
- [x] add command to set max-visible while running
- [x] add markdown metadata for options (persit: false, filter-done: true, ...)
- [x] add functionality to edit pervious seen files (saved in locale share or config)
- [x] add frontmatter for hide done/filter-done

## Reddit requests

- [ ] add date when task is due (@due(2025-11-24))
- [ ] add priorities and a priority-sort command (p1, p2, ...)
- [x] add nested tasks
- [x] add tags and f shortcut to filter for tags (#tag)

## Other features

- [x] make wrap the default
- [x] add new items after cursor (maybe command option to set after cursor or end of file)
- [ ] add command or keyboard shortcut (# or h) to add a heading before? after? the current cursor maybe that should go together with the insert setting?

## fix bugs

- [x] fix move bug that it sometimes (I don't know when - previous move?) doesn't add it to the bottom of the next heading group - seems to be the case if filter-done: true is active
- [x] fix delete should move the cursor to the nearest task also if the done tasks are hidden. We should seriously check that the ui handles hiddden stuff well. maybe we do another tree or something for what is visible.
- [x] fix new entry should also respent word-wrap
- [x] make sure we don't delete frontmatter

## Maybe

- [ ] add tui theme picker via command ": theme" which live previews
- [ ] add keyboard shortcut h to toggle hide for the done tasks (+config)
- [ ] expose all commands in frontmatter
- [ ] provide a way to render code segments which can be copied and are rendered as code maybe line code? maybe ctrl/cmd+shift+c
