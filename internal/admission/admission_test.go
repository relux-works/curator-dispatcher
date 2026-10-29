package admission

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func limits() Config {
	return Config{
		Global: Counts{8, 8}, PerCaller: Counts{4, 4}, MaxRecords: 16,
		MaxRetainedBytes: 4096, MaxPrincipalBytes: 32, MaxRequestIDBytes: 32,
		MaxGenerationBytes: 32, MaxRequestBytes: 64, MaxEvidenceBytes: 16,
	}
}

func openGates() Gates { return Gates{Open, Open, Open, Open, Open, Open, Open} }

func newStore(t *testing.T, c Config) *Store {
	t.Helper()
	s, err := New(c, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetGates(1, openGates()); err != nil {
		t.Fatal(err)
	}
	return s
}

func request(principal, id string, resources Resources) Request {
	return Request{Principal: principal, RequestID: id, Generation: "generation-a", Bytes: []byte("immutable request"), Resources: resources}
}

func admit(t *testing.T, s *Store, q Request) Record {
	t.Helper()
	r, err := s.Admit(s.Snapshot().Epoch, q)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func issue(t *testing.T, s *Store, r Record) Record {
	t.Helper()
	r, err := s.Issue(r.Binding)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v; want %#v", got, want)
	}
}

func rejection(t *testing.T, s *Store, want error, action func() error) {
	t.Helper()
	before := s.Snapshot()
	if got := action(); !errors.Is(got, want) {
		t.Fatalf("got error %v; want %v", got, want)
	}
	equal(t, s.Snapshot(), before)
}

// Each invalid finite limit is rejected; an ordinary complete config and an
// int-boundary config construct successfully without eager large allocation.
func TestConfiguration(t *testing.T) {
	mutations := []func(*Config){
		func(c *Config) { c.Global.Allocations = 0 },
		func(c *Config) { c.Global.Executions = -1 },
		func(c *Config) { c.PerCaller.Allocations = 0 },
		func(c *Config) { c.PerCaller.Executions = 0 },
		func(c *Config) { c.MaxRecords = 0 },
		func(c *Config) { c.MaxRetainedBytes = 0 },
		func(c *Config) { c.MaxPrincipalBytes = 0 },
		func(c *Config) { c.MaxRequestIDBytes = 0 },
		func(c *Config) { c.MaxGenerationBytes = 0 },
		func(c *Config) { c.MaxRequestBytes = 0 },
		func(c *Config) { c.MaxEvidenceBytes = 0 },
		func(c *Config) { c.MaxEvidenceBytes = c.MaxRetainedBytes },
	}
	for i, mutate := range mutations {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			c := limits()
			mutate(&c)
			if s, err := New(c, 1); s != nil || !errors.Is(err, ErrConfig) {
				t.Fatalf("got %v, %v; want nil, %v", s, err, ErrConfig)
			}
		})
	}
	if s, err := New(limits(), 0); s != nil || !errors.Is(err, ErrConfig) {
		t.Fatalf("zero epoch: %v, %v", s, err)
	}
	newStore(t, limits())
	c := limits()
	c.MaxRetainedBytes = int(^uint(0) >> 1)
	c.MaxEvidenceBytes = c.MaxRetainedBytes / 2
	s := newStore(t, c)
	rejection(t, s, ErrBytes, func() error {
		_, err := s.Admit(1, request("a", "r", Resources{Execution: true}))
		return err
	})
}

// Exact retries retain one record through holds; changed bytes or immutable
// metadata conflict without writes. Distinct authenticated principals do not
// collide even when they supply the same request ID.
func TestRequestIdentity(t *testing.T) {
	s := newStore(t, limits())
	q := request("caller-a", "same-id", Resources{true, true})
	original := admit(t, s, q)
	for _, change := range []func(*Request){
		func(q *Request) { q.Bytes = bytes.Clone(q.Bytes); q.Bytes[0] ^= 1 },
		func(q *Request) { q.Bytes = []byte("changed request") },
		func(q *Request) { q.Bytes = nil },
		func(q *Request) { q.Bytes = bytes.Repeat([]byte{'x'}, 65) },
		func(q *Request) { q.Generation = "generation-b" },
		func(q *Request) { q.Resources.Execution = false },
	} {
		other := q
		change(&other)
		rejection(t, s, ErrRequestConflict, func() error { _, err := s.Admit(1, other); return err })
	}
	g := openGates()
	g.Hold = ReadFailure
	if err := s.SetGates(1, g); err != nil {
		t.Fatal(err)
	}
	before := s.Snapshot()
	retry, err := s.Admit(1, q)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, retry, original)
	equal(t, s.Snapshot(), before)
	if err := s.SetGates(1, openGates()); err != nil {
		t.Fatal(err)
	}
	q.Principal = "caller-b"
	other := admit(t, s, q)
	if other.Binding.RecordID == original.Binding.RecordID {
		t.Fatal("distinct principals collided")
	}
	equal(t, s.Snapshot().Used, Counts{2, 2})
}

