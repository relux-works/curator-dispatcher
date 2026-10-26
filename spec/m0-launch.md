# DRAFT: bounded existing-worker launch authority

Status: **proposal for joint acceptance; unavailable and inactive**. MUST and
MUST NOT below describe this candidate, not shipped capability. This supplement
proposes one closed M0 subset under the `dispatcher-request/1` signing profile.
Keymaker, Architect, broker and integration must review the exact candidate;
curator and independent review precede runtime code. Owner acceptance is still
required before protected operator installation or activation. Neither a valid
fixture nor a schema publication satisfies those gates.

## 1. Scope and relation to the full contract

One nonfederated trusted Linux worker, one existing active UID/generation, one
immutable worker plan, one physical launch maximum, and owned-run cancel. There
is no provision, standalone start, delegation, renewal, restart, pool, session,
remote target, untrusted placement or automatic retirement in this subset.
macOS remains separately native-gated. All applicable confinement, network,
capacity, export/custody, time, restore and operator guarantees in the
[main specification](dispatcher.md) remain.

The proposed protected policy substitutes **only in this separately selected
subset** for the full path's caller/dispatcher grant chains and fresh broker
`revocation.check` launch-authority dependency (main §§3.1–3.4). Activated broker
v0 supplies a lease for an exact operator-allowlisted worker UID. It supplies no
caller authority, freshness verdict or binding. M0 neither calls a fictitious
broker grant/bind/revocation operation nor produces an empty or fake grant.
The full grant-bearing `dispatch-op/1` grammar and its acceptance remain closed
and unchanged. An authenticated key association is necessary, never sufficient
launch authority. There is no fallback flag or credential-copy path.

Before dispatch, the operator performs ordinary `user.create`, waits for durable
`active`, then ordinary Keymaker register for that exact active/unbound
generation, installs the exact v0 UID allowlist and protected association/policy.
Provision/confirm is bump-only with a removed predecessor; M0 cannot fabricate a
predecessor or import pre-active provisioning. A lease-only worker does not need
a Keymaker key merely to execute; any selected enrollment is still operator-only.
Caller and worker are separately pinned and may be distinct. **No qualified
first caller is established by this proposal.** Enrollment, time/custody and
purpose-6 bootstrap staging/currentness remain owning-lane gates. Dispatch never
creates an account, adopts/rebinds one, changes generation or adds an allowlist
entry. A recreated generation requires fresh explicit operator setup.

Purpose-6 registry publication is not required for this local association. That
does not waive actual QT-2c/G17 custody, freshness and old-instance exclusion
dependencies imported by the selected bootstrap/time path. Local approval or
watermarks cannot stand in for their independent evidence.

## 2. Closed proposed forms and signing

The complete proposed shapes are [request](../schemas/m0-request-1.schema.json),
[policy](../schemas/m0-policy-1.schema.json) and
[protected signing plan](../schemas/m0-signing-plan-1.schema.json), JSON Schema
2020-12. All object members are required and closed; only the two operation
branches differ. Unknown or cross-operation members refuse, including nested
ones. These are new proposal discriminators, not an already negotiated socket
capability or a relaxation of the full grammar. Selection/version negotiation
and receiver results remain publication gates; no incomplete response grammar
is enabled here and no `hello` or `method` member is imported into a request.

