//go:build linux

package transport

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Real socket sessions exercise the prevalidated protocol mechanism only.
// Config/deployment/peer admission is not bypassed by any exported API.
func TestProtocolSessions(t *testing.T) {
	for _, op := range []string{"capacity", "start"} {
		a, b := pair(t)
		serverDone := make(chan error, 1)
		checks := 0
		go func() {
			serverDone <- serveOperation(b, time.Now(), func() error { checks++; return nil })
			wakeClose(b)
		}()
		client := wireFor(t, a)
		if err := client.writeFrame(hello("a")); err != nil {
			t.Fatal(err)
		}
		reply, err := client.readFrame()
		if err != nil || string(reply) != helloResult("a") {
			t.Fatal("hello", err)
		}
		if err = client.writeFrame(`{"op":"` + op + `","request_id":"a"}`); err != nil {
			t.Fatal(err)
		}
		reply, err = client.readFrame()
		want := hold("a")
		if op != "capacity" {
			want = invalid("a")
		}
		if err != nil || string(reply) != want {
			t.Fatalf("response %q: %v", reply, err)
		}
		if err = <-serverDone; err != nil {
			t.Fatal(err)
		}
		if checks != 2 {
			t.Fatal("pin rechecks", checks)
		}
	}
	for _, first := range []string{capacity("a"), hello("a") + " ", hello(""), `{"request_id":"a","versions":[2]}`} {
		a, b := pair(t)
		done := make(chan error, 1)
		go func() { done <- serveOperation(b, time.Now(), func() error { return nil }); wakeClose(b) }()
		wire := wireFor(t, a)
		if err := wire.writeFrame(first); err != nil {
			t.Fatal(err)
		}
		if _, err := wire.readFrame(); err == nil {
			t.Fatal("reply before valid hello")
		}
		if err := <-done; !errors.Is(err, ErrInvalid) {
			t.Fatal("bad stage rejection", err)
		}
	}
	for _, op := range []string{hello("a"), capacity("wrong"), `{"op":"dispatch","plan":{},"request_id":"a"}`} {
		a, b := pair(t)
		done := make(chan error, 1)
		go func() { done <- serveOperation(b, time.Now(), func() error { return nil }); wakeClose(b) }()
		wire := wireFor(t, a)
		if err := wire.writeFrame(hello("a")); err != nil {
			t.Fatal(err)
		}
		if _, err := wire.readFrame(); err != nil {
			t.Fatal(err)
		}
		if err := wire.writeFrame(op); err != nil {
			t.Fatal(err)
		}
		if _, err := wire.readFrame(); err == nil {
			t.Fatal("invalid op received result")
		}
		if err := <-done; !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
}

// A pin failure immediately before either result sends no result; prior valid
// hello does not keep authority alive. The actual FS pin recheck is tested nearby.
func TestPinFailureBeforeResult(t *testing.T) {
	for _, failAt := range []int{1, 2} {
		a, b := pair(t)
		done := make(chan error, 1)
		count := 0
		go func() {
			done <- serveOperation(b, time.Now(), func() error {
				count++
				if count == failAt {
					return ErrIdentity
				}
				return nil
			})
			wakeClose(b)
		}()
		wire := wireFor(t, a)
		if err := wire.writeFrame(hello("a")); err != nil {
			t.Fatal(err)
		}
		if failAt == 2 {
			if _, err := wire.readFrame(); err != nil {
				t.Fatal(err)
			}
			if err := wire.writeFrame(capacity("a")); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := wire.readFrame(); err == nil {
			t.Fatal("reflected result after pin failure")
		}
		if err := <-done; !errors.Is(err, ErrIdentity) {
			t.Fatal(err)
		}
	}
}

// Exact client result checks prevent hello-stage errors and mismatched results
// from triggering any capacity write. No production authentication is mocked.
func TestClientStageValidation(t *testing.T) {
	for _, response := range []string{invalid("a"), helloResult("wrong"), helloResult("a") + " "} {
		a, b := pair(t)
		done := make(chan error, 1)
		go func() { _, err := capacityExchange(a, time.Now(), "a"); done <- err; wakeClose(a) }()
		server := wireFor(t, b)
		if _, err := server.readFrame(); err != nil {
			t.Fatal(err)
		}
		if err := server.writeFrame(response); err != nil {
			t.Fatal(err)
		}
		if err := <-done; !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
		if _, err := server.readFrame(); err == nil {
			t.Fatal("capacity sent after invalid hello")
		}
	}
	a, b := pair(t)
	done := make(chan error, 1)
	go func() { done <- serveOperation(b, time.Now(), func() error { return nil }); wakeClose(b) }()
	got, err := capacityExchange(a, time.Now(), strings.Repeat("a", 64))
	if err != nil || got != hold(strings.Repeat("a", 64)) {
		t.Fatal("valid hold control", err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
}

// Reservations enforce exact global/per-UID boundaries and release post-failure
// state; stop prevents new reservations and a second shutdown preserves its clock.
func TestSlotsAndShutdownRequest(t *testing.T) {
	s := newSlots()
	for i := range 64 {
		if !s.global() {
			t.Fatal("early global refusal", i)
		}
	}
	if s.global() {
		t.Fatal("65th global")
	}
	for i := range 64 {
		s.release(nil)
		_ = i
	}
	if s.total != 0 {
		t.Fatal("global debt")
	}
	for i := range 8 {
		if !s.global() || !s.uid(1000) {
			t.Fatal("early UID refusal", i)
		}
	}
	if !s.global() {
		t.Fatal("global control")
	}
	if s.uid(1000) {
		t.Fatal("9th UID")
	}
	s.release(nil)
	uid := uint32(1000)
	for range 8 {
		s.release(&uid)
	}
	if len(s.byUID) != 0 || s.total != 0 {
		t.Fatal("UID debt")
	}
	s.stop()
	if s.global() || s.uid(1000) {
		t.Fatal("reservation after stop")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := &shutdown{cancel: cancel}
	stop.request(nil)
	first, _ := stop.snapshot()
	stop.request(nil)
	second, _ := stop.snapshot()
	if !first.Equal(second) || ctx.Err() == nil {
		t.Fatal("shutdown clock reset")
	}
	if !transientAccept(syscall.EINTR) || !transientAccept(syscall.ECONNABORTED) || transientAccept(syscall.EBADF) || transientAccept(syscall.EMFILE) {
		t.Fatal("transient accept classes")
	}
}

// The actual accept/peer/pin/worker path is exercised with kernel current UID
// and an owned local socket. This deliberately does not assert distinct-role or
// root-protected fixed-path deployment admission from its internal config.
func TestServeAndShutdown(t *testing.T) {
	e, c := fixtureEndpoint(t)
	pindir := protectedFixture(t, 0700)
	id := pindir.ids[len(pindir.ids)-1]
	cfg := serverConfig{commonConfig: c, Callers: []principal{{Dev: id.dev, Ino: id.ino, UID: uint32(os.Geteuid()), Dir: pindir.path}}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stop := &shutdown{cancel: cancel}
	done := make(chan error, 1)
	go func() { done <- serve(ctx, e, cfg, stop) }()
	f, err := connectSocket(e.dir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyPeer(f, uint32(os.Geteuid())); err != nil {
		t.Fatal(err)
	}
	got, err := capacityExchange(f, time.Now(), "a")
	wakeClose(f)
	if err != nil || got != hold("a") {
		t.Fatal("accept exchange", err)
	}
	// A second connection stalls in its first prefix; shutdown must wake it.
	stalled, err := connectSocket(e.dir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer wakeClose(stalled)
	stop.request(nil)
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown blocked")
	}
	if err = e.close(); err != nil {
		t.Fatal(err)
	}
}

// Unknown genuine kernel UIDs receive no hello, and eight actual sessions
// consume the UID limit before the ninth is refused. These are current-UID
// internal mechanism fixtures, not managed/distinct-UID deployment admission.
func TestAcceptUIDRefusalAndBound(t *testing.T) {
	for _, unknown := range []bool{true, false} {
		e, c := fixtureEndpoint(t)
		pindir := protectedFixture(t, 0700)
		id := pindir.ids[len(pindir.ids)-1]
		uid := uint32(os.Geteuid())
		if unknown {
			uid++
		}
		cfg := serverConfig{commonConfig: c, Callers: []principal{{Dev: id.dev, Ino: id.ino, UID: uid, Dir: pindir.path}}}
		ctx, cancel := context.WithCancel(context.Background())
		stop := &shutdown{cancel: cancel}
		done := make(chan error, 1)
		go func() { done <- serve(ctx, e, cfg, stop) }()
		t.Cleanup(func() {
			stop.request(nil)
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Error("fixture server did not stop")
			}
		})
		count := 1
		if !unknown {
			count = 9
		}
		for i := 0; i < count; i++ {
			f, err := connectSocket(e.dir, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { wakeClose(f) })
			wire := wireFor(t, f)
			if err = wire.writeFrame(hello("a")); err != nil {
				if !(unknown || i == 8) || !errors.Is(err, syscall.EPIPE) && !errors.Is(err, syscall.ECONNRESET) {
					t.Fatal(err)
				}
				continue
			}
			reply, err := wire.readFrame()
			if unknown || i == 8 {
				if reply != nil || !errors.Is(err, io.EOF) && !errors.Is(err, syscall.ECONNRESET) {
					t.Fatalf("expected closed admission without reply: %q %v", reply, err)
				}
			} else if err != nil || string(reply) != helloResult("a") {
				t.Fatal("valid session refused", err)
			}
		}
		stop.request(nil)
		// Cleanup waits on done; do not consume that notification here.
	}
}

// Extra queued operations are never read or answered after the first result.
// This tests server closure, not detection of client pipelining.
func TestOneOperationThenClose(t *testing.T) {
	a, b := pair(t)
	done := make(chan error, 1)
	checks := 0
	go func() {
		done <- serveOperation(b, time.Now(), func() error { checks++; return nil })
		wakeClose(b)
	}()
	wire := wireFor(t, a)
	if err := wire.writeFrame(hello("a")); err != nil {
		t.Fatal(err)
	}
	if got, err := wire.readFrame(); err != nil || string(got) != helloResult("a") {
		t.Fatal("hello", err)
	}
	// Queue both operations in one send before the server can close its side.
	body := capacity("a")
	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], uint32(len(body)))
	one := append(prefix[:], []byte(body)...)
	queued := append(append([]byte(nil), one...), one...)
	sendBytes(t, a, queued, nil)
	if got, err := wire.readFrame(); err != nil || string(got) != hold("a") {
		t.Fatal("first operation", err)
	}
	if err := <-done; err != nil || checks != 2 {
		t.Fatal("extra operation processed", err, checks)
	}
	if got, err := wire.readFrame(); got != nil || !errors.Is(err, io.EOF) && !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("extra result: %q %v", got, err)
	}
}
