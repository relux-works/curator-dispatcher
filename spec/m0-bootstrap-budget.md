# DRAFT: dispatcher contribution to the B0–B5 installation budget

Status: **bounded proposal for owning-lane and independent review; admission
closed**. Keymaker owns the complete installation. These are candidate dispatcher
admission ceilings, not approved numbers, measured service guarantees, a protocol
or qualification. No implementation, installation or activation is authorized.
MUST and MUST NOT describe only the proposed contract.

## 1. Pinned basis and preserved scope

The normative partner input is [Keymaker §4.1 at 1453290][budget], source SHA-256
`ba785b2f13cff37478d6ced975eacef559cfdf10c44ebb6943832cee89dca607`, cumulative
PR54 diff SHA-256
`335e156362f8cc7d518bd39d43978b28512f14528eef06faffd65831f8176e1e`.
Earlier a5981e4/c4b90c3/beabd39 are historical. The [conditional bootstrap
publication at 79b37d2][bootstrap] remains unqualified and unactivated; 1453290
supplies its budget and caller/worker delta. Publication is not deployment or
owner acceptance of these ceilings.

The [existing-worker M0 candidate](m0-launch.md) remains the local authority
contract: ordinary create -> durable active -> authorized register-after-active
(setup A), one caller association, one explicitly mapped existing worker
UID/generation, one dispatcher/scope and one policy lineage. Caller and worker
may differ; equal accounts require an explicit protected-policy selection too.
No signature, lease or association authorizes its own installation. The
independently qualified operator control/installation entry is unpublished.
No wire member, method, acknowledgement, receipt or refusal code is allocated.

Preserve launch C/Rv/expiry/residual/P7, retained reduce-only owned cancel,
WC1/common synchronous independent G17 with separate importing histories/approved
genesis and lifetime fence, worker-owned broker I/O/no dispatcher observer and
actual policy horizon (b). The full specification, closed schemas, 20 JSON
fixtures and 904-byte specimen are unchanged. Linux/macOS remain native-gated;
the separate operator service root retains its own authority.
The separate G2 credential-final-release policy-classification coordination gate
remains open; this installation-budget proposal does not select or close it.

## 2. Original clocks and complete phase endpoints

Retain one original authenticated operator occurrence and canonical arguments
before lookup, preparation, confirmation or reservation: operator UID/generation,
caller target UID/generation and purpose-1 binding, public stage/path/key/epoch,
association and receiver/scope, worker UID/generation and selected mapping,
policy/catalog/adapter/qualification tuple, exact predecessors, activation decision
and durable outcomes. These are semantic bindings, not added M0 schema fields.
Recreation of either role requires fresh explicit association/policy installation
and worker setup verification; equal public bytes do not reactivate anything.

Use original Keymaker B0 entry **t_B**, before configuration/provider lookup,
locks, closure, human confirmation or reservation, and **D_B = t_B + 120 s**.
Intersect shorter original caller/authority/component deadlines. Checked,
suspend-aware elapsed arithmetic and independently qualified authority
time/continuity are both required; unavailable original boot/deadline holds.
For phase i, e_i is its original first entry, A_i its allowance and C_i its
cumulative cutoff. Every complete dispatcher endpoint must satisfy
**endpoint < min(e_i + A_i, t_B + C_i, D_B, shorter original limits)**.
Equality refuses; message receipt or successful reservation never starts a timer.

Recommend retaining the partner phase ceilings as necessary dispatcher ceilings,
with the endpoints below. Each is a **shared phase envelope**, not independently
consumable dispatcher service time or a fresh per-call allowance. B1 is included
only to account for the original schedule.

