# swarma-dispatcher: specification

- **Status:** draft v0.3 (2026-10-11). Nothing is implemented.
- **Normative language:** MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.
- **Companions:** [swarma-user-manager](https://github.com/relux-works/swarma-user-manager) (agent accounts, the launcher, the stop operation), [swarma-credential-broker](https://github.com/relux-works/swarma-credential-broker) (bindings, caller-authorization dependencies, leases), curator-spec CIP-0011 (the Curator side), the platform architecture §7.1 (launch dispatcher), §5.1 (launch contract), §5.3 (fences and restarts), §7.2 (key keeper), §7.5 (isolation).

### Revision 0.3

This documentation revision aligns R-DP1 with the [platform contract table](https://github.com/relux-works/curator/blob/177cb554/.research/261010_platform-contract-table.md), the [docs audit](https://github.com/relux-works/curator/blob/48789217/.research/261010_platform-docs-audit.md) and the owner decisions of 2026-10-10/11. It separates root correlation from effect IDs, v0 capacity refusal from v1 queueing, and one-shot execution from session holding; it records the narrow identity cleanup authority and the remaining owner questions (§16). These are design contracts, not claims of implementation or qualification.

### Revision 0.2

An architecture review on 2026-10-09 (0 P0, 15 P1, 3 P2) kept the module split and asked for a closed contract. This revision:

- binds a caller's grant to the caller's authenticated principal and request signature, and keeps the caller's authorization as a dependency the broker re-checks on every lease, so revoking a caller cuts its agents at once (§3; owner decision 2026-10-10);
- freezes the grant vocabulary and the constructible chain shapes (§3.4);
- derives a distinct, durable effect id for every downstream call (§4.5);
- stores immutable inputs and effect intents and results, distinguishes known, absent, unknown and contradictory observations, and keeps a run non-terminal until its cleanup is confirmed (§4, §7);
- separates **agent allocations** (`agent/1`) from **executions** (`run/1`), so `provision` and repeated `start` have a lifecycle (§5);
- replaces stop-by-signal-through-sudo with an execution claim, a launcher start receipt and the helper's authorised `exec.stop` (§6);
- uses owned capacity claims and the architecture's host gates (§9);
- publishes one payload budget and complete schemas for the v0 command and the v1 socket (§8, §10);
- defines the v0→v1 activation as an operator transaction and names what later integrations (session host, board, key keeper) must define before they are enabled (§11, §14).

## 1. Purpose

The dispatcher is where a launch becomes an execution. It takes from today's `task-board spawn` the part that starts executions and keeps their records, and makes it a module that works with or without a board:

1. it accepts a request from a person, an orchestrator, the board or a tool, authenticates the caller by its OS account and its request signature, and checks the caller's grant;
2. it checks and reserves this machine's capacity;
3. it provisions an agent: an OS account through swarma-user-manager, a binding in swarma-credential-broker with a leaf grant narrowed for the agent and the caller's authorization recorded as a dependency;
4. it starts the agent's executor under that account through the helper's launcher, tracks the execution, and stops it through the helper when asked;
5. it cleans up per policy, and after any crash reconciles its records with the helper, the broker and the process evidence before accepting new work.

Without a dispatcher, Curator can still launch directly, without accounting; with one, everybody launches through it. The board and its toolchain are optional clients, never dependencies of this module.

### 1.1 Non-goals

- It does not choose roles, models or efforts, and does not compose plans; a plan is opaque bytes with a digest.
- It does not hold credentials and does not run as root.
- With a session host present, it does not hold processes (§12).
- It does not decide when board work is accepted; the board's own completion barrier stays with the board (§14).

### 1.2 Names

| Name | Meaning |
|---|---|
| dispatcher | this component, under its service account `swarma-dispatcher` (names follow the platform root `/opt/swarma`) |
| caller | the principal that sends a request: a person's account, an orchestrator agent, the board's runner, the session host |
| agent allocation | one provisioned agent account generation and what the dispatcher keeps about it (`agent/1`) |
| execution | one start of one plan under one allocation (`run/1`) |
| host launcher | `swarma-user-launcher`, the privileged transition into an agent account (helper repository) |
| executor | `curator-run` from `curator-agent-launcher`, in its versioned, narrow one-shot exec mode under the agent account: verifies and executes a plan composed by the caller, prepares the working directory, leases its credential and execs the harness; it never composes or resolves profiles in this mode (§5.3). Long-lived sessions use `swarma-session-runner` under the agent account instead (SH2+, §12). |
| supervisor | the dispatcher-owned process that holds one execution (v0 detached, v1 inside the daemon) |

## 2. Versions

Figure M-DP1: [process and component boundaries](../diagrams/components.puml).

| Version | Shape | Adds |
|---|---|---|
| v0 | the `swarma-dispatcher` binary run under its service account through `sudo -u`, one request per invocation; detached supervisors; an internal periodic sweep; temporary capacity closure refuses | the core: requests, allocations, executions, authority, capacity claims, provisioning, start, stop, cleanup, reconciliation |
| v1 | the same `swarma-dispatcher` binary in daemon mode on the same core, a Unix socket with peer authentication; temporary capacity closure queues | queue with handles, events, long waits, drain; then the session-host backend and key keeper registration, each behind its activation gate (§14) |

Quotas and authority failures refuse in both versions; neither is queueable (§3.5, §9).

The installation mode (`command` or `daemon`) is recorded in the dispatcher's configuration and in its state; it changes only through the activation transaction of §11.

## 3. Identities and authority

### 3.1 The dispatcher's account

The dispatcher runs as the unprivileged service account `swarma-dispatcher`, created by the helper for the installer. Only this account is a **dispatcher** in the helper policy and in the broker, so the root transitions and bindings are reachable only through it. The dispatcher is a trusted principal for its scope: a compromised dispatcher can sign leaves within its own grant and administer the agents it created, but cannot exceed the broker-verified root bounds or obtain a general root shell from the helper's closed operations.

### 3.2 Callers

- **v0:** a sudoers rule lets each configured caller run the dispatcher command as `swarma-dispatcher` with no arguments (`<caller> ALL=(swarma-dispatcher) NOPASSWD: /opt/swarma/libexec/swarma-dispatcher ""`). The command refuses unless: it runs as `swarma-dispatcher` (`not_dispatcher_account`); `SUDO_UID` is present and decimal (`not_invoked_by_sudo`); its executable and every parent directory are root-owned and not writable by others; no argument is given; it reads its state root from its configuration, never from an inherited `HOME`; the environment and descriptors are reset; and the request is read within bounded size and time (§10) **before** the state lock is taken.
- **v1:** the caller is the kernel-reported peer UID on the socket, with the connection rules of broker §4.1.
- Principals follow broker §4.2: managed callers by `(uid, generation)` with the helper ledger showing the generation `active` (re-checked for every operation, every queued start and every reconciliation); unmanaged callers pinned by UID and directory identity. An unreadable or invalid ledger refuses; a recycled UID never inherits old allocations or executions.

### 3.3 Caller keys and request signatures

- The dispatcher's configuration (owned by its account, written by the operator through the installer) maps each **principal** to its **caller keys** (Ed25519). An orchestrator agent's key is registered when the dispatcher provisions it with the `orchestrator` capability (§5.2): the executor generates the key in the agent's home on first start and registers the public key with the dispatcher over an authenticated call from that agent's UID; the dispatcher records it against the allocation's `(uid, generation)`.
- Every mutating request carries `auth: { key, sig }`: an Ed25519 signature by a caller key over the domain tag `swarma-dispatcher request/1`, a newline and the CCJ-1 bytes of the request without `auth`. The key must be associated with the authenticated principal (`caller_key_not_associated`), and the leaf of the caller's grant chain must have that key as its subject (`grant_subject_mismatch`).
- **Wire naming (D-WIRE-NAMES).** Versioned wire identifiers use the repository name: `swarma-<component> <object>/<n>`. The signed request domain above is exactly `swarma-dispatcher request/1`; the request-signature newline does not apply to the effect hash (§4.5). Broker signed domains are `swarma-credential-broker grant/1` and `swarma-credential-broker revocation/1`. Nothing has shipped, so these clean names require no migration or legacy-domain acceptance. This decision supersedes the contract table's W2 retention baseline; it does not publish the helper v2 or launch-ticket signature grammar (§16). The existing dispatcher transport/schema labels `dispatch-op/1` and `dispatch/1` remain distinct from the signed domain.
- A valid chain presented without the matching principal and signature is refused; a copied chain is useless to another caller.

### 3.4 Grants and their constructible shapes

- **Action vocabulary (frozen for v0):** `agent.spawn` (may ask the dispatcher to provision and start agents within the grant's accounts, harnesses and profiles) and `credential.lease` (may lease those accounts). Unknown actions refuse (`grant_unsupported`).
- **Caller chain:** the caller's leaf grant names the caller key, allows `agent.spawn` and `credential.lease`, and has remaining depth ≥ 1. It chains to a configured root under the broker's verifier (broker §6.2), at most 4 envelopes.
- **Dispatcher chain:** the dispatcher's own grant names the dispatcher key, allows `credential.lease`, has depth ≥ 1, and is at most 3 envelopes long, so an agent leaf can be appended within the 4-envelope bound. The installer refuses to configure a dispatcher chain outside this shape.
- **Agent leaf:** signed by the dispatcher key under its own chain; subject `agent:<generation>`, depth 0, action `credential.lease` only; accounts, harnesses, profiles and window are the intersection of the request, the caller's grant and the dispatcher's grant. A request whose intersection is empty or whose leaf could not be constructed is refused before any account is created (`grant_unconstructible`).
- **Caller dependency:** the binding carries the caller's chain and the exact intersection (broker §7, revision 0.3). The broker verifies both chains, separately, at bind, lease and every renewal, against the current revocation state: revoking the caller's grant or key stops new leases and renewals for every agent provisioned under it immediately. Tokens already delivered remain valid until they expire or are revoked at the vendor.
- **Delayed and recovered work** (a queued request, a start after `provision`, a resumed cleanup that would start anything) re-checks the caller principal, signature validity window, caller chain and revocation state before acting.

### 3.5 Quotas

Configuration sets, per caller and in total: live allocations, concurrent executions, the maximum allocation lifetime and the maximum execution runtime. Requests beyond them are refused (`quota_exceeded`) — a permanent refusal, never a queueable condition.

Defaults (owner decision 2026-10-10), all configurable:

| Limit | Per caller | Per node |
|---|---:|---:|
| concurrent executions | 2 | 6 |
| live allocations | 4 | 16 |
| allocation lifetime | default 24 h, maximum 7 days | — |
| execution runtime | default 4 h, maximum 12 h | — |

The per-caller execution limit matches the host's existing rule of two runs per orchestrator.

## 4. Records and state

Figures M-DP2: [execution states](../diagrams/run-states.puml) and [allocation and claim states](../diagrams/agent-claim-states.puml); M-DP3: [storage ownership and durable ordering](../diagrams/storage-layout.puml).

### 4.1 State directory

The state lives under `/opt/swarma/services/swarma-dispatcher/state` (mode 0700): the immutable inputs, the record files, the effect table, the journal, the index, the claims, the dispatcher key and the lock. Every state change and every reconciliation holds the lock. Record files are replaced atomically (temporary file, flush, rename, directory flush); the journal is appended and flushed before any external effect; on start-up the journal is replayed over the record files, so a crash between the two leaves no ambiguity.

### 4.2 Immutable input `input/1`

Before a request is accepted, its caller principal and root `request_id`, its canonical bytes, the decoded plan bytes, the full caller chain and the computed digests are written once and never changed. Every later step, including a queued start or a recovery, reads from this object.

### 4.3 Agent allocation `agent/1`

```
{ v: 1, kind: "agent/1",
  agent_id:   "AG-<yymmdd>-<random>",
  owner:      principal reference, request input id,
  generation, uid, username, label,
  capabilities: ["orchestrator"]?,          // §5.2
  binding_id, leaf_grant_id, caller_chain_ids,
  caller_key?:  registered orchestrator key (§3.3),
  state:      "provisioning" | "idle" | "running" | "retiring" | "removed" | "failed",
  active_execution: run_id | null,          // v0: at most one
  expires_at, cleanup: { home: "archive" | "delete" },
  created_at, updated_at }
```

When keeper integration is activated, the allocation must retain the registration reference and the one-shot flag recorded at registration (§14); the exact keeper API/schema is an activation prerequisite.

### 4.4 Execution `run/1`

```
{ v: 1, kind: "run/1",
  run_id:     "RUN-<yymmdd>-<random>",
  agent_id, owner, input id, plan_digest, executor,
  claim:      { execution_id, epoch, state: "claimed" | "launching" | "started" | "stopping" | "absent" },
  receipt:    { pid, start_time, boot_id } | null,   // from the launcher, §6
  state:      "accepted" | "queued" | "starting" | "running" | "stopping" | "cleaning"
            | "done" | "failed" | "cancelled" | "expired",
  outcome:    { kind: "exit" | "failed" | "cancelled" | "expired" | "lost", exit_code?, signal?, stage?, code? } | null,
  deadline:   { max_runtime, ends_at },
  parent, board_ref (opaque, kept whole),
  created_at, updated_at }
```

### 4.5 Effect ids and the effect table

| Identifier | Meaning and owner |
|---|---|
| root `request_id` + caller principal | caller-scoped request correlation and repeat lookup; kept in immutable input, journal and audit metadata, not reused as every receiver's idempotency key |
| effect id | dispatcher-derived idempotency key for one downstream operation; retries reuse it, different effects do not |
| `agent_id` | allocation handle (`agent/1`), distinct from account generation |
| `run_id` | dispatcher execution handle (`run/1`), distinct from board run/attempt handles |
| `execution_id` | execution claim and launcher/stop receipt identity; its exact construction from `run_id` is not frozen here |
| account `generation`, broker `binding_id` | never-reused account generation and binding identity; UID or label alone is not authority |
| broker `launch_id` | correlation only (plan digest or run ID), never authorization |
| ticket `epoch`, backend `controller_epoch`, `op_id` | ticket claim epoch, host controller fence epoch and backend operation identity; none is an account generation (§12) |

Every downstream call has an **effect id**: the first 32 hex characters of SHA-256 over the domain tag `swarma-dispatcher effect/1` and the CCJ-1 bytes of `{ caller principal, request_id, record id, component, operation, attempt }`. Its inputs are journaled before the call; a retry of the same effect reuses its id; create, bind, launch, stop, unbind and retire use distinct ids. The effect table records, per effect: id, component, operation, intent time, result (`ok` with the returned identity, `refused` with its code, or `unknown`), and the last observation. The root caller correlation is retained alongside the effect's input reference for tracing. Two callers using the same `request_id` never collide downstream. No newline is added between the effect domain and its CCJ-1 bytes.

For one `dispatch` root request `R` by principal `C`, the following are symbolic IDs derived by the formula above, not literal IDs or new downstream members:

| Call | Receiver idempotency key | Root correlation kept by dispatcher |
|---|---|---|
| helper `user.create` | `request_id: E-create` | `(C, R)` |
| broker `bind` | `request_id: E-bind` | `(C, R)` |
| launcher `launch.start` | `request_id: E-launch` | `(C, R)` |
| helper `exec.stop` | `request_id: E-stop` | `(C, R)` |
| broker `unbind` | `request_id: E-unbind` | `(C, R)` |
| helper `user.retire` | `request_id: E-retire` | `(C, R)` |

Each `E-*` uses its record, component, operation and journaled attempt; a retry replays the same intent and ID. The root `R` is never substituted for an `E-*`. Separate `provision` and `start` commands have separate root request IDs; their effects remain correlated to their own originating request and the shared allocation.

### 4.6 Observations

Reconciliation and recovery classify every observation of the helper ledger, the broker binding, the claim and the process evidence as **known present**, **known absent**, **unknown** (unreadable, timed out) or **contradictory**. Unknown and contradictory observations never authorise forward execution, a second launch or a successful cleanup; the record stays where it is and the observation is retried.

## 5. Lifecycle

Figure M-DP1: [provision, launch, refusal and recovery sequence](../diagrams/dispatch-lifecycle.puml).

### 5.1 Allocation

`provision` (or the first half of `dispatch`):

1. **prepare** — schema, principal, signature, chains, quotas, constructibility, plan digest; write `input/1`. Refusal ends here with no effect.
2. **capacity** — take an owned claim (§9) or refuse (v0) / queue (v1).
3. **provision** — helper `user.create { kind: agent, label, request_id: <effect id> }` → generation; (helper v1) `fw.apply`; sign the leaf; broker `bind { uid, generation, lease chain, caller chain, intersection, request_id: <effect id> }`. The allocation becomes `idle`.

On failure the allocation goes to `retiring`: unbind (if bound) and `user.retire` (if created), each as its own effect, retried until confirmed; the allocation becomes `failed` only after those effects and any applicable identity cleanup (§14) are confirmed.

### 5.2 Orchestrator allocations

The home-generated caller key is distinct from a keeper-retained identity key; broker v0 software roots are a third key class. Keeper adoption requires the versioned broker migration adapter (§14), not relabelling a caller key as a keeper identity.

An allocation with the `orchestrator` capability may itself call the dispatcher as a caller. Its caller chain is a grant the dispatcher issues to the agent's registered key (§3.3), narrowed from the requesting caller's grant, with depth one less. The dispatcher refuses a start of an orchestrator allocation whose key is not yet registered (`caller_key_pending`) except the first start, which registers it.

### 5.3 Execution

`start` (or the second half of `dispatch`), on an `idle` allocation owned by the caller:

1. re-check the caller's authority and the allocation's lifetime; take a capacity claim for one execution;
2. **claim** — under the lock, write an execution claim `{ execution_id, epoch }` and mark the allocation `running`; only one claim per allocation in v0 (`agent_busy`);
3. **supervisor** — start the supervisor; it registers its own process identity and acknowledges durably before anything else;
4. **launch** — the supervisor calls the launcher (`launch.start { request_id: <launch effect id>, generation, execution_id, epoch, executor, plan }`); the launcher writes a **start receipt** for that execution (pid, start time, boot id) while still root, before clearing groups, setting GID/UID and exec (helper §7); only the immutable plan is passed on FD3 and no broker connection is inherited; the supervisor records the receipt;
5. **running** — `curator-run` in its narrow exec mode verifies the caller-composed plan, prepares its working directory in the agent's home, resolves the protected credential binding, opens its own broker connection, leases its credential and execs the harness; `max_runtime` and `cancel` lead to **stopping**;
6. **stopping** — the supervisor (or recovery) calls helper `exec.stop { request_id: <stop effect id>, generation, execution_id }`, which terminates the execution's process session and returns a termination receipt only after it has verified that no process of that execution remains;
7. **finish** — record the outcome; the allocation returns to `idle` (kept agents) or proceeds to `retiring` (`dispatch`, or an expired allocation).

**One-shot intake gate (D-EXECUTOR).** `curator-run` under the agent account is the first broker consumer. The board runner and session host ask the dispatcher; neither embeds a broker client. This mode never composes or resolves profiles and has no network access except to the broker; harness egress is governed separately by the admitted network policy. It shares one launch library with `swarma-session-runner`. A dedicated entry such as `curator-run exec --plan-fd N` illustrates the mode, not a published CLI grammar. The launcher owner (R-LR1) publishes the versioned Decision 0021 amendment and intake contract (immutable plan verification, FD transport, credential injection capability, signals, exit/receipts and typed refusals). This launch path stays disabled until that contract is published and qualified; the older interactive scope is not evidence of one-shot support. A separate minimal executor binary is a later option, not the current target.

The managed plan records the resolved PATH and covers it with its digest (D-PATH): protected launch tool directory (CIP-0007), then admitted base PATH in its existing order, then one protected manager dispatcher directory. Project commands pass that dispatcher's checks; a repository `.agents/bin` is never executed in a managed launch. The exact plan schema is owned with the intake contract.

A cancel is serialised with the claim: before the launch, it marks the claim `absent` and the launcher is never called; during the launch, the supervisor waits for the receipt or the launcher's refusal and then stops; after the launch, it stops. A request never issues a second launch for an execution whose state is unknown.

### 5.4 Retirement and expiry

`retire`, an expired allocation, or a finished `dispatch`: unbind, then `user.retire { home }` (helper archive policy), each a confirmed effect. With keeper integration active, the dispatcher then revokes an identity only if its registration recorded it as one-shot (§14), as a separate, confirmed effect after account retirement. Standing identities are left to the operator. The allocation becomes `removed` only after its applicable cleanup effects are confirmed; executions become terminal only after their stop is confirmed and their allocation's cleanup obligations are recorded. The periodic sweep (§7.2) owns lifetimes while no request is running. Dispatcher retirement requests admit only `archive` and `delete`; `keep` is not dispatcher authority, and the helper's conflicting request/policy wording remains an explicit owner question (§16).

## 6. Holding and stopping

- **Supervisor (v0 and v1).** A per-execution process under `swarma-dispatcher`: new session, no controlling terminal, caller descriptors closed, bounded private logs of the executor's output in the dispatcher state, durable registration before the launch, and the start and termination receipts recorded. In v1 the daemon runs the supervisors itself.
- **Stop.** Signals are not relayed through sudo (sudo cannot forward SIGKILL and may not stay as a parent). The only stop is the helper's `exec.stop`, authorised for the dispatcher's own generations; ephemeral executions may also end through `user.retire`, which stops the account's processes before removal. The qualification distinguishes: the caller's command killed, its terminal closed, sudo killed, an executor that ignores SIGTERM, and descendants that escaped the session.

## 7. Reconciliation and the sweep

### 7.1 Reconciliation

Companion figure: [state-by-state reconciliation](../diagrams/reconcile.puml).

Runs at every v0 invocation before the request is read into state, at v1 daemon start and periodically, and on `reconcile`. For every allocation and execution that is not terminal, it compares the journal and effect table with the observations (§4.6), state by state:

| State | Observation | Action |
|---|---|---|
| provisioning | create effect unknown | query the helper ledger by effect id; never create again until known |
| provisioning | account present, bind effect unknown | query the broker by effect id; bind if known absent and the request is still valid, else retire |
| idle | binding known absent, allocation not expired | rebind with fresh authority checks, or retire if authority is gone |
| running, claim `claimed` | no supervisor registration | mark the claim `absent`; no launch happens later |
| running, claim `launching` | receipt unknown | keep waiting for the receipt or the launcher's answer; never relaunch |
| running, `started` | supervisor known absent | `exec.stop`, then record `lost` |
| stopping | termination receipt missing | repeat `exec.stop` (same effect id) |
| retiring / cleaning | effects pending | repeat them; terminal only when all are confirmed |

### 7.2 The sweep

A timer (launchd job or systemd timer as `swarma-dispatcher`) runs the internal service-only CLI entry `swarma-dispatcher sweep` every minute in v0, directly under that service account rather than through the caller sudo entry: reconciliation, lifetimes, claim expiry and log rotation. The internal entry requires the dispatcher service account and recorded command mode. It is outside the closed public `dispatch-op/1` operation table; callers still use the no-argument sudo entry, and their `reconcile` is caller-scoped. A public sweep would need a separate versioned authorization decision. In v1 the daemon does the same internally. While neither runs, lifetimes are not enforced; the status of every allocation says so.

## 8. Operations, results and authorisation

### 8.1 Request `dispatch-op/1`

A discriminated union on `op`; every operation has a closed set of members, and members of other operations are refused (`request_invalid`). Times are RFC 3339 UTC; durations are integers in seconds; lists have bounds.

| `op` | Members | Allowed to |
|---|---|---|
| `provision` | `request_id`, `label`, `profile`, `harness`, `accounts`, `capabilities?`, `lifetime`, `cleanup`, `network?`, `parent?`, `grants`, `auth` | callers with `agent.spawn` |
| `start` | `request_id`, `agent_id`, `plan`, `plan_digest`, `executor`, `max_runtime`, `board_ref?`, `auth` | the allocation's owner |
| `dispatch` | the union of `provision` and `start` members except `agent_id`, plus `cleanup.retire = "on-exit"` | callers with `agent.spawn` |
| `cancel` | `request_id`, `run_id`, `auth` | the execution's owner |
| `retire` | `request_id`, `agent_id`, `auth` | the allocation's owner |
| `status` | `run_id` or `agent_id` | the owner |
| `wait` | `run_id`, `until_state`, `timeout ≤ 30 s` (v0), `≤ 600 s` (v1) | the owner |
| `list` | `kind`, `cursor?`, `limit ≤ 200` | own records |
| `capacity` | — | any configured caller |
| `reconcile` | `run_id` or `agent_id` | the owner, or operators for any |
| `drain` (v1) | `mode: "park" \| "reject"` | operators |
| `register_key` | `public_key` | an orchestrator allocation's own UID (§3.3) |

### 8.2 Results

`dispatch-result/1`: `{ v: 1, request_id, ok: true, result: { … record … } }` or `{ v: 1, request_id, ok: false, error: { code, message, stage? } }`. A repeat of a completed request returns the stored result with the record's **current** state. The meaning of terminal outcomes is fixed: `exit` (the harness ended), `failed` (a stage failed and cleanup is confirmed), `cancelled`, `expired` (lifetime or runtime), `lost` (the holder vanished and the execution was stopped and confirmed absent).

### 8.3 v1 socket `dispatch/1`

Companion figure: [v1 daemon and gated SH2+ backend](../diagrams/v1-daemon.puml).

The daemon `swarma-dispatcher` runs as `swarma-dispatcher` and listens in `/opt/swarma/run/dispatcher/`. It uses the framing and connection rules of broker §4.1 and §13 with one declared difference: frames up to 128 KiB (§10). Clients use the transport the installation mode names; an authentication, permission, version or malformed-response failure never makes a client fall back to the other transport. After an ambiguous reply the client repeats the same request with the same `request_id` and signature.

### 8.4 Relation to the platform launch contract

The dispatcher implements a **local headless subset** of `dispatch(LaunchRequest)` (architecture §7.1, §5.1): it starts already composed plans; it records the opaque provenance (`board_ref`, `parent`) whole; it refuses interactive transport, native resume, seeded continuity and remote targets (`unsupported_combination`). Interactive executions under agent accounts arrive with the session host's platform phase (its backend gate, §12), not in this dispatcher (owner decision 2026-10-10); goals are optional. For now only the launching caller at start (through the launcher), or the board/orchestrator for board goals, authors a session goal (D-GOALS); an agent account has no goal capability on the session-host socket. This keeps authorship with the initiating caller while a separate agent namespace and its authority bounds await SH2+ design. The host must function without goals, a board, an orchestrator or their toolchain; board goals are an optional client of a generic goal interface. Choosing self-goals in a separate namespace versus caller/board-only authorship remains deferred (§16). A versioned LCH adapter, owned with the launch client, composes plans, compares pins and maps `LaunchRequest` onto these operations; until it exists, nothing claims full LCH conformance.

## 9. Capacity

Figure M-DP2: [capacity ownership alongside allocation and execution claims](../diagrams/agent-claim-states.puml).

| Admission condition | v0 | v1 |
|---|---|---|
| temporary capacity gate closed | refuse with `capacity:*` | queue with a durable handle; re-check authority and gates before starting |
| quota exceeded | refuse `quota_exceeded` | refuse `quota_exceeded`; never queue |
| authority invalid | refuse | refuse; never queue |

- **Claims.** A capacity claim is a durable record `{ claim_id, owner record, class, resources, expires_at }`, taken at `capacity`, transferred to the execution at `start`, released after confirmed end/absence, including cancel, expiry and reconciliation; unknown process evidence never releases it. Re-checks before the launch exclude the claim's own resources, so an admitted request never refuses itself.
- **Counters.** Allocations and executions are counted separately, per caller and in total.
- **Gates** (architecture §5.1.4): `slots`; `disk` on every volume the run touches (agents root, dispatcher state, broker store); `cpu_idle` over a window; `hold` (the host-wide hold flag, read fail-closed: unreadable means held); `build_lock` and `syspolicyd` for job classes that build or create executables. v0 admits the job classes `agent-headless` and `agent-headless-build`; the second requires the build-lock and syspolicyd gates, which read the host's flags.
- **Queue (v1).** Temporary gate closures queue a request with a handle; permanent refusals (quota, authority) never queue. The queue is bounded per caller and in total, fair across callers (round-robin by caller), durable, and re-checks authority and every gate at dequeue. `drain` either parks queued requests or rejects them, as the operator chooses.

## 10. Payload budget

| Item | Bound |
|---|---|
| decoded plan | 48 KiB |
| plan in a request (base64) | 64 KiB |
| `dispatch-op/1` request (v0 stdin) | 96 KiB, read within 5 s |
| launcher request `launch-op/1` | 80 KiB (helper §7) |
| `dispatch/1` frame (v1) | 128 KiB |

A request whose plan exceeds the smallest bound on its path is refused at `prepare`, before any account exists (`payload_too_large`). Exact-bound and one-over vectors, counted after base64 and UTF-8 encoding, are part of the qualification.

## 11. Installation and activation

- **Installation.** The installer creates `swarma-dispatcher` with the helper; places the binary in `/opt/swarma/libexec/`; writes the sudoers rules (callers to `swarma-dispatcher`; `swarma-dispatcher` to the helper and the launcher, each with no arguments); registers `swarma-dispatcher` in the helper policy and the broker configuration; generates the dispatcher key; asks the operator to issue the dispatcher's grant in the constructible shape (§3.4); installs the sweep timer; records the installation mode `command`.
- **Activation of v1** is an operator transaction: stop admission; take exclusive ownership of the state; check and migrate the state version; let v0 supervisors finish their executions (the first activation drains, it does not adopt live supervisors); reconcile; switch the recorded mode to `daemon` and publish readiness. Every v0 mutation and every v0 supervisor checks the recorded mode and backend generation and stops when they change. Rollback reverses the same steps; a v0 binary refuses state written by a newer version.

## 12. Session-host backend (later; activation gate)

Companion figure: [backend process identities, ticket refusals and recovery](../diagrams/v1-daemon.puml).

Before the SH2+ session-host module may hold executions through its agent-side runner (not the service UID):

- **Launch ticket.** The dispatcher issues a signed ticket `{ generation, execution_id, epoch, executor, plan_digest, expires_at }`; the launcher, under an explicit host role in the helper policy, accepts a launch from the host only with a valid, unused ticket for a generation the dispatcher owns. Wrong host principal, reuse, other generation, executor or plan, and revoked claims are refused.
- **Fences and epochs.** The backend API carries `(logical_session_id, execution_id, controller_epoch, op_id)`; the host owns the durable per-session fences (architecture §5.3) and validates the fence and the current epoch atomically with each effect; the dispatcher never infers controller authority from a live pid.
- **Ownership table.** Admission, records and cleanup stay with the dispatcher; the host service owns durable session metadata, logs, receipts, fences and input admission; `swarma-session-runner` under the agent UID owns the PTY master or per-session app-server transport, stays alive for the execution, and owns credential lease renewal; the handoff of each is specified before activation. `curator-run` remains the first broker consumer; board and host clients request launches through the dispatcher. The runner integration is a later qualified consumer, not an embedded broker client in the host service.
- **Ticket publication gate.** D-WIRE-NAMES fixes the component naming rule, not the ticket signature domain/version, canonical bytes or host-role schema. These require the joint owner decision in §16; this backend remains disabled until they are published.
- **Native resume (D-RESUME; later backend only).** On the same machine, a non-breaking harness version change is allowed by semver: same major, and for 0.x the same minor, without an extra qualification row. A different major or 0.x minor refuses. Retain the same runtime family, managed home, profile pin, transcript integrity, exclusive native-session lock, ready authentication and current grants. A harness resume failure is reported as `resume-failed` for the caller to decide; no automatic fallback or portable seed is part of resume. A per-harness note records vendors known not to follow semver. This does not enable resume in the current local headless subset (§8.4).

## 13. Security

- Unprivileged; its powers are its helper role (its own agents), its broker role (their bindings), the launcher rule, the stop operation, and its key, which signs leaves only inside its own grant.
- Callers are authenticated by the kernel and by request signatures; plans are opaque; the dispatcher never reads Curator fragments or credentials.
- An agent that compromises its own account cannot reach the dispatcher's state, key or sudo rules; it can call the dispatcher only as a caller with its own grant.
- Logs: the executor's output is stored with size bounds in the dispatcher state; the qualification scans these logs, the executor and broker artifacts for credential material; the trusted boundary is the executor and the harness release it qualified (broker §12.2).
- The audit log records each request, decision, effect and cleanup as typed metadata.

## 14. Later integrations and their gates

| Integration | Must be defined before it is enabled |
|---|---|
| session host (§12) | launch tickets, fences and epochs, the ownership table |
| board spawn path | a board-owned adapter that keeps the board's run ids, attempt lineage and recovery successors; distinct dispatcher execution handles; a second completion barrier (an execution's exit never satisfies the board's `wait`); durable, replayable exit delivery |
| key keeper | registration tuples `(uid, generation, identity path, owner, registration generation)` plus the one-shot flag recorded at registration, with active-ledger checks and detachment before UID reuse; the broker's versioned key-migration procedure |
| one-shot executor | R-LR1 versioned narrow `curator-run` intake and Decision 0021 amendment, plus broker credential-injection qualification (§5.3) |

**Identity cleanup (D-IDENTITY).** The dispatcher may revoke only identities it registered as one-shot, with that flag recorded at registration; it MUST NOT infer the flag from an account name or a later retirement request. After confirmed account retirement, it performs identity revocation as a separate journaled effect, reusing that effect ID on retry. Unknown or refused revocation keeps the applicable cleanup obligation pending. A standing orchestrator or fixed team role's account retirement never revokes its identity; only the operator may revoke it with a signed record. This narrow exception does not grant general identity administration. The keeper registration/revocation API and broker migration procedure must exist before enabling this integration.

## 15. Qualification

On hosted CI runners only, through the deployed clients (`curator agent-user` and `curator-dispatch`) and exactly the deployed sudoers rules:

- `dispatch`, and `provision` then `start` twice in sequence, with the broker's slice 0 and one Claude launch each;
- authority: a copied chain from another caller, a missing or wrong signature, an unregistered key, an unconstructible request, a revoked caller grant (the agent's next lease refused), a recycled UID;
- crashes on both sides of every acknowledgement (input, effect intent, effect result, claim, supervisor registration, start receipt, termination receipt), each followed by reconciliation;
- cancel in every state; `max_runtime`; allocation expiry with and without the sweep; a lost supervisor with escaped descendants;
- capacity: quota 1 admits its own request and refuses a competitor; each gate; hold read failure;
- payload bounds at, and one over, each limit;
- v1: peer authentication, server identity, no transport fallback, queue fairness and drain, the activation transaction and its rollback.

## 16. Open questions

1. **Retirement enum (contract P7; OWNER DECISION NEEDED, user-manager owner):** admit `keep` as an operator-only request value and align the helper schema, or remove it from policy and retain only `archive` / `delete`. Neither option extends dispatcher authority; this spec admits only archive/delete for dispatcher cleanup (§5.4).
2. **Helper v2 signed requests (contract W3; OWNER DECISION NEEDED, user-manager/contract owners):** publish a component-specific format or a versioned shared adapter. D-WIRE-NAMES supplies the naming rule but not the signature domain/version and canonical bytes. v0/v1 use authenticated sudo caller and role policy, not an invented signed-grant verifier.
3. **Launch tickets (contract I4; OWNER DECISION NEEDED, dispatcher/user-manager/session-host owners):** jointly version and publish the signature domain, canonical bytes and host-role schema, then activate the gate, or leave the SH2 backend disabled (§12). No concrete ticket-signature grammar is claimed here.
4. **Agent self-goals (contract G2; deferred SH2+ owner decision):** (a) a separate agent namespace with its own bounds, or (b) retain caller/board-only authorship. Neither requires goals or a board for host operation (§8.4).
5. **Log-storage handoff:** retain dispatcher execution logs or transfer them to the host's durable log store when the backend activates; the ownership handoff must be specified first (§12).

The executor and first consumer are decided by D-EXECUTOR, managed PATH by D-PATH, wire names by D-WIRE-NAMES, and resume compatibility by D-RESUME; their older open questions are superseded. Default quotas and lifetimes are already decided and configurable (§3.5). Interactive execution remains gated on the session host. Donor provisioning belongs to a separately specified small privileged component, not a user-manager extension (D-DONOR-PROVISIONING); variant B stays disabled until that component is specified, and remote targets remain outside this subset.

## Appendix A. Refusal codes

`not_dispatcher_account`, `not_invoked_by_sudo`, `caller_not_permitted`, `caller_key_not_associated`, `caller_key_pending`, `signature_invalid`, `request_invalid`, `request_too_large`, `request_timeout`, `request_id_conflict`, `payload_too_large`, `plan_digest_mismatch`, `grant_invalid`, `grant_unsupported`, `grant_subject_mismatch`, `grant_not_delegable`, `grant_unconstructible`, `quota_exceeded`, `capacity:slots`, `capacity:disk`, `capacity:cpu_idle`, `capacity:hold`, `capacity:build_lock`, `capacity:syspolicyd`, `expired_in_queue`, `agent_busy`, `agent_not_found`, `agent_not_owned`, `agent_retired`, `run_not_found`, `run_not_owned`, `run_terminal`, `unsupported_combination`, `observation_unknown`, `mode_mismatch`, `state_version_unsupported`.

## Appendix B. Diagrams

PlantUML sources are design views of this spec, not implementation or qualification evidence. The audit's module rows map to:

| Audit row | Figures | Defining sections |
|---|---|---|
| M-DP1 | [Components](../diagrams/components.puml), [dispatch lifecycle](../diagrams/dispatch-lifecycle.puml) | §§2–6, 9, 12, 14 |
| M-DP2 | [Run states](../diagrams/run-states.puml), [agent and claim states](../diagrams/agent-claim-states.puml) | §§4.3–4.6, 5–7, 9 |
| M-DP3 | [Storage layout](../diagrams/storage-layout.puml) | §§4.1–4.5, 6, 8.3, 11 |

Companion views: [reconciliation](../diagrams/reconcile.puml) (§7.1) and [v1 daemon / gated SH2+ backend](../diagrams/v1-daemon.puml) (§§8.3, 12). Unspecified details are marked in the figures; their notes do not introduce contract behavior.