// The owner rechecks successfully at quota one, a competing request fails that
// quota, and a different caller can reserve until the independent global limit.
func TestOwnClaimAndCompetitor(t *testing.T) {
	c := limits()
	c.PerCaller = Counts{1, 1}
	c.Global = Counts{2, 2}
	s := newStore(t, c)
	q := request("a", "one", Resources{true, true})
	r := admit(t, s, q)
	checked, err := s.Recheck(r.Binding)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, checked, r)
	equal(t, admit(t, s, q), r)
	rejection(t, s, ErrQuota, func() error {
		_, err := s.Admit(1, request("a", "two", q.Resources))
		return err
	})
	admit(t, s, request("b", "one", q.Resources))
	rejection(t, s, ErrSlots, func() error {
		_, err := s.Admit(1, request("c", "one", q.Resources))
		return err
	})
	issued := issue(t, s, r)
	equal(t, issued.State, Issued)
	equal(t, s.Snapshot().Used, Counts{2, 2})
}

// Allocation and execution counts are independent globally and per caller;
// execution absence releases only execution, never allocation or dedup history.
func TestIndependentCounters(t *testing.T) {
	for _, perCaller := range []bool{false, true} {
		t.Run(fmt.Sprint(perCaller), func(t *testing.T) {
			c := limits()
			want := ErrSlots
			if perCaller {
				c.PerCaller = Counts{1, 1}
				want = ErrQuota
			} else {
				c.Global = Counts{1, 1}
			}
			s := newStore(t, c)
			allocation := admit(t, s, request("a", "allocation", Resources{Allocation: true}))
			execution := admit(t, s, request("a", "execution", Resources{Execution: true}))
			equal(t, s.Snapshot().Used, Counts{1, 1})
			for _, demand := range []Resources{{Allocation: true}, {Execution: true}, {true, true}} {
				rejection(t, s, want, func() error {
					_, err := s.Admit(1, request("a", "competitor", demand))
					return err
				})
			}
			absent, err := s.ConfirmAbsent(execution.Binding, []byte("fenced absence"))
			if err != nil {
				t.Fatal(err)
			}
			equal(t, s.Snapshot().Used, Counts{1, 0})
			equal(t, admit(t, s, execution.Request), absent)
			equal(t, admit(t, s, allocation.Request), allocation)
			admit(t, s, request("a", "next-execution", Resources{Execution: true}))
			rejection(t, s, want, func() error {
				_, err := s.Admit(1, request("a", "next-allocation", Resources{Allocation: true}))
				return err
			})
		})
	}
}

// A failed combined reservation cannot spend the other still-free resource.
// Each counter direction is checked against caller quota and global slots,
// followed by successful admission using only that free resource.
func TestAtomicCombinedClaim(t *testing.T) {
	for _, perCaller := range []bool{false, true} {
		for _, allocationFull := range []bool{false, true} {
			t.Run(fmt.Sprintf("caller=%v/allocation=%v", perCaller, allocationFull), func(t *testing.T) {
				c := limits()
				want := ErrSlots
				if perCaller {
					c.PerCaller = Counts{1, 1}
					want = ErrQuota
				} else {
					c.Global = Counts{1, 1}
				}
				s := newStore(t, c)
				full := Resources{Allocation: allocationFull, Execution: !allocationFull}
				free := Resources{Allocation: !allocationFull, Execution: allocationFull}
				admit(t, s, request("a", "first", full))
				rejection(t, s, want, func() error {
					_, err := s.Admit(1, request("a", "combined", Resources{true, true}))
					return err
				})
				admit(t, s, request("a", "free", free))
				equal(t, s.Snapshot().Used, Counts{1, 1})
			})
		}
	}
}

