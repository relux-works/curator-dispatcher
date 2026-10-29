//go:build linux

package transport

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/relux-works/swarma-dispatcher/internal/frame"
)

func pair(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, 0)
	if err != nil {
		t.Fatal(err)
	}
	a, b := os.NewFile(uintptr(fds[0]), "test socket"), os.NewFile(uintptr(fds[1]), "test socket")
	t.Cleanup(func() { wakeClose(a); wakeClose(b) })
	for _, f := range []*os.File{a, b} {
		if err := tuneSocket(f); err != nil {
			t.Fatal(err)
		}
		if err := f.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	return a, b
}
func wireFor(t *testing.T, f *os.File) *socketIO {
	t.Helper()
	s, err := newSocketIO(f, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func sendBytes(t *testing.T, f *os.File, b, rights []byte) {
	t.Helper()
	raw, err := f.SyscallConn()
	if err != nil {
		t.Fatal(err)
	}
	for len(b) > 0 {
		var n int
		var call error
		err = raw.Write(func(fd uintptr) bool {
			n, call = syscall.SendmsgN(int(fd), b, rights, nil, syscall.MSG_NOSIGNAL|syscall.MSG_DONTWAIT)
			return call != syscall.EAGAIN && call != syscall.EWOULDBLOCK
		})
		if err != nil || call != nil || n < 1 {
			t.Fatalf("send fixture: %v %v %d", err, call, n)
		}
		b = b[n:]
		rights = nil
	}
}
func endWrite(t *testing.T, f *os.File) {
	t.Helper()
	if err := control(f, func(fd int) error { return syscall.Shutdown(fd, syscall.SHUT_WR) }); err != nil {
		t.Fatal(err)
	}
}
func fdCount(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// Real Linux streams prove frame boundaries, prefix/payload EOF distinctions
// and preallocation refusal. They are unprivileged mechanism tests, not CLI deployment proof.
func TestSocketFrames(t *testing.T) {
	for _, size := range []int{1, frame.MaxPayloadSize} {
		t.Run(string(rune('a'+size%26)), func(t *testing.T) {
			a, b := pair(t)
			wire := wireFor(t, b)
			payload := make([]byte, size)
			for i := range payload {
				payload[i] = 'x'
			}
			done := make(chan error, 1)
			go func() {
				var header [4]byte
				binary.BigEndian.PutUint32(header[:], uint32(size))
				if _, err := a.Write(header[:]); err != nil {
					done <- err
					return
				}
				_, err := a.Write(payload)
				done <- err
			}()
			got, err := wire.readFrame()
			if err != nil || !bytes.Equal(got, payload) {
				t.Fatalf("frame: %d %v", len(got), err)
			}
			if err = <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, size := range []uint32{0, frame.MaxPayloadSize + 1} {
		a, b := pair(t)
		var prefix [4]byte
		binary.BigEndian.PutUint32(prefix[:], size)
		sendBytes(t, a, prefix[:], nil)
		wire := wireFor(t, b)
		_, err := wire.readFrame()
		want := frame.ErrEmptyFrame
		if size != 0 {
			want = frame.ErrFrameTooLarge
		}
		if !errors.Is(err, want) || wire.body != nil {
			t.Fatalf("allocation gate %d: %v", size, err)
		}
	}
	for _, tc := range []struct {
		data []byte
		want error
	}{
		{nil, io.EOF}, {[]byte{0}, io.ErrUnexpectedEOF}, {[]byte{0, 0}, io.ErrUnexpectedEOF}, {[]byte{0, 0, 0}, io.ErrUnexpectedEOF},
		{[]byte{0, 0, 0, 2}, io.ErrUnexpectedEOF}, {[]byte{0, 0, 0, 2, 'x'}, io.ErrUnexpectedEOF},
	} {
		a, b := pair(t)
		if len(tc.data) > 0 {
			sendBytes(t, a, tc.data, nil)
		}
		endWrite(t, a)
		wire := wireFor(t, b)
		got, err := wire.readFrame()
		if got != nil || !errors.Is(err, tc.want) {
			t.Fatalf("EOF %v: %v", tc.data, err)
		}
	}
	a, b := pair(t)
	sendBytes(t, a, []byte{0, 0, 0, 1, 'a', 0, 0, 0, 1, 'b'}, nil)
	wire := wireFor(t, b)
	for _, want := range []string{"a", "b"} {
		got, err := wire.readFrame()
		if err != nil || string(got) != want {
			t.Fatalf("read ahead: %q %v", got, err)
		}
	}
}

// Rights on prefix and payload are rejected using actual recvmsg; counting
// descriptors checks delivered-FD cleanup, not kernel behavior on every platform.
func TestSocketRightsRefusal(t *testing.T) {
	for _, payloadStage := range []bool{false, true} {
		a, b := pair(t)
		file, err := os.Open("/dev/null")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		wire := wireFor(t, b)
		before := fdCount(t)
		if payloadStage {
			sendBytes(t, a, []byte{0, 0, 0, 1}, nil)
		}
		rights := syscall.UnixRights(int(file.Fd()), int(file.Fd()))
		data := []byte{0, 0, 0, 1}
		if payloadStage {
			data = []byte{'x'}
		}
		sendBytes(t, a, data, rights)
		got, err := wire.readFrame()
		if got != nil || !errors.Is(err, ErrAncillary) {
			t.Fatal("rights accepted", err)
		}
		if after := fdCount(t); after != before {
			t.Fatalf("delivered rights leaked: %d -> %d", before, after)
		}
	}
}

// A real small recvmsg control buffer creates MSG_CTRUNC without weakening the
// production 4096-byte buffer. Both delivered cleanup and FD-count restoration are checked.
func TestActualControlTruncation(t *testing.T) {
	a, b := pair(t)
	file, err := os.Open("/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	descriptors := make([]int, 64)
	for i := range descriptors {
		descriptors[i] = int(file.Fd())
	}
	before := fdCount(t)
	sendBytes(t, a, []byte{'x'}, syscall.UnixRights(descriptors...))
	var payload [1]byte
	small := make([]byte, syscall.CmsgSpace(4))
	var n, flags int
	err = control(b, func(fd int) error {
		_, n, flags, _, err = syscall.Recvmsg(fd, payload[:], small, syscall.MSG_CMSG_CLOEXEC|syscall.MSG_DONTWAIT)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if flags&syscall.MSG_CTRUNC == 0 {
		t.Fatal("truncation plant not applied")
	}
	if err = disposeControls(small[:n], flags); !errors.Is(err, ErrAncillary) {
		t.Fatal(err)
	}
	if after := fdCount(t); after != before {
		t.Fatalf("truncated rights leaked: %d -> %d", before, after)
	}
}

// This deterministic error+control result is not a claim that Linux returns
// that combination. Real duplicated FDs ensure rejection actually closes them.
func TestReceiveErrorOrdering(t *testing.T) {
	file, err := os.Open("/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	planted := errors.New("planted receive failure")
	for _, flags := range []int{0, syscall.MSG_CTRUNC, syscall.MSG_TRUNC} {
		fd, err := syscall.Dup(int(file.Fd()))
		if err != nil {
			t.Fatal(err)
		}
		controls := syscall.UnixRights(fd)
		if n, err := received(0, controls, flags, planted); n != 0 || !errors.Is(err, ErrAncillary) {
			syscall.Close(fd)
			t.Fatalf("control before error: %d %v", n, err)
		}
		var st syscall.Stat_t
		if err = syscall.Fstat(fd, &st); !errors.Is(err, syscall.EBADF) {
			syscall.Close(fd)
			t.Fatal("right not closed", err)
		}
		if err = syscall.Fstat(int(file.Fd()), &st); err != nil {
			t.Fatal("source fd lost", err)
		}
	}
	if n, err := received(0, nil, 0, planted); n != 0 || !errors.Is(err, planted) {
		t.Fatal("lost original error", err)
	}
	if n, err := received(1, nil, 0, nil); n != 1 || err != nil {
		t.Fatal("ordinary data rejected", err)
	}
}

// Genuine kernel credentials are accepted only against the expected UID;
// failure occurs before any write and leaves the peer stream empty.
func TestKernelPeerBeforeWrite(t *testing.T) {
	a, b := pair(t)
	if err := verifyPeer(a, uint32(os.Geteuid())); err != nil {
		t.Fatal(err)
	}
	if err := verifyPeer(a, uint32(os.Geteuid())+1); !errors.Is(err, ErrIdentity) {
		t.Fatal(err)
	}
	wire := wireFor(t, b)
	wire.start = time.Now().Add(-sessionBudget + 25*time.Millisecond)
	if _, err := wire.readFrame(); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("unexpected peer bytes", err)
	}
	var flags uintptr
	if err := control(a, func(fd int) error {
		v, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
		flags = v
		if e != 0 {
			return e
		}
		return nil
	}); err != nil || flags&syscall.FD_CLOEXEC == 0 {
		t.Fatalf("CLOEXEC: %v %d", err, flags)
	}
}

// Expired session caps win even with pending stage time; wakeClose interrupts
// an actual runtime-poll receive. MSG_NOSIGNAL makes a closed-peer send an error.
func TestSocketDeadlinesWakeAndSendFailure(t *testing.T) {
	a, b := pair(t)
	wire := wireFor(t, b)
	wire.start = time.Now().Add(-sessionBudget + 30*time.Millisecond)
	begin := time.Now()
	if _, err := wire.readFrame(); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal(err)
	}
	if time.Since(begin) > time.Second {
		t.Fatal("session cap ignored")
	}
	wire = wireFor(t, b)
	done := make(chan error, 1)
	go func() { _, err := wire.readFrame(); done <- err }()
	wakeClose(b)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("closed receive passed")
		}
	case <-time.After(time.Second):
		t.Fatal("receive did not wake")
	}
	sender := wireFor(t, a)
	if err := sender.writeFrame(hello("a")); err == nil {
		t.Fatal("closed-peer send passed")
	}
	now := time.Now()
	if got := stageDeadline(now.Add(-4*time.Second), now); !got.Equal(now.Add(time.Second)) {
		t.Fatal("deadline composition")
	}
}

// A small hello-sized frame followed by a maximum frame reuses one backing
// allocation. Prefix and payload stalls keep the original session deadline.
func TestBufferReuseAndPartialFrameDeadlines(t *testing.T) {
	a, b := pair(t)
	wire := wireFor(t, b)
	sendBytes(t, a, []byte{0, 0, 0, 1, 'a'}, nil)
	first, err := wire.readFrame()
	if err != nil || string(first) != "a" || cap(wire.body) != frame.MaxPayloadSize {
		t.Fatal("first bounded buffer", err)
	}
	address := &first[0]
	payload := bytes.Repeat([]byte{'z'}, frame.MaxPayloadSize)
	done := make(chan error, 1)
	go func() {
		var prefix [4]byte
		binary.BigEndian.PutUint32(prefix[:], uint32(len(payload)))
		if _, err := a.Write(prefix[:]); err != nil {
			done <- err
			return
		}
		_, err := a.Write(payload)
		done <- err
	}()
	second, err := wire.readFrame()
	if err != nil || !bytes.Equal(second, payload) || &second[0] != address {
		t.Fatal("second payload allocation or corruption", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	for _, partial := range [][]byte{{0}, {0, 0, 0, 3, 'a'}} {
		a, b := pair(t)
		sendBytes(t, a, partial, nil)
		wire := wireFor(t, b)
		wire.start = time.Now().Add(-sessionBudget + 30*time.Millisecond)
		if got, err := wire.readFrame(); got != nil || !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("partial frame deadline: %q %v", got, err)
		}
	}
}

// Real kernel backpressure makes a frame write time out under the session cap.
// Oversized output is refused before any write; a nearby normal frame succeeds.
func TestStalledWriteAndOutputLimit(t *testing.T) {
	a, b := pair(t)
	wire := wireFor(t, a)
	if err := wire.writeFrame(string(bytes.Repeat([]byte{'x'}, 509))); !errors.Is(err, ErrLimit) {
		t.Fatal("output budget", err)
	}
	if err := wire.writeFrame(hello("a")); err != nil {
		t.Fatal("valid output after pre-write rejection", err)
	}
	reader := wireFor(t, b)
	if got, err := reader.readFrame(); err != nil || string(got) != hello("a") {
		t.Fatal("output limit wrote extra bytes", err)
	}
	// Bounded fill, stopping only at the intended kernel backpressure sentinel.
	full := false
	if err := control(a, func(fd int) error {
		var fill [4096]byte
		for range 128 {
			_, err := syscall.SendmsgN(fd, fill[:], nil, nil, syscall.MSG_DONTWAIT|syscall.MSG_NOSIGNAL)
			if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				full = true
				return nil
			}
			if err != nil {
				return err
			}
		}
		return ErrLimit
	}); err != nil || !full {
		t.Fatal("backpressure plant not applied", err)
	}
	wire.start = time.Now().Add(-sessionBudget + 30*time.Millisecond)
	if err := wire.writeFrame(hello("a")); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatal("stalled write deadline", err)
	}
	// The caller closes on this error; it never starts a second frame.
	wakeClose(a)
}
