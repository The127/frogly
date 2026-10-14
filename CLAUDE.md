# frogly

frogly is a task planner for self-organized teams. You host it
yourself. It keeps planning light, so the team spends its time on the work
and not on the tool.

## Checks

Run `just setup` once after you clone. It installs the git hooks and runs
every check. Run `just` alone to list all recipes.

## Commits

- Use Conventional Commits without a scope: `type: description`. The types
  are `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `perf`, `build`,
  `ci` and `revert`.
- Sign off every commit with `git commit -s`.
- If an AI tool helped write a commit, add an `Assisted-by` line with the
  tool and the exact model:

  ```
  Assisted-by: Claude:claude-opus-5-5
  ```

  Don't add a `Co-Authored-By` line for an AI, and don't link to an AI
  session.
- Keep each commit to one change.

Your own notes for this machine go in `CLAUDE.local.md`, which git ignores:

@CLAUDE.local.md
