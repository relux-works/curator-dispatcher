package admission

import (
	"bytes"
	"errors"
	"strings"
	"sync"
)

var (
	ErrConfig           = errors.New("invalid_config")
	ErrRequest          = errors.New("request_invalid")
	ErrRequestTooLarge  = errors.New("request_too_large")
	ErrRequestConflict  = errors.New("request_id_conflict")
	ErrRecords          = errors.New("record_limit")
	ErrBytes            = errors.New("retained_byte_limit")
	ErrQuota            = errors.New("quota_exceeded")
	ErrSlots            = errors.New("capacity:slots")
	ErrHold             = errors.New("capacity:hold")
	ErrDisk             = errors.New("capacity:disk")
	ErrCPU              = errors.New("capacity:cpu_idle")
	ErrMaintenance      = errors.New("maintenance_hold")
	ErrRestore          = errors.New("restore_hold")
	ErrIntegrity        = errors.New("integrity_hold")
	ErrAllStop          = errors.New("allstop_hold")
	ErrGate             = errors.New("invalid_gate")
	ErrNotFound         = errors.New("record_not_found")
	ErrVersion          = errors.New("stale_version")
	ErrEpoch            = errors.New("stale_daemon_epoch")
	ErrGeneration       = errors.New("stale_generation")
	ErrCancelled        = errors.New("cancel_requested")
	ErrUnresolved       = errors.New("observation_unknown")
	ErrAlreadyIssued    = errors.New("already_issued")
	ErrNotInFlight      = errors.New("not_in_flight")
	ErrCall             = errors.New("stale_call")
	ErrTerminal         = errors.New("record_terminal")
	ErrEvidence         = errors.New("invalid_evidence")
	ErrEvidenceConflict = errors.New("evidence_conflict")
	ErrOutcome          = errors.New("invalid_outcome")
	ErrVersionExhausted = errors.New("version_exhausted")
)

// Counts deliberately keeps allocation and execution limits independent.
type Counts struct {
	Allocations int
	Executions  int
}

type Config struct {
	Global, PerCaller  Counts
	MaxRecords         int
	MaxRetainedBytes   int
	MaxPrincipalBytes  int
	MaxRequestIDBytes  int
	MaxGenerationBytes int
	MaxRequestBytes    int
	MaxEvidenceBytes   int
}

// Resources reserves at most one allocation and one execution in one claim.
type Resources struct {
	Allocation bool
	Execution  bool
}

type Request struct {
	Principal  string // Authenticated by the trusted adapter, never extracted as authority from JSON.
	RequestID  string
	Generation string
	Bytes      []byte
	Resources  Resources
}

// Unknown is the safe zero value, including before the first gate read.
type Gate uint8

const (
	Unknown Gate = iota
	Open
	Closed
	ReadFailure
)

// Each field is an independent barrier. Only Open allows forward transitions.
// Updating these observations neither establishes process absence nor releases
// claims. Authorization for widening/recovery must be checked by the adapter.
type Gates struct {
	Hold, Disk, CPUIdle, Maintenance, Restore, Integrity, AllStop Gate
}

func (g Gates) valid() bool {
	return g.Hold <= ReadFailure && g.Disk <= ReadFailure && g.CPUIdle <= ReadFailure &&
		g.Maintenance <= ReadFailure && g.Restore <= ReadFailure &&
		g.Integrity <= ReadFailure && g.AllStop <= ReadFailure
}

func (g Gates) check() error {
	for _, barrier := range [...]struct {
		gate Gate
		err  error
	}{
		{g.Hold, ErrHold}, {g.Disk, ErrDisk}, {g.CPUIdle, ErrCPU},
		{g.Maintenance, ErrMaintenance}, {g.Restore, ErrRestore},
		{g.Integrity, ErrIntegrity}, {g.AllStop, ErrAllStop},
	} {
		if barrier.gate != Open {
			return barrier.err
		}
	}
	return nil
}

type State uint8