| Phase | A_i (s) | C_i (s) | Endpoint rule | Work charged through endpoint |
|---|---:|---:|---|---|
| B0 | 10 | 10 | < | Dispatcher closure/drain-or-containment, original intent/capacity and durable closure evidence; operator checks, Keymaker reservation and delivery/retention share this envelope. |
| B1 | 10 | 20 | < | Keymaker custody/stage/result/audit and protected handoff; no dispatcher installation or delegated work scheduled here. |
| B2 | 20 | 40 | < | Inert exact-predecessor association install, durable outcome/audit, authenticated acknowledgement, permitted exact lookup/retry and Keymaker local retention. No authority opens. |
| B3 | 20 | 60 | < | Joint exact association/revision/currentness verification, original-outcome reconciliation and both owning checkpoint/audit retention. |
| B4 | 40 | 100 | < | Dispatcher-only policy installation/currentness, caller association plus independently pinned worker/mapping, owning outcome/audit and acknowledgement/lookup/retention. Broker share is zero. |
| B5 | 20 | 120 | < | Both owners' final currentness/setup-evidence validation, complete durable outcome/audit/custody/barriers AND every actual applicable admission opening. |

The allowances total 120 s; this is not proof of feasible joint scheduling.
The nominal 80 s for B2–B4 cannot mean 80 s of dispatcher work plus free human,
transport, Keymaker or custody work. This proposal objects to that interpretation.
No internal RPC topology or nominal timing split is invented to make arithmetic
green. Before acceptance, enumerate actual serial/parallel preparation, delivery,
local/dependency work, human confirmation, retention and final openings, and show
accepted complete bounds fit every original phase and D_B at maximum admitted
load. Unknown partner/final-endpoint work leaves the joint envelope incomplete.
A measured failure requires the workload and a reviewed whole-operation revision,
not merely a higher ceiling. E-I 60 s, E-C 10 s, native 2 s and approval TTL supply
no bootstrap allowance; shorter applicable original limits still win.

Human think-time, retries, exact status lookup, reconnect and reconciliation
consume the same original phase/installation bounds. Unused allowance cannot
transfer. Begin none unless its accepted complete bound fits strictly within
remaining original limits. Retain the original occurrence/canonical arguments,
predecessor and counters. B2/B3/B4 finish before t_B+40/60/100 s; B5 is not
overflow installation/retry time.

## 3. Actual opening, all-writer exclusion and unknown debt

All required owning state, full originals/results, audit, independent custody
confirmations and checked file/namespace durability barriers MUST precede opening.
The same bounds cover **durable outcome AND actual opening**, not a success
marker, accepted write, scheduled callback or response deadline. After each sole
qualified final read, the positive enforced remaining act/use/outcome/opening
bound covers pauses, storage stalls and release ordering. A timer, cancellation
or post-write error alone cannot prevent a late effect.

Every installer, invalidator, recovery, restore, prepared and old-instance writer
participates in durable close -> install -> verify -> activate exclusion. Close
affected scope and drain/contain prior work before replacement; retain original
occurrence, exact predecessor and nonregressing revisions. The selected separate
lifetime fence prevents late writes/effects/openings, not just late responses.
Acquire external evidence/acknowledgements outside final state/generation locks,
then revalidate the exact attempt under local exclusion. No RPC/external wait
under those locks. A refreshed receipt, second read, new use admission or fresh
ID cannot repair expiry. Each service consumes the same protected activation
decision and its own current records without waiting for another service's open
bit. No atomic cross-service commit is claimed. The concrete exclusion/protocol
and enforced post-read endpoints remain unpublished and unqualified.

At any cutoff retain original partial/unknown effects, provenance, capacity,
holds, closed affected signing/new-start authority and stop/recovery debt. Lost
response is unknown, never no effect. Partial on-time opening needs containment
and reconciliation, not a joint-success claim. Late/duplicate acknowledgements
can establish evidence, never open expired authority. Proven full durability and
all openings on time followed only by lost notification is the original success;
it is neither refunded nor activated twice.

Separately authorized recovery needs its own finite original scope/budget to
establish/contain only the old outcome, never activate the expired installation.
A fresh explicit operator decision is possible only after old effects/holds are
conclusively resolved under the accepted recovery protocol. No new ID/Run/epoch,
process/host, compensating grant/key/policy, rollback, prune, refund or rearm clears
debt. Stop deadlines and retained cancel consent preserve their prerequisites and
original clocks. The 120 s ceiling is not policy lifetime or a P7 extension.

## 4. Independently proposed dispatcher load ceilings

These are **aggregate candidate ceilings for one original installation across
all dispatcher phases**, not per request, phase or retry:

| Quantity | Candidate ceiling | Charging boundary |
|---|---:|---|
| Dispatcher-attributable byte-work | 2 MiB | Every read/validation/copy/write/transfer and repeated access, including full originals, state/results, acknowledgement evidence, audit, cleanup and custody. |
| Dispatcher-attributable dependency invocations | 32 calls | Every initial/retry/status/reconnect/reconciliation and nested delegated generation/time/custody/audit invocation; no cache, retry or one-logical-call exemption. |
| Retained dispatcher installation inventory | 512 KiB | All original/partial/unknown state, replay/results, audit, receipt, custody and cleanup inventory, including the existing <=64 KiB control/result subset; no evicting debt to fit. |

Rationale: one small lineage limits repeated byte-work to 32 times the existing
64 KiB control subset, with an aggregate retained ceiling eight times that subset
to leave finite room to specify other inventories. The 32-call cap limits total
amplification rather than granting each retry another allowance. These are
deliberate admission choices, not estimates that unpublished formats/dependencies
fit, throughput measurements or maximum-load proof. They do not copy Keymaker's
local 8 MiB/16-call caps or borrow its owning 256 MiB state budget.

The 512 KiB includes the 64 KiB subset; it allocates no additional store or owner
allowance. Before work, enumerate/reserve the full actual inventories against
separately published owning capacities. Unallocated audit/receipt/cleanup storage
remains closed even if a hypothetical sum fits. Frame/preimage bounds do not
change. No batch, second target/worker or arbitrary inventory/history scan.
Full originals are charged; truncation/pruning cannot make an oversize install fit.

The ceilings include all dispatcher-attributable nested work, but **the actual
complete joint workload is unknown**: encodings/versions, participants, fan-out,
per-invocation byte/endpoint bounds, repeats and state/barrier/custody inventories
are unpublished. Every dependency owner must bound its internal work, charged
without pretending one outer call covers it. A transfer can charge both owners'
local caps without additive state allowances. Keymaker work, human steps and
separate setup also require enumeration. Unknown inventory, participant, byte or
call totals is unqualified, never zero, unlimited or one logical call. Unknown or
over-cap work refuses before new effects while retaining earlier evidence/debt.
Changed bounds/source/test/configuration/environment invalidate exact qualification.

## 5. Pre-B0 setup and worker status boundary

Broker policy Replace is separate one-time operator setup **before B0**, with
admission closed and no active leases anywhere on that broker: it closes all
connections and invalidates every old-generation lease. The operator replaces
the protected file and signals reload; no broker install/confirmation API exists.
No-error reload is insufficient. Operator prohibition of unsafe reload is not a
claim that the daemon rejects it.

Existing peer-scoped v0 `status` runs under the exact kernel worker UID and proves
expected current `policy_generation` plus required allowed account, without
token/lease payload. Pair with a foreign UID's empty/denied result, including the
caller UID when distinct. Broker checks peer UID, not caller-to-worker mapping.
Dispatcher/integration setup owns that explicit qualified protected mapping:
retain both roles at B0, pin both in B4, independently recheck both and setup
evidence at B5 and at ordinary prepare/start. Same-account selection is explicit
too. UID-only allowlisting cannot prove generation; caller/D status is no substitute.

`lease.request` is forbidden as a setup probe: it issues a real vendor token whose
lifetime release cannot revoke. These controls create no token/lease debt. Any
forbidden probe's real debt must be retained/contained, not relabelled read-only
or erased by release. Status controls and their qualification belong to the
separate setup budget, not B0–B5. No broker operation, including reload, status or
confirmation, occurs in B0–B5; only independently qualified retained setup evidence
is consumed. Its currentness through activation and exclusion of policy drift
remain joint gates.

Broker/operator owners still owe finite original setup entry/deadline, complete
file-write/reload/status-positive/status-negative/evidence endpoint, human time,
aggregate bytes/calls/nested work, retry and unknown-outcome recovery bounds.
This contribution assigns none of those owners' numbers and objects to free or
unbounded setup. The earlier proposed 5 s broker-daemon work is not that complete
endpoint. Setup cannot borrow B4 or the bootstrap clock. Unknown setup or policy
drift closes B0; replacement remains separate and forbidden with active leases.
No setup/control is executed here.

