# curator-dispatcher: specification

- **Status:** draft v0.1 (2026-10-09). Nothing is implemented.
- **Normative language:** MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.
- **Companions:** [curator-host-helper](https://github.com/relux-works/curator-host-helper) (agent accounts; its launcher starts one approved executor under one agent account), [curator-credential-broker](https://github.com/relux-works/curator-credential-broker) (bindings and credential leases), curator-spec CIP-0011 (the Curator side), the platform architecture §7.1 (launch dispatcher), §5.1 (launch contract), §5.3 (fences and restarts), §7.2 (key keeper), §7.5 (isolation).

## 1. Purpose

The dispatcher is where a launch becomes a run. It takes the part of today's `task-board spawn` that starts runs and keeps their records, and makes it a module that works with or without a board:

1. it accepts a launch request from a person, an orchestrator, the board or a tool, and authenticates the caller by its OS account;
2. it checks the caller's authority (its grant) and this machine's capacity;
3. it provisions the agent: an OS account through curator-host-helper, a binding of that account in curator-credential-broker with a grant narrowed for the agent, and later a registration with the key keeper;
4. it starts the agent's executor under that account through the helper's launcher, and tracks the run until it ends;
5. it cleans up: unbinds, retires the account, keeps or archives what the policy says;
6. after any crash it reconciles its records with the helper, the broker and the process table.

Every launch is one durable **run record**, keyed by the caller's request id. Without a dispatcher, Curator can still launch directly, without accounting; with one, everybody launches through it: the board, orchestrators without a board, and ordinary sessions.

### 1.1 Non-goals

- It does not choose roles, models or efforts (the playbook core and the model router do), and it does not compose plans (Curator composes; the dispatcher treats a plan as opaque bytes with a digest).
- It does not hold credentials (the broker does) and does not run as root (the helper and its launcher do, one operation at a time).
- Once a session host exists, it does not hold processes either: it asks the host to hold them (§6.2).

### 1.2 Names

| Name | Meaning |
|---|---|
| dispatcher | this component, running under its own service account `cur-s-dispatch` |
| caller | the OS account that sends a request (a person's account, an orchestrator agent's account, the board's runner, the session host) |
| host launcher | `curator-host-launcher`, the privileged transition into an agent account (helper repository); not the launch client of the architecture (M4) |
| executor | the trusted process that runs under the agent account, reads the plan, obtains its credential lease and execs the harness |
| holder | the process that is the parent of a running executor: a per-run supervisor (v0, v1) or the session host (later) |
| run | one launch from acceptance to cleanup, with one record `run/1` |

## 2. Versions

| Version | Shape | Adds |
|---|---|---|
| v0 | a command, run under the dispatcher's service account through `sudo -u cur-s-dispatch`, one request per invocation; a detached supervisor per run | the core: requests, records, authority, capacity gates (refusing when full), provisioning, start, tracking, cancel, cleanup, reconciliation |
| v1 | a daemon on the same core and the same state, listening on a Unix socket with peer authentication | a queue with handles, an event stream, long waits, drain, the session-host holder backend, key keeper registration |

v0 and v1 share the core library, the record formats and the state directory. Moving to v1 means installing the daemon; clients use the socket when it exists and fall back to the v0 command otherwise. Records written by v0 are reconciled and continued by v1.

## 3. Identities and authority

### 3.1 The dispatcher's account

The dispatcher runs as the unprivileged service account `cur-s-dispatch`, created by curator-host-helper for the installer (`user.create { kind: "service" }`). Only this account is configured as a **dispatcher** in the helper policy (helper §3) and in the broker (broker §4.2), so the root transitions and the bindings are reachable only through the dispatcher, not through every program a person or an agent runs.

### 3.2 Callers

- **v0:** a sudoers rule lets configured callers run the dispatcher command as its service account with no arguments, for example `orch-a ALL=(cur-s-dispatch) NOPASSWD: /usr/local/libexec/curator-dispatcher ""`. The caller is `SUDO_UID`, set by sudo during that transition; the command refuses to run as any account but `cur-s-dispatch` (`not_dispatcher_account`).
- **v1:** the caller is the kernel-reported peer UID on the dispatcher socket (§8.2), with the connection rules of broker §4.1.
- Principals are recorded the same way as in the broker: managed accounts by `(uid, generation)` checked against the helper ledger, unmanaged accounts pinned by UID and directory identity.

### 3.3 Grants

- A caller presents its grant chain (broker `grant/1` envelopes, verified with the same library). To start an agent, the leaf must allow the action `agent.spawn` and, for every account the agent may use, `credential.lease` with remaining delegation depth of at least 1, for the requested harness and profile.
- The dispatcher holds its own signing key (a software Ed25519 key in its home, mode 0600, in v0; signing through the key keeper later) and its own grant chain from a configured root.
- For each provisioned agent, the dispatcher signs a **leaf grant** for `agent:<generation>`, depth 0, narrowed to the intersection of the request, the caller's grant and its own grant, with the run's lifetime as its window. This leaf, under the dispatcher's own chain, is what the broker verifies at bind and lease time. The caller's chain is checked by the dispatcher and recorded in the run; it is not part of the broker chain.
- A caller can never obtain for an agent more than its own grant allows, and the dispatcher can never grant more than its own grant allows.

### 3.4 Quotas

Configuration sets the maximum number of live agents per caller and in total, and the maximum lifetime of a run. Requests beyond them are refused (`quota_exceeded`).

## 4. Requests and records

### 4.1 Operations

| Operation | Meaning |
|---|---|
| `dispatch` | provision an agent, start one plan under it, clean up when it ends (the default for ephemeral agents) |
| `provision` | provision an agent and keep it, without starting anything (the agent stays until `retire` or its lifetime ends) |
| `start` | start one plan under an agent the same caller provisioned |
| `status`, `wait` | read a run; wait for a state with a bounded timeout |
| `cancel` | stop a run at whatever stage it is in, with compensation |
| `retire` | end a provisioned agent: unbind and retire its account |
| `list`, `capacity` | the caller's runs and agents; this machine's capacity |
| `reconcile` | re-check one run against the helper, the broker and the process table |

### 4.2 Request `dispatch-request/1`

```
{ v: 1,
  op:         "dispatch",
  request_id: string (≤ 64),            // idempotency key, scoped to the caller
  label:      "dev-7f3",                // display only
  profile:    "dev",                    // Curator profile of the launch
  harness:    "claude_code",
  accounts:   ["ivan/claude/personal"], // what the agent may lease
  mode:       "headless",               // v0: headless only
  plan:       base64 (≤ 256 KiB),       // composed by Curator (compose-only); opaque here
  plan_digest: "sha256:…",
  executor:   "curator-executor",       // an approved executor id (helper launcher)
  cwd:        "work",                   // relative to the agent's home
  lifetime:   { until: time, max_runtime: duration },
  cleanup:    { retire: "on-exit" | "keep", home: "archive" | "delete" },
  network:    { profile_ref, profile_digest } | null,
  parent:     { run_id } | null,        // the run that asked, for orchestrator trees
  board_ref:  { board, element, run_id } | null,   // kept, never interpreted
  grants:     [ grant envelopes, leaf first ] }
```

- Unknown members, duplicate keys and trailing documents are refused (`request_invalid`). The plan's digest is recomputed and must match (`plan_digest_mismatch`); the dispatcher never parses the plan.
- `(caller, request_id)` identifies a request. A repeat with the same canonical request returns the existing run; a repeat with a different one is refused (`request_id_conflict`). The board passes its spawn key as `request_id`.

### 4.3 Run record `run/1`

```
{ run_id:      "RUN-<yymmdd>-<random>",
  request_id, request_digest, caller: principal reference,
  op, label, profile, harness, accounts, plan_digest, executor,
  agent:       { generation, uid, username } | null,
  binding_id, leaf_grant_id, caller_grant_ids: [...],
  state:       "accepted" | "queued" (v1) | "provisioning" | "starting" | "running" | "stopping"
             | "cleaning" | "done" | "failed" | "cancelled" | "refused",
  stage:       the last completed stage (§5),
  holder:      { kind: "supervisor" | "session-host", pid, start_time, epoch } | null,
  executor_pid, exit: { code, signal, at } | null,
  cleanup:     { retire, home, state: "pending" | "unbound" | "retired" },
  refusal:     { stage, code, message } | null,
  parent, board_ref,
  created_at, updated_at, expires_at }
```

Records contain no secret material.

### 4.4 State

The state directory lives in the dispatcher's home (mode 0700): one file per run (replaced atomically), an index by `(caller, request_id)`, an append-only journal of stage transitions (flushed before each external effect), the audit log, the dispatcher key, and one lock file. Every state change and every reconciliation holds the lock. Callers read state only through the dispatcher.

## 5. Lifecycle

The stages follow the launch contract of the architecture (§5.1.2). Each stage is journaled before its external effect and confirmed after it; a stage never starts unless its predecessor is confirmed.

| Stage | What happens | On failure |
|---|---|---|
| `prepare` | schema, caller principal, grant chains, quotas, plan digest | `refused { code }`; nothing to undo |
| `capacity` | the host gates (§9) | v0: `refused { capacity:<gate> }`; v1: `queued { handle }` |
| `provision` | helper `user.create { kind: agent, label, request_id }` → generation; (helper v1) `fw.apply` for the network profile; sign the leaf grant for `agent:<generation>`; broker `bind { request_id }`; (later) key keeper `register(uid, path)` | compensation in reverse: unbind, retire the account; `failed { provision }` |
| `start` | the holder runs the helper launcher `launch.start { generation, executor, cwd, plan, request_id }`; the executor reads the plan, leases from the broker and execs the harness | compensation: stop the executor, verify no process of the account remains; `failed { start }` (the account is cleaned up per policy) |
| `running` | the holder waits; `cancel` and `max_runtime` stop the executor through the holder | — |
| `stopping` | the executor exited or was stopped; the exit code and signal are recorded | — |
| `cleaning` | broker `unbind`; helper `user.retire { home }` unless `retire: keep` | retried with backoff; the run stays in `cleaning` until confirmed |
| `done` / `failed` / `cancelled` | terminal | — |

`cancel` before `running` performs the same compensation as a failure at the current stage; a failed bind or start never leaves an orphaned account or process.

```
accepted → provisioning → starting → running → stopping → cleaning → done
    │            │             │          │                    
    └→ refused   └→ failed     └→ failed  └→ (cancel) → stopping
```

## 6. Holding processes

### 6.1 Per-run supervisor (v0 and v1)

For each started run the dispatcher starts a supervisor under its own account. The supervisor:

- starts `sudo /usr/local/libexec/curator-host-launcher` with the `launch-op/1` request on stdin, so that it is the parent of sudo, which is the parent of the executor (helper §7);
- writes the executor's standard output and error to the run's log files in the dispatcher's state (no credential is placed there by design; the harness's own output is the harness's responsibility);
- relays `cancel` as SIGTERM, then SIGKILL after a grace period, through sudo;
- records the exit and runs the cleanup stage.

In v0 the supervisor is a detached process (it survives the command that started it); in v1 the daemon runs supervisors itself. Each supervisor start increments the run's `epoch`; only the current epoch may change the run (architecture §5.3).

### 6.2 Session-host backend (v1, later)

When a session host exists, the dispatcher asks it to start and hold the executor (the host uses the same launcher) and receives exit events from it; the dispatcher keeps the run record, admission and cleanup. The host's fences (architecture §5.3) take precedence over the dispatcher's wishes: a run whose session is fenced is not started.

## 7. Reconciliation

Reconciliation runs before anything else when the dispatcher starts (v0: on every invocation, for runs not touched within a grace period; v1: at daemon start and periodically), and on `reconcile`. For each run that is not terminal, under the lock:

| Observed | Decision |
|---|---|
| provisioning journaled, no account in the helper ledger | roll back: mark `failed { provision }` |
| account created, no binding | roll back: retire the account (or bind, if the request is still within its window and capacity holds) |
| holder alive (pid and start time match) | leave it |
| holder gone, executor gone | record `exit` as unknown, go to `cleaning` |
| holder gone, processes of the agent account remain | `lost`: stop them through `user.retire` (which terminates the account's processes), then `failed { lost }` |
| `cleaning` not confirmed | resume the cleanup |

The dispatcher never runs two supervisors for one run: the lock and the holder's pid and start time decide.

## 8. Interfaces

### 8.1 v0 command

- Binary: `/usr/local/libexec/curator-dispatcher`, run as `sudo -u cur-s-dispatch /usr/local/libexec/curator-dispatcher` with one `dispatch-op/1` JSON request on stdin (at most 512 KiB); one JSON result on stdout.
- Front-ends: `curator agent-user create | run | retire | list` (a Curator provider; `create` is `provision`, `run` is `start`, and `curator agent-user run --dispatch` is `dispatch`) and a standalone `curator-dispatch` command. Both compose the request and call the binary; both switch to the v1 socket when it exists.

### 8.2 v1 daemon and socket `dispatch/1`

- The daemon `curator-dispatcherd` runs as `cur-s-dispatch` (LaunchDaemon with `UserName`, or a systemd unit with `User=`).
- Socket: a Unix-domain stream socket in its own directory, with the framing, connection rules and deadlines of broker §4.1 and §13 (4-byte length plus canonical JSON, close-on-exec connections, server identity check, one role per connection).
- Methods: the operations of §4.1, plus `subscribe` (a stream of run events for the caller's runs: state changes, exits, refusals) and `drain` (operators: stop admitting, finish running work).
- Queue: when a capacity gate is closed, `dispatch` returns `queued { handle }`; the run starts when the gate opens, in order of arrival per caller, unless cancelled or its window passes (`expired_in_queue`).
- Clients: Curator and `curator-dispatch`, orchestrators, the board's spawn path (which keeps its records as references to `RUN-*`), the session host.

## 9. Capacity

`capacity()` returns `{ slots: { used, max, by_caller }, disk: { free, floor }, load: { load5, ceiling }, hold: bool }`. The gates follow the architecture (§5.1.4):

| Gate | Rule | Version |
|---|---|---|
| `slots` | live agents below the total and the caller's quota | v0 |
| `disk` | free space on the agents' volume above a floor | v0 |
| `load` | five-minute load average below a ceiling | v0 |
| `hold` | no host-wide hold flag set by the operator or the ramp guard | v0 |
| `build_lock`, `syspolicyd` | one build at a time; no executable-creating work on a host with the syspolicyd fault flag | v1 |

The gates are checked at `capacity` and again immediately before `start`.

## 10. Security

- The dispatcher is unprivileged. Its only powers are its helper role (agents it created), its broker role (bindings for those agents), the launcher rule, and its key, which signs leaves only inside its own grant.
- Callers are identified by the kernel; requests have a closed schema; plans are opaque bytes with a digest; the dispatcher never reads Curator fragments or credentials.
- An agent that compromises its own account cannot reach the dispatcher's state, key or sudo rules; it can at most call the dispatcher as a caller, with its own (empty, unless delegated) grant.
- The audit log records each request, decision, refusal and cleanup as typed metadata.

## 11. Installation

The installer creates `cur-s-dispatch` with the helper, installs the binary (and in v1 the daemon unit), writes the sudoers rules (callers to `cur-s-dispatch`; `cur-s-dispatch` to the helper and the launcher, each with no arguments), registers `cur-s-dispatch` as the dispatcher in the helper policy and the broker configuration, generates the dispatcher key, and asks the operator to issue the dispatcher's grant (`curator broker grant --to-key <dispatcher key> … --depth 1`).

## 12. Qualification

Tests that create accounts or start processes run on hosted CI runners only:

- `dispatch` end to end with the broker's slice 0 (a Claude launch under an agent account); `provision` then `start`; idempotent repeat; `request_id` conflict;
- refusals: no grant, a grant that is too narrow or not delegable, quota, each capacity gate, plan digest mismatch, malformed requests, a caller that is not configured;
- crash after each stage, then reconciliation forward or back; cancel at each stage; `max_runtime`; a lost holder with remaining processes; two invocations racing for one run;
- v1: peer authentication, an inherited or passed connection rejected, the queue, the event stream, drain.

## 13. Delivery order

1. **v0** with the helper's v0 and launcher and the broker's slice 0: the dispatcher replaces the planned `curator agent-user` internals, and `curator agent-user` becomes its client.
2. **v1** on the same core: the daemon and socket, the queue, events and drain; then the session-host backend and key keeper registration.

## 14. Open questions

1. Interactive runs (terminal attach) in v0, or only with the session host in v1.
2. Where run logs live long-term: the dispatcher's state, the agent's archived home, or the session host.
3. Default quotas per caller and in total.
4. When the board's spawn path moves its run records to the dispatcher (coordination with the board's maintainers).

## Appendix A. Refusal codes

`not_dispatcher_account`, `caller_not_permitted`, `request_invalid`, `request_too_large`, `request_id_conflict`, `plan_digest_mismatch`, `grant_invalid`, `grant_not_delegable`, `grant_too_narrow`, `quota_exceeded`, `capacity:slots`, `capacity:disk`, `capacity:load`, `capacity:hold`, `capacity:build_lock`, `capacity:syspolicyd`, `expired_in_queue`, `provision_failed`, `bind_failed`, `start_failed`, `executor_not_approved`, `run_not_found`, `run_not_owned`, `run_terminal`, `agent_not_provisioned`, `lost`.

## Appendix B. Diagrams

PlantUML sources in [`diagrams/`](../diagrams/): `components.puml`, `dispatch-lifecycle.puml`, `run-states.puml`, `reconcile.puml`, `v1-daemon.puml`.