// Every independent barrier fails closed for closed, unknown and failed reads.
// Snapshots, cancellation and sound absence remain reachable while closed; a
// fresh all-open control admits and issues. This tests local state, not OS gates.
func TestGates(t *testing.T) {
	barriers := []struct {
		name string
		set  func(*Gates, Gate)
		err  error
	}{
		{"hold", func(g *Gates, v Gate) { g.Hold = v }, ErrHold},
		{"disk", func(g *Gates, v Gate) { g.Disk = v }, ErrDisk},
		{"cpu", func(g *Gates, v Gate) { g.CPUIdle = v }, ErrCPU},
		{"maintenance", func(g *Gates, v Gate) { g.Maintenance = v }, ErrMaintenance},
		{"restore", func(g *Gates, v Gate) { g.Restore = v }, ErrRestore},
		{"integrity", func(g *Gates, v Gate) { g.Integrity = v }, ErrIntegrity},
		{"allstop", func(g *Gates, v Gate) { g.AllStop = v }, ErrAllStop},
	}
	for _, barrier := range barriers {
		for _, state := range []Gate{Closed, Unknown, ReadFailure} {
			t.Run(fmt.Sprintf("%s/%d", barrier.name, state), func(t *testing.T) {
				s := newStore(t, limits())
				r := admit(t, s, request("a", "reserved", Resources{true, true}))
				g := openGates()
				barrier.set(&g, state)
				if err := s.SetGates(1, g); err != nil {
					t.Fatal(err)
				}
				equal(t, s.Snapshot().Used, Counts{1, 1})
				for _, action := range []func() error{
					func() error { _, err := s.Admit(1, request("b", "new", Resources{Execution: true})); return err },
					func() error { _, err := s.Recheck(r.Binding); return err },
					func() error { _, err := s.Issue(r.Binding); return err },
				} {
					rejection(t, s, barrier.err, action)
				}
				cancelled, err := s.Cancel(r.Binding)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := s.ConfirmAbsent(cancelled.Binding, []byte("fenced absence")); err != nil {
					t.Fatal(err)
				}
				equal(t, s.Snapshot().Used, Counts{1, 0})
				if err := s.SetGates(1, openGates()); err != nil {
					t.Fatal(err)
				}
				issue(t, s, admit(t, s, request("b", "valid", Resources{Execution: true})))
			})
		}
	}
	s, err := New(limits(), 1)
	if err != nil {
		t.Fatal(err)
	}
	rejection(t, s, ErrHold, func() error { _, err := s.Admit(1, request("a", "r", Resources{true, true})); return err })
	if err := s.SetGates(1, openGates()); err != nil {
		t.Fatal(err)
	}
	admit(t, s, request("a", "r", Resources{true, true}))
}

// Releasing one barrier does not clear ALLSTOP or release a retained claim.
// Invalid and old-epoch gate updates leave the complete snapshot unchanged.
func TestIndependentGateRelease(t *testing.T) {
	s := newStore(t, limits())
	r := admit(t, s, request("a", "r", Resources{true, true}))
	g := openGates()
	g.Hold, g.AllStop = Closed, Closed
	if err := s.SetGates(1, g); err != nil {
		t.Fatal(err)
	}
	g.Hold = Open
	if err := s.SetGates(1, g); err != nil {
		t.Fatal(err)
	}
	rejection(t, s, ErrAllStop, func() error { _, err := s.Issue(r.Binding); return err })
	bad := openGates()
	bad.Hold = Gate(255)
	rejection(t, s, ErrGate, func() error { return s.SetGates(1, bad) })
	rejection(t, s, ErrEpoch, func() error { return s.SetGates(2, openGates()) })
	if err := s.SetGates(1, openGates()); err != nil {
		t.Fatal(err)
	}
	equal(t, s.Snapshot().Used, Counts{1, 1})
	issue(t, s, r)
}

