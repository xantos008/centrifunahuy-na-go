# ADR-0026: AI tooling stays project-local; no GitHub MCP / Context7 — DEC-08

- Status: **accepted** (user decision DEC-08, 2026-11-15)
- Date: 2026-11-15

## Decision
- GitHub MCP and Context7 are not installed. The `gh` CLI and the local clones of pinned sources are enough.
- Every tool, skill, MCP server and config stays in the project:
  - Go dev tools pinned in `tools/go.mod`, built into `.bin/`;
  - skills in `.claude/skills/`;
  - any future MCP config in the project `.mcp.json`, never user-level.
- Global installs are forbidden. A tool that needs one is reported to the user instead of installed.
- Every change is recorded in `docs/ai/tooling-lock.md`.

## Sources
User decision DEC-08; CLAUDE.md "Claude Code skills and MCP setup".
