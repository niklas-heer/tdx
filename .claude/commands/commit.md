---
description: Create a conventional commit from staged changes.
---
Analyze staged changes and create an appropriate conventional commit.

**Steps**
1. Run `git diff --cached` to see staged changes
2. Run `git log --oneline -10` to understand commit style
3. Determine the commit type (feat, fix, chore, docs, refactor, test, etc.)
4. Identify the scope if applicable
5. Write a concise commit message following conventional commits spec
6. Create the commit

**Conventional Commit Format**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting, no code change
- `refactor`: Code restructuring without behavior change
- `test`: Adding/updating tests
- `chore`: Maintenance tasks
- `ci`: CI/CD changes
- `perf`: Performance improvements

Include breaking change footer (`BREAKING CHANGE:`) if applicable.
