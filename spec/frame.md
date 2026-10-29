# Bounded dispatch byte frames

Status: draft normative library contract, version 1. Imperatives below are requirements. This closes the local framing definition used by [dispatcher §8.3](dispatcher.md#83-socket-dispatch1); it defines neither the socket handshake nor caller authority.

## Wire bytes and bounds

One frame is a four-byte **unsigned big-endian payload length N**, followed by exactly N opaque payload bytes. N is in **1..131072 inclusive** (128 KiB of payload). The prefix is outside that bound; maximum total wire length is **131076 bytes**. Zero and any larger N are rejected before allocating or reading payload bytes. Check the bound before narrowing N to the implementation's allocation size. A successful read consumes exactly one frame, leaving the next frame unread.

These rules are defined here independently of an external broker revision. Broker-derived connection rules in dispatcher §8.3 remain additional requirements. The byte codec does not validate JSON, CCJ-1, object shape, versions, identities, signatures or grants. A syntactically accepted byte frame grants no authority.

Independent literal prefixes (hex):

| N | Prefix | Outcome |
|---:|---|---|
| 0 | `00 00 00 00` | refuse without payload read/allocation |
| 1 | `00 00 00 01` | accept one payload byte |
| 258 | `00 00 01 02` | accept 258 payload bytes |
| 131071 | `00 01 ff ff` | accept (maximum minus one) |
| 131072 | `00 02 00 00` | accept (maximum) |
| 131073 | `00 02 00 01` | refuse without payload read/allocation |

For example, the complete frame for the one-byte payload `61` is `00 00 00 01 61`. These are contract fixtures, not runtime evidence.

## Reads, writes and stream disposition

The Go library returns a complete payload or a nil payload with an error; it never returns partially read payload bytes. Clean `io.EOF` means zero bytes were read before the next frame's header. EOF after any header byte, including a complete nonempty header followed by zero payload bytes, is `io.ErrUnexpectedEOF`. EOF after a partial payload has that same truncation verdict. Invalid N keeps its specific length refusal even if no payload follows; it is not an EOF verdict. Non-EOF reader errors retain their identity. Complete header/payload reads retain `io.ReadFull` full-count semantics: an underlying error accompanying all requested bytes does not invalidate those complete bytes. There is no speculative read of a following frame.

Writes reject empty or over-limit payloads before any writer call, and never modify the caller's payload. Emit the prefix, then the payload. Every non-nil writer error is returned unchanged, even with a full reported byte count. A short count with nil error becomes `io.ErrShortWrite`; this library makes no automatic retry, including for a progressing short write. Success means acceptance by the writer, not durable storage, peer validation or request execution. Readers/writers must obey their Go interface contracts; nil interfaces and concurrent mutation of supplied byte slices are programmer errors outside this contract.

A failed read may consume a prefix and a failed write may emit a prefix. After a framing error, discard the stream; never resynchronize or retry on it. Callers serialize each direction of a shared stream. Callers also own finite deadlines and cancellation: a generic blocking reader/writer cannot be interrupted by the byte library. The library has no descriptor-aware receive path. A socket adapter must independently reject ancillary descriptors from the first prefix byte and must not use this codec to bypass that obligation.

## Required implementation checks

Pair every rejection with a nearby valid control. Verify literal wire bytes independently of round trips; minimum, asymmetric, maximum-minus-one and maximum payloads; zero and maximum-plus-one before payload consumption/allocation; fragmented and consecutive frames; clean empty-stream EOF versus each partial-header EOF and complete-header/zero-payload EOF; partial payloads and retained non-EOF errors; nil error with short writes and non-nil errors with partial or full counts. All read failures return nil payload; refused writes emit nothing; allowed partial failed writes are recorded as such rather than misdescribed as rollback. Deadline, peer credentials, ancillary disposal and daemon behavior require separate socket/OS qualification.
