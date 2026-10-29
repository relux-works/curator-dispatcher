// Package admission provides bounded, in-memory request identity, capacity and
// cancellation state. It implements no wire format, authorization, verification,
// journal, process observation, launcher or downstream I/O.
//
// A trusted future adapter must authenticate Principal independently of request
// JSON, supply the exact immutable request bytes and generation, establish fresh
// authority and gate observations, and verify all receiver evidence. Binding and
// Call are correlation/CAS values, never capabilities or reusable launch permits.
// A caller must not mutate byte inputs concurrently with a method reading them.
//
// The mutex linearizes each method. Issue's state transition is the local
// cancellation boundary: an earlier Cancel prevents issue; a later Cancel keeps
// the original call and records intent. No method calls external code under the
// lock. A future integration must journal and flush intent before any external
// effect, perform I/O outside its state lock, and revalidate the current binding
// on completion. This package alone cannot safely drive external effects.
// Local cancellation is not a receiver launch/create/ingress fence. Even after
// cancellation, an already issued call can execute until the receiver fences it.
//
// ConfirmAbsent accepts only evidence already verified by that trusted adapter:
// fresh generation-bound, account-wide process absence, original pending-call
// resolution, and receiver exclusion of later starts/ingress. Arbitrary bytes,
// a missing receipt, refusal, timeout, timer, gate release or daemon change are
// not such evidence. The reducer bounds and copies evidence; it cannot establish
// its truth or grant authority to obtain it. A valid absence releases only the
// execution slot. Allocation claims, evidence and dedup history are never evicted;
// retirement/storage disposition is outside this primitive.
//
// MaxRecords bounds records, map entries and fixed metadata. MaxRetainedBytes
// bounds owned variable input bytes plus two MaxEvidenceBytes reservations per
// record (receipt and absence). It is a conservative logical charge, not a Go
// heap measurement; allocator overhead is bounded by record count. Snapshots
// allocate independent bounded copies owned by the caller. There is no clock,
// expiry or automatic retry. A new Store loses all history: this is neither
// durable recovery nor evidence that any real station or process is absent.
package admission
