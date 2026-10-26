# DRAFT inert M0 counterexamples

These are synthetic, nonproduction **document records**, not requests, parser
inputs, a new wire grammar or an evidence handoff format. They are deliberately
outside the root `*.valid.json` / `*.invalid.json` shape-fixture inventory.
Runtime enforcement of every case is **PENDING**; signatures are placeholders.
The [candidate](../../spec/m0-launch.md) defines the required behavior.

Each JSON block names one pair/group. `control` and `counterexamples` describe
inert constructions or hypothetical protected facts with expected dispositions;
no operation is executed. `base` resolves within this directory. `replace` means
replace that decoded member only; it does not authorize mutation of a protected
record. The byte recipes are concatenation/repetition descriptions, not a CCJ
codec. Passing record consistency checks cannot attest runtime refusal.

## Domain and final LF

```json
{
  "id": "exact-preimage",
  "control": {
    "base": "dispatch.preimage.txt",
    "bytes": 904,
    "expect": "exact documented preimage; crypto and authority still pending"
  },
  "counterexamples": [
    {
      "replace_prefix": {"from": "swarma-dispatcher request/1\n", "to": "swarma-dispatcher request/2\n"},
      "expect": "wrong signing domain; must not verify as the control request"
    },
    {
      "append_hex": "0a",
      "bytes": 905,
      "expect": "final LF forbidden; must not verify as the control request"
    }
  ]
}
```

## Well-shaped authority mismatch

The protected current receiver and generation are hypothetically exactly those
in `dispatch.valid.json`. Each altered request remains schema-valid; its
signature and authority are not asserted valid. `generation-lf.invalid.json`
checks spelling only and does not cover stale authority.

```json
{
  "id": "current-pins",
  "control": {
    "base": "dispatch.valid.json",
    "expect": "pins match hypothetical protected current values; other gates pending"
  },
  "counterexamples": [
    {
      "replace": {"/receiver": "99999999999999999999999999999999"},
      "expect": "schema-valid wrong receiver; semantic refusal before effects"
    },
    {
      "replace": {"/association/generation": "44444444444444444444444444444444"},
      "expect": "schema-valid stale caller generation; semantic refusal before effects"
    }
  ]
}
```

## Generic Keymaker size boundary, outside M0

This probes **only the generic length inequality**, not a valid CCJ body or an
admitted M0 request. A 28-byte domain/LF plus the described ASCII repetition is
not the closed request grammar. M0 maximum preimages stay dispatch 964 / cancel
1019 bytes: four serials grow from one to sixteen digits, adding 60 bytes to
904 / 959. Generic cap credit must never be described as an M0 positive.

```json
{
  "id": "generic-preimage-length",
  "control": {
    "prefix": "swarma-dispatcher request/1\n",
    "repeat_ascii": "x",
    "repeat_count": 24548,
    "bytes": 24576,
    "expect": "within generic length cap only; outside M0 and not valid CCJ"
  },
  "counterexamples": [
    {
      "prefix": "swarma-dispatcher request/1\n",
      "repeat_ascii": "x",
      "repeat_count": 24549,
      "bytes": 24577,
      "expect": "over generic length cap; no signing; outside M0"
    }
  ]
}
```

## Worker owner and normal teardown

[Broker v0 §§2, 6–7][broker] supplies this documentary source witness. An
allowlisted **worker** kernel peer requests the lease and normally releases it
after teardown. Dispatcher is not added to the allowlist. The foreign-peer
counterexample assumes a separately admitted foreign peer to isolate ownership;
it does not propose admitting dispatcher. A non-allowlisted dispatcher fails
admission earlier. No real broker refusal is measured here.

```json
{
  "id": "worker-lease-owner",
  "control": {
    "request_peer": "worker",
    "release_peer": "worker",
    "private_descendants": "confirmed absent",
    "expect": "same-owner durable release may succeed; token lifetime remains independent"
  },
  "counterexamples": [
    {
      "request_peer": "worker",
      "release_peer": "admitted foreign peer",
      "expect": "lease_not_owner; no foreign release or invented receipt"
    },
    {
      "request_peer": "non-allowlisted dispatcher",
      "expect": "admission refuses; no dispatcher lease acquisition"
    }
  ]
}
```