// New invalid/oversize requests retain nothing; each neighboring at-limit valid
// request is admitted. Existing-identity conflicts are tested separately.
func TestRequestBounds(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Request)
		err  error
	}{
		{"principal-empty", func(q *Request) { q.Principal = "" }, ErrRequest},
		{"id-empty", func(q *Request) { q.RequestID = "" }, ErrRequest},
		{"generation-empty", func(q *Request) { q.Generation = "" }, ErrRequest},
		{"bytes-empty", func(q *Request) { q.Bytes = nil }, ErrRequest},
		{"no-resources", func(q *Request) { q.Resources = Resources{} }, ErrRequest},
		{"principal-large", func(q *Request) { q.Principal = strings.Repeat("p", 33) }, ErrRequestTooLarge},
		{"id-large", func(q *Request) { q.RequestID = strings.Repeat("r", 33) }, ErrRequestTooLarge},
		{"generation-large", func(q *Request) { q.Generation = strings.Repeat("g", 33) }, ErrRequestTooLarge},
		{"bytes-large", func(q *Request) { q.Bytes = bytes.Repeat([]byte{'b'}, 65) }, ErrRequestTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t, limits())
			q := request("a", "r", Resources{true, true})
			tc.edit(&q)
			rejection(t, s, tc.err, func() error { _, err := s.Admit(1, q); return err })
			valid := Request{
				Principal: strings.Repeat("p", 32), RequestID: strings.Repeat("r", 32),
				Generation: strings.Repeat("g", 32), Bytes: bytes.Repeat([]byte{'b'}, 64), Resources: Resources{true, true},
			}
			admit(t, s, valid)
		})
	}
}

// Record and byte exhaustion never evict even resolved history. Evidence space
// reserved at admission allows receipt and absence at exact data capacity.
func TestRetainedBounds(t *testing.T) {
	for _, byteBound := range []bool{false, true} {
		t.Run(fmt.Sprint(byteBound), func(t *testing.T) {
			q := request("a", "r", Resources{true, true})
			c := limits()
			want := ErrRecords
			if byteBound {
				c.MaxRetainedBytes = len(q.Principal) + len(q.RequestID) + len(q.Generation) + len(q.Bytes) + 2*c.MaxEvidenceBytes
				want = ErrBytes
			} else {
				c.MaxRecords = 1
			}
			s := newStore(t, c)
			r := issue(t, s, admit(t, s, q))
			charged := s.Snapshot().ChargedBytes
			if byteBound {
				equal(t, charged, c.MaxRetainedBytes)
			}
			fullEvidence := bytes.Repeat([]byte{'e'}, c.MaxEvidenceBytes)
			present, err := s.Complete(r.Binding, r.Call, ResultPresent, fullEvidence)
			if err != nil {
				t.Fatal(err)
			}
			absent, err := s.ConfirmAbsent(present.Binding, fullEvidence)
			if err != nil {
				t.Fatal(err)
			}
			equal(t, s.Snapshot().ChargedBytes, charged)
			equal(t, s.Snapshot().Used, Counts{1, 0})
			rejection(t, s, want, func() error {
				_, err := s.Admit(1, request("b", "new", Resources{Execution: true}))
				return err
			})
			equal(t, admit(t, s, q), absent)
			conflict := q
			conflict.Bytes = []byte("substitution")
			rejection(t, s, ErrRequestConflict, func() error { _, err := s.Admit(1, conflict); return err })
		})
	}
	q := request("a", "r", Resources{Execution: true})
	c := limits()
	c.MaxRetainedBytes = len(q.Principal) + len(q.RequestID) + len(q.Generation) + len(q.Bytes) + 2*c.MaxEvidenceBytes - 1
	s := newStore(t, c)
	rejection(t, s, ErrBytes, func() error { _, err := s.Admit(1, q); return err })
	q.Bytes = q.Bytes[:len(q.Bytes)-1]
	admit(t, s, q)
	equal(t, s.Snapshot().ChargedBytes, c.MaxRetainedBytes)
}

