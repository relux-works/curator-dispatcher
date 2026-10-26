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
