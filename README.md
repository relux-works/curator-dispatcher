# swarma-dispatcher

The launch dispatcher for agent systems. It accepts launch requests from people, orchestrators and tools, decides whether this machine can take the run now, provisions a per-agent OS account and its credential binding, starts the agent's executor under that account, tracks the run to its end, and reconciles after crashes.

It works with a privileged account manager and launcher, and an unprivileged credential broker, each kept narrow:

- [swarma-user-manager](https://github.com/relux-works/swarma-user-manager) creates and retires per-agent OS accounts; its launcher starts one approved executor under one agent account;
- [swarma-credential-broker](https://github.com/relux-works/swarma-credential-broker) binds the agent's account and leases it the subscription credential its grant allows.

The one-shot executor is `curator-run` from `curator-agent-launcher`, in a versioned narrow exec mode under the agent account: it verifies and executes a caller-composed plan, never composes or resolves profiles, and has no network access except to the broker. It is the first broker consumer; the board runner and session host ask the dispatcher rather than embedding broker clients. Its intake and the Decision 0021 amendment must be published and qualified before activation. SH2+ long sessions instead use the agent-side `swarma-session-runner`, sharing the launch library; the host service owns session metadata and fences, while the runner owns PTY or app-server transport and lease renewal.

**Status: design in progress.** Nothing is implemented yet. The specification (v0: a command run under the dispatcher's own service account; v1: the same `swarma-dispatcher` binary as a daemon with a socket for orchestrators and the session host, on the same core) is [`spec/dispatcher.md`](spec/dispatcher.md); diagrams are in [`diagrams/`](diagrams/).

## Principles

- The dispatcher runs under its own unprivileged service account; root stays in the helper and the launcher.
- A root request ID is scoped to the caller and kept for correlation; create, bind, launch, stop, unbind and retire each use a distinct durable effect ID (§4.5). Repeating a request returns the same records; retrying an effect reuses its ID.
- A caller gets only what its own grant allows; the dispatcher narrows it further for the agent.
- Capacity closures refuse in v0 and queue in v1; quota and authority failures refuse in both versions and never queue.
- Mutating requests sign exactly `swarma-dispatcher request/1`, newline, then CCJ-1 bytes without `auth` (§3.3). The repository-based wire naming decision needs no migration because nothing has shipped; transport labels are distinct from signed domains.
- Accounting, admission and cleanup belong here; SH2+ holding moves to the session host module's runner under the agent account after the backend gate.
- The dispatcher, board and goals are optional to direct Curator launches. The session host must work without a board, goals, an orchestrator or its toolchain.
- With keeper integration enabled, the dispatcher revokes only identities it registered with the one-shot flag, in a separate operation after account retirement. Standing identities remain operator-only, with a signed revocation record (§14).
- The v0 timer invokes `swarma-dispatcher sweep` directly as the service account every minute; this is an internal maintenance entry, outside the public no-argument sudo request path. The v1 daemon sweeps internally (§7.2).
- After any crash, the dispatcher reconciles its records with the helper, the broker and the process table before it accepts new work.

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