const (
	Reserved State = iota
	Issued
	Uncertain
	Present
	Absent
)

// Binding is an expected record version and the current daemon epoch. Epochs
// must be nonzero and never reused by the adapter. They confer no authority.
type Binding struct {
	RecordID   uint64
	Version    uint64
	Epoch      uint64
	Generation string
}

// Call identifies the single original issue. Cancellation changes the current
// record version, but not this marker. Applying a result requires both this
// original marker and a fresh Binding. Old-epoch calls cannot be adopted.
type Call struct {
	RecordID     uint64
	IssueVersion uint64
	Epoch        uint64
	Generation   string
}

type Record struct {
	Binding         Binding
	Request         Request
	State           State
	Claims          Resources
	CancelRequested bool
	Call            Call
	Receipt         []byte
	Absence         []byte
}

type Snapshot struct {
	Epoch        uint64
	Gates        Gates
	Used         Counts
	ChargedBytes int
	Records      []Record // Admission order; no map iteration nondeterminism.
}

type requestKey struct{ principal, id string }

// Store must be constructed with New, must not be copied, and has no reset API.
type Store struct {
	mu      sync.Mutex
	config  Config
	epoch   uint64
	gates   Gates
	charged int
	index   map[requestKey]int
	records []Record
}

func New(c Config, epoch uint64) (*Store, error) {
	if epoch == 0 || c.Global.Allocations <= 0 || c.Global.Executions <= 0 ||
		c.PerCaller.Allocations <= 0 || c.PerCaller.Executions <= 0 ||
		c.MaxRecords <= 0 || c.MaxRetainedBytes <= 0 ||
		c.MaxPrincipalBytes <= 0 || c.MaxRequestIDBytes <= 0 ||
		c.MaxGenerationBytes <= 0 || c.MaxRequestBytes <= 0 ||
		c.MaxEvidenceBytes <= 0 || c.MaxEvidenceBytes > c.MaxRetainedBytes/2 {
		return nil, ErrConfig
	}
	return &Store{config: c, epoch: epoch, index: make(map[requestKey]int)}, nil
}

// SetGates records trusted current observations. It does not reconcile records.
func (s *Store) SetGates(epoch uint64, g Gates) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if epoch != s.epoch {
		return ErrEpoch
	}
	if !g.valid() {
		return ErrGate
	}
	s.gates = g
	return nil
}

// AdvanceEpoch fences writers and invalidates open gate observations without
// clearing existing barriers, calls or claims. An old issued call stays
// unresolved until independently verified absence; it cannot report completion
// as a supervisor of the replacement daemon.
func (s *Store) AdvanceEpoch(current, next uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current != s.epoch || next <= current {
		return ErrEpoch
	}
	s.epoch = next
	for _, gate := range [...]*Gate{&s.gates.Hold, &s.gates.Disk, &s.gates.CPUIdle,
		&s.gates.Maintenance, &s.gates.Restore, &s.gates.Integrity, &s.gates.AllStop} {
		if *gate == Open {
			*gate = Unknown
		}
	}
	return nil
}

