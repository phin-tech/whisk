@AGENTS.md

## Claude Code

- Treat `AGENTS.md` as the project source of truth.
- If an instruction references Codex-only tools or MCP graph tools that are unavailable, use the closest Claude Code/local equivalent and state the substitution.
- Follow the TDD phase protocol in `AGENTS.md`; in RED phase, write tests only and stop.
- For agentbridge/hook testing, `go tool testagent` provides a deterministic fake `claude`/`codex` CLI that fires real hook payloads without an API key. See the Testing section of `AGENTS.md`.