| Form | Exact semantic meaning, in addition to the schemas |
|---|---|
| Common request R | `v=1`, `kind="swarma-dispatcher m0-request/1"`, random 128-bit lowercase-hex `request_id`, exact protected `receiver` instance identity, `policy` ID/revision, complete `association`, existing `worker` tuple and exact `plan` pins. |
| Dispatch | Common R plus `op="dispatch"`; consumes the single launch budget for that existing worker. Unlike main §8.1 combined dispatch, it has no provision members, grants, caller bytes, command, arguments or environment. |
| Cancel | Common R plus `op="cancel"`, `dispatch_request_id` naming the original owned dispatch. This lookup resolves only that original run/effect; it is not a caller-selected PID or a new run selector. All pins equal the original record. |
| Auth | Wire request adds exactly `auth={key,sig}`. Key equals the protected association key; signature verifies R. Schema-valid example signatures do not verify. |
| Association | Protected ID/revision, kernel caller UID, independently active generation, purpose-6 public key, Keymaker registration number and master epoch. Caller-supplied copies must exactly match current protected evidence and kernel peer; they select no identity. |
| Plan pins | `worker_plan`, Curator `profile`, `harness`, `exec_profile`: exact content-addressed identities, no path/URL/wildcard. Protected catalog pins authenticated immutable origin, full typed bodies and ownership. Digest equality alone never validates an opaque executable payload. |
| Policy | Protected ID/revision, receiver, full association, worker, plan pins, protected `catalog` and exact accepted `qualification` identity; operations exactly `["dispatch","cancel"]`; one worker/start and finite time/stop bounds. It is local authority data, **not** an owner-signature/install envelope. |
| Signing plan | Exactly `v`, `kind="swarma-dispatcher m0-signing-plan/1"`, `profile="dispatcher-request/1"`, unsigned `request`. Native `plan_id` is `sha256:` plus lowercase SHA-256 of CCJ-1 of this entire object, without a prefix or final LF. This is dispatcher adapter data, not invented Keymaker bootstrap fields. |

Keymaker resolves the protected typed signing plan referenced by
`sign_station_caller {profile,plan_id}`, independently validates its full body,
digest, authenticated origin, current authority and receiver, constructs R and
its preimage itself, and returns `{key,sig}`. The adapter independently resolves
the same validated values, assembles matching R and inserts auth without changing
R. It does not submit a free-form body for signing. The dispatcher independently
validates the same schema, catalog provenance and protected authority source.
The worker-plan/executor intake, exec-profile, catalog provenance/install and
receiver-internal encodings must be fully published and jointly selected before
use. A reference is not permission to admit an unpublished body as opaque bytes.

Use registrar [CCJ-1 §2.1 at 5f44ebf][ccj], with its public [ccj][codec] and
[envelope][envelope] boundaries, **not Keymaker native JSON spelling**. Reject
duplicate names at every depth (including escaped aliases), invalid UTF-8/BOM,
lone surrogates, non-integer numeric tokens, negative zero and integers outside
the CCJ safe range; depth is at most 16. CCJ-1 sorts names by Unicode scalar,
preserves array order, applies no Unicode normalization, emits shortest integer
spelling and only its prescribed string escapes. Input whitespace/order can
differ; the resulting signed semantic object cannot.

```text
preimage = ASCII("swarma-dispatcher request/1") || 0x0a || CCJ-1(R)
R        = request with only top-level auth omitted
```