func (s *Store) Admit(epoch uint64, q Request) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if epoch != s.epoch {
		return Record{}, ErrEpoch
	}
	if q.Principal == "" || q.RequestID == "" {
		return Record{}, ErrRequest
	}
	if len(q.Principal) > s.config.MaxPrincipalBytes || len(q.RequestID) > s.config.MaxRequestIDBytes {
		return Record{}, ErrRequestTooLarge
	}
	key := requestKey{q.Principal, q.RequestID}
	if i, ok := s.index[key]; ok {
		r := &s.records[i]
		if q.Generation != r.Request.Generation || q.Resources != r.Request.Resources || !bytes.Equal(q.Bytes, r.Request.Bytes) {
			return Record{}, ErrRequestConflict
		}
		return s.copyRecord(r), nil
	}
	if q.Generation == "" || len(q.Bytes) == 0 || (!q.Resources.Allocation && !q.Resources.Execution) {
		return Record{}, ErrRequest
	}
	if len(q.Generation) > s.config.MaxGenerationBytes || len(q.Bytes) > s.config.MaxRequestBytes {
		return Record{}, ErrRequestTooLarge
	}
	if err := s.gates.check(); err != nil {
		return Record{}, err
	}
	if len(s.records) >= s.config.MaxRecords {
		return Record{}, ErrRecords
	}
	// Subtractions avoid overflow even for configuration near the int limit.
	remaining := s.config.MaxRetainedBytes - s.charged
	for _, n := range [...]int{len(q.Principal), len(q.RequestID), len(q.Generation), len(q.Bytes), s.config.MaxEvidenceBytes, s.config.MaxEvidenceBytes} {
		if n > remaining {
			return Record{}, ErrBytes
		}
		remaining -= n
	}
	if err := s.capacity(q.Principal, q.Resources, nil); err != nil {
		return Record{}, err
	}
	// No retained copies or mutations occur until every refusal check passed.
	q.Principal = strings.Clone(q.Principal)
	q.RequestID = strings.Clone(q.RequestID)
	q.Generation = strings.Clone(q.Generation)
	q.Bytes = bytes.Clone(q.Bytes)
	r := Record{
		Binding: Binding{RecordID: uint64(len(s.records)) + 1, Version: 1, Generation: q.Generation},
		Request: q, State: Reserved, Claims: q.Resources,
	}
	s.index[requestKey{q.Principal, q.RequestID}] = len(s.records)
	s.records = append(s.records, r)
	s.charged = s.config.MaxRetainedBytes - remaining
	return s.copyRecord(&s.records[len(s.records)-1]), nil
}

func (s *Store) Recheck(b Binding) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.bound(b)
	if err != nil {
		return Record{}, err
	}
	if err := s.forward(r); err != nil {
		return Record{}, err
	}
	return s.copyRecord(r), nil
}

// Issue records a local boundary only. No returned value authorizes I/O.
func (s *Store) Issue(b Binding) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.bound(b)
	if err != nil {
		return Record{}, err
	}
	if err := s.forward(r); err != nil {
		return Record{}, err
	}
	if err := bump(r); err != nil {
		return Record{}, err
	}
	r.State = Issued
	r.Call = Call{RecordID: b.RecordID, IssueVersion: r.Binding.Version, Epoch: s.epoch, Generation: r.Binding.Generation}
	return s.copyRecord(r), nil
}

func (s *Store) Cancel(b Binding) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.bound(b)
	if err != nil {
		return Record{}, err
	}
	if r.CancelRequested {
		return s.copyRecord(r), nil
	}
	if r.State == Absent {
		return Record{}, ErrTerminal
	}
	if err := bump(r); err != nil {
		return Record{}, err
	}
	r.CancelRequested = true
	return s.copyRecord(r), nil
}

type Outcome uint8

const (
	ResultUnknown Outcome = iota
	ResultPresent
)

// Complete records only the original issue's unknown result or verified receipt.
// It never clears a gate, cancellation or claim, and never enables delivery.
// After cancellation the worker's saved Binding is stale: the adapter must read
// current state and reconcile that result using the unchanged original Call.
func (s *Store) Complete(b Binding, call Call, outcome Outcome, receipt []byte) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.bound(b)
	if err != nil {
		return Record{}, err
	}
	if call.Epoch != s.epoch {
		return Record{}, ErrEpoch
	}
	if r.State != Issued && r.State != Uncertain {
		return Record{}, ErrNotInFlight
	}
	if call != r.Call {
		return Record{}, ErrCall
	}
	if outcome != ResultUnknown && outcome != ResultPresent {
		return Record{}, ErrOutcome
	}
	if (outcome == ResultUnknown && len(receipt) != 0) ||
		(outcome == ResultPresent && (len(receipt) == 0 || len(receipt) > s.config.MaxEvidenceBytes)) {
		return Record{}, ErrEvidence
	}
	if err := bump(r); err != nil {
		return Record{}, err
	}
	if outcome == ResultUnknown {
		r.State = Uncertain
	} else {
		r.State = Present
		r.Receipt = bytes.Clone(receipt)
	}
	return s.copyRecord(r), nil
}