## 6. Adjacent inert contract witnesses

All rows are **PENDING runtime / NOT RUN**, synthetic semantic witnesses, not
runnable control grammar, tests or qualification. Times are seconds relative to
t_B=0; shorter original limits still win. First-entry examples are not scheduling
reservations. Each negative preserves the other positive facts unless changed.

| Obligation group / future test | Adjacent valid witness | Negative witness / forbidden outcome |
|---|---|---|
| BB-1 OriginalClock | B2 first entry 20, complete endpoint 39 < min(40,40,120). | Endpoint 40 equals cutoff: refuse. Retry at 39 cannot restart B2 or borrow B1 time. Shorter original limit 38 excludes 39 too; human/suspend time before reservation counts. |
| BB-2 ActualOpening | B5 first entry 100, full outcome/audit/custody/barriers 118 and every opening 119 < 120. Notification loss cannot repeat activation. | Same marker 118 but opening 120/121 cannot succeed. Post-read pause/prepared/old writer cannot bypass enforcement; post-write error is insufficient. |
| BB-3 UnknownDebt | Exact authenticated B2 lookup proves original outcome/currentness and retention at 39 without duplicate install. | Lost acknowledgement at 40 retains hold/capacity/debt. Late ack/fresh ID cannot reopen; expired recovery only proves/contains. Partial B5 opening is not full success. |
| BB-4 CompleteLoad | Enumerated retained inventory <=512 KiB, all byte-work <=2 MiB and nested invocations <=32 meet only candidate load ceilings. | 1 MiB accessed three times exceeds work; 32 outer plus one nested call exceeds calls; 512 KiB plus one retained byte exceeds inventory. Unknown inventory/participants refuse even below known local counts. |
| BB-5 CallerWorker | Distinct caller/worker independently current pins and explicit mapping; worker status succeeds, distinct caller empty/denied. Explicit selected same-account mapping also permitted. | Caller status/key/generation substituted for worker, swapped/stale roles or inferred equality cannot admit. Either recreation needs fresh policy/setup. |
| BB-6 SetupNoToken | Before B0: closed admission/no active lease, worker status expected generation/account without payload; foreign control empty/denied, all inside accepted separate setup bounds. | No-error reload, wrong generation/account, allowed foreign UID (including a distinct caller), missing setup bound or active lease at Replace cannot satisfy setup. lease.request probe issues token and is forbidden, never read-only. |

Production coverage: **0 of 6 obligation groups driven**. Production call sites
for BB-1 through BB-6 are **absent**: no dispatcher operator bootstrap, owning
install/lookup/currentness, opening or setup-evidence consumer is implemented.
Existing broker status is a source dependency, not an executed control or
dispatcher call site here. Documentary arithmetic/source checks prove no runtime
refusal, real signature, late-writer exclusion or qualified workload.

## 7. Outstanding acceptance inputs

Keymaker must review common phase ceilings, dispatcher aggregate caps and the
objection to a supposedly complete 80 s share. Dispatcher/Keymaker must publish
closed protected operator/install/ack/lookup/replay/recovery formats, mapping and
currentness, store allocations, all-writer exclusion and post-read enforcement.
Broker/operator must publish separate setup bounds and qualify worker/foreign
status evidence without token issuance. Custody/time/generation owners owe bounded
complete dependency work and exact native evidence. Missing inputs keep closed.

Exact Keymaker, broker, integration and Architect consumer review, independent
review, curator/budget-owner decisions, complete maximum-load/native qualification
and explicit operator acceptance/activation remain gates before their respective
implementation and activation stages. Source publication, documentary checks or
this producer proposal discharge none. No runtime build/test result is claimed.

[budget]: https://github.com/relux-works/swarma-keymaker/blob/14532907e28cbfda6d049955e5e2b7de0bf42554/spec/keymaker-station-caller.md#original-installation-budget-and-complete-endpoint
[bootstrap]: https://github.com/relux-works/swarma-keymaker/blob/79b37d2d0a6fbacae1174ec97c8f07d09681fe00/spec/keymaker-station-caller.md#41-m0-operator-bootstrap-transaction
