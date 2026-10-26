# DRAFT M0 executor entry /1

This is one closed **proposal**, pending independent review and joint
user-manager, credential-broker and catalog/policy owners exact-content selection. It publishes no
runtime, installed executor, selected Result channel or activation authority.
The full [dispatcher design](dispatcher.md) and [M0 gates](m0-launch.md#7-publication-review-and-activation-gates)
remain intact. The input is not a generic command, workspace or environment API.

## 1. Exact owning sources and admission boundary

Manager [helper §7][manager] at `5db5b8bbeba8af7ed6962c87586ef48a0cd5b0b3`
owns approved executor path/digest/verified handle, durable root receipt, verified
supplementary-group removal then real/effective/saved GID and UID drop, locks,
FD3, account-wide stop and absence. Only caller-owned descriptors 0–2 and complete
immutable read-only FD3 reach executor code. No executable, directory, lock,
receipt, service, broker or root handle survives into it. Existing 0–2 are the
caller's pipes/terminal; this proposal adds no manager output channel. Decoded
plan <=48 KiB, base64 <=64 KiB, whole launch request <=80 KiB. Staging, interruption
and cleanup under the generation lock require published finite platform bounds;
no unqualified pipe capacity, path/stdin fallback or extra launcher wire member.

Broker `0b29c086608d808903b9abf286497d941297a078` has an experimental Linux
[command dispatch][main], [CLI parser][options], [runner][runner] and
[protected client][client]. The older [v0 §6][v0] command paragraph still says
“not yet implemented”; it is not current implementation-absence evidence.
Required production observers remain unavailable: ordinary execution refuses
69 `lease_harness_unqualified` with no lease and no admitted app-server; owner
remains stopped. Protected qualification measurements occur **during**
[`protectedAuthority.Resolve` through `productionQualifiedSet`][authority] →
[`static.Config.Qualified`][qualified] → [`qualification.Repository.Read`][qualification-read].
With readable protected root and lock and neither `in-progress` nor `pending-activation`, Read
invokes [`Observe` (native `Version` first)][observe] before reading `qualified.json`.
Absent or malformed `qualified.json` therefore does not imply that no native
qualification measurement occurred. The later runner `versionProbe` follows
successful protected qualification; it is a separate step before lease admission.
These source facts authorize no target execution or probe here; native/OS,
typed-plan/authority and Result gates remain closed. Source availability is not
a qualified deployed/installable package.
`internal/runner` is not a public Go API; the proposed adapter uses the existing
command, never imports that internal package.

The approved adapter MUST be a native executable, never a shebang script:
manager's retained-handle exec does not pin an interpreter or make its script
descriptor readable after close-on-exec. Broker policy must likewise pin a native
Codex binary, not an npm/node wrapper. The second exec of the absolute pinned
broker command, its supervisor, Codex app-server and every transitive shell/tool
exec remain subject to manager R-EXEC's OS allowlist and complete invocation
qualification (entrypoint, arguments, environment and interpreted configuration).
PATH alone enforces nothing. This proposed task is one bounded text-in/text-out
turn; D requires no checkout, workspace or task-tool invocation. That does not
establish an empty transitive exec list: the actual minimum is still an owning
inventory/allowlist/qualification gate, including explicit protected interpreter
entries and exclusions for compilers, package managers and agent binaries.

## 2. Closed input grammar

Proposed discriminator: `kind="swarma-dispatcher m0-executor-entry/1"`, `v=1`.
Every member below is required, non-null and unique after JSON name unescaping;
every object is closed. No other union, optional field, array or opaque body is
admitted. FD3 contains exactly one UTF-8 JSON object, no BOM or trailing document.
Use the existing normative [JSON/CCJ-1 restrictions](m0-launch.md#2-closed-proposed-forms-and-signing):
reject invalid UTF-8, lone surrogates, duplicate/escaped duplicate names,
noninteger tokens (including `1.0`/`1e0`), `-0`, and unsafe integers. Whitespace
and member order may vary within the encoded bound; no Unicode normalization.
Canonical content identity uses CCJ-1 without final LF, never a pretty-printed
fixture's bytes. Depth <=8 is an intentional stricter subset of upstream <=16.

Notation: `ID` is exactly 32 lowercase hexadecimal ASCII bytes; `HASH` is exactly
`sha256:` plus 64 lowercase hexadecimal ASCII bytes. `SERIAL` is an integer in
1..9007199254740991; `UID` is 30000..59999 (shape only, never range admission).
The key is exactly the existing canonical `ed25519:` + 43-character unpadded
base64url public-key spelling, including the final-bit restriction in the
[signing-plan schema](../schemas/m0-signing-plan-1.schema.json).

| Object | Exactly these members and value types |
|---|---|
| Entry | `v:1`, `kind` as above, `request:Request`, `execution:Execution`, `catalog:HASH`, `qualification:HASH`, `input:Input` |
| Request | `v:1`, `kind:"swarma-dispatcher m0-request/1"`, `request_id:ID`, `receiver:ID`, `policy:Policy`, `association:Association`, `worker:Worker`, `plan:Plan`, `op:"dispatch"` |
| Policy | `id:ID`, `revision:SERIAL` |
| Association | `id:ID`, `revision:SERIAL`, `uid:UID`, `generation:ID`, `key` as above, `registration_number:SERIAL`, `master_epoch:SERIAL` |
| Worker | `uid:UID`, `generation:ID` |
| Plan | `worker_plan:HASH`, `profile:HASH`, `harness:HASH`, `exec_profile:HASH` |
| Execution | `effect_id:ID`, `execution_id` matching ASCII `EX-` followed by exactly 32 lowercase hexadecimal bytes, `controller_epoch:SERIAL` |
| Input | `account:Selector`, `profile:Selector`, `model:CatalogString`, `reasoning_effort:CatalogString`, `placement:"station-one-shot-v1"`, `prompt:Prompt` |

The proposed execution-ID spelling is an intentionally restricted adapter subset,
not a change to manager's unpublished complete identifier grammar. Owners must
select compatibility before use. `Selector` is 1..64 ASCII bytes: first `[a-z0-9]`,
then `[a-z0-9_-]`. `CatalogString` is 1..128 printable ASCII bytes U+0021..U+007E,
first byte not `-`; equality with the exact qualified catalog is mandatory.
These shape restrictions neither invent an effort enum nor admit every matching
string. `Prompt` is 1..24576 actual decoded UTF-8 bytes; U+0000 is refused.
Whitespace and newlines within Prompt are preserved. No prompt file/home lookup,
credential content, resume, provider fallback or implicit additional input exists.

Hard parser/work ceilings: encoded FD3 <=49152 bytes, canonical entire entry
<=49152 bytes, total decoded string values <=32768 UTF-8 bytes (excluding member
names), exactly 8 objects/40 members, no arrays, <=512 lexical tokens including
member names and punctuation. Each member name <=32 ASCII bytes; every non-prompt
string <=128 bytes. Check raw bytes before allocation; scan at most limit+1 only
to detect excess. Reuse one bounded parse and one bounded CCJ traversal, each
linear in <=49152 bytes/512 tokens; no recursive dependency expansion, retries,
network resolution, catalog fetching or directory tree walk. Reserve <=256 KiB
owned parse/canonical/prompt/argv buffers and <=4 protected metadata resolutions
(each body <=49152 bytes), <=256 KiB total metadata bytes; these are proposed
ceilings, not measured runtime guarantees. Complete platform/library/syscall work,
locks, metadata latency and wall-time qualification remain gates. These limits
cannot borrow the runner's 1 MiB prompt allowance or widen the manager request.
JSON escapes count toward encoded/canonical bounds: 24576 decoded bytes do not
guarantee fit. Refuse rather than truncate or normalize. No Schema is supplied:
a parsed-object shape checker could not prove lexical or UTF-8 byte restrictions.

## 3. Content selection and semantic authority

`request` is the binding copy of the admitted unsigned dispatch, not a new signed
request. Equality is with the independently authenticated durable original R;
its association/kernel caller, receiver, worker UID/generation, current policy,
registration/epoch, all plan pins and creating-dispatcher association must match
protected originals. Execution copies equal the original effect/receipt/claim
and controller epoch; they allocate no new execution. Catalog and qualification
match the protected policy's exact pins, not a caller assertion.

For this proposal, `request.plan.worker_plan = HASH(CCJ-1(input))`. This defines
one new closed worker-input body, pending catalog-owner selection. Other plan
pins retain their existing meanings. The entry identity is separately
`HASH(CCJ-1(entire entry))`, bound to the manager-admitted FD3 digest in protected
original launch evidence; it is not recursively its own worker-plan hash. The
protected catalog must select this exact input body and the complete typed
profile/harness/exec-policy bodies, authenticated immutable origins, publication,
installation, ownership, currentness and equality rules. This leaf does **not**
publish those bodies or admit digest-matching opaque enabled payloads. Until
those joins exist, even the positive specimen is inadmissible for execution.

Protected resolutions are four bounded, preselected snapshots: original
request/effect/receipt/claim; current policy/association/generation/time; complete
selected catalog/profile/harness/exec-policy/qualification; installation and
placement/containment tuple. No self-minted success field, missing evidence,
failed read, stale snapshot, or restored old reader/writer is authority. The full
typed representation and aggregation within the proposed four-resolution budget
must be jointly published; an oversized or unavailable snapshot closes use.

## 4. Deterministic proposed command and stdin mapping

The installation tuple supplies `E`, the absolute approved broker executable,
verified digest/handle, and `H`, the manager-verified station home. They are not
input fields, PATH lookups or arbitrary caller paths. Proposed fixed derivation:
`J=H + "/tmp/m0/" + execution.execution_id`, `L=J + "/home"`,
`W=H + "/work"`, `O=J + "/last-message.txt"`. Paths are absolute UTF-8,
NUL-free and <=1024 bytes each including E/H/J/L/W/O; no alternate spelling,
symlink component, reuse of an unresolved J, or fallback directory is admitted.
The worker, after verified drop, prepares its work/tmp/J/L content under a
published bounded ownership/mode and no-follow contract (launch home 0700).
D-owned approval/intake/root is distinct from worker-created content. Manager
creates no generic workspace. No repository, archive, home traversal or tool
input is imported. These are candidate derivations, **not** proof that same-UID
races, filesystem/config authority, permissions or protected Result access are
resolved. Selection must qualify the exact filesystem/identity tuple, stale
writers/readers/restore fences and safe verified-handle exec mechanism, or refuse.

For one admitted tuple, argv is exactly these 19 elements, in this order, with
separate option/value elements and no shell, `=` spelling, positional tail,
duplicate option, injected flag, prompt-file, resume or credential/config flag:

```text
[E, "run-codex", "--account", input.account,
 "--profile", input.profile, "--launch-home", L,
 "--model", input.model, "--reasoning-effort", input.reasoning_effort,
 "--output-last-message", O, "--prompt-stdin", "--cwd", W,
 "--timeout", decimal(T)]
```

Actual argv UTF-8 storage, sum of each element's byte length plus one NUL, is
<=8192 bytes; each element <=1024 bytes, and OS exec representation/pointer/env
limits must independently qualify. No caller environment map: use only the
manager's fixed HOME, USER, LOGNAME, PATH, LANG and TMPDIR inside the home;
the adapter passes it unchanged to the runner and adds no CODEX_HOME, model,
endpoint or credential variable. It remains qualified for runner conflict checks;
never silently clear a reserved conflicting variable to gain admission. Broker
reads protected `/etc/swarma-credential-broker/config.json` with its current
same-user R2 reader. Dedicated-service unreadability refuses; no public copy,
weakened mode, `--config`, environment override or invented privileged read.

The broker [independently constructs its vendor environment][preflight]: HOME and CODEX_HOME
equal L, PATH is `/usr/bin:/bin`, LANG and LC_ALL are `C.UTF-8`; TMPDIR and other
incoming variables are not forwarded. Selected M0 PATH(b) pins that fixed vendor
PATH as M0's plan-digest-covered D-PATH, distinct from the helper bootstrap PATH.
The protected full profile/plan relation must establish that equality before use;
no new entry member, adapter PATH rewrite or broker flag is selected. Protected
environment conformance and OS exec enforcement are separate gates: the kernel
does not refuse a PATH change. Qualify actual forbidden exec attempts (absolute
paths, writable aliases/copies and unentered interpreters), independently of
environment equality. This content
selection grants no published complete profile, OS allowlist or native credit.

FD3 is the immutable plan, **never runner stdin**. Validate it completely, retain
the admitted identity, and close EVERY descriptor except 0–2 before runner exec,
including FD3 and any adapter-owned metadata, feeder or executable handle not
consumed/closed by the qualified exec mechanism. Never reopen FD3 by a path.
Feed exactly decoded
Prompt UTF-8 bytes once through a newly owned bounded stdin pipe, then EOF; no
extra LF, normalization, truncation, home file, second prompt or lease retry.
Pipe capacity/feeder/cancellation/join and exec/close ordering must be qualified;
no write-before-reader deadlock. Only 0–2 reach the invoked command; its internal
qualified guard/vendor descriptor needs are broker-owned, not extra inherited
manager/adapter descriptors. Caller stdout/stderr remain existing observation
paths; stdin feeding does not grant the adapter new inherited FD authority.
The plan object and ALL its duplicates must be absent downstream, verified by
kernel object identity (for example device/inode or qualified open-file-description
comparison), not descriptor number or adapter logs. Broker-created internal
descriptors may later reuse number 3; that alone is no violation.
The current runner does not inspect/refuse an inherited plan descriptor: a native negative must
observe accidental plan-object/duplicate inheritance and fail the adapter/profile qualification,
not claim the runner implements a refusal. Feeder lifecycle and any surviving
adapter process remain in the original generation's whole-account stop scope.
Unknown launch/prompt/lease effect is reconciled on the original execution;
restart cannot perform a second physical start or repeat the prompt.


A separate current broker handle gate remains OPEN. At the pinned revision,
[containment_linux.go lines 62–64][containment] starts app-server through
`/proc/self/fd/3` with `ExtraFiles=[executable]`: app-server holds the pinned
Codex executable without CLOEXEC. Tool inheritance/re-exec is a source-identified
risk, not a native observation here. A path/mount allowlist alone cannot prohibit
re-exec of the same already-allowed agent file through an inherited handle.
Credential-broker must fix that transfer and provide native trace/regression
proof before M0 use. The positive control starts app-server once from the pinned
digest; negatives must prevent model tools holding/reopening the agent object or
re-executing it through inherited descriptors, `/proc/.../fd`, `/proc/.../exe`,
`fexecve` or `execveat`/`AT_EMPTY_PATH`. Evidence must name the actual enforcement
and object identity; do not infer this property from PATH or path allowlisting.
This is not a blanket exemption for broker-internal descriptors. No fix, trace,
qualification or waiver is supplied by this entry proposal.

## 5. Original clock, stop and result boundaries

The adapter is target/executor code. Selected A's receiver-owned durable C must
be after root receipt/verified drop and **before any adapter/loader/constructor
code**. A pre-C decoder is not permitted by this proposal. Serialized C/Rv/expiry,
equality refusal, original spent claim/debit, qualified late S/residual and
all-resumer closure remain [M0 §3](m0-launch.md#3-protected-installation-currentness-and-ordering).
The owning mechanism is unpublished/unqualified; userspace validation cannot
replace it. Do not add policy evaluation/argv fields to launcher.

No fresh 300 s timeout: at the qualified handoff instant let D be the earliest
original applicable end-to-end completion deadline, and S the separately
qualified positive slack covering handoff, remaining preparation, cleanup,
storage/publication and independent stop/absence completion. Propose
`T=min(240, floor((D-now-S)/1 second))`; require T>=1 and recheck the original
deadline at exec. Decimal T is an integer 1..240, exactly representable by the
current float-seconds CLI conversion; never round fractional remainder upward.
A tighter original authority/phase deadline also shortens/refuses this cap.
For example D=290 s, now=20 s, S=10 s gives T=240; D-now-S=0.9 s refuses.
This arithmetic is a stricter proposal, not qualification of any actual S.
Runner timeout bounds initialization+turn only; cleanup is separate. No flag,
stdout deadline, context or timer proves bounded regular-file read/write/fsync,
metadata latency, process absence, or the global D deadline. Missing finite
storage/stop slack qualification closes use, never invents a constant safe S.

Preserve original observation <=30 s, detection-to-stop <=270 s and intent/incident/
expiry-to-completion <=300 s, earliest shorter bound or real original P7 wins.
Intent0/detection20 requires completion290, real P7 deadline280 requires280;
291 is late. Receipt, Rv, ACK, retries, nested preparation and recovery reset none.
Original bootstrap t_B+120/B0–B5 and proposed 2 MiB work/32 dependency/512 KiB
retained inventory budgets remain [separate unchanged gates](m0-bootstrap-budget.md).

Runner's statuses 0,1,64,69,75,124,130 and [phase precedence][runner] remain its
own facts. Foreign adapter context errors/timeouts are not runner-owned 124.
Successful final atomic rename fixes runner terminal0 despite racing/later stop;
D must not rewrite that fact to75/130. D disposition/deadline failure is separate.
Only matching completed turn plus confirmed cleanup/release permits0600 atomic
replacement. A known runner refusal before publication preserves old O, or leaves
a previously absent O absent. Successful `output.root.Rename` is the irreversible
publication point ([output commit][output], [runner publication][run]): it replaces
old O or creates O for a fresh J. D can lose reliable completion observation
afterward, so unknown completion can coexist with **new O**;
it does not imply preservation of old output or absence of publication. Runner
terminal0 after successful rename is distinct from D's observed completion and
disposition. Bytes or mode0600 grant no D read/consumption authority, new success
or failure disposition, or fresh admission. Retain original publication/Result/
effect debt on unknown completion. Known staging is removed on prepublication
failure; unknown cleanup retains its original debt. The [adjacent inert boundary
witnesses](../fixtures/m0-executor-entry/boundary.witnesses.json) cover both old
and previously absent O; they establish no runtime behavior or Result authority.
Cleanup shares5 s, process confirmation<=3.5 s, release<=remaining2 s; no claimed
bound on all syscalls/guard lifecycle follows. Fresh connection/kernel identity
and actual protected policy rechecks remain broker-owned. D never requests,
renews/releases leases or becomes broker observer. Runner success supplies no
D durable receipt, account-wide absence or token-revocation proof. Manager's
account-wide absence and independently installed stop stay authoritative.

Result channel/sink/encoding/authentication/ACL/atomic durability, byte/frame/
aggregate/state/retention/overflow budgets and old-writer/reader/restore fences
are jointly **unselected**. Existing stdio is a capture candidate, not already
authenticated durable Result. O is a proposed worker output location, not D read
authority: worker0600 alone grants none. No arbitrary sink/extra FD, raw vendor
stderr/frames/errors, prompt/secret logging or token hashes. Preserve sanitized
assistant deltas; they are nonterminal. This leaf selects no capture quota; any
future candidate quota must reserve full storage before admission and retain
unknown original outcome/effects/claim/custody/stop/lease/token debt on overflow,
disconnect or delivery failure. No quarantine/wipe/retire/rebind/purge permission.
G2B/planned removal vs security-window/EIC and actual policy horizon(b)/issuance
coverage/custody remain unchanged; no v0 security/cancel predicate or NEW-lease hook.

## 6. Inert conformance and activation

[Specimens and forbidden controls](../fixtures/m0-executor-entry/README.md) are
inert documentary obligations, not a parser, validator or native test harness.
Production entry coverage is **0/8 obligation groups**, with no production call
sites: lexical/shape; finite byte/work; protected identity/provenance; argv/stdin;
handle/drop/FD; original clocks/stop/replay; qualification/config; Result/debt.
Native cases are PENDING. Shape success proves no authorization, original time,
permissions, descriptor immutability, process absence or Result authenticity.

Independent review, joint user-manager + credential-broker review and catalog/policy
owner exact-content selection precede adapter code; owning typed catalog/install/profile,
FD3/C/platform qualification and authenticated Result joins precede activation.
Required hosted checks and signed delivery remain delivery gates.
No local runtime/build/project test or live broker/credential operation is
required or claimed for this documentary slice.

[manager]: https://github.com/relux-works/swarma-user-manager/blob/5db5b8bbeba8af7ed6962c87586ef48a0cd5b0b3/spec/helper.md#7-the-launcher-swarma-user-launcher
[runner]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/docs/runner.md
[client]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/docs/broker-client.md
[options]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/runner/options.go
[main]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/cmd/swarma-credential-broker/main.go
[preflight]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/runner/preflight_linux.go
[v0]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/spec/v0-codex-refresher.md#6-external-token-delivery-and-one-shot-runner

[containment]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/runner/containment_linux.go#L62
[authority]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/runner/authority_linux.go
[qualified]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/policy/qualification_linux.go
[qualification-read]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/qualification/storage_linux.go
[observe]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/qualification/observer_linux.go
[output]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/runner/output.go
[run]: https://github.com/relux-works/swarma-credential-broker/blob/0b29c086608d808903b9abf286497d941297a078/internal/runner/run.go