// Pre-issue cancellation prevents issue with no call marker; cancellation itself
// retains both claims until explicit sound absence. Current cancellation retries
// are idempotent, and the earlier version cannot mutate the record.
func TestCancelBeforeIssue(t *testing.T) {
	s := newStore(t, limits())
	r := admit(t, s, request("a", "r", Resources{true, true}))
	cancelled, err := s.Cancel(r.Binding)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, cancelled.Call, Call{})
	equal(t, cancelled.CancelRequested, true)
	equal(t, s.Snapshot().Used, Counts{1, 1})
	rejection(t, s, ErrCancelled, func() error { _, err := s.Issue(cancelled.Binding); return err })
	rejection(t, s, ErrCancelled, func() error { _, err := s.Recheck(cancelled.Binding); return err })
	rejection(t, s, ErrVersion, func() error { _, err := s.Cancel(r.Binding); return err })
	before := s.Snapshot()
	again, err := s.Cancel(cancelled.Binding)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, again, cancelled)
	equal(t, s.Snapshot(), before)
	if _, err := s.ConfirmAbsent(cancelled.Binding, []byte("fenced absence")); err != nil {
		t.Fatal(err)
	}
	equal(t, s.Snapshot().Used, Counts{1, 0})
	issue(t, s, admit(t, s, request("b", "valid", Resources{Execution: true})))
}

// Post-issue cancellation preserves the original call and claims. A late worker
// using its old version fails; current reconciliation can retain its receipt
// without reopening delivery. Only explicit absence releases execution once.
func TestCancelAfterIssue(t *testing.T) {
	s := newStore(t, limits())
	r := issue(t, s, admit(t, s, request("a", "r", Resources{true, true})))
	cancelled, err := s.Cancel(r.Binding)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, cancelled.Call, r.Call)
	equal(t, s.Snapshot().Used, Counts{1, 1})
	rejection(t, s, ErrVersion, func() error { _, err := s.Complete(r.Binding, r.Call, ResultPresent, []byte("receipt")); return err })
	rejection(t, s, ErrCancelled, func() error { _, err := s.Issue(cancelled.Binding); return err })
	present, err := s.Complete(cancelled.Binding, r.Call, ResultPresent, []byte("receipt"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, present.CancelRequested, true)
	equal(t, present.State, Present)
	equal(t, s.Snapshot().Used, Counts{1, 1})
	rejection(t, s, ErrNotInFlight, func() error {
		_, err := s.Complete(present.Binding, r.Call, ResultPresent, []byte("receipt"))
		return err
	})
	absent, err := s.ConfirmAbsent(present.Binding, []byte("fenced absence"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, absent.Claims, Resources{Allocation: true})
	equal(t, absent.CancelRequested, true)
	before := s.Snapshot()
	again, err := s.ConfirmAbsent(absent.Binding, []byte("fenced absence"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, again, absent)
	equal(t, s.Snapshot(), before)
	rejection(t, s, ErrEvidenceConflict, func() error { _, err := s.ConfirmAbsent(absent.Binding, []byte("other evidence")); return err })
	rejection(t, s, ErrVersion, func() error {
		_, err := s.Complete(present.Binding, r.Call, ResultPresent, []byte("receipt"))
		return err
	})
	rejection(t, s, ErrNotInFlight, func() error {
		_, err := s.Complete(absent.Binding, r.Call, ResultPresent, []byte("receipt"))
		return err
	})
	rejection(t, s, ErrTerminal, func() error { _, err := s.Issue(absent.Binding); return err })
}

// Unknown results retain reservations and the original marker. Opening gates
// cannot release that uncertainty or permit another launch. A later verified
// receipt for the same call is a valid control and still retains capacity.
func TestUnknownRetention(t *testing.T) {
	c := limits()
	c.PerCaller = Counts{1, 1}
	s := newStore(t, c)
	r := issue(t, s, admit(t, s, request("a", "r", Resources{true, true})))
	g := openGates()
	g.Hold = Closed
	if err := s.SetGates(1, g); err != nil {
		t.Fatal(err)
	}
	unknown, err := s.Complete(r.Binding, r.Call, ResultUnknown, nil)
	if err != nil {
		t.Fatal(err)
	}
	equal(t, unknown.State, Uncertain)
	equal(t, unknown.Call, r.Call)
	equal(t, s.Snapshot().Used, Counts{1, 1})
	equal(t, s.Snapshot().Gates, g)
	if err := s.SetGates(1, openGates()); err != nil {
		t.Fatal(err)
	}
	for _, action := range []func() error{
		func() error { _, err := s.Issue(unknown.Binding); return err },
		func() error { _, err := s.Recheck(unknown.Binding); return err },
	} {
		rejection(t, s, ErrUnresolved, action)
	}
	rejection(t, s, ErrQuota, func() error {
		_, err := s.Admit(1, request("a", "competitor", Resources{Execution: true}))
		return err
	})
	admit(t, s, request("b", "confined", Resources{Execution: true}))
	present, err := s.Complete(unknown.Binding, r.Call, ResultPresent, []byte("receipt"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, present.State, Present)
	equal(t, s.Snapshot().Used, Counts{1, 2})
	rejection(t, s, ErrAlreadyIssued, func() error { _, err := s.Issue(present.Binding); return err })
}

// Each stale record/call dimension rejects without any state change; the exact
// current completion succeeds. These values are CAS data, not authentication.
func TestStaleCompletionBindings(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Binding, *Call)
		err  error
	}{
		{"missing", func(b *Binding, _ *Call) { b.RecordID = 0 }, ErrNotFound},
		{"other-record", func(b *Binding, _ *Call) { b.RecordID = 999 }, ErrNotFound},
		{"version", func(b *Binding, _ *Call) { b.Version-- }, ErrVersion},
		{"generation", func(b *Binding, _ *Call) { b.Generation = "other" }, ErrGeneration},
		{"daemon", func(b *Binding, _ *Call) { b.Epoch++ }, ErrEpoch},
		{"call-version", func(_ *Binding, c *Call) { c.IssueVersion-- }, ErrCall},
		{"call-generation", func(_ *Binding, c *Call) { c.Generation = "other" }, ErrCall},
		{"call-record", func(_ *Binding, c *Call) { c.RecordID++ }, ErrCall},
		{"call-daemon", func(_ *Binding, c *Call) { c.Epoch++ }, ErrEpoch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := newStore(t, limits())
			r := issue(t, s, admit(t, s, request("a", "r", Resources{true, true})))
			b, c := r.Binding, r.Call
			tc.edit(&b, &c)
			rejection(t, s, tc.err, func() error { _, err := s.Complete(b, c, ResultPresent, []byte("receipt")); return err })
			present, err := s.Complete(r.Binding, r.Call, ResultPresent, []byte("receipt"))
			if err != nil {
				t.Fatal(err)
			}
			equal(t, present.State, Present)
			equal(t, s.Snapshot().Used, Counts{1, 1})
		})
	}
}

