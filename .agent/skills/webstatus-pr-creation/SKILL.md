---
name: webstatus-pr-creation
description: How to commit code and create a Pull Request
---

# Creating a Pull Request (PR)

When you have finished implementing a feature or fixing a bug and the user asks you to "create a PR" or "commit these changes", you MUST follow these specific steps to ensure the repository remains clean and PRs are descriptive.

1. **Verify the State**: Run `git status` to see what files have been modified.
2. **Review the Diffs**: Before committing, use `git diff` to quickly review the changes. Ensure you haven't left any stray `console.log` statements, commented-out debugger code, or unresolved merge conflict markers.
3. **Format, Lint, and Style**: Run standard project linters (`make precommit`, `make go-lint`, or `make node-lint`).
   - **Environment**: Ensure you are in the `nix develop` shell or the project's DevContainer so you use the correct tool versions.
   - **Regeneration**: If you modified OpenAPI specs, JSON schemas, or ANTLR grammars, you MUST run `make gen` before committing to ensure generated code is in sync.
   - **Style Guides**: Cross-reference your changes against the project's specific skills (e.g., `webstatus-backend`, `webstatus-frontend`) and standard Google Style Guides. If you are unsure about a style rule, you may search for it or ask the user.
4. **Create a New Branch**: NEVER commit directly to `main`.
   - Run `git checkout -b feature/<short-description>` or `git checkout -b fix/<short-description>`
5. **Stage Files**: Add the specific files you modified using `git add <file1> <file2>`. Try to avoid `git add .` unless you are absolutely certain no unrelated files (like local IDE configs) are present.
   - **Nix**: If you modified `flake.nix`, ensure you also run `nix flake update` and stage the updated `flake.lock`.
6. **Write a Descriptive Commit Message**: You MUST use the Conventional Commits format (`type(scope): subject`). Make sure to prefix the commit with `feat:`, `fix:`, `chore:`, `docs:`, `test:`, or `refactor:`.
   - _Example_: `feat: implement dark mode component`
   - _Example_: `fix(api): handle missing search name in payload`
   - _Example_: `docs: update knowledge base and skills`
7. **Commit the Changes**: Run `git commit -m "<your message>"`. You can omit the `-m` flag and let it open an editor if you prefer to write a multi-line body explaining _why_ the change was made (encouraged for complex bug fixes!).
8. **Prompt for Issue Number**: Before creating the PR, ask the user if there is an associated GitHub issue number for this work, if they haven't already provided one.
9. **Push the Branch**: Run `git push -u origin <branch-name>`
10. **Create the PR**: Use the GitHub CLI (`gh`) to create the PR.
    - Run `gh pr create --title "<Commit Subject>" --body "<Detailed Description>"`
    - **CRITICAL**: The `--body` must explain:
      - **What** was changed.
      - **Why** it was changed (context, bugs fixed).
      - **How** it was implemented (architectural decisions).
      - **Corrected Assumptions / Learnings**: Document any edge cases, architecture rules, or misconceptions discovered during the process that the AI should learn from.
      - Include the issue number provided by the user (e.g., `Fixes #123`).

> [!IMPORTANT]
> A PR with just "fixed the bug" in the body is unacceptable. Take the time to write a body that a human reviewer can immediately understand without having to read every line of code first.

## Dependent PRs (Splitting Big Changes)

When tasked with a massive feature or refactor, always aim to split the work into smaller, focused PRs that build upon each other (dependent PRs). This makes reviewing significantly easier for humans.

### Creating Dependent PRs

1. Branch off `main` for the first piece of work (`PR 1`).
2. Once the first PR is submitted for review, create your next branch _directly off of_ the first branch (e.g., `git checkout -b feature/pr2`).
3. If you need a chain of 4 PRs, branch `pr4` off of `pr3`, `pr3` off of `pr2`, etc.

### Updating Dependent PRs After Merges

When the base branch (`origin/main`) advances (e.g., after `PR 1` merges), you must rebase your dependent branches so they stack cleanly on the new `main`. Use the `git rebase --onto` strategy.

1. Fetch latest: `git fetch`
2. Determine how many commits belong to your current branch (e.g., `N=2` if `PR 2` added 2 commits on top of `PR 1`).
3. Rebase onto the new base: `git rebase --onto origin/main HEAD~<N> <BranchName>`
4. Repeat this for subsequent stacked branches, swapping out `origin/main` for the new base branch as needed.
