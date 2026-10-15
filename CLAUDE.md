# frogly

frogly is a task planner for self-organized teams. You host it
yourself. It keeps planning light, so the team spends its time on the work
and not on the tool.

## Checks

Run `just setup` once after you clone. It installs the git hooks and runs
every check. Run `just` alone to list all recipes.

## Go

- All Go code lives in one module. Binaries go in `cmd/<name>`, everything
  else in `internal/`.
- Every package needs a rule in `arch-go.yml` that says what it may depend
  on and why. A new package arrives together with its rule.
- Tests follow "Writing tests" in CONTRIBUTING.md.

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

## Pull requests

Work reaches `main` only through a pull request, never by pushing to
`main`. Open it from a branch named `type/topic`, wait for the checks, then
squash merge it.

## Working on a story

A story is one thing a person can do with frogly that they couldn't do
before, like "log in and see your name". One story is one pull request.

1. First, write the story as a GitHub issue: the problem, the decisions
   still open as questions, and "Done when". The pull request closes it.
2. Work test first, from the outside in. The first commit is a failing
   integration test for the whole story. Then small cycles of a failing
   test, the code that makes it pass and a cleanup make the story test
   pass bit by bit.
3. Commit when the tests pass. Each commit is one small step and passes
   `just ci`. The failing story test at the start is the only commit with a
   red test.
4. Open the pull request as a draft early, so CI runs on every push.
   CodeRabbit reviews it once it's marked ready.
5. A Sonnet subagent reviews the diff of each commit before the next one
   starts. Fixes from a review go into their own commit.
6. Anything that doesn't belong to the story goes into a new issue, not
   into the code.
7. When the story test passes, run the simplify skill and an Opus code
   review over the whole pull request, fix what they find, then mark it
   ready.

Your own notes for this machine go in `CLAUDE.local.md`, which git ignores:

@CLAUDE.local.md
