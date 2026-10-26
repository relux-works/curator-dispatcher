# DRAFT M0 shape and byte fixtures

Synthetic, nonproduction data only. Repeated hexadecimal identities/digests and
the public example key are placeholders, not installed principals or provenance.
`auth.sig` is a zero-byte encoding placeholder: **none of these requests is a
cryptographically valid or authorized launch**. No private key is included.

The [candidate](../../spec/m0-launch.md) owns semantics. Four `*.valid.json`
files satisfy shape only: `dispatch` and `cancel` use
[request schema](../../schemas/m0-request-1.schema.json), `policy` uses
[policy schema](../../schemas/m0-policy-1.schema.json), and `signing-plan` uses
[signing-plan schema](../../schemas/m0-signing-plan-1.schema.json).

`dispatch.preimage.txt` is exactly 904 ASCII bytes: dispatcher domain + LF +
CCJ-1 of `dispatch.valid.json` without auth, with no final LF.
`dispatch.preimage.sha256` contains its lowercase SHA-256 plus a text-file LF.
This bounded ASCII specimen does not replace registrar's Unicode/duplicate/
numeric-token codec conformance vectors. Changing JSON whitespace or key order
must preserve this preimage; signing its SHA-256 text, appending LF, inserting
auth, changing the domain or prepending a native length frame must not verify
as this request.

Each `*.invalid.json` is rejected by the named schema, with a nearby positive
control from the first table. These are instance counterexamples, not runtime
tests or evidence of a working launch, signing service, currentness source or
stop bound. No mock receiver is supplied.

| Invalid file stem | Schema | Boundary |
|---|---|---|
| request-extra | request | Caller command is forbidden. |
| nested-extra | request | Worker environment/extension map is forbidden. |
| grant-fallback | request | Even an empty grants member is not M0 grammar. |
| unsupported-op | request | Provision is not admitted. |
| dispatch-cancel-member | request | Dispatch cannot carry cancel target. |
| missing-cancel-target | request | Cancel must name original dispatch. |
| generation-lf | request | Exact generation spelling cannot end in LF. |
| hash-case | request | Digest spelling is lowercase only. |
| signature-pad | request | Base64url has no padding. |
| signature-tail-bits | request | Unused signature bits must be zero. |
| native-frame | request | Native method/hello fields are not signed request fields. |
| policy-two-starts | policy | The budget is exactly one physical start. |
| policy-stop-bound | policy | Proposed post-detection stop bound is <=270 s. |
| policy-extra | policy | No enable/fallback boolean. |
| signing-opaque | signing-plan | No opaque payload extension. |
| signing-auth | signing-plan | Keymaker constructs unsigned R before auth insertion. |

Wrong but well-shaped caller/key/worker/plan pins, reversed/expired policy times,
forged authority, stale revisions, changed-body replay, unknown receipt and
leader-only exit are **semantic negatives**, not schema-invalid fixtures. See
the adjacent semantic vectors in candidate §6. Meta-schema/fixture validation
does not prove these gates; their owning production implementations must supply
paired tests and narrowing controls before acceptance/activation.

## Coverage and enforcement boundary

[Inert counterexample records](counterexamples.md) are separate document pairs,
not additional shape fixtures. The 20 existing JSON shape fixtures are unchanged.

| Scope | Executable shape check | Inert pair / reference | Runtime enforcement |
|---|---|---|---|
| Unknown/nested members, operation branches, required members, field spelling/bounds | Four positive and sixteen negative fixtures above; JSON Schema 2020-12 | Candidate §2 | PENDING |
| Exact domain and no final LF | Not a schema property; existing 904-byte specimen/digest can be compared | `exact-preimage` control and wrong-domain/final-LF records | PENDING, no crypto/CCJ implementation |
| Exact receiver/current caller generation | Both changed requests remain well-shaped | `current-pins` control and wrong-receiver/stale-generation records | PENDING, no authority test |
| Finite closed M0 preimage maximum | Shape field bounds permit an ASCII length calculation, not general canonicalization | Candidate §2: dispatch 28+872+64=964; cancel 1019; specimen 904 | PENDING |
| Generic Keymaker preimage cap | Not an M0 shape positive | `generic-preimage-length`: 24576 / 24577, length only, both outside M0 | PENDING |
| Broker peer ownership | Not a request schema property | `worker-lease-owner`: normal owner release / foreign-owner refusal | PENDING; documentary pinned-source witness only |
| Killed worker and lease disposition | Not a schema property | `killed-worker-horizon`: sound horizon / unavailable horizon, unchanged deadline and false-release records | PENDING; no broker observer exists for dispatcher |
| Selected-A C/Rv/expiry and exact protected facts | No runtime entry point | `m0-c-order`: valid C / Rv-first, expiry equality/past and stale protected facts | PENDING |
| A delayed S and residual inside original bounds; B preserved | No native mechanism qualified | `m0-a-delayed-s`: bounded original start / missing qualification, false strict-B claim and added allowance | PENDING |
| Late refusal retains original effect/debit/claim/stop | No runtime entry point | `m0-late-refusal`: timely completion / missed interval or deadline, crash/relaunch | PENDING |
| All-resumer closure and absence | No runtime entry point | `m0-all-resumers`: fenced full absence / late resumer, writer bypass and false absence | PENDING |
| Close before publish and unavailable receiver | No control/store protocol published | `m0-close-before-publish`: journal/close/unlock/Rv/stop/ACK/publish / pending contact, shifted clocks and file edit | PENDING |
| Earliest original component/completion limits | Document arithmetic only | `m0-earliest-deadline`: removal 0, detection 20, ceiling 290 / 291, earlier 280 and reset/unknown-time negatives | PENDING |
| Raw duplicates/escaped aliases, BOM/encoding, numeric tokens, depth, framing, CCJ and signatures | Outside decoded shape checks | Registrar CCJ-1 and candidate §2; JSON decoding may lose duplicate or lexical information | UNRUN; no parser/codec qualification |
| Other authority/replay/currentness/stop/custody semantics | Outside shape checks | Candidate §§3–7 adjacent vectors and closed owning-lane gates | PENDING |

No schema pass, document record, finite size calculation or source witness
attests a real signature, service refusal, current caller, launch, stop, cleanup
or accepted numeric bound. In particular integral decimal/exponent lexical forms
and duplicate members can be lost by decoding before JSON Schema sees them.