// ConfirmAbsent records sound evidence supplied by the trusted adapter as
// described in the package contract. It is not a failure/timeout fallback.
// Evidence space was reserved at admission; full data capacity cannot block it.
func (s *Store) ConfirmAbsent(b Binding, evidence []byte) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, err := s.bound(b)
	if err != nil {
		return Record{}, err
	}
	if len(evidence) == 0 || len(evidence) > s.config.MaxEvidenceBytes {
		return Record{}, ErrEvidence
	}
	if r.State == Absent {
		if !bytes.Equal(evidence, r.Absence) {
			return Record{}, ErrEvidenceConflict
		}
		return s.copyRecord(r), nil
	}
	if err := bump(r); err != nil {
		return Record{}, err
	}
	r.State = Absent
	r.Claims.Execution = false
	r.Absence = bytes.Clone(evidence)
	return s.copyRecord(r), nil
}

// Snapshot is available even when all gates are closed. Its data is not aliased
// with the store, so caller edits cannot change accounting or retained evidence.
func (s *Store) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Snapshot{Epoch: s.epoch, Gates: s.gates, ChargedBytes: s.charged, Records: make([]Record, len(s.records))}
	for i := range s.records {
		out.Records[i] = s.copyRecord(&s.records[i])
		add(&out.Used, s.records[i].Claims)
	}
	return out
}

func (s *Store) bound(b Binding) (*Record, error) {
	if b.Epoch != s.epoch {
		return nil, ErrEpoch
	}
	if b.RecordID == 0 || b.RecordID > uint64(len(s.records)) {
		return nil, ErrNotFound
	}
	r := &s.records[b.RecordID-1]
	if b.Generation != r.Binding.Generation {
		return nil, ErrGeneration
	}
	if b.Version != r.Binding.Version {
		return nil, ErrVersion
	}
	return r, nil
}

func (s *Store) forward(r *Record) error {
	if r.State == Absent {
		return ErrTerminal
	}
	if r.CancelRequested {
		return ErrCancelled
	}
	if r.State == Uncertain {
		return ErrUnresolved
	}
	if r.State != Reserved {
		return ErrAlreadyIssued
	}
	if err := s.gates.check(); err != nil {
		return err
	}
	return s.capacity(r.Request.Principal, r.Claims, r)
}

func (s *Store) capacity(principal string, demand Resources, self *Record) error {
	var total, caller Counts
	for i := range s.records {
		r := &s.records[i]
		if r == self {
			continue
		}
		add(&total, r.Claims)
		if r.Request.Principal == principal {
			add(&caller, r.Claims)
		}
	}
	if !fits(caller, demand, s.config.PerCaller) {
		return ErrQuota
	}
	if !fits(total, demand, s.config.Global) {
		return ErrSlots
	}
	return nil
}

func fits(used Counts, demand Resources, limit Counts) bool {
	return (!demand.Allocation || used.Allocations < limit.Allocations) &&
		(!demand.Execution || used.Executions < limit.Executions)
}

func add(c *Counts, r Resources) {
	if r.Allocation {
		c.Allocations++
	}
	if r.Execution {
		c.Executions++
	}
}

func bump(r *Record) error {
	if r.Binding.Version == ^uint64(0) {
		return ErrVersionExhausted
	}
	r.Binding.Version++
	return nil
}

func (s *Store) copyRecord(r *Record) Record {
	out := *r
	out.Binding.Epoch = s.epoch
	out.Request.Bytes = bytes.Clone(r.Request.Bytes)
	out.Receipt = bytes.Clone(r.Receipt)
	out.Absence = bytes.Clone(r.Absence)
	return out
}