// All record-level controls share generation/version/epoch validation. An
// invalid binding cannot cancel, issue or release another version's claim;
// the current binding still supports each requested control.
func TestStaleControlBindings(t *testing.T) {
	for _, dimension := range []string{"generation", "version", "epoch", "record"} {
		for _, operation := range []string{"recheck", "issue", "cancel", "absence"} {
			t.Run(dimension+"/"+operation, func(t *testing.T) {
				s := newStore(t, limits())
				r := admit(t, s, request("a", "r", Resources{true, true}))
				b := r.Binding
				var want error
				switch dimension {
				case "generation":
					b.Generation = "other"
					want = ErrGeneration
				case "version":
					b.Version++
					want = ErrVersion
				case "epoch":
					b.Epoch++
					want = ErrEpoch
				case "record":
					b.RecordID = 0
					want = ErrNotFound
				}
				apply := func(binding Binding) error {
					var err error
					switch operation {
					case "recheck":
						_, err = s.Recheck(binding)
					case "issue":
						_, err = s.Issue(binding)
					case "cancel":
						_, err = s.Cancel(binding)
					case "absence":
						_, err = s.ConfirmAbsent(binding, []byte("fenced absence"))
					}
					return err
				}
				rejection(t, s, want, func() error { return apply(b) })
				if err := apply(r.Binding); err != nil {
					t.Fatal(err)
				}
				if operation == "absence" {
					equal(t, s.Snapshot().Used, Counts{1, 0})
				} else {
					equal(t, s.Snapshot().Used, Counts{1, 1})
				}
			})
		}
	}
}

