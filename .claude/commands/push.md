---
description: Smart Git push with automatic fetch, rebase, and conflict handling.
---
Push local commits to remote, handling common issues automatically.

**Steps**

1. Run `git fetch origin` to get latest remote state
2. Get current branch name with `git branch --show-current`
3. Check if branch has an upstream with `git rev-parse --abbrev-ref @{upstream}`
   - If no upstream, set it: `git push -u origin <branch>`
4. Compare local vs remote with `git status -sb` and `git rev-list --left-right --count HEAD...@{upstream}`
5. Based on status:
   - **Up-to-date or ahead only**: Run `git push`
   - **Behind only**: Run `git pull --rebase` then `git push`
   - **Diverged**: See divergence handling below

**Divergence Handling**

1. Check if current branch is protected (main, master, develop, release/*)
   - If protected: STOP and inform user to resolve manually - never force push protected branches
2. For feature branches:
   - Run `git rebase origin/<branch>`
   - If rebase succeeds: Run `git push --force-with-lease`
   - If rebase has conflicts: Run `git rebase --abort` and inform user to resolve manually

**Protected Branch Rules**

These branches should never be force-pushed:
- `main`
- `master`
- `develop`
- `release/*`
- Any branch matching common protection patterns

On protected branches, if diverged, inform the user and suggest:
- Merging remote changes: `git merge origin/<branch>`
- Or creating a PR instead of direct push

**Error Handling**

- If push is rejected due to hooks: Show the hook output and stop
- If authentication fails: Inform user to check credentials
- If remote doesn't exist: Suggest `git remote add` or check remote name

**Safety Guarantees**

- Always fetch before any operation
- Never use `--force` (only `--force-with-lease`)
- Never force push protected branches
- Abort rebase on any conflicts - don't auto-resolve
- Show what's being pushed before pushing (commit count, summary)
