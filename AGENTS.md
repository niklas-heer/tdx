<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->
# General conventions

## Documentation Folder Guidelines
- **`docs/` is for end-user documentation**: The docs folder is reserved for user-facing specifications, guides, and API documentation.
- **LLM content is not welcome**: Do NOT use the docs folder to store LLM-generated notes, temporary files, or work-in-progress content.
- **Use OpenSpec instead**: For internal development documentation, specifications, and change tracking, use OpenSpec.
- **Keep docs clean**: The docs folder should only contain polished, finalized content intended for end users and maintainers.

## References Folder
- **Read-only reference**: The `references/` folder is provided as documentation and context for the LLM only.
- **Do NOT modify**: Never create, edit, delete, or alter any files in the references folder.
- **Research purposes only**: Use files in this folder to understand concepts, APIs, specifications, and project context.
- **Preserve as-is**: This folder should remain untouched and serve solely as a knowledge base for reference during development.
- **Read this folder before implementation**: Read the contents of this folder before you implement any feature. (`references/`)


## LLM Repository Hygiene
- **Minimize Markdown files**: Do NOT scatter Markdown files throughout the repository.
- **Keep it bare minimum**: Only create Markdown files if absolutely necessary.
- **Single temporary file only**: If needed, maintain at most ONE temporary `.md` file (e.g., `_llm_notes.md`) for tracking tasks, notes, and work-in-progress items.
- **Prefer OpenSpec**: Use OpenSpec for change proposals, specifications, and documentation instead of creating ad-hoc Markdown files.
- **No pollution**: Resist the urge to create explanatory `.md` files, TODO lists, or documentation snippets scattered across the repo.

## Commit message format
- All commits MUST follow the [Conventional Commits](https://www.conventionalcommits.org/) specification.
- Format: `<type>[optional scope]: <description>`
- Common types:
  - `feat:` – new feature (triggers MINOR version bump)
  - `fix:` – bug fix (triggers PATCH version bump)
  - `docs:` – documentation changes
  - `test:` – add or update tests
  - `refactor:` – code change that neither fixes a bug nor adds a feature
  - `chore:` – maintenance tasks, dependency updates
  - `ci:` – CI/CD configuration changes
  - `perf:` – performance improvements
- Use `!` after the type or add `BREAKING CHANGE:` footer for breaking changes (triggers MAJOR version bump).
- Examples:
  - `feat(api): add user authentication endpoint`
  - `fix: resolve memory leak in data processor`
  - `docs: update installation instructions`
  - `feat!: remove deprecated API endpoints`

## Using Context7
- Before starting any task, check if you have access to [Context7](https://context7.com/) MCP tools.
- ALWAYS use Context7 automatically for:
  - Code generation and scaffolding
  - Setup and configuration steps
  - Library and API documentation lookups
- Use Context7 MCP tools to resolve library IDs and fetch docs without waiting for explicit requests.
- Context7 provides up-to-date documentation and examples—prefer it over outdated knowledge.
- If Context7 is unavailable, fall back to standard methods and mention it in your response.

## Development workflow
- Follow this standard workflow for all feature and fix work:
  1. **Create and switch to a new branch**: `git checkout -b <branch-name>`
  2. **Make OpenSpec change**: Create a change proposal with `/openspec` or `openspec` command
  3. **Review the change**: Validate the proposal, tasks, and spec updates using `openspec show <change>`
  4. **Implement the changes**: Execute the tasks and write the code
  5. **Commit with conventional commits**: Use proper format (e.g., `feat(api): add new endpoint`)
     - Commit packages separately if needed: `git add packages/<package_name> && git commit`
  6. **Archive the OpenSpec change**: Run `openspec archive <change>` to merge
  7. **Push the branch**: `git push origin <branch-name>` approved updates into specs
- Always keep the OpenSpec specs and implementation in sync.
- Don't commit the implementation before the spec change is reviewed and approved.
- Each package change should have its own descriptive conventional commit.

## TypeScript and Bun development
- Use Bun as the default runtime and task runner for TypeScript code in this repository.
- Prefer running scripts with `bun run <script>` instead of `node` or `ts-node`.
- Run tests with `bun test` where possible, and co-locate tests next to the implementation file (e.g., `foo.test.ts` or `foo.spec.ts`).
- Write modern, ESM-first TypeScript:
  - Use `import`/`export` syntax.
  - Avoid new CommonJS entrypoints unless interacting with legacy tooling.
- Keep TypeScript configurations strict:
  - Enable `strict` mode and related options such as `noImplicitAny`, `strictNullChecks`, and `noUnusedLocals`.
  - Fix type errors instead of relying on `any`, `unknown` without narrowing, or `// @ts-ignore`.
  - If you must suppress an error, use `// @ts-expect-error` with a short justification comment.
- Be explicit about runtime assumptions:
  - Prefer Web-standard APIs that Bun supports (`fetch`, `URL`, `Request`, `Response`, etc.).
  - Only use Node-specific APIs when the code is clearly documented as Node-only.
- Dependencies and tooling:
  - Prefer libraries that are runtime-agnostic or documented as working with Bun.
  - Avoid adding dependencies that rely heavily on Node core modules unless necessary.
  - Use `bunx` instead of `npx` when running CLIs.
- Performance and build pipeline:
  - Let Bun handle TypeScript transpilation; do not add redundant build steps if Bun already supports the workflow.
  - Do not commit compiled `.js` outputs for Bun-run TypeScript; treat `.ts` files as the source of truth.