There is **no final LF**, recursive auth stripping, digest substitution,
envelope wrapper, native frame, SSHSIG or caller-chosen domain. Whole preimage
MUST be <=24576 bytes. Ed25519 key is `ed25519:` plus canonical unpadded
base64url of 32 bytes; signature is canonical unpadded base64url of 64 bytes.
Decode/re-encode, strict key/point/scalar and signature-equation verification
remain mandatory beyond shape validation. Keymaker's <=65536-byte native frame
is independent; dispatcher keeps main [§8.3](dispatcher.md#83-socket-dispatch1)
framing, connection/peer/FD rules and 128 KiB frame cap. Neither frame cap
enlarges the preimage bound. The registrar envelope API does not accept a new
dispatcher domain merely because its framing is similar; no API extension is
claimed by this reference.

### Proposed finite bounds

These choices require joint acceptance; they are not owner-decided facts. IDs
are 32 lowercase hex characters to match a 128-bit generation/correlation space;
digests are `sha256:` plus 64 lowercase hex characters. UID range is the existing
agent range 30000–59999, never authority to admit the range. Revision, registration
and epoch integers are 1..9007199254740991. Policy timestamps are integer Unix
seconds in 0..253402300799, avoiding alternate time spellings; qualified authority
time remains required. Require `not_before < not_after` and window <=3600 s.
Maximum runtime is 1..3600 s, additionally cut by policy expiry/removal. M0 is an
ordinary bounded run, not a standing allocation or a change to standing-runtime
rules. One hour limits this candidate's exposure and state horizon rather than
claiming an existing deployment default.

One selected policy lineage, one worker, one admitted dispatch ID and one cancel
ID; same-ID retries reuse records. Further IDs refuse before reservation (one
dispatch already consumed) or reuse the original owned stop through authorized
recovery. Protected catalog has at most two signing plans, one per operation,
with the cancel target pinned to the selected dispatch ID. Reserve <=64 KiB
durable control/result state for this lineage, **in addition to** separately
bounded receiver, audit, receipt and cleanup stores whose complete accounting
is a gate. Tombstones/claims cannot be evicted on expiry or capacity pressure;
refuse forward work when reserves cannot cover the full transaction. This is not
an unlimited request log or an invented share of Keymaker's owning-store budget.
SC-Q6 must separately allocate and fence that store. Bounds on internal evidence
must be selected before activation, not filled with unbounded opaque records.

## 3. Protected installation, currentness and ordering

Only the protected operator path may install, replace or remove policy,
association, catalog or qualification pins, after exact-content owner acceptance.
Agent-writable files, a signed request, an active station, key availability and
allowlist membership cannot modify authority. Atomic policy update/removal must
durably advance a monotonic revision/tombstone and invalidate all consumers.
Reusing an ID/revision or reinstalling an older body refuses. The single launch
debit belongs to the policy lineage **and worker generation**; editing revision,
changing an ID or uninstall/reinstall cannot replenish it. A fresh launch scope
requires new explicit operator setup/acceptance, outside this subset.

Local high-water marks cannot detect coherent rollback. Restore/reboot/lost
counter or uncertain currentness holds forward work until main §11.1 external
fence, receiver coverage/inventory and independent restore evidence are reconciled.
Neither a newer wall clock nor an internally consistent snapshot reopens it.
Use main §3.4 qualified time, durable watermark and boot-inclusive monotonic
elapsed checks; unknown time closes forward work and time-based reclamation.
Revocation of association, epoch, plan/provenance, qualification or tuple also
closes forward admission and initiates the same retained stop obligation.

1. **Prepare.** Acquire external peer/generation/association/time/catalog/policy
   evidence outside the short dispatcher state lock. Re-read current protected
   policy and exact revisions. Independently validate R, signature, currentness,
   catalog and exact worker active generation, confinement/capacity and tuple.
   Under local serialization revalidate evidence versions/fences; reserve the
   durable request, capacity claim and launch debit before recording launch intent.
2. **Serialized start.** Re-read policy and time/currentness at the actual
   generation-excluded start boundary, recheck every pin, stop/cancel intent and
   original deadline. Journal a durable in-flight launch with its original
   execution/effect identity; unlock for service I/O. The receiver must participate
   in qualified generation/start/revocation exclusion through physical start.
   A prepare check or dispatcher-local mutex alone is insufficient.
3. **Removal versus physical start.** Let Rv be the protected durable removal/
   expiry-invalidation boundary observed by admission and the receiver's shared
   exclusion, and S be receiver-serialized physical-start authorization immediately
   before child code. If Rv precedes S, no physical start is permitted. If S wins,
   that original execution is in scope for mandatory stop even when its receipt
   arrives after Rv. No new admission or launch authorization can linearize after
   observed Rv. For expiry, S independently checks the qualified original expiry,
   not a cached prepare result. This specifies observable ordering, not
   instantaneous kernel visibility of file removal. The owning lanes must publish
   how removal reaches that shared boundary without a cross-service lock cycle;
   until they do, M0 start is closed.
4. **Late reply.** Lock only to revalidate the original record, controller fence,
   generation, current policy and cancel/stop intent after I/O. A successful late
   launch receipt attaches to that original effect and advances stop, never
   reopens admission. A lost reply is unknown, not a failed/no-effect verdict.
   Never hold the dispatcher state lock across Keymaker, broker, user-manager or
   launcher I/O. Durable intent and generation exclusion must survive crashes.

Unknown evidence, missing source/exclusion or failed reads refuse forward work.
Authorized original-record status/reconciliation/stop remain available under main
§8.3 control prerequisites. They do not require resurrecting expired launch
authority: cancel after removal uses the retained original owned-run stop consent,
sound current caller identity and exact original pins; stale caller association
cannot impersonate its successor. If current caller signing is unavailable, the
dispatcher's already-authorized automatic stop or separately authorized operator
stop remains required. This subset grants no new status or operator wire grammar.

## 4. Durable replay and unknown effects

Journal canonical R and its digest, authenticated caller tuple, all pins,
original deadlines, reservation/debit, original run identity and each distinct
lease/launch/stop/release effect identity before its call. Main §4.5's original
effect derivation and distinct operation scopes apply; receiver encodings remain
gated. Serialize competing IDs and both operations against the same lineage.
An accepted dispatch spends the sole physical-start entitlement durably; failure
or confirmed absence does not give another attempt a new entitlement.

The replay key is `(caller association ID, caller generation, request_id)`.
Same ID and identical canonical R (auth verifies independently) recovers only the
original record after current gates for the requested operation. Changed body
is `request_id_conflict` before any new effect; the remembered body is immutable.
Current authority is needed for any still-unperformed forward step; replay of a
receipt cannot renew it. Safe original-record observation/stop is distinct from
forward recovery. A second dispatch ID, concurrent admission, a new native signer
ID or a replaced policy cannot reset the worker's one-start ledger.

If a crash occurs before durable reservation, no effect could have been issued.
After reservation but before intent, retain the debit and reconcile; after durable
intent, unknown receipt or unknown receiver observation retains the claim and
same effect ID. Query/reconcile the original receiver effect; **do not issue a
second physical launch, recreate/adopt the account or allocate a replacement**.
Even a retry using the same effect ID is allowed only when the selected receiver
has qualified durable one-launch replay/observation; otherwise observation/stop
only. Lost or restored evidence cannot be classified as never launched. The
receiver's durable deduplication must ensure at most one child start under every
crash cut; inability to establish that closes this receiver selection.

## 5. P7 stop and retained cleanup

Removal/expiry closes new admission at observed Rv and triggers durable
account-generation-wide stop. Proposed budget: protected invalidation/time
observation <=30 s and fenced confirmed stop <=270 s, with
`detect_seconds + stop_seconds <=300 s`, or shorter actual qualified bounds.
The P7 clock begins at authoritative removal commit or original expiry, **not**
at convenient receipt arrival or eventual polling detection. Detection includes
watch/poll loss, time scheduling and restart; stop includes in-flight launch
reconciliation, fencing further starts, all descendants and the receiver's quiet
period/confirmation. The ordinary minute sweep alone proves no such bound.

This is a required proposed worst-case bound, **not a demonstrated capability**.
Before activation, time/currentness and receiver owners must qualify the source,
maximum detection delay, scheduler/failure assumptions, stop duration and
independent stop progress during dispatcher failure for the exact Linux tuple.
If those cannot fit P7, refuse activation; do not reset the deadline or declare
success. On overrun/unknown, retain the live claim, durable stop/incident and
cleanup debt, close affected forward work and report the unmet bound. Stop work
continues under its original authority and effect; unknown time does not delay
an explicit authorized stop. Delivered vendor tokens retain their real lifetime;
stopping processes does not retract credentials already issued.

Only fresh, receiver-owned **account-wide confirmed absence**, with generation,
complete launch-history coverage, start/ingress fences and required quiet period,
can release a live claim. Leader exit, PID disappearance, socket EOF, expired
lease, missing receipt and failed inventory are not absence. A known never-started
generation still needs authoritative no-launch coverage and fenced absence;
unknown history cannot use that disposition.

Cleanup is the selected published broker-v0 lease-release disposition and
receiver stop evidence, with durable obligations retained on unknown/refusal.
There is **no binding to unbind**. No new broker/helper operation is invented:
the exact v0 release/observation and receiver stop/absence join must be selected
and accepted by their owners. If an additional helper cleanup join is needed,
it is a publication gate, not implicit authority. The preprovisioned station and
its enrollment remain operator-owned; M0 does not retire it or borrow station-v0
operator-only export-waiver retirement. Output/export/custody and ordinary
retirement guarantees remain intact; if the selected worker produces workspace
content, preserve the full export/custody obligations and keep the claim/debt
until their separately published path is available. A terminal execution verdict
does not assert cleanup completion while any obligation remains.

## 6. Adjacent contract vectors

All examples are synthetic and nonproduction. The [fixture index][fixtures]
distinguishes executable **shape checks** from these semantic obligations;
neither is runtime qualification. Each negative below has its adjacent allowed
control, conditional on every gate in §7. Schema-valid semantic violations must
still refuse at signer/receiver; a schema pass never attests authority.

| Surface | Positive control | Negative/control boundary |
|---|---|---|
| Exact scope | Current pinned caller/key/generation, worker plan/profile/harness and dispatch; owned original cancel. | Wrong caller, key, generation, receiver, epoch, plan or op; caller command/environment; delegation or cross-run stop: refuse before effects. |
| Canonical values | Valid R yields domain + one LF + CCJ-1(R); input key order/whitespace variation yields identical bytes. | Extra/nested member, duplicate alias, float/-0, wrong domain/hash/wrapper, final LF, base64 padding/tail bits or native-frame confusion: cannot sign/verify the candidate. |
| Current policy | Current exact protected source at prepare **and S**, with qualified time/exclusion. | Removed/expired/rolled-back/unknown source, generation change or missed invalidation; Rv before S refuses; S before Rv mandates original stop even on late receipt. |
| One launch/replay | One reserved dispatch; exact retry recovers original effect with current gates. | Concurrent/changed-body/new-ID/reinstalled-policy/crash replay cannot launch twice; unknown receipt retains claim and original effect. |
| Recovery boundary | Sound owned cancel/recovery after launch authority closes, or independently authorized operator stop. | Reconciliation renews launch, old signature/association revives authority, cancel selects a foreign account, unknown time blocks required explicit stop: forbidden. |
| Absence/cleanup | Fenced generation-bound account-wide quiet-period absence plus selected lease release and retained custody evidence. | Leader exit/lost receipt/failed read becomes absence, missing release is success, automatic station retirement/unbind or discarded output debt: forbidden. |

## 7. Publication, review and activation gates

| Owner(s) | Required exact input; current disposition |
|---|---|
| Integration, Architect, dispatcher and Keymaker | Accept this single policy/request/signing-plan variant and numeric bounds; jointly amend SC-Q2 for existing-worker dispatch/cancel rather than the full provision/start form. Exact candidate consumer review pending. |
| Operator/curator and installer | Owner acceptance, protected install/removal transaction, source path/ACL/atomicity/currentness/audit, acyclic purpose-6 public staging, rotation and association version/epoch rules. No install/activation performed or delegated. |
| Keymaker + generation/time/storage owners | SC-Q4 final generation/exclusion/release ordering and authority time; SC-Q6 owning-store budget, durable result/replay/audit and old-reader fence; qualified first caller and published purpose-6 bootstrap. No guessed bootstrap fields. |
| User-manager/launcher + dispatcher | Selected launcher/FD3/exec-profile/start/stop/effect.observe intake and encoding, generation/revocation exclusion, durable one-launch replay, original-effect/account-wide absence and independent P7 stop. Station `active` alone is insufficient. |
| Executor/harness/catalog owners | Compatible one-shot executor intake and full typed immutable plan/profile/harness/exec-policy provenance/body/ownership, confinement, export/custody and finite evidence/result accounting. No arbitrary command or opaque enabled plan. |
| Broker + integration | **Actual v0 activation**, exact versioned lease adapter, allowlist UID mapping, final credential/currentness/release/observation predicates and all additional activation audit rows required by broker. Not just peer + runner evidence; no copied credential fallback. |
| Native qualification authority + all consumers | Exact Linux OS/kernel/architecture/hardware/firmware, boot/node/backend/image/profile, all service/harness/collector revisions, source/test/config, caller/worker UID-generation/association/epoch, units/users/groups/FDs/sandbox/mount/env/network tuple. Qualify the production entry and invalidation/P7, not support labels. Linux CI is not deployment; macOS stays native-gated. |
| Keymaker, Architect, broker, integration; independent reviewer; curator | Review exact draft and subsequent implementation separately, preserve negative controls and exact evidence, then explicit owner acceptance before installation/activation. No consumer or independent acceptance is claimed here. |

[ccj]: https://github.com/relux-works/swarma-registrar/blob/5f44ebf/spec/registrar.md#21-json-and-ccj-1-normative
[codec]: https://github.com/relux-works/swarma-registrar/blob/5f44ebf/ccj/ccj.go
[envelope]: https://github.com/relux-works/swarma-registrar/blob/5f44ebf/envelope/envelope.go
[fixtures]: ../fixtures/m0-launch/README.md
