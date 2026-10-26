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

## Killed worker, original horizon and debt

[Broker v0 §§6–7][broker] says killed-runner cleanup has no assumed callback;
original `authorized_until` persists. The sound upper horizon below is a
hypothetical result of the separately published/qualified executor evidence or
protected actual broker-policy/original-occurrence/time coverage in candidate
§5, not a caller assertion. Stop/absence is required even when horizon evidence
is unavailable. Earlier delivered tokens retain their actual independent lifetime.

```json
{
  "id": "killed-worker-horizon",
  "control": {
    "worker": "killed before release",
    "receiver_absence": "confirmed account-wide",
    "original_horizon": "sound covered upper bound reached under qualified time",
    "release_receipt": "none",
    "expect": "record lease expiry bound, never released; retain separate token and custody debt"
  },
  "counterexamples": [
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