// Epoch replacement invalidates open observations and old writers, without
// forgetting original calls or releasing claims. New-epoch sound absence can
// reconcile them while admission remains closed, without adopting a supervisor.
func TestDaemonReplacement(t *testing.T) {
	s := newStore(t, limits())
	r := issue(t, s, admit(t, s, request("a", "r", Resources{true, true})))
	g := openGates()
	g.AllStop = Closed
	if err := s.SetGates(1, g); err != nil {
		t.Fatal(err)
	}
	for _, epochs := range [][2]uint64{{2, 3}, {1, 1}, {1, 0}} {
		rejection(t, s, ErrEpoch, func() error { return s.AdvanceEpoch(epochs[0], epochs[1]) })
	}
	if err := s.AdvanceEpoch(1, 2); err != nil {
		t.Fatal(err)
	}
	current := s.Snapshot().Records[0]
	equal(t, current.Call, r.Call)
	equal(t, s.Snapshot().Used, Counts{1, 1})
	equal(t, s.Snapshot().Gates.AllStop, Closed)
	equal(t, s.Snapshot().Gates.Hold, Unknown)
	rejection(t, s, ErrEpoch, func() error { _, err := s.Complete(r.Binding, r.Call, ResultPresent, []byte("receipt")); return err })
	rejection(t, s, ErrEpoch, func() error {
		_, err := s.Complete(current.Binding, r.Call, ResultPresent, []byte("receipt"))
		return err
	})
	rejection(t, s, ErrEpoch, func() error { _, err := s.Cancel(r.Binding); return err })
	rejection(t, s, ErrEpoch, func() error { _, err := s.ConfirmAbsent(r.Binding, []byte("fenced absence")); return err })
	rejection(t, s, ErrEpoch, func() error { return s.SetGates(1, openGates()) })
	rejection(t, s, ErrEpoch, func() error { _, err := s.Admit(1, r.Request); return err })
	rejection(t, s, ErrHold, func() error { _, err := s.Admit(2, request("b", "new", Resources{Execution: true})); return err })
	cancelled, err := s.Cancel(current.Binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ConfirmAbsent(cancelled.Binding, []byte("fenced absence")); err != nil {
		t.Fatal(err)
	}
	equal(t, s.Snapshot().Used, Counts{1, 0})
	if err := s.SetGates(2, openGates()); err != nil {
		t.Fatal(err)
	}
	issue(t, s, admit(t, s, request("b", "valid", Resources{Execution: true})))
}

// Invalid/oversize evidence and unsupported outcome values preserve the entire
// issued state. Exact bounded receipt and absence are nearby valid controls.
func TestEvidenceValidation(t *testing.T) {
	s := newStore(t, limits())
	r := admit(t, s, request("a", "r", Resources{true, true}))
	rejection(t, s, ErrNotInFlight, func() error { _, err := s.Complete(r.Binding, Call{Epoch: 1}, ResultUnknown, nil); return err })
	r = issue(t, s, r)
	rejection(t, s, ErrAlreadyIssued, func() error { _, err := s.Issue(r.Binding); return err })
	for _, tc := range []struct {
		outcome  Outcome
		evidence []byte
		err      error
	}{
		{Outcome(255), nil, ErrOutcome},
		{ResultUnknown, []byte("receipt"), ErrEvidence},
		{ResultPresent, nil, ErrEvidence},
		{ResultPresent, bytes.Repeat([]byte{'x'}, 17), ErrEvidence},
	} {
		rejection(t, s, tc.err, func() error { _, err := s.Complete(r.Binding, r.Call, tc.outcome, tc.evidence); return err })
	}
	for _, proof := range [][]byte{nil, bytes.Repeat([]byte{'x'}, 17)} {
		rejection(t, s, ErrEvidence, func() error { _, err := s.ConfirmAbsent(r.Binding, proof); return err })
	}
	present, err := s.Complete(r.Binding, r.Call, ResultPresent, bytes.Repeat([]byte{'r'}, 16))
	if err != nil {
		t.Fatal(err)
	}
	absent, err := s.ConfirmAbsent(present.Binding, bytes.Repeat([]byte{'a'}, 16))
	if err != nil {
		t.Fatal(err)
	}
	rejection(t, s, ErrTerminal, func() error { _, err := s.Cancel(absent.Binding); return err })
	rejection(t, s, ErrTerminal, func() error { _, err := s.Recheck(absent.Binding); return err })
	equal(t, s.Snapshot().Used, Counts{1, 0})
}

// All byte inputs and every returned record/snapshot are detached. Mutating a
// caller-owned copy cannot substitute immutable input, evidence or accounting.
func TestImmutableCopies(t *testing.T) {
	s := newStore(t, limits())
	q := request("a", "r", Resources{true, true})
	originalBytes := bytes.Clone(q.Bytes)
	r := admit(t, s, q)
	q.Bytes[0] ^= 1
	r.Request.Bytes[0] ^= 2
	equal(t, s.Snapshot().Records[0].Request.Bytes, originalBytes)
	r = issue(t, s, s.Snapshot().Records[0])
	receipt := []byte("receipt")
	present, err := s.Complete(r.Binding, r.Call, ResultPresent, receipt)
	if err != nil {
		t.Fatal(err)
	}
	receipt[0] ^= 1
	present.Receipt[0] ^= 2
	equal(t, s.Snapshot().Records[0].Receipt, []byte("receipt"))
	proof := []byte("fenced absence")
	absent, err := s.ConfirmAbsent(present.Binding, proof)
	if err != nil {
		t.Fatal(err)
	}
	proof[0] ^= 1
	absent.Absence[0] ^= 2
	before := s.Snapshot()
	mutated := s.Snapshot()
	mutated.Records[0].Request.Bytes[0] ^= 4
	mutated.Records[0].Receipt[0] ^= 4
	mutated.Records[0].Absence[0] ^= 4
	mutated.Records[0].Claims = Resources{}
	mutated.Records[0].Binding.Version++
	mutated.Used = Counts{}
	mutated.Gates = Gates{}
	equal(t, s.Snapshot(), before)
	equal(t, before.Records[0].Absence, []byte("fenced absence"))
	q.Bytes = originalBytes
	retry := admit(t, s, q)
	retry.Request.Bytes[0] ^= 8
	equal(t, s.Snapshot(), before)
}

// Concurrent distinct requests at caller quota one have exactly one winner;
// each loser has the specific quota verdict, and no duplicate reservations exist.
// Concurrent exact retries all obtain the same record. No scheduling order is
// assumed. These assertions do not prove race freedom; an independent
// isolated Linux -race run is a separate check.
func TestConcurrentAdmission(t *testing.T) {
	for _, sameID := range []bool{false, true} {
		t.Run(fmt.Sprint(sameID), func(t *testing.T) {
			c := limits()
			c.PerCaller = Counts{1, 1}
			s := newStore(t, c)
			const workers = 24
			type result struct {
				record Record
				err    error
			}
			results := make(chan result, workers)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := range workers {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					id := fmt.Sprint(i)
					if sameID {
						id = "same"
					}
					r, err := s.Admit(1, request("a", id, Resources{true, true}))
					results <- result{r, err}
				}()
			}
			close(start)
			wg.Wait()
			close(results)
			wins, losses := 0, 0
			for result := range results {
				if result.err == nil {
					wins++
					equal(t, result.record.Binding.RecordID, uint64(1))
				} else if errors.Is(result.err, ErrQuota) {
					losses++
				} else {
					t.Fatalf("unexpected error: %v", result.err)
				}
			}
			if sameID {
				equal(t, wins, workers)
				equal(t, losses, 0)
			} else {
				equal(t, wins, 1)
				equal(t, losses, workers-1)
			}
			equal(t, len(s.Snapshot().Records), 1)
			equal(t, s.Snapshot().Used, Counts{1, 1})
		})
	}
}