## Selected importing custody and separate old-instance fence

These hypothetical facts describe separate Keymaker bootstrap/custody, broker
time settlement (W/hold/in-flight) and broker D5 (acknowledged store head/start
fence) importing scopes in candidate §1. The two broker scopes have separate
histories, enrollment, identities, installation bindings, namespaces, credentials,
original occurrences and independently operator-approved first checkpoint/genesis.
Common WC1/G17 infrastructure merges none of these; neither broker scope can
substitute for the other. Missing or mismatched genesis cannot be repaired by
automatic re-enrollment, first-seen adoption or the other scope's history.
All provider/placement, typed protocol, complete-original/currentness/continuity,
endpoint/load and native gates remain PENDING. Ordinary manager
create/retire/launch/stop is outside this consuming scope; later PC2 MC-Q5/A3d-2
requires separate publication and qualification. These are inert hypothetical
facts, not wire/profile members or an executable control grammar.

```json
{
  "id": "m0-selected-custody",
  "control": {
    "scope": "one actual importing scope: Keymaker bootstrap/custody, broker time settlement, or broker D5 (the broker scopes are separate)",
    "custody": "required WC1 plus common synchronous independent G17",
    "history": "complete originals and currentness/continuity for this scope's own history, enrollment, identities, installation binding, namespace, credentials, original occurrences and independently operator-approved first checkpoint/genesis",
    "fence": "separate lifetime exclusion of restored/prepared/old-instance effects, writes, signing and releases",
    "expect": "only eligible when every owning provider/protocol/native gate is qualified; no runtime authority asserted"
  },
  "counterexamples": [
    {
      "custody": "manual restore or local watermark replaces required WC1/G17",
      "expect": "refuse imported authority before effects, writes, signing or releases; retain recovery debt"
    },
    {
      "history": "another consumer's otherwise qualified history or credentials",
      "expect": "refuse cross-consumer substitution; common custodian does not merge authority"
    },
    {
      "custody": "qualified WC1/common synchronous independent G17",
      "fence": "missing separate old-instance lifetime exclusion",
      "expect": "remain closed; custody alone cannot exclude restored/prepared/old-instance effects"
    },
    {
      "history": "broker D5 history substituted for broker time settlement history",
      "expect": "refuse cross-scope import before effects, writes, signing or releases; retain recovery debt even when the other scope is qualified"
    },
    {
      "history": "broker time settlement history substituted for broker D5 history",
      "expect": "refuse cross-scope import before effects, writes, signing or releases; retain recovery debt even when the other scope is qualified"
    },
    {
      "genesis": "missing or mismatched independently approved genesis",
      "expect": "refuse import before effects, writes, signing or releases; retain recovery debt; no automatic re-enrollment or first-seen adoption or other-scope substitution"
    },
    {
      "genesis": "shared genesis across broker time settlement and broker D5",
      "expect": "refuse import before effects, writes, signing or releases; retain recovery debt; no automatic re-enrollment or first-seen adoption or other-scope substitution"
    }
  ],
  "scope_controls": [
    {
      "scope": "broker time settlement",
      "history": "own history, own enrollment, own identities, own installation binding, own namespace, own credentials, own original occurrences and independently operator-approved first checkpoint/genesis; no other scope substitution",
      "custody": "required WC1 plus common synchronous independent G17",
      "fence": "separate lifetime exclusion of restored/prepared/old-instance effects, writes, signing and releases",
      "expect": "only eligible when every owning provider/protocol/native gate is qualified; no runtime authority asserted"
    },
    {
      "scope": "broker D5",
      "history": "own history, own enrollment, own identities, own installation binding, own namespace, own credentials, own original occurrences and independently operator-approved first checkpoint/genesis; no other scope substitution",
      "custody": "required WC1 plus common synchronous independent G17",
      "fence": "separate lifetime exclusion of restored/prepared/old-instance effects, writes, signing and releases",
      "expect": "only eligible when every owning provider/protocol/native gate is qualified; no runtime authority asserted"
    }
  ]
}
```

## Killed worker, original horizon and debt

