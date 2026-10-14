# Security policy

## Reporting a vulnerability

Report vulnerabilities privately through
[GitHub's private vulnerability reporting](https://github.com/The127/frogly/security/advisories/new).
Don't open a public issue for a security problem.

Include enough detail to reproduce the problem: the version or commit, how
frogly is set up, and the steps. You'll get an answer as soon as possible.
There is no bug bounty.

## Supported versions

frogly has no releases yet. Only the `main` branch is supported. Once there
are releases, this section says which of them get security fixes.

## What's in scope

frogly is a planner that a team hosts for itself. Of particular interest
are problems where someone gets past what they're allowed to see or do:

- reading or changing a project you aren't a member of
- planting ideas or acting as someone else, through the web, the API, an
  email or a form
- an agent or a script doing more than its person may do
- data from a project leaking into another one, for example through "me
  too" suggestions or search
