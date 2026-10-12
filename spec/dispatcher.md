# swarma-dispatcher: specification

- **Status:** draft v0.6 (2026-10-12). Nothing is implemented.
- **Normative language:** MUST, MUST NOT, SHOULD and MAY are used as in RFC 2119.
- **Companions:** [swarma-user-manager](https://github.com/relux-works/swarma-user-manager) (agent accounts, the launcher, the stop operation), [swarma-credential-broker](https://github.com/relux-works/swarma-credential-broker) (bindings, caller-authorization dependencies, leases), curator-spec CIP-0011 (the Curator side), the platform architecture §7.1 (launch dispatcher), §5.1 (launch contract), §5.3 (fences and restarts), §7.2 (key keeper), §7.5 (isolation).

### Revision 0.6

This documentation revision applies the accepted review findings of 2026-10-12: the Unix-socket service starts in slice 0 (F-5); dispatcher-owned `host_start` and the host admission permit replace launch tickets (F-4); operator maintenance has a bounded stop-and-reconcile attempt (F-2); orchestrator delegation is requester-signed and recursive provisioning stays gated (F-3); admission uses the broker's `revocation.check` verdict (F-8); receiver-owned `effect.observe` states map to the four reconciliation classes (F-10); retirement references deletion and explicit operator retention in the user manager (F-1). It aligns every affected figure and lists unpublished formats in §16. These remain design contracts, not implementation or qualification claims.

### Revision 0.5

This documentation revision applies the owner decision of 2026-10-12 on standing-session runtime: executions of standing allocations have no `max_runtime` quota or runtime deadline, and the sweep never stops them for runtime (§§3.5, 4.4, 5.3, 7.2, 8.1). Other executions retain the 4 h default and 12 h maximum. It closes R3 as owner-decided (§16), adds negative qualification rows (§15), and aligns the figures. These remain design contracts, not implementation or qualification claims.

### Revision 0.4

This documentation revision applies the owner decisions of 2026-10-12: standing team allocations have no expiry or lifetime quota and only the operator retires or re-creates them with a signed record; an operator-signed team description brings them up idempotently (§§3.5, 4.3, 5.4–5.5, 7). It states caller-side plan composition by an orchestrator in its agent account (§5.2), and records the standing-session runtime proposal and unpublished formats as open questions (§16). The execution-runtime limits are unchanged. These remain design contracts, not implementation or qualification claims.

### Revision 0.3

This documentation revision aligns R-DP1 with the [platform contract table](https://github.com/relux-works/curator/blob/177cb554/.research/261010_platform-contract-table.md), the [docs audit](https://github.com/relux-works/curator/blob/48789217/.research/261010_platform-docs-audit.md) and the owner decisions of 2026-10-10/11. It separates root correlation from effect IDs, initial capacity refusal from later queueing, and one-shot execution from session holding; it records the narrow identity cleanup authority and the remaining owner questions (§16). These are design contracts, not claims of implementation or qualification.

### Revision 0.2

An architecture review on 2026-10-09 (0 P0, 15 P1, 3 P2) kept the module split and asked for a closed contract. This revision:

- binds a caller's grant to the caller's authenticated principal and request signature, and keeps the caller's authorization as a dependency the broker re-checks on every lease, so revoking a caller cuts its agents at once (§3; owner decision 2026-10-10);
- freezes the grant vocabulary and the constructible chain shapes (§3.4);
- derives a distinct, durable effect id for every downstream call (§4.5);
- stores immutable inputs and effect intents and results, distinguishes known, absent, unknown and contradictory observations, and keeps a run non-terminal until its cleanup is confirmed (§4, §7);
- separates **agent allocations** (`agent/1`) from **executions** (`run/1`), so `provision` and repeated `start` have a lifecycle (§5);
- replaces stop-by-signal-through-sudo with an execution claim, a launcher start receipt and the helper's authorised `exec.stop` (§6);
- uses owned capacity claims and the architecture's host gates (§9);
- publishes the payload budget and request schemas (§8, §10), updated to the single service transport in revision 0.6;
- defines operator activation, superseded by the maintenance contract of revision 0.6, and names what later integrations (session host, board, key keeper) must define before they are enabled (§11, §14).

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
| supervisor | the dispatcher-owned process that holds one execution (inside the daemon) |

## 2. Versions

Figure M-DP1: [process and component boundaries](../diagrams/components.puml).

| Slice | Shape | Adds |
|---|---|---|
| initial protected-launch slice (slice 0) | unprivileged `swarma-dispatcher` Unix-socket service on `dispatch/1`, using the broker's connection and framing rules (§8.3); temporary capacity closure refuses | requests, allocations, executions, authority, capacity claims, provisioning, start, stop, cleanup and reconciliation; direct workers and operator-enrolled orchestrators |
| later capability-gated slices | the same service, core and transport | queue with handles, events, long waits and drain; session-host backend, recursive orchestrator provisioning and key keeper registration each require their publication and qualification gates (§§5.2, 12, 14) |

Quotas and authority failures always refuse; neither is queueable (§3.5, §9). Initial admission omits queueing, events, long waits and advanced integrations. Capability negotiation and refusal mappings for unsupported capabilities would need publication (§16). The service completes recovery and publishes readiness before admission. An absent or unready daemon refuses; there is no command fallback. Binary, state-schema and backend-mode changes require the fenced operator maintenance transaction (§11).

## 3. Identities and authority

### 3.1 The dispatcher's account

The dispatcher runs as the unprivileged service account `swarma-dispatcher`, created by the helper for the installer. Only this account is a **dispatcher** in the helper policy and in the broker, so the root transitions and bindings are reachable only through it. The dispatcher is a trusted principal for its scope: a compromised dispatcher can sign leaves within its own grant and administer the agents it created, but cannot exceed the broker-verified root bounds or obtain a general root shell from the helper's closed operations.

### 3.2 Callers

- The caller is the kernel-reported peer UID on `dispatch/1`, with the broker's connection rules (§4.1): protected socket directory and endpoint, client verification of the server UID, bounded authenticated connections and descriptor-inheritance/passing restrictions. Caller-to-service sudo authority is absent; dispatcher-to-root helper and launcher authority remains (§11).
- Principals follow broker §4.2: managed callers by `(uid, generation)` with the helper ledger showing the generation `active` (re-checked for every operation, every queued start and every reconciliation); unmanaged callers pinned by UID and directory identity. An unreadable or invalid ledger refuses; a recycled UID never inherits old allocations or executions.

### 3.3 Caller keys and request signatures

- The dispatcher's configuration (owned by its account, written by the operator through the installer) maps each **principal** to its **caller keys** (Ed25519). An orchestrator's caller key is generation-bound. `register_key` records its public key separately from caller-authority activation; the requester signs its delegation only after registration (§5.2). Operator-enrolled orchestrators are supported in the initial slice with generation-bound keys and valid caller chains installed through operator configuration; this does not require the unpublished pre-key exception. Recursive provisioning is refused until its bootstrap is published and qualified.
- Except for the narrow pre-key `register_key` exception below, every mutating `dispatch-op/1` request carries `auth: { key, sig }`: an Ed25519 signature by a caller key over the domain tag `swarma-dispatcher request/1`, a newline and the CCJ-1 bytes of the request without `auth`. The key must be associated with the authenticated principal (`caller_key_not_associated`), and the leaf of the caller's grant chain must have that key as its subject (`grant_subject_mismatch`).
- **Wire naming (D-WIRE-NAMES).** Versioned wire identifiers use the repository name: `swarma-<component> <object>/<n>`. The signed request domain above is exactly `swarma-dispatcher request/1`; the request-signature newline does not apply to the effect hash (§4.5). Broker signed domains are `swarma-credential-broker grant/1` and `swarma-credential-broker revocation/1`. Nothing has shipped, so these clean names require no migration or legacy-domain acceptance. This decision supersedes the contract table's W2 retention baseline; it does not publish the helper v2 signature grammar (§16). The existing dispatcher transport/schema labels `dispatch-op/1` and `dispatch/1` remain distinct from the signed domain.
- A valid chain presented without the matching principal and signature is refused; a copied chain is useless to another caller.
- **Pre-key registration exception (R-DELEG).** Only the authenticated allocation's own active `(uid, generation)`, already selected for orchestrator enrollment by valid requester/operator authority, may register its first caller key. This exception permits key registration with proof of possession of that key, never `provision`, `start` or spawn authority. It does not use a nonexistent previously registered key. The dispatcher durably binds the enrollment intent, generation, key and registration replay evidence before acknowledgement. An exact replay returns the same registration; a changed key or enrollment intent refuses, a reused UID cannot replay a former generation's registration, and unavailable replay evidence keeps registration closed. The proof-of-possession domain, canonical bytes, freshness window and replay identity are unpublished (§16); the exception is disabled until these are published and qualified.
- **Rotation.** A new key requires proof of possession and authorization by the currently enrolled key within its current authority, or explicit operator recovery if that key is unavailable. Rotation is journaled and serialized with activation: the old key loses its association, stale registration replays cannot restore it, and a replacement delegation to the new subject must be verified before caller authority is active. Registration never silently rotates a key or transfers authority. The rotation signature/replay format and operator recovery authorization would need publication (§16).

### 3.4 Grants and their constructible shapes

- **Action vocabulary:** `agent.spawn` (may ask the dispatcher to provision and start agents within the grant's accounts, harnesses and profiles) and `credential.lease` (may lease those accounts). Unknown actions refuse (`grant_unsupported`).
- **Caller chain:** the caller's leaf grant names the caller key, allows `agent.spawn` and `credential.lease`, and has remaining depth ≥ 1. It chains to a configured root under the broker's verifier (broker §6.2), at most 4 envelopes.
- **Dispatcher chain:** the dispatcher's own grant names the dispatcher key, allows `credential.lease`, has depth ≥ 1, and is at most 3 envelopes long, so an agent leaf can be appended within the 4-envelope bound. The installer refuses to configure a dispatcher chain outside this shape.
- **Agent leaf:** signed by the dispatcher key under its own chain; subject `agent:<generation>`, depth 0, action `credential.lease` only; accounts, harnesses, profiles and window are the intersection of the request, the caller's grant and the dispatcher's grant. A request whose intersection is empty or whose leaf could not be constructed is refused before any account is created (`grant_unconstructible`).
- **Caller dependency:** the binding carries the caller's chain and the exact intersection (broker §7, revision 0.3). The broker verifies both chains, separately, at bind, lease and every renewal, against the current revocation state: revoking the caller's grant or key stops new leases and renewals for every agent provisioned under it immediately. Tokens already delivered remain valid until they expire or are revoked at the vendor.
- **Orchestrator delegation preflight (R-DELEG).** For requester C to delegate spawn authority to O, C's chain must leave depth for O's `agent.spawn` and `credential.lease` leaf: C's leaf depth ≥ 2, O's depth strictly smaller and ≥ 1, and C's chain plus O's envelope ≤ 4. Each edge uses the parent subject as issuer. The dispatcher chain remains lease-only and never substitutes for C's caller ancestry. Insufficient depth or chain length refuses before account creation.
- **Revocation verdict (R-VERDICT).** At prepare, dequeue (when enabled) and every start, the configured dispatcher calls broker `broker/1` method `revocation.check` with the exact caller and dispatcher chains being checked and context `prepare`, `dequeue` or `start`. The broker answers `not_revoked`, `revoked` or `unavailable`, together with its revocation sequence at the check. Only `not_revoked` passes this check; `revoked` and `unavailable` refuse before new effects. Fenced broker restore answers `unavailable`. The verdict is point-in-time evidence, never a grant or a future-start permit; no cached verdict substitutes for a new admission check. Exact encoding and refusal-code mappings would need publication (§16).
- **Delayed and recovered work** (a queued request, a start after `provision`, a resumed cleanup that would start anything) re-checks the caller principal, request signature and validity window, caller chain and a fresh `revocation.check` verdict before acting. The broker's final bind, lease and renewal checks remain mandatory; an earlier verdict cannot prevent later revocation.

### 3.5 Quotas

Configuration sets, per caller and in total: live allocations, concurrent executions, and, for ordinary allocations and their executions, the maximum allocation lifetime and the maximum execution runtime. Requests beyond the applicable quotas are refused (`quota_exceeded`) — a permanent refusal, never a queueable condition.

Defaults (owner decision 2026-10-10), all configurable:

| Limit | Per caller | Per node |
|---|---:|---:|
| concurrent executions | 2 | 6 |
| live allocations | 4 | 16 |
| allocation lifetime (one-shot and ordinary kept allocations) | default 24 h, maximum 7 days | — |
| execution runtime (all executions other than those of standing allocations) | default 4 h, maximum 12 h | — |

Standing allocations cover the standing team's orchestrator and fixed roles, including other orchestrators and standing agents. They have no `expires_at` and no lifetime quota. Only the operator may retire or re-create them, with a signed record (§§4.3, 5.4–5.5); the sweep never expires them (§7.2). Per the owner decision of 2026-10-12 (§16, question 6), their executions MUST NOT acquire a `max_runtime` quota or runtime deadline, in either supervisor or session-host backends. The sweep MUST NOT stop them for runtime. They end only by an explicit stop (`cancel`/stop through `exec.stop`), operator action, their own exit or an incident hard stop (§5.3). Live-allocation and concurrency quotas still apply. Every other execution retains the 4 h default and 12 h maximum.

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
  active_execution: run_id | null,          // at most one
  expires_at?: allocation expiry (absent for standing allocations),
  cleanup: { home: "delete" },             // ordinary retirement; explicit retention is operator-only
  created_at, updated_at }
```

`agent/1` MUST retain a designation that marks an allocation as standing, tied to the operator-signed team description (§5.5). A standing allocation MUST NOT carry `expires_at` or acquire a lifetime quota. The designation's wire encoding and the signed retirement/re-creation record format are specified-but-unpublished; no member name or value for them is defined here (§16). Ordinary allocations retain their expiry and §3.5 lifetime bounds. Only the operator may retire or re-create a standing allocation, with a signed record; a caller's `agent.spawn` is insufficient.

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
  deadline?:  { max_runtime, ends_at },     // ordinary executions only; absent for standing
  parent, board_ref (opaque, kept whole),
  created_at, updated_at }
```

An execution of a standing allocation MUST NOT carry `deadline`, `max_runtime` or `ends_at` as a runtime limit. An ordinary execution retains its runtime deadline under §3.5. Absence is the standing contract; no sentinel duration, null deadline or additional wire member is defined.

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
| claim `epoch`, backend `controller_epoch`, `op_id` | execution claim epoch, host controller fence epoch and backend operation identity; none is an account generation (§12) |

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

Each `E-*` uses its record, component, operation and journaled attempt; a retry replays the same intent and ID. The root `R` is never substituted for an `E-*`. Separate `provision` and `start` requests have separate root request IDs; their effects remain correlated to their own originating request and the shared allocation.

Receiver-owned read-only `effect.observe` is published by `helper-op/1` and `broker/1`. The dispatcher asks with the original effect's `request_id`, operation and digest of canonical arguments; the lookup never performs a mutation. Only the original receiver caller principal (here, the dispatcher) or an operator may observe; the receiver serializes the read with its journal. It remains usable for reconciliation after the initiating caller/grant expires and grants no fresh mutation authority. Completed mutation replay still returns its recorded result and current target state. Broker unbind uses the matching removal/tombstone observation; retention and exact encoding would need joint publication (§16). Missing or expired evidence is `unavailable`, never absence.

### 4.6 Observations

Reconciliation and recovery classify every observation of the helper ledger, the broker binding, the claim and the process evidence as **known present**, **known absent**, **unknown** (unreadable, timed out) or **contradictory**. Unknown and contradictory observations never authorise forward execution, a second launch or a successful cleanup; the record stays where it is and the observation is retried.

R-OBS maps the receiver's six states exactly:

| Receiver state | Dispatcher observation class |
|---|---|
| `recorded`, with recorded result and current target state | known present or known absent according to the current state |
| `pending` | unknown |
| `rolled_back` | known absent for that effect only |
| `not_recorded`, with complete authoritative evidence it was never recorded | known absent for that effect only |
| `unavailable`, including missing, expired or unreadable evidence | unknown |
| `contradictory` | contradictory |

`rolled_back` and `not_recorded` never authorize an automatic new create. Absence for an effect does not establish absence of another effect or of every process. A partial list, refused observation, timeout or unreadable evidence cannot establish absence, release capacity or satisfy cleanup. Process absence and launch/stop receipts remain separately required.

## 5. Lifecycle

Figure M-DP1: [provision, launch, refusal and recovery sequence](../diagrams/dispatch-lifecycle.puml).

### 5.1 Allocation

`provision` (or the first half of `dispatch`):

1. **prepare** — schema, principal, signature, chains, delegation depth preflight when applicable, quotas, constructibility, plan digest and fresh `revocation.check` with context `prepare`; write `input/1`. Refusal ends here with no effect.
2. **capacity** — take an owned claim (§9) or refuse initially; queue only when that capability is enabled, with a fresh `dequeue` check before any effect.
3. **provision** — helper `user.create { kind: agent, label, request_id: <effect id> }` → generation; (helper v1) `fw.apply`; sign the leaf; broker `bind { uid, generation, lease chain, caller chain, intersection, request_id: <effect id> }`. The allocation becomes `idle`.

On failure an ordinary allocation goes to `retiring`: unbind (if bound) and `user.retire` (if created), each as its own effect, retried until confirmed; the allocation becomes `failed` only after those effects and any applicable identity cleanup (§14) are confirmed.

For a standing allocation, provisioning failure does not authorise automatic retirement or re-creation. Keep it pending for reconciliation or an operator-signed decision (§§5.4–5.5, 7.1); unknown observations never permit a duplicate effect.

### 5.2 Orchestrator allocations

The home-generated caller key is distinct from a keeper-retained identity key; broker software roots are a third key class. Keeper adoption requires the versioned broker migration adapter (§14), not relabelling a caller key as a keeper identity.

**Requester-signed delegation (R-DELEG).** Requester C signs O's caller delegation, narrowed from C's own grant, after O's public key has been registered under O's active generation (§3.3). C is the issuer and O's registered key is the subject; the dispatcher D verifies and records this chain and never signs O's caller delegation. D continues to sign only its separate credential-lease agent leaf (§3.4). Descendants retain C in their ordinary caller chain, so ancestor revocation remains an ordinary verifier/broker dependency check.

Registration is separate from activating caller authority. Before provisioning, preflight remaining depth and the four-envelope bound (§3.4). At activation, recheck C's authenticated principal, request signature, delegation issuer/parent-subject continuity, O's generation/key association, attenuated actions/scope, depth, chain length, validity window and a fresh `revocation.check` with context `prepare`. A missing C signature, unavailable requester or missing/unavailable evidence leaves enrollment pending with no spawn authority. O cannot obtain implicit authority from its UID, registration, capability designation or the dispatcher's signature. Rotation needs a newly valid delegation to the replacement key.

The pre-key registration exception, replay and rotation rules are in §3.3; their wire format and the authority-install transaction remain unpublished (§16). Recursive orchestrator provisioning MUST refuse until this bootstrap is jointly published and qualified. Direct worker launches and operator-enrolled orchestrators form the initial slice. Operator enrollment does not waive registration or valid caller-chain requirements; no first-start bypass grants spawn authority.

An orchestrator composes its children's plans on the caller side, under its own agent account, using its own Curator configuration in its home. The child's `curator-run` only verifies the caller-composed plan before executing it (§5.3); it does not compose or resolve profiles. Which profiles and environments the orchestrator may read and use for its children, and how its narrowed grant constrains them, are unspecified and remain an open question (§16).

### 5.3 Execution

`start` (or the second half of `dispatch`), on an `idle` allocation owned by the caller:

1. re-check the caller's authority and a fresh `revocation.check` with context `start` before any launch effect and, for an ordinary allocation, its lifetime and §3.5 runtime quota; take a capacity claim for one execution. A standing execution has no `max_runtime` quota or runtime deadline; an ordinary execution retains its runtime deadline;
2. **claim** — under the lock, write an execution claim `{ execution_id, epoch }` and mark the allocation `running`; only one claim per allocation (`agent_busy`);
3. **supervisor** — the daemon starts the supervisor; it registers its own process identity and acknowledges durably before anything else;
4. **launch** — the supervisor calls the launcher (`launch.start { request_id: <launch effect id>, generation, execution_id, epoch, executor, plan }`); the launcher writes a **start receipt** for that execution (pid, start time, boot id) while still root, before clearing groups, setting GID/UID and exec (helper §7); the user manager §7 owns the descriptor lifecycle (R-FD3): scrub untrusted inherited descriptors first, retain the generation lock through the root receipt and completed UID/GID transition, retain the verified executable handle until exec, and release internal handles at last use/close-on-exec and on every failure. The executor receives only 0–2 and read-only FD3 with completely staged, bounded immutable plan bytes; the per-platform staging mechanism remains a publication gate, and any pipe would need qualified capacity/feeder/deadline/cleanup ordering. No broker or root handle is inherited; the supervisor records the receipt;
5. **running** — `curator-run` in its narrow exec mode verifies the caller-composed plan, prepares its working directory in the agent's home, resolves the protected credential binding, opens its own broker connection, leases its credential and execs the harness. For an ordinary execution, `max_runtime` expiry leads to **stopping**. A standing execution has no runtime deadline and ends only by an explicit stop (`cancel`/stop through `exec.stop`), operator action, its own exit or an incident hard stop; the sweep never stops it for runtime. Exit, explicit stop, operator action or an incident hard stop leads to **stopping**, with absence confirmed in step 6;
6. **stopping** — the supervisor (or recovery) calls helper `exec.stop { request_id: <stop effect id>, generation, execution_id }`, which terminates the execution's process session and returns a termination receipt only after it has verified that no process of that execution remains;
7. **finish** — record the outcome; the allocation returns to `idle` (ordinary kept or standing agents) or proceeds to `retiring` (ordinary `dispatch`, an expired ordinary allocation, or operator-signed retirement of a standing allocation).

**One-shot intake gate (D-EXECUTOR).** `curator-run` under the agent account is the first broker consumer. The board runner and session host ask the dispatcher; neither embeds a broker client. This mode never composes or resolves profiles and has no network access except to the broker; harness egress is governed separately by the admitted network policy. It shares one launch library with `swarma-session-runner`. A dedicated entry such as `curator-run exec --plan-fd N` illustrates the mode, not a published CLI grammar. The launcher owner (R-LR1) publishes the versioned Decision 0021 amendment and intake contract (immutable plan verification, FD transport, credential injection capability, signals, exit/receipts and typed refusals). This launch path stays disabled until that contract is published and qualified; the older interactive scope is not evidence of one-shot support. A separate minimal executor binary is a later option, not the current target.

The managed plan records the resolved PATH and covers it with its digest (D-PATH): protected launch tool directory (CIP-0007), then admitted base PATH in its existing order, then one protected manager dispatcher directory. Project commands pass that dispatcher's checks; a repository `.agents/bin` is never executed in a managed launch. The exact plan schema is owned with the intake contract.

A cancel is serialised with the claim: before the launch, it marks the claim `absent` and the launcher is never called; during the launch, the supervisor waits for the receipt or the launcher's refusal and then stops; after the launch, it stops. A request never issues a second launch for an execution whose state is unknown.

### 5.4 Retirement and expiry

`retire` of an ordinary allocation, an expired ordinary allocation, or a finished one-shot `dispatch`: unbind, then `user.retire` under the user manager's delete-at-retirement rule (§4.4.1), each a confirmed effect. The user manager quiesces the account, cleans declared keys and deletes its home. There is no archive-by-default policy. Any retained retirement export is limited to service-generated metadata in a closed schema with no agent-controlled strings, names, links or bodies. Code/results follow their separate git/acceptance policy, which does not prove secret-free content.

Keeping a home is a separate explicit operator retention operation owned by the user manager; it must state plainly that the home's confidentiality is not guaranteed. Dispatcher cleanup has no retention authority and admits `delete` only (§4.3); the retention operation's unpublished format would need publication by the user manager (§16).

With keeper integration active, the dispatcher then revokes an identity only if its registration recorded it as one-shot (§14), as a separate, confirmed effect after account retirement. Standing identities are left to the operator. The allocation becomes `removed` only after its applicable cleanup effects are confirmed; executions become terminal only after their stop is confirmed and their allocation's cleanup obligations are recorded. The daemon's internal sweep (§7.2) owns ordinary allocation lifetimes while no request is running.

Standing allocations do not expire and are not retired on execution exit, provisioning failure or loss of caller authority. Only the operator may retire or re-create them, with a signed record. Once such retirement is authorized, the same confirmed cleanup effects apply; reconciliation may retry those authorized effects but cannot initiate retirement on its own. Account retirement never revokes a standing identity (§14). Re-creation is an operator decision, not an automatic recovery from an absent account or binding. The signed-record format remains unpublished (§16).

### 5.5 Bring up team (specified-but-unpublished)

Companion figure: [operator-signed team bring-up and per-role reconciliation](../diagrams/bring-up-team.puml).

The dispatcher has an operator-only bring-up-team operation from an operator-signed description of the standing team. Its request object is named **`swarma-dispatcher team/1`** under D-WIRE-NAMES. It is specified-but-unpublished; it is not an added `dispatch-op/1` value. The description lists the standing roles, each with only the following specified fields:

| Field | Meaning |
|---|---|
| `label` | the standing role's label |
| `profile` | the role's profile |
| `harness` | the role's harness |
| `accounts` | the role's accounts |
| `capabilities` | the role's capabilities |

Only an authenticated operator with a valid operator-signed description may invoke it. A caller with `agent.spawn`, including an orchestrator allocation, has no bring-up-team authority. The signature grammar, remaining envelope/record fields, role matching, change handling and transport are unspecified (§16); this section does not invent them or new refusal codes.

From the description, the dispatcher creates or reconciles one standing allocation per role, idempotently, with one logical bring-up effect per role: repeating bring-up or recovering it must not create a second allocation or duplicate a role's already-recorded effect. Provisioning uses the usual distinct, journaled effect IDs for create and bind (§4.5), reusing each ID on retry, and the existing provisioning/idle/running/retiring states and known-present/known-absent/unknown/contradictory observations (§§4.3, 4.6, 7.1). An existing allocation is reconciled, not automatically replaced; running allocations are not relaunched. An unknown or contradictory effect remains pending until observed. Retirement or re-creation always needs the operator's signed record (§5.4). Bring-up allocates roles; execution remains a separate start (§5.3).

## 6. Holding and stopping

- **Supervisor.** The daemon owns the per-execution supervisor under `swarma-dispatcher`: new session, no controlling terminal, caller descriptors closed, bounded private logs of the executor's output in dispatcher state, durable registration before launch, and start/termination receipts recorded. Daemon loss does not authorize live adoption or a second launch; reconcile the existing effect/receipt and stop as required (§7).
- **Stop.** The only stop is the helper's `exec.stop`, authorized for the dispatcher's own generations; ordinary ephemeral executions may also end through `user.retire`, which stops the account's processes before removal. Operator maintenance authority is explicit (§§8.1, 11). Qualification distinguishes caller disconnect/death, daemon death, executor ignoring SIGTERM and descendants escaping the session; a client connection is never the execution's lifetime authority.

## 7. Reconciliation and the sweep

### 7.1 Reconciliation

Companion figure: [state-by-state reconciliation](../diagrams/reconcile.puml).

Runs at daemon start before readiness/admission, periodically inside the daemon, and on `reconcile`. For every allocation and execution that is not terminal, it compares the journal and effect table with the observations (§4.6), state by state:

| State | Observation | Action |
|---|---|---|
| provisioning | create effect unknown | call helper `effect.observe` with original effect ID/operation/digest; apply §4.6; no automatic new create |
| provisioning | account present, bind effect unknown | call broker `effect.observe`; absence is effect-scoped only; resume only the same admitted bind effect with fresh `prepare` authority/verdict checks, never infer authority for a new effect; otherwise retire only an ordinary allocation or keep standing pending |
| idle | binding known absent, ordinary allocation not expired or standing allocation | reconcile original bind/unbind observations; a new bind requires explicit fresh authority and `prepare` verdict, never automatic creation from effect absence; if authority is gone, retire only an ordinary allocation; standing stays pending |
| running, claim `claimed` | supervisor registration conclusively absent | mark the claim `absent`; no launch happens later |
| running, claim `launching` | receipt unknown | keep waiting for the receipt or the launcher's answer; never relaunch |
| running, `started` | supervisor known absent | `exec.stop`, then record `lost` |
| stopping | termination receipt missing | repeat `exec.stop` (same effect id) |
| retiring / cleaning | effects pending | repeat them; terminal only when all are confirmed |

Reconciliation uses `effect.observe` rather than mutation solely to discover state: `recorded` maps by current target state; `rolled_back`/`not_recorded` are known absent for that effect only and never trigger automatic new create; `pending`/`unavailable` remain unknown; `contradictory` stays contradictory. Observe helper create/retire and broker bind/unbind, including retained removal/tombstone evidence. Launch/stop use qualified receipt and process evidence; observations do not waive confirmed process absence. After caller/grant expiry, read-only observation can resolve prior effects without renewing authority. Any forward admission requires fresh §3.4 checks; missing evidence keeps claims held.

Reconciliation of a standing allocation never initiates retirement or re-creation without the operator's signed record, including on a provisioning failure or known-absent account. It may resume already-authorised effects; unknown and contradictory observations remain pending. The absence of allocation expiry does not waive fresh authority checks.

### 7.2 The sweep

The daemon runs an internal sweep every minute: reconciliation, ordinary allocation lifetimes, claim maintenance and log rotation. There is no external sweep timer or public sweep operation; caller `reconcile` remains scoped by §8.1. The sweep never expires standing allocations: they have neither `expires_at` nor a lifetime quota. It MUST NOT stop their executions for runtime, because they carry no `max_runtime` quota or runtime deadline. Ordinary executions retain §3.5's 4 h default and 12 h maximum; the sweep stops an ordinary execution whose runtime deadline has elapsed through confirmed `exec.stop` (§§5.3, 6). It reconciles standing allocations' authorized effects/executions and maintains claims and logs. While the daemon is absent or unready, new admission refuses; status must report that ordinary lifetimes are unenforced when it can be served, and distinguish standing allocations where expiry does not apply. Recovery completes before readiness and reopening admission.

## 8. Operations, results and authorisation

### 8.1 Request `dispatch-op/1`

A discriminated union on `op`; every operation has a closed set of members, and members of other operations are refused (`request_invalid`). Rows marked as gated or partly unpublished specify semantics only; they cannot be enabled as an incomplete wire schema. Times are RFC 3339 UTC; durations are integers in seconds; lists have bounds.

| `op` | Members | Allowed to |
|---|---|---|
| `provision` | `request_id`, `label`, `profile`, `harness`, `accounts`, `capabilities?`, `lifetime`, `cleanup`, `network?`, `parent?`, `grants`, `auth` | callers with `agent.spawn` |
| `start` | `request_id`, `agent_id`, `plan`, `plan_digest`, `executor`, `max_runtime?` (ordinary only), `board_ref?`, `auth` | the allocation's owner |
| `dispatch` | the union of `provision` and `start` members except `agent_id`, plus `cleanup.retire = "on-exit"` | callers with `agent.spawn` |
| `cancel` | `request_id`, `run_id`, `auth` | the execution's owner; operators may stop selected executions only under explicit maintenance authority (§11), whose signed authorization format is unpublished |
| `retire` | `request_id`, `agent_id`, `auth` | the ordinary allocation's owner; standing retirement is operator-only with a signed record whose format is unpublished (§§5.4, 16) |
| `status` | `run_id` or `agent_id` | the owner |
| `wait` | `run_id`, `until_state`, `timeout ≤ 30 s` initially; `≤ 600 s` only with the later long-wait capability | the owner |
| `list` | `kind`, `cursor?`, `limit ≤ 200` | own records |
| `capacity` | — | any configured caller |
| `reconcile` | `run_id` or `agent_id` | the owner, or operators for any |
| `drain` (later queue capability) | `mode: "park" \| "reject"` | operators |
| `register_key` | `public_key`; pre-key proof/replay and rotation encoding unpublished (§3.3) | the allocation's own authenticated active generation under the narrow enrollment exception; registration never activates spawn authority |
| `host_start` (gated §12) | `logical_session_id`, `execution_id`, `controller_epoch`, `op_id`; refers to the host admission permit bound to these values; permit reference/authentication encoding unpublished | only the configured session-host service principal |

For an ordinary allocation, omitted `max_runtime` uses §3.5's 4 h default; a supplied value is subject to the 12 h maximum and configured quotas. For a standing allocation, `max_runtime` MUST be absent; supplying it MUST be refused before launch, never used to install a deadline or silently ignored. The refusal-code mapping is unpublished (§16, question 8); no new code is asserted here. Allocation designation, not the requested duration, determines which policy applies.

### 8.2 Results

`dispatch-result/1`: `{ v: 1, request_id, ok: true, result: { … record … } }` or `{ v: 1, request_id, ok: false, error: { code, message, stage? } }`. A repeat of a completed request returns the stored result with the record's **current** state. The meaning of terminal outcomes is fixed: `exit` (the harness ended), `failed` (a stage failed and cleanup is confirmed), `cancelled`, `expired` (ordinary allocation lifetime or ordinary execution runtime; neither applies to standing allocations or their executions), `lost` (the holder vanished and the execution was stopped and confirmed absent).

### 8.3 Socket `dispatch/1`

Companion figure: [service daemon and gated SH2+ backend](../diagrams/v1-daemon.puml).

The daemon `swarma-dispatcher` runs as `swarma-dispatcher` and listens in `/opt/swarma/run/dispatcher/`. It uses the framing and connection rules of broker §4.1 and §13 with one declared difference: frames up to 128 KiB (§10). This is the service transport from slice 0, including the Curator CLI client. The broker rules include peer authentication, server-UID verification, protected endpoint/connection lifecycle and descriptor restrictions; descriptors are not a request transport. An absent/unready daemon or authentication, permission, version or malformed-response failure refuses with no command fallback. Recovery completes and readiness is published before admission. Later capabilities reuse this transport; queueing/events/long waits are omitted initially. After an ambiguous reply the client repeats the same request with the same `request_id` and signature.

### 8.4 Relation to the platform launch contract

The dispatcher implements a **local headless subset** of `dispatch(LaunchRequest)` (architecture §7.1, §5.1): it starts already composed plans; it records the opaque provenance (`board_ref`, `parent`) whole; it refuses interactive transport, native resume, seeded continuity and remote targets (`unsupported_combination`). Interactive executions under agent accounts arrive with the session host's platform phase (its backend gate, §12), not in this dispatcher (owner decision 2026-10-10); goals are optional. For now only the launching caller at start (through the launcher), or the board/orchestrator for board goals, authors a session goal (D-GOALS); an agent account has no goal capability on the session-host socket. This keeps authorship with the initiating caller while a separate agent namespace and its authority bounds await SH2+ design. The host must function without goals, a board, an orchestrator or their toolchain; board goals are an optional client of a generic goal interface. Choosing self-goals in a separate namespace versus caller/board-only authorship remains deferred (§16). A versioned LCH adapter, owned with the launch client, composes plans, compares pins and maps `LaunchRequest` onto these operations; until it exists, nothing claims full LCH conformance.

## 9. Capacity

Figure M-DP2: [capacity ownership alongside allocation and execution claims](../diagrams/agent-claim-states.puml).

| Admission condition | Initial slice | Later queue capability |
|---|---|---|
| temporary capacity gate closed | refuse with `capacity:*` | queue with a durable handle; re-check authority and gates before starting |
| quota exceeded | refuse `quota_exceeded` | refuse `quota_exceeded`; never queue |
| authority invalid | refuse | refuse; never queue |

- **Claims.** A capacity claim is a durable record `{ claim_id, owner record, class, resources, expires_at }`, taken at `capacity`, transferred to the execution at `start`, released after confirmed end/absence, including cancel, expiry and reconciliation; unknown process evidence never releases it. Re-checks before the launch exclude the claim's own resources, so an admitted request never refuses itself.
- **Counters.** Allocations and executions are counted separately, per caller and in total.
- **Authority at admission.** Prepare and every start require a fresh `revocation.check` with the matching `prepare` or `start` context (§3.4); `revoked`/`unavailable` refuse before new effects, including broker restore fencing. Dequeue requires its own `dequeue` check, never a saved prepare verdict. Broker final checks remain independent and mandatory.
- **Gates** (architecture §5.1.4): `slots`; `disk` on every volume the run touches (agents root, dispatcher state, broker store); `cpu_idle` over a window; `hold` (the host-wide hold flag, read fail-closed: unreadable means held); `build_lock` and `syspolicyd` for job classes that build or create executables. The initial slice admits the job classes `agent-headless` and `agent-headless-build`; the second requires the build-lock and syspolicyd gates, which read the host's flags.
- **Queue (later capability).** Temporary gate closures queue a request with a handle; permanent refusals (quota, authority) never queue. The queue is bounded per caller and in total, fair across callers (round-robin by caller), durable, and re-checks principal, signature/window, quota, every gate and a fresh `revocation.check` with context `dequeue` at dequeue. `drain` either parks queued requests or rejects them, as the operator chooses.

## 10. Payload budget

| Item | Bound |
|---|---|
| decoded plan | 48 KiB |
| plan in a request (base64) | 64 KiB |
| launcher request `launch-op/1` | 80 KiB (helper §7) |
| `dispatch/1` frame (all slices) | 128 KiB |

A request whose plan exceeds the smallest bound on its path is refused at `prepare`, before any account exists (`payload_too_large`). Exact-bound and one-over vectors, counted after base64 and UTF-8 encoding, are part of the qualification.

## 11. Installation and activation

Companion figure: [service activation and bounded maintenance](../diagrams/activation.puml).

- **Installation.** The installer creates `swarma-dispatcher` with the helper, places the binary in `/opt/swarma/libexec/`, writes only dispatcher-to-helper/launcher no-argument sudo rules, registers the service principal in helper policy and broker configuration, generates its key, and obtains the operator-issued constructible dispatcher grant (§3.4). It installs the daemon and protected Unix socket using the broker's service/connection rules. There is no caller-to-dispatcher sudo rule, detached-command bootstrap, external sweep timer or command-to-daemon conversion.
- **Readiness.** Start with admission closed, take exclusive state ownership, validate state reader/writer compatibility, replay the journal and reconcile effects, receipts and process evidence. Publish readiness and open admission only after recovery is resolved. Unknown or contradictory evidence keeps admission closed; an absent or unready service refuses without fallback.
- **Bounded maintenance attempt (R-MAINT).** Before a backend-mode change, mutable-state migration or binary swap: (1) close new admission; (2) durably inventory existing executions and selected stops; (3) require explicit operator authority to stop the selected executions through the dispatcher and journaled helper `exec.stop`; (4) require confirmed process absence and reconciliation for all affected executions before changing mode/state/binary. Closing admission or draining queued requests does not stop a running execution. The maintenance deadline bounds this attempt, never a standing execution's lifetime. Unknown/refused stop, uncertain absence, incomplete reconciliation or deadline exhaustion defers maintenance with admission closed; no swap or migration is permitted. Progress and unresolved obligations remain durable for operator review and resumption using the same stop effects/receipts.
- **Retention and restart.** Maintenance keeps standing allocations and identities; it performs no automatic retirement, re-creation, live adoption or lifetime expiry. Persist intended restarts for operator review; a restart needs fresh authority, fresh `prepare`/`start` verdicts and an admitted plan/transport. Unsupported rows stay visibly stopped. Native resume is only available after its §12 gate; maintenance promises no seamless context continuation.
- **Cutover and rollback.** After confirmed absence/reconciliation, take exclusive state ownership, validate/migrate the schema and activate the selected binary/backend. Repeat recovery before publishing readiness. Rollback follows the same fenced absence and compatibility order, retains later durable records and never restores an old snapshot as a repair. A binary refuses unreadable or unsupported state; no live supervisor adoption is allowed. Maintenance authorization, deadline/progress and restart-review formats remain unpublished (§16); their wire fields/codes are not invented here.

## 12. Session-host backend (later; activation gate)

Companion figure: [backend process identities, host admission and recovery](../diagrams/v1-daemon.puml).

Before the SH2+ session-host module may hold executions through its agent-side runner (not the service UID):

- **Host start (R-START).** Only the configured session-host service principal may invoke `host_start` on `dispatch-op/1`, carrying `(logical_session_id, execution_id, controller_epoch, op_id)` and referring to a **host admission permit**. The permit is a durable dispatcher record bound to those four values, created when a session-host-executor start is admitted. It spans physical start; it is not a signed launch ticket or delegated root authority. The host first persists session intent and a closed delivery fence, then asks the dispatcher for the admitted start. A fence change withdraws the permit or sends it to reconciliation.
- **Privileged start and cancellation.** Only the dispatcher invokes `swarma-user-launcher` for agent starts, through its existing claim, distinct launch effect and root-receipt path (§§4.5, 5.3). At `host_start`, it rechecks the original admitted caller authority and a fresh `revocation.check` with context `start`; the permit is no substitute for those checks. The host has no launcher sudo authority or helper host role; no ticket signature/schema or redemption journal exists. Cancellation before the call prevents it. After an ambiguous call, resolve the same effect and root receipt and stop the execution if required; never launch twice. Host failure retains the claim and closed delivery fence until reconciliation.
- **Fences and epochs.** The backend API carries `(logical_session_id, execution_id, controller_epoch, op_id)`; the host owns durable per-session fences (architecture §5.3) and validates the fence/current epoch atomically with each effect. The dispatcher never infers controller authority from a live pid. Controller-only re-adoption of a live runner keeps `execution_id`; actual execution replacement uses a new one and fresh admission. This does not permit live supervisor adoption during dispatcher maintenance (§11).
- **Registration before delivery.** The approved agent-side runner opens a fresh authenticated channel to the host after the dispatcher starts it. Registration in the current controller epoch and readiness precede delivery; a closed fence or stale epoch prevents delivery. The fresh runner still checks the exclusive native-session lock (R-HOLD-OLD/R2).
- **Ownership table.** Admission, privileged starts, records, stop and cleanup stay with the dispatcher; the host service owns durable session metadata, logs, receipts, fences and input admission; `swarma-session-runner` under the agent UID owns the PTY master or per-session app-server transport, stays alive for the execution and owns credential lease renewal. Account generation, execution claim epoch and host controller epoch remain distinct. Each handoff is specified before activation. `curator-run` remains the initial first broker consumer; board/host clients request launches through the dispatcher. Runner integration is a later qualified consumer, not an embedded broker client in the host service.
- **Permit publication gate (I4).** Tickets are removed. The host admission permit reference/encoding, `host_start` authentication, admission durability, fence withdrawal, claim/cancel/crash ordering, root/host receipts, recovery and ownership handoffs would need joint publication and qualification (§16). The backend remains disabled until then; naming these semantics does not publish wire members beyond R-START.
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
| session host (§12) | `host_start`, host admission permit, fences and epochs, cancellation/recovery ordering, registration/readiness before delivery and the ownership table |
| board spawn path | a board-owned adapter that keeps the board's run ids, attempt lineage and recovery successors; distinct dispatcher execution handles; a second completion barrier (an execution's exit never satisfies the board's `wait`); durable, replayable exit delivery |
| key keeper | registration tuples `(uid, generation, identity path, owner, registration generation)` plus the one-shot flag recorded at registration, with active-ledger checks and detachment before UID reuse; the broker's versioned key-migration procedure |
| one-shot executor | R-LR1 versioned narrow `curator-run` intake and Decision 0021 amendment, plus broker credential-injection qualification (§5.3) |

**Identity cleanup (D-IDENTITY).** The dispatcher may revoke only identities it registered as one-shot, with that flag recorded at registration; it MUST NOT infer the flag from an account name or a later retirement request. After confirmed account retirement, it performs identity revocation as a separate journaled effect, reusing that effect ID on retry. Unknown or refused revocation keeps the applicable cleanup obligation pending. A standing orchestrator or fixed team role's account retirement never revokes its identity; only the operator may revoke it with a signed record. This narrow exception does not grant general identity administration. The keeper registration/revocation API and broker migration procedure must exist before enabling this integration.

## 15. Qualification

On hosted CI runners only, through the deployed clients (`curator agent-user` and `curator-dispatch`), the slice-0 `dispatch/1` socket and exactly the deployed dispatcher-to-helper/launcher sudoers rules. These are qualification requirements, not locally executed evidence:

- `dispatch`, and `provision` then `start` twice in sequence, with the broker's slice 0 and one Claude launch each;
- authority: a copied chain from another caller, a missing or wrong signature, an unregistered key, an unconstructible request, a revoked caller grant (the agent's next lease refused), a recycled UID;
- crashes on both sides of every acknowledgement (input, effect intent, effect result, claim, supervisor registration, start receipt, termination receipt), each followed by reconciliation;
- cancel in every state; ordinary execution `max_runtime`; ordinary allocation expiry with and without the sweep; a lost supervisor with escaped descendants;
- standing allocations: no expiry/lifetime quota; sweep never expires them; caller `agent.spawn` cannot retire, re-create or bring up the team; absent or invalid operator signatures authorise no team effect; operator-signed retirement/re-creation only;
- standing runtime: start without `max_runtime` creates no runtime quota or `deadline`; with the supervisor and the activated session-host backend, advance beyond both 4 h and 12 h and run repeated sweeps and recovery; no runtime stop effect or `expired` outcome, no deadline introduced on recovery;
- standing start with any supplied `max_runtime` refuses before launch and creates no runtime deadline; refusal-code assertions await publication (§§8.1, 16);
- ordinary runtime: omitted `max_runtime` retains the 4 h deadline, an admitted explicit 12 h retains that deadline, and a value above 12 h refuses; sweep after each admitted deadline invokes `exec.stop` and records runtime expiry only after confirmed absence; the standing exception never admits an over-limit ordinary execution or suppresses its runtime stop;
- standing termination: explicit stop (`cancel`/stop through `exec.stop`), operator action, own exit and incident hard stop still reach confirmed absence and applicable cleanup; exemption from runtime never suppresses these paths;
- team bring-up after wire publication: repeat and crash recovery per role reuse recorded effect IDs without duplicate allocations; unknown/contradictory observations remain pending; provisioning failure and authority loss do not automatically retire standing allocations;
- capacity: quota 1 admits its own request and refuses a competitor; each gate; hold read failure;
- payload bounds at, and one over, each limit;
- service from slice 0: peer authentication, server UID verification, protected endpoint, no inherited/passed service descriptors, absent/unready daemon refuses, recovery before readiness and no command fallback; later capabilities: queue fairness, dequeue verdict, events/wait gates and drain;
- maintenance: authorized selected stop of a standing execution within the bounded attempt, retained allocation/identity and fresh authority for restart; unauthorized stop, unknown/refused stop, escaped/live process, deadline exhaustion and incomplete reconciliation defer mode/state/binary change; rollback retains later records and verifies compatibility, never adopts a live supervisor;
- delegation bootstrap (before recursive activation): C != D with a valid C-to-O chain, dispatcher-signed/wrong issuer refusal, insufficient depth/overlong chain refused before create, pre-key replay/mismatched key or intent/old generation refused, rotation replay cannot restore the old key, missing C signature leaves O without spawn authority; revoke C before O activates and after O creates W, and refuse O/W new admission and subsequent lease/renewal under the retained ancestry; delivered tokens retain broker lifetime limits;
- revocation verdict: revoked before prepare/dequeue/start, unavailable broker, stale prior verdict and fenced restore all refuse before new effects; final bind/lease/renewal still check independently after a prior `not_revoked`;
- effect observation: all six states and exact §4.6 mapping; crashes before intent, during effect, after durable result/before acknowledgement, rolled-back create, complete not-recorded evidence, missing/expired/unreadable evidence, contradictory results and reconciliation after caller/grant expiry; effect absence never creates automatically or releases capacity without process absence; bind/unbind removal/tombstone retention and unauthorized/digest-mismatched lookup;
- host backend before activation: only configured host principal can use `host_start`; mismatching permit values, stale epoch, fence change/cancellation before call, host failure, ambiguous root call, receipt recovery and required stop never produce a second launch; fresh runner channel, current-epoch registration and readiness precede delivery;
- retirement: confirmed quiesce/declared-key cleanup and home deletion, closed-schema service metadata only, no automatic retention; explicit operator retention states confidentiality is not guaranteed; no standing identity revocation;
- R-FD3 launcher qualification is owned by the user manager: maximum admitted plan staging, launch/stop/retire interlocks, staging/drop/exec failures and absence of internal handles in the executor must be qualified before dispatcher activation.

## 16. Open questions

1. **Retirement policy (F-1; CLOSED — owner-decided 2026-10-12):** ordinary retirement deletes the home after quiesce and declared-key cleanup. Dispatcher cleanup admits only `delete`; retention is a separate explicit operator operation with no confidentiality guarantee (§5.4). The user manager would need to publish its retention operation and closed-schema service-metadata export; no general secret classifier or archive-by-default alternative remains.
2. **Helper v2 signed requests (contract W3; OWNER DECISION NEEDED, user-manager/contract owners):** publish a component-specific format or a versioned shared adapter. D-WIRE-NAMES supplies the naming rule but not the signature domain/version and canonical bytes. The helper's initial contracts use authenticated dispatcher sudo caller and role policy, not an invented signed-grant verifier.
3. **Host admission permit (contract I4 / R-START):** launch tickets and the old launcher-redemption resolution are removed by the owner decision of 2026-10-12. Joint publication would need the durable permit encoding/reference bound to `(logical_session_id, execution_id, controller_epoch, op_id)`, configured-host authentication, signature/canonical bytes where required, admission and fence-withdrawal durability, claim/cancel/crash ordering, root/host receipt observation, registration/readiness, recovery and ownership handoffs. Exact refusal mappings would need publication; no new code is invented. The SH2+ backend stays disabled (§12).
4. **Agent self-goals (contract G2; deferred SH2+ owner decision):** (a) a separate agent namespace with its own bounds, or (b) retain caller/board-only authorship. Neither requires goals or a board for host operation (§8.4).
5. **Log-storage handoff:** retain dispatcher execution logs or transfer them to the host's durable log store when the backend activates; the ownership handoff must be specified first (§12).
6. **Standing-session execution runtime (R3; CLOSED — owner-decided 2026-10-12):** the owner decided that the orchestrator and standing team members (other orchestrators and standing agents) have no predefined allocation expiry or maximum execution runtime; only an explicit stop or the operator stops them. This supersedes the R3 proposal and applies to executions of standing allocations in all backends, not only SH2+. They have no `max_runtime` quota or runtime deadline, and the sweep never stops them for runtime (§§3.5, 4.4, 5.3, 7.2). Own exit and incident hard stop still end an execution through confirmed absence. The 4 h default and 12 h maximum apply to every other execution.
7. **Orchestrator child-plan access:** which profiles and environments may an orchestrator read and use for its children from its agent account, and how does its narrowed grant constrain that access? Caller-side composition with the orchestrator's own home configuration is decided (§5.2); the read/access policy is unspecified.
8. **Standing/team wire publication:** publish the standing designation in `agent/1`, the operator-signed retirement/re-creation record, and the remaining envelope, signature grammar, transport, role matching and change semantics for the specified-but-unpublished `swarma-dispatcher team/1` (§§4.3, 5.4–5.5). Only per-role `label`, `profile`, `harness`, `accounts` and `capabilities` are specified here; no additional fields or values are asserted. Publish the refusal-code mapping for `start` with `max_runtime` on a standing allocation (§8.1); if an existing code cannot represent this refusal, it would need a published refusal code. No code is invented here; the runtime policy itself is decided in question 6.

9. **Orchestrator bootstrap (R-DELEG):** would need the generation-bound enrollment intent, pre-key proof-of-possession domain/canonical bytes, authenticated freshness/replay window and identity, rotation/operator-recovery authorization, requester-signed delegation installation operation and durable activation/acknowledgement ordering (§§3.3–3.4, 5.2, 8.1). Depth/chain preflight and requester ancestry are fixed; wire members and refusal mappings remain unpublished. Recursive provisioning stays refused until publication and qualification.
10. **Revocation verdict (R-VERDICT):** would need exact `broker/1 revocation.check` chain/context and sequence encoding, configured-dispatcher authentication, bounded response/freshness handling and dispatcher refusal-code mapping for `revoked`/`unavailable`, including restore fencing. Contexts and verdict names are fixed (§3.4); final broker checks stay mandatory.
11. **Receiver observations (R-OBS):** would need exact `effect.observe` encoding, canonical argument digest format, original-caller/operator authentication and journal serialization, helper and broker retention floors, matching unbind removal/tombstone retention and lost-evidence behavior. Six states and their mapping are fixed (§§4.5–4.6, 7); expired/unavailable evidence never proves absence.
12. **Maintenance authorization (R-MAINT):** would need operator-signed selected-stop authorization, bounded attempt deadline/progress persistence and restart-review formats, interruption/resumption ordering and reader/writer compatibility qualification (§§8.1, 11). No maintenance deadline limits standing runtime.
13. **Service capability publication / R-FD3:** later queue/events/long-wait negotiation and unsupported-capability refusal mappings would need publication on the same `dispatch/1` transport. User manager §7 would need the per-platform fully staged, bounded immutable read-only FD3 mechanism (or qualified pipe capacity/feeder/deadline/cleanup ordering) and lifecycle qualification before launch activation; this dispatcher references that single contract (§5.3).

The executor and first consumer are decided by D-EXECUTOR, managed PATH by D-PATH, wire names by D-WIRE-NAMES, and resume compatibility by D-RESUME; their older open questions are superseded. Default quotas and ordinary allocation lifetimes are already decided and configurable (§3.5); the owner decision of 2026-10-12 closes R3: standing allocations have no expiry/lifetime quota and their executions have no runtime quota/deadline. Every other execution retains the 4 h default and 12 h maximum. Interactive execution remains gated on the session host. Donor provisioning belongs to a separately specified small privileged component, not a user-manager extension (D-DONOR-PROVISIONING); variant B stays disabled until that component is specified, and remote targets remain outside this subset.

## Appendix A. Refusal codes

`caller_not_permitted`, `caller_key_not_associated`, `caller_key_pending`, `signature_invalid`, `request_invalid`, `request_too_large`, `request_timeout`, `request_id_conflict`, `payload_too_large`, `plan_digest_mismatch`, `grant_invalid`, `grant_unsupported`, `grant_subject_mismatch`, `grant_not_delegable`, `grant_unconstructible`, `quota_exceeded`, `capacity:slots`, `capacity:disk`, `capacity:cpu_idle`, `capacity:hold`, `capacity:build_lock`, `capacity:syspolicyd`, `expired_in_queue`, `agent_busy`, `agent_not_found`, `agent_not_owned`, `agent_retired`, `run_not_found`, `run_not_owned`, `run_terminal`, `unsupported_combination`, `observation_unknown`, `state_version_unsupported`.

## Appendix B. Diagrams

PlantUML sources are design views of this spec, not implementation or qualification evidence. The audit's module rows map to:

| Audit row | Figures | Defining sections |
|---|---|---|
| M-DP1 | [Components](../diagrams/components.puml), [dispatch lifecycle](../diagrams/dispatch-lifecycle.puml) | §§2–6, 9, 12, 14 |
| M-DP2 | [Run states](../diagrams/run-states.puml), [agent and claim states](../diagrams/agent-claim-states.puml) | §§4.3–4.6, 5–7, 9 |
| M-DP3 | [Storage layout](../diagrams/storage-layout.puml) | §§4.1–4.5, 6, 8.3, 11 |

Companion views: [bring up team](../diagrams/bring-up-team.puml) (§5.5), [reconciliation](../diagrams/reconcile.puml) (§7.1), [service daemon / gated SH2+ backend](../diagrams/v1-daemon.puml) (§§8.3, 12) and [activation / maintenance](../diagrams/activation.puml) (§11). Unspecified details are marked in the figures; their notes do not introduce contract behavior.