// An isolated fixture sets the otherwise impractical version boundary. Every
// advancing method fails closed without wrapping or changing claims; a version
// immediately below the boundary can still advance once.
func TestVersionExhaustion(t *testing.T) {
	for _, operation := range []string{"issue", "cancel", "complete", "absence"} {
		t.Run(operation, func(t *testing.T) {
			s := newStore(t, limits())
			r := admit(t, s, request("a", "r", Resources{true, true}))
			if operation == "complete" {
				r = issue(t, s, r)
			}
			s.records[0].Binding.Version = ^uint64(0)
			r = s.Snapshot().Records[0]
			rejection(t, s, ErrVersionExhausted, func() error {
				var err error
				switch operation {
				case "issue":
					_, err = s.Issue(r.Binding)
				case "cancel":
					_, err = s.Cancel(r.Binding)
				case "complete":
					_, err = s.Complete(r.Binding, r.Call, ResultUnknown, nil)
				case "absence":
					_, err = s.ConfirmAbsent(r.Binding, []byte("fenced absence"))
				}
				return err
			})
			s.records[0].Binding.Version--
			r = s.Snapshot().Records[0]
			if _, err := s.Cancel(r.Binding); err != nil {
				t.Fatal(err)
			}
			equal(t, s.Snapshot().Records[0].Binding.Version, ^uint64(0))
		})
	}
}