[Broker v0 §§6–7][broker] says killed-runner cleanup has no assumed callback;
original `authorized_until` persists. The sound upper horizon below is a
hypothetical result of selected horizon (b) only: protected actual broker-policy
maximum plus qualified complete original/in-flight/latest possible-issuance
coverage, retained original occurrences/authorized absolute horizons, actual
policy-version history and sound time in candidate §5. Executor evidence may
corroborate, never substitute. Stop/absence is required even when coverage is
unavailable. Earlier delivered tokens retain their actual independent lifetime.

```json
{
  "id": "killed-worker-horizon",
  "control": {
    "worker": "killed before release",
    "receiver_absence": "confirmed account-wide",
    "coverage": "qualified complete original/in-flight/latest issuance under protected actual policy maximum and actual policy-version history",
    "provenance": "retained original occurrences and authorized absolute horizons; sound time joined to receiver stop/account-wide absence",
    "original_horizon": "sound covered upper bound reached under qualified time",
    "release_receipt": "none",
    "expect": "record lease expiry bound, never released; retain separate token and custody debt"
  },
  "counterexamples": [
    {
      "receiver_absence": "confirmed account-wide",
      "coverage": "executor-only original-horizon report; schemas pass",
      "expect": "retain unknown lease debt; executor evidence cannot replace actual-policy/all-issuance coverage; continue required stop"
    },
    {
      "receiver_absence": "confirmed account-wide",
      "coverage": "actual policy maximum and original issuance covered, but one in-flight or latest issuance uncovered",
      "expect": "retain unknown lease debt despite worker absence and schema pass; refuse completion and continue required stop"
    },
    {
      "worker": "killed before release",
      "receiver_absence": "confirmed account-wide",
      "original_horizon": "unknown or caller-asserted",
      "release_receipt": "none",
      "expect": "retain lease debt; refuse its completion; continue required stop"
    },
    {
      "worker": "killed before release",
      "original_horizon": "known but not reached",
      "signal": "disconnect, restart or lease.renew",
      "expect": "original authorized_until unchanged; no release or extension"
    },
    {
      "receiver_absence": "confirmed account-wide",
      "release_receipt": "none",
      "claim": "released using dispatcher broker observation",
      "expect": "forbidden false release; v0 provides no dispatcher lease observer or foreign receipt"
    }
  ]
}
```

[broker]: https://github.com/relux-works/swarma-credential-broker/blob/8709e215a1b6bc7f42094a14e4a5620e2baed290/spec/v0-codex-refresher.md

## Selected-A interlock document pairs

These pairs follow the [landed receiver design][interlock] and candidate §§3–5.
The bounds are qualification predicates, not invented numeric residual allowances
or runnable protocol fields. All runtime controls and native instrumentation
remain **PENDING**. B retains its separate strict native-release obligations.

### C versus Rv and expiry

```json
{
  "id": "m0-c-order",
  "control": {
    "order": "root receipt < verified UID/GID drop < C < Rv/expiry",
    "facts": "exact protected publisher/projection/generation/claim/controller/validity; original unspent debit",
    "expect": "one original committed execution; all other gates and original stop limits apply"
  },
  "counterexamples": [
    {
      "order": "prepare < Rv <= C",
      "expect": "refuse C; no target code; retain preparatory evidence/cleanup until fenced absence"
    },
    {
      "order": "prepare < expiry = C",
      "expect": "refuse C at equality even with dispatcher unavailable"
    },
    {
      "order": "prepare < expiry < C",
      "expect": "refuse C after expiry; cached prepare cannot authorize"
    },
    {
      "order": "verified drop < C",
      "facts": "stale projection or wrong publisher/claim/controller/generation",
      "expect": "refuse before target code; active account alone is insufficient"
    }
  ]
}
```

### Committed C, delayed S and bounded residual

```json
{
  "id": "m0-a-delayed-s",
  "control": {
    "order": "receipt < verified drop < C < expiry < S",
    "bounds": "qualified C-to-exec-or-abort and residual intervals fit every original component/completion deadline",
    "stop": "explicitly installed independent receiver authority progresses during dispatcher failure",
    "expect": "A permits only original committed execution with retained stop; not strict B"
  },
  "counterexamples": [
    {
      "order": "C < expiry < S",
      "bounds": "unqualified handoff/residual or independent stop",
      "expect": "M0 remains closed; timer alone is not qualification"
    },
    {
      "order": "C < Rv < S",
      "claim": "A proves strict B or no worker instructions after expiry",
      "expect": "forbidden strict claim; B retains ExpiryAtNativeS and atomic native release gates"
    },
    {
      "order": "C < expiry < S",
      "bounds": "residual added after original stop deadline",
      "expect": "forbidden new allowance; residual fits inside every original deadline"
    }
  ]
}
```

