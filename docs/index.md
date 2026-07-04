# Whisk

Whisk is a daemon-owned agent workspace. The desktop app and CLI are clients;
`whiskd` owns sessions, PTYs, agent processes, work items, and runtime state.

## Install

Install the macOS app from a release, or install the CLI-only archive on remote
machines.

```sh
whisk daemon run
whisk session create
```

## Agent Sessions

Whisk can launch regular shells, Claude Code, Codex, or other agents in
daemon-managed PTYs. For interactive Claude Code questions, Whisk mirrors the
question in notifications and sends the selected number back to the PTY.

## References

- [Agent Interface](agent-interface.md)
- [Browser CDP Evaluation](browser-cdp-evaluation.md)
- [Plugins](plugins.md)
- [Style Guide](STYLEGUIDE.md)
