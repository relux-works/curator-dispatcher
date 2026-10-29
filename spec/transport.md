# Reference transport subset: `dispatch/1`

This proposed Linux reference subset supplements [dispatcher §§8.1–8.3 and 10](dispatcher.md#8-operations-results-and-authorisation).
It specifies transport negotiation and the existing `capacity` read only, not launch authority or activation.
`dispatch-op/1` operation names and `dispatch-result/1` results remain authoritative.
Launch, signatures, journal replay, accounting and all other operations remain unimplemented in this subset;
unpublished nested grammars cannot be accepted as opaque payloads. No shell, operation substitution or alternate launch fallback exists.
The opaque frame codec is a separate pending dependency (PR #10, `dcd4821e388384899aaeef8d91dab9d4311c11f8`); it does not validate or enable JSON operations.
Linux-only is a reference implementation limit, not a claim about macOS kernel credentials or platform admission.

## Peer and endpoint ownership

- Use Unix `SOCK_STREAM`. Both ends obtain Linux `SO_PEERCRED` on the connected socket before protocol I/O;
  unavailable credentials close. The server matches the kernel UID to a protected configured numeric caller-role UID allowlist.
  Only its `capacity` read role is admitted here. JSON UIDs, names, PID claims and group membership confer no protocol authority.
  Credentials are connect-time authority: never pass or inherit an established connection. Same-UID processes are the same principal.
- The client checks the peer UID against its protected configured numeric service UID **before sending any bytes**, including hello.
  An absent daemon, wrong UID, permission, version or malformed-response failure closes with no fallback.
- The endpoint directory is outside the data tree, owned by the service UID, mode `0711`;
  every ancestor is root/service-owned and not group/other-writable. Pin directory descriptors and reject symlinks at every component.
  The service UID is trusted as endpoint custodian; callers must not share that UID unless trusted with its filesystem powers.
  The socket is service-owned, mode `0660`, with an explicitly configured connect group; DAC access is additional to the UID allowlist.
  The client validates these owners/modes and socket type through the pinned directory and rechecks endpoint identity around connect.
- Use `dispatcher.sock` in that directory and a nonblocking exclusive lifetime lock on its service-owned regular `0600`, single-link `lifetime.lock` file,
  opened without symlink following and with close-on-exec. Retain its descriptor until endpoint cleanup; never unlink the lock file.
  This singleton lock is separate from the future journal/state-transition lock and is never released for receiver calls.
- Startup refuses any existing endpoint, including stale sockets, live sockets, regular files and symlinks; no automatic unlink/adoption.
  Residual endpoints require explicit operator cleanup. Bind without replacement under the pinned directory;
  do not accept clients before final owner/mode/type verification. Any race or changed identity refuses.
  Record the created socket's device/inode; cleanup unlinks only that exact socket in the same pinned directory while holding the lifetime lock.
  A missing/replaced path is left untouched. No foreign path, directory or lock is removed.

## Frames and descriptor refusal

- Each frame is four-byte unsigned big-endian length followed by exactly that many body bytes, length `1..131072` inclusive.
  Reject zero/one-over before body allocation; EOF in prefix/body and timeout close. No resynchronization or unbounded read-ahead.
- Set `CLOEXEC` on listener, accepted/client sockets and ownership descriptors atomically when created/opened.
  No inherited service sockets, socket activation or descriptor request transport; no child execution.
  Close unexpected inherited descriptors before serving; standard input/output/error are not transport capabilities.
- Every socket read on both ends uses a descriptor-aware receive mechanism, including prefix reads:
  on Linux, `recvmsg` with `MSG_CMSG_CLOEXEC` and a fixed 4096-byte ancillary buffer.
  Any ancillary data or `MSG_CTRUNC` refuses the connection before interpreting its payload; enumerate and close all delivered `SCM_RIGHTS` descriptors.
  Safe refusal must cover truncation (kernel disposal of undelivered rights), malformed controls and error paths without leaks;
  qualification of that mechanism is required. Never trust, use, forward or retain received descriptors.

## Closed JSON exchange

Bodies are exact UTF-8 CCJ-1 canonical bytes, without BOM, whitespace outside strings or final LF.
This implementation subset uses only printable ASCII strings without quote/backslash characters or escapes,
objects with unique lexically sorted member names, arrays, booleans and the integer token `1`.
It accepts only the complete shapes below, not arbitrary CCJ-1. Reject unsupported Unicode/escapes, duplicate keys at every depth,
noncanonical order/spelling, extra members, nulls, other numbers, trailing bytes and malformed JSON rather than silently normalizing them.
Maximum JSON nesting is four containers. `request_id` is 1–64 ASCII characters from `[A-Za-z0-9._:-]`.
All request/response objects carry that exact client ID; hello and operation use the same ID on this connection.

1. Client hello is exactly `{"request_id":"r1","versions":[1]}` with the caller's ID substituted.
   Only `[1]` is supported; empty, duplicate, mixed or unknown versions close without negotiation or operation effects.
2. Server hello success is exactly `{"ok":true,"request_id":"r1","result":{"version":1},"v":1}`.
   This is the existing result envelope with a closed negotiation result, not a new readiness operation.
   It asserts transport version only, never replay, accounting, authority, launch availability or OS qualification.
3. After validating hello, the client sends exactly `{"op":"capacity","request_id":"r1"}`.
   There is no `v`, UID, auth, plan, launch payload or other member in this read request.
4. With journal/accounting uninitialized or otherwise unavailable, the only capacity response is exactly
   `{"error":{"code":"capacity:hold","message":"admission closed"},"ok":false,"request_id":"r1","v":1}`.
   It carries no counters, zeroes, free slots or implied available state. This subset defines no successful capacity payload;
   publishing/implementing truthful initialized accounting must precede one. Hello never opens forward admission.

An authenticated, canonical two-string object containing exactly `op` and valid matching `request_id`,
whose `op` is not `capacity`, receives the existing `request_invalid` refusal with message `unsupported request` in the same closed error envelope.
This rejection recognizer authorizes no operation or incomplete launch/auth grammar. All other invalid requests close without a result;
an untrusted/missing ID is never reflected. Unknown/unauthorized peers receive no hello/result.
No error has optional `stage` or extra members here. The client validates the full closed envelope and matching ID before reporting any result.
One connection carries one hello and at most one operation/result, then closes; no multiplexing, pipelining, retry or pooling on that socket.
An ambiguous exchange gives no durable-effect evidence; reconnect with the same ID, never substitute another operation.

## Finite reference resource choices

These are transport resource limits, not launch, quota, queue or authority policy; none extends a standing execution's lifetime.
Use a listen backlog of 16, at most 64 accepted active connections and at most 8 per authenticated UID; excess connections close without queuing work.
Authenticate immediately after accept; no application work/goroutine or body allocation before taking a global slot.
Use at most one 131072-byte inbound frame buffer and one 512-byte outbound frame buffer per active connection,
plus its fixed ancillary buffer; no unbounded pending tasks or retained frames. Release hello storage before reading the operation.
Client connect, each whole-frame read (prefix plus body) and each write have a 2 s monotonic deadline;
the entire accepted/connected session has a non-resetting 5 s deadline. Partial progress never renews either deadline.
Shutdown closes the listener, cancels/closes all active sockets and releases buffers within 1 s, then performs identity-checked endpoint cleanup
and releases the lifetime lock. No wait for partial clients, no unrelated process stop and no state/launch cleanup is implied.

## Required adjacent qualification cases (not runtime evidence)

- Allowlisted same-UID callers negotiate; an unauthorized kernel UID or forged JSON UID cannot obtain a result/authority.
- Correct service UID permits hello; wrong/unavailable server credentials produce zero client protocol bytes.
- Exact hello/capacity/result controls pass; unsupported/duplicate versions, extra members, malformed/duplicate-key/escaped-alias JSON and wrong echoed IDs fail.
- Length 131072 passes the framing boundary (JSON still validated); 131073 and zero fail before body allocation.
- Complete frames within deadlines pass; partial prefix/body, slow drip and stalled writes expire; concurrent clients stay within global/per-UID/buffer limits.
- Protected fresh endpoints work; symlinked ancestors/socket/lock, stale/live endpoints, lock contention and replacement races refuse without unlinking foreign paths.
- Ordinary data works; rights on prefix/body, multiple/truncated rights and error paths refuse with no leaked or inherited descriptors.
- Clean shutdown removes only its own unchanged socket; a replacement, missing endpoint or stalled client cannot cause foreign deletion or unbounded shutdown.
- Hello succeeds while accounting is unavailable; capacity returns `capacity:hold` without invented state; forward/gated operations remain disabled with no effects.
