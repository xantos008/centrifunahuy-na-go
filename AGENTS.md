# Cedrazeka repository instructions

Use [CLAUDE.md](CLAUDE.md) for repository workflow and [SPEC.md](SPEC.md) for product requirements. Use the [feature matrix implementation order](docs/compatibility/feature-matrix.md#implementation-order) for current phase numbering and the corresponding phase plan for acceptance criteria.

Project skills are maintained in `.claude/skills` and exposed to Codex through `.agents/skills`. Read the relevant skill before changing a wire contract, test infrastructure, or product subsystem.

## Subagents and delegation

GPT-6.2 Astra with high reasoning leads the work. It handles complex analysis, planning, investigation of subtle bugs, and authors new or complex code. It owns integration and final review.

Delegate routine implementation following an established pattern, test execution, and small edits to GPT-6.2 Sol with medium reasoning. Give each subagent a bounded assignment, the relevant files, and acceptance checks. Keep simultaneous edits and test stacks isolated. Have the subagent return changed paths, commands, results, and unexpected observations. Bring unexplained failures back to Astra for diagnosis.

## Local Git and handoff

Local branches, commits and merges are authorized. Integrate each verified stage into local `main` and start the next phase branch from the updated `main`. Check branch ancestry before merging or branching. Track phase acceptance criteria separately from local merge status.

The user handles pushes and remote pull requests. If a later phase branch already descends from an earlier phase branch, preserve its commits and specify the order in which the PRs should reach remote `main`. Give each proposed or completed commit a descriptive subject, a body explaining behavior and verification, and the exact paths included. Provide a PR title, target and source branches, summary and test results when a change is ready to publish.