### Missed completion and original unknown effects

```json
{
  "id": "m0-late-refusal",
  "control": {
    "order": "C < exec-or-abort within qualified interval and original deadlines",
    "expect": "original result may resolve under current gates; spent debit never refills"
  },
  "counterexamples": [
    {
      "order": "C < missed qualified completion interval or original deadline",
      "effect": "attempted/unknown, receipt delayed",
      "expect": "refuse success; retain original receipt/incarnation/effect, spent debit, claim, stop debt and reconciliation; refusal is not absence"
    },
    {
      "order": "C < crash/lost reply/restore",
      "action": "new child/effect or retry launch or rearmed debit",
      "expect": "forbidden; observe/stop same original effect, including conclusive non-start without refill"
    }
  ]
}
```

### Every resumer and writer participates

```json
{
  "id": "m0-all-resumers",
  "control": {
    "participants": "all resumers, recovery, invalidators, generation/stop/emergency writers",
    "state": "original execution reconciled; whole-account quiet-period absence under complete history and fences",
    "expect": "absence discharges only corresponding obligation; every later resume stays closed"
  },
  "counterexamples": [
    {
      "state": "confirmed account-wide absence",
      "action": "late original or alternate resumer executes",
      "expect": "forbidden even with valid old C; no new child/effect"
    },
    {
      "participants": "happy-path launcher only",
      "action": "recovery or generation/stop writer bypasses shared C/Rv gate",
      "expect": "forbidden; receiver selection remains unqualified"
    },
    {
      "state": "leader exit, failed scan or missing receipt only",
      "expect": "not absence; retain claim and stop debt"
    }
  ]
}
```

### Pending invalidation preserves original clocks

```json
{
  "id": "m0-close-before-publish",
  "control": {
    "order": "journal original removal intent and close admission < unlock < durable receiver Rv plus original stop < ACK < completed publication",
    "expect": "only receiver closure may justify completed removal; still not stop receipt or absence"
  },
  "counterexamples": [
    {
      "state": "receiver contact unavailable",
      "expect": "pending removal, admission closed and debt retained; no completed revocation/absence"
    },
    {
      "state": "Rv/ACK/publication delayed",
      "expect": "retain original removal intent/incident/detection/expiry and earlier deadlines; idempotent closed retry only"
    },
    {
      "state": "out-of-protocol file edit",
      "expect": "incident with detection and stop; no retrospective proof of ordered Rv"
    }
  ]
}
```

### Earliest component and completion deadlines

```json
{
  "id": "m0-earliest-deadline",
  "control": {
    "removal": 0,
    "detection": 20,
    "p7_ceiling": 320,
    "detection_stop_ceiling": 290,
    "m0_total_ceiling": 300,
    "stop_ceiling": 290,
    "expect": "stop including residual/reconciliation/quiet period by 290 or any earlier original P7/shorter qualified deadline; detection <=30 independently"
  },
  "counterexamples": [
    {
      "stop_at": 291,
      "claim": "within M0 total 300 and P7 320 so compliant",
      "expect": "refuse compliance; detection+270 was missed; keep stop/debt"
    },
    {
      "earlier_original_deadline": 280,
      "stop_at": 290,
      "expect": "refuse compliance; earlier outstanding P7/shorter qualified bound wins"
    },
    {
      "state": "restart, delayed receipt or pending invalidation resets clock",
      "expect": "forbidden; original absolute component/completion deadlines survive"
    },
    {
      "state": "unknown time",
      "action": "wait for time confidence before explicit authorized stop",
      "expect": "forbidden; close forward C but progress authorized stop"
    }
  ]
}
```

[interlock]: https://github.com/relux-works/swarma-user-manager/blob/e5a6ca608d5ecc92a36a8b8cd1f6a81340318abc/docs/design/m0-start-invalidation.md
