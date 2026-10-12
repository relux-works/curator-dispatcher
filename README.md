# swarma-dispatcher

The launch dispatcher for agent systems. It accepts launch requests from people, orchestrators and tools, decides whether this machine can take the run now, provisions a per-agent OS account and its credential binding, starts the agent's executor under that account, tracks the run to its end, and reconciles after crashes.

It works with a privileged account manager and launcher, and an unprivileged credential broker, each kept narrow:

- [swarma-user-manager](https://github.com/relux-works/swarma-user-manager) creates and retires per-agent OS accounts; its launcher starts one approved executor under one agent account;
- [swarma-credential-broker](https://github.com/relux-works/swarma-credential-broker) binds the agent's account and leases it the subscription credential its grant allows.

The one-shot executor is `curator-run` from `curator-agent-launcher`, in a versioned narrow exec mode under the agent account: it verifies and executes a caller-composed plan, never composes or resolves profiles, and has no network access except to the broker. It is the first broker consumer; the board runner and session host ask the dispatcher rather than embedding broker clients. Its intake and the Decision 0021 amendment must be published and qualified before activation. SH2+ long sessions instead use the agent-side `swarma-session-runner`, sharing the launch library; the host service owns session metadata and fences, while the runner owns PTY or app-server transport and lease renewal.

**Status: design in progress.** Nothing is implemented yet. The specification (the `swarma-dispatcher` Unix-socket service on `dispatch/1` from slice 0; later queueing, events, long waits and integrations behind capability gates on the same core and transport) is [`spec/dispatcher.md`](spec/dispatcher.md); diagrams are in [`diagrams/`](diagrams/).

## Principles

- The dispatcher runs under its own unprivileged service account; root stays in the helper and the launcher.
- A root request ID is scoped to the caller and kept for correlation; create, bind, launch, stop, unbind and retire each use a distinct durable effect ID (§4.5). Repeating a request returns the same records; retrying an effect reuses its ID.
- A caller gets only what its own grant allows; the dispatcher narrows it further for the agent.
- Capacity closures refuse initially; later queue capability adds durable queueing. Quota and authority failures always refuse and never queue. Prepare, dequeue and every start use fresh broker `revocation.check`; `revoked` and `unavailable` refuse, while final broker checks stay mandatory.
- Except for the narrowly gated pre-key registration exception (§3.3), mutating `dispatch-op/1` requests sign exactly `swarma-dispatcher request/1`, newline, then CCJ-1 bytes without `auth` (§3.3). The repository-based wire naming decision needs no migration because nothing has shipped; transport labels are distinct from signed domains.
- Accounting, admission and cleanup belong here; SH2+ holding moves to the session host module's runner under the agent account after the backend gate.
- The dispatcher, board and goals are optional to direct Curator launches. The session host must work without a board, goals, an orchestrator or its toolchain.
- Standing team allocations have no expiry or lifetime quota. Only the operator retires or re-creates them with a signed record; the sweep never expires them (§§3.5, 5.4, 7.2). Their executions have no `max_runtime` quota or runtime deadline and the sweep never stops them for runtime, per the owner decision of 2026-10-12 (R3 closed in §16). They end only by explicit stop through `exec.stop`, operator action, own exit or an incident hard stop. Ordinary allocations keep the 24 h default and 7-day maximum; every other execution keeps the 4 h default and 12 h maximum.
- The operator-only bring-up-team operation creates or reconciles one standing allocation per role from an operator-signed description, idempotently. Its request object `swarma-dispatcher team/1` is specified-but-unpublished (§5.5).
- A new orchestrator's requester signs its narrowed caller delegation after generation-bound key registration; the dispatcher verifies and records it, never signs it. Registration alone gives no spawn authority. Recursive provisioning stays refused until pre-key replay/rotation and depth preflight are published and qualified (§5.2).
- Retirement follows the user manager's quiesce, declared-key cleanup and home deletion rule. Retention is a separate explicit operator operation with no confidentiality guarantee (§5.4).
- Receiver-owned `effect.observe` answers map exactly to the dispatcher's four observation classes; missing/expired evidence stays unknown, and effect absence never authorizes an automatic new create (§§4.5–4.6, 7).
- An orchestrator composes child plans from its own Curator configuration in its agent home; the child executor only verifies them. Child profile/environment access and narrowed-grant constraints remain open (§§5.2, 16).
- For the gated SH2+ backend, the configured host principal asks the dispatcher through `host_start`, referring to a durable host admission permit bound to the session, execution, controller epoch and operation. Only the dispatcher invokes the root launcher; current-epoch runner registration and readiness precede delivery (§12). Permit formats and crash ordering remain unpublished.
- With keeper integration enabled, the dispatcher revokes only identities it registered with the one-shot flag, in a separate operation after account retirement. Standing identities remain operator-only, with a signed revocation record (§14).
- The daemon sweeps internally every minute (§7.2). Recovery and readiness precede admission; an absent or unready daemon refuses with no command fallback. Operator maintenance closes admission, inventories executions and explicitly stops selected executions through `exec.stop`; confirmed absence and reconciliation precede mode/schema/binary change. Its deadline bounds the attempt, never standing runtime; standing allocations and identities are retained (§11).
- After any crash, the dispatcher reconciles its records with the helper, the broker and the process table before it accepts new work.

## Diagrams

These PlantUML sources describe the draft service design and label later SH2+ gates; they do not claim implementation. Each figure links its defining spec sections and marks unspecified details.

| Audit row | Figures | Spec |
|---|---|---|
| M-DP1 | [Components](diagrams/components.puml), [dispatch lifecycle](diagrams/dispatch-lifecycle.puml) | [Versions](spec/dispatcher.md#2-versions), [lifecycle](spec/dispatcher.md#5-lifecycle) |
| M-DP2 | [Run states](diagrams/run-states.puml), [agent and claim states](diagrams/agent-claim-states.puml) | [Records](spec/dispatcher.md#4-records-and-state), [capacity](spec/dispatcher.md#9-capacity) |
| M-DP3 | [Storage layout](diagrams/storage-layout.puml) | [State directory](spec/dispatcher.md#41-state-directory), [socket](spec/dispatcher.md#83-socket-dispatch1) |

Companion views: [bring up team](diagrams/bring-up-team.puml) ([§5.5](spec/dispatcher.md#55-bring-up-team-specified-but-unpublished)), [reconciliation](diagrams/reconcile.puml) ([§7.1](spec/dispatcher.md#71-reconciliation)), [service daemon / gated SH2+ backend](diagrams/v1-daemon.puml) ([§12](spec/dispatcher.md#12-session-host-backend-later-activation-gate)) and [activation / maintenance](diagrams/activation.puml) ([§11](spec/dispatcher.md#11-installation-and-activation)).

## License

Apache License 2.0 — see [LICENSE](LICENSE) and [NOTICE](NOTICE).
