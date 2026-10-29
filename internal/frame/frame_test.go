package frame

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

// These cases prove rejected declarations leave the finite sentinel tail unread
// and return no payload. The valid control proves the same probe detects payload
// consumption. Allocation ordering is additionally checked by source review.
func TestReadLengthGate(t *testing.T) {
	tests := []struct {
		name   string
		header []byte
		want   error
	}{
		{"empty", []byte{0, 0, 0, 0}, ErrEmptyFrame},
		{"maximum plus one", []byte{0, 2, 0, 1}, ErrFrameTooLarge},
		{"maximum uint32", []byte{255, 255, 255, 255}, ErrFrameTooLarge},
	}
	tail := []byte("probe")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bytes.NewReader(append(bytes.Clone(tt.header), tail...))
			got, err := Read(r)
			if err != tt.want || got != nil {
				t.Fatalf("Read = (%v, %v), want (nil, %v)", got, err, tt.want)
			}
			if r.Len() != len(tail) {
				t.Fatalf("unread bytes = %d, want %d", r.Len(), len(tail))
			}
			remaining, err := io.ReadAll(r)
			if err != nil || !bytes.Equal(remaining, tail) {
				t.Fatalf("remaining = (%q, %v), want (%q, nil)", remaining, err, tail)
			}
		})
	}
	t.Run("consumption control", func(t *testing.T) {
		r := bytes.NewReader(append([]byte{0, 0, 0, 1}, tail...))
		got, err := Read(r)
		if err != nil || !bytes.Equal(got, tail[:1]) || r.Len() != len(tail)-1 {
			t.Fatalf("Read = (%q, %v), unread = %d; want one payload byte consumed", got, err, r.Len())
		}
	})
}

// One-byte chunks exercise fragmented header/payload reads at the minimum and
// inclusive maximum. Binary and non-object bytes confirm content stays opaque.
func TestReadFragmented(t *testing.T) {
	tests := []struct {
		name    string
		header  []byte
		payload []byte
	}{
		{"minimum", []byte{0, 0, 0, 1}, []byte{0xff}},
		{"non-object", []byte{0, 0, 0, 3}, []byte("[1]")},
		{"maximum", []byte{0, 2, 0, 0}, bytes.Repeat([]byte{0xa5}, 131072)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire := append(bytes.Clone(tt.header), tt.payload...)
			r := &fragmentReader{r: bytes.NewReader(wire)}
			got, err := Read(r)
			if err != nil || !bytes.Equal(got, tt.payload) || r.r.Len() != 0 {
				t.Fatalf("Read length = %d, err = %v, unread = %d", len(got), err, r.r.Len())
			}
		})
	}
}

// Each truncation preserves io.ReadFull's exact error and exposes no partial
// payload. No-payload EOF intentionally differs from a partially read payload.
func TestReadTruncated(t *testing.T) {
	tests := []struct {
		name string
		wire []byte
		want error
	}{
		{"no header", nil, io.EOF},
		{"one header byte", []byte{0}, io.ErrUnexpectedEOF},
		{"two header bytes", []byte{0, 0}, io.ErrUnexpectedEOF},
		{"three header bytes", []byte{0, 0, 0}, io.ErrUnexpectedEOF},
		{"no payload", []byte{0, 0, 0, 3}, io.EOF},
		{"one payload byte", []byte{0, 0, 0, 3, 'a'}, io.ErrUnexpectedEOF},
		{"two payload bytes", []byte{0, 0, 0, 3, 'a', 'b'}, io.ErrUnexpectedEOF},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Read(bytes.NewReader(tt.wire))
			if err != tt.want || got != nil {
				t.Fatalf("Read = (%v, %v), want (nil, %v)", got, err, tt.want)
			}
		})
	}
}

// An injected non-EOF error in either part is returned by identity, including
// when the failing reader returns some bytes with it; no partial payload escapes.
func TestReadErrors(t *testing.T) {
	failure := errors.New("read failure")
	tests := []struct {
		name   string
		prefix []byte
		last   []byte
	}{
		{"header", nil, nil},
		{"partial header", []byte{0}, []byte{0}},
		{"payload", []byte{0, 0, 0, 3}, nil},
		{"partial payload", []byte{0, 0, 0, 3, 'a'}, []byte{'b'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := io.MultiReader(bytes.NewReader(tt.prefix), &errorReader{data: tt.last, err: failure})
			got, err := Read(r)
			if err != failure || got != nil {
				t.Fatalf("Read = (%v, %v), want (nil, injected error)", got, err)
			}
		})
	}
}

// Independent literal headers check big-endian wire output at the minimum,
// an asymmetric length, and the maximum; successful writes preserve caller data.
func TestWriteWire(t *testing.T) {
	tests := []struct {
		name    string
		header  []byte
		payload []byte
	}{
		{"minimum", []byte{0, 0, 0, 1}, []byte{0xff}},
		{"asymmetric length", []byte{0, 0, 1, 2}, bytes.Repeat([]byte{'x'}, 258)},
		{"maximum", []byte{0, 2, 0, 0}, bytes.Repeat([]byte{0xa5}, 131072)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.payload)
			var w bytes.Buffer
			if err := Write(&w, tt.payload); err != nil {
				t.Fatal(err)
			}
			want := append(bytes.Clone(tt.header), tt.payload...)
			if !bytes.Equal(w.Bytes(), want) || !bytes.Equal(tt.payload, before) {
				t.Fatal("wire bytes or caller data differ")
			}
		})
	}
}

// Invalid write lengths must produce the intended error without invoking the
// writer. Successful cases in TestWriteWire guard against rejecting all inputs.
func TestWriteLengthGate(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    error
	}{
		{"nil", nil, ErrEmptyFrame},
		{"empty", []byte{}, ErrEmptyFrame},
		{"maximum plus one", make([]byte, 131073), ErrFrameTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := bytes.Clone(tt.payload)
			w := &scriptWriter{}
			if err := Write(w, tt.payload); err != tt.want {
				t.Fatalf("Write = %v, want %v", err, tt.want)
			}
			if w.calls != 0 || w.output.Len() != 0 || !bytes.Equal(tt.payload, before) {
				t.Fatal("invalid length caused output, a writer call, or payload mutation")
			}
		})
	}
}

// Each scripted failure checks the exact error, emitted prefix, call count and
// unchanged input. A failed header never reaches the payload, and nil-error
// short counts (including zero) never become success. This cannot undo output.
func TestWriteFailures(t *testing.T) {
	failure := errors.New("write failure")
	tests := []struct {
		name  string
		steps []writeStep
		want  error
		wire  []byte
	}{
		{"header zero", []writeStep{{0, nil}}, io.ErrShortWrite, nil},
		{"header short", []writeStep{{2, nil}}, io.ErrShortWrite, []byte{0, 0}},
		{"header error", []writeStep{{0, failure}}, failure, nil},
		{"header partial error", []writeStep{{2, failure}}, failure, []byte{0, 0}},
		{"header full error", []writeStep{{4, failure}}, failure, []byte{0, 0, 0, 3}},
		{"payload zero", []writeStep{{4, nil}, {0, nil}}, io.ErrShortWrite, []byte{0, 0, 0, 3}},
		{"payload short", []writeStep{{4, nil}, {2, nil}}, io.ErrShortWrite, []byte{0, 0, 0, 3, 'a', 'b'}},
		{"payload error", []writeStep{{4, nil}, {0, failure}}, failure, []byte{0, 0, 0, 3}},
		{"payload partial error", []writeStep{{4, nil}, {2, failure}}, failure, []byte{0, 0, 0, 3, 'a', 'b'}},
		{"payload full error", []writeStep{{4, nil}, {3, failure}}, failure, []byte{0, 0, 0, 3, 'a', 'b', 'c'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte("abc")
			w := &scriptWriter{steps: tt.steps}
			if err := Write(w, payload); err != tt.want {
				t.Fatalf("Write = %v, want %v", err, tt.want)
			}
			if w.calls != len(tt.steps) || !bytes.Equal(w.output.Bytes(), tt.wire) || string(payload) != "abc" {
				t.Fatalf("calls = %d, wire = %v, payload = %q", w.calls, w.output.Bytes(), payload)
			}
		})
	}
}

// A finite stream of three writes is read one frame at a time, with exact unread
// byte counts and clean end-of-stream; shared encode/decode agreement alone is
// not the wire-format proof (literal wire cases above provide that control).
func TestConsecutiveFrames(t *testing.T) {
	payloads := [][]byte{[]byte("one"), {0, 0xff}, []byte("three")}
	var wire bytes.Buffer
	for _, payload := range payloads {
		if err := Write(&wire, payload); err != nil {
			t.Fatal(err)
		}
	}
	r := bytes.NewReader(wire.Bytes())
	remaining := r.Len()
	for _, want := range payloads {
		got, err := Read(r)
		remaining -= 4 + len(want)
		if err != nil || !bytes.Equal(got, want) || r.Len() != remaining {
			t.Fatalf("Read = (%v, %v), unread = %d, want %d", got, err, r.Len(), remaining)
		}
	}
	if got, err := Read(r); err != io.EOF || got != nil {
		t.Fatalf("final Read = (%v, %v), want (nil, EOF)", got, err)
	}
}

type fragmentReader struct {
	r *bytes.Reader
}

func (r *fragmentReader) Read(p []byte) (int, error) {
	if len(p) > 1 {
		p = p[:1]
	}
	return r.r.Read(p)
}

type errorReader struct {
	data []byte
	err  error
}

func (r *errorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, r.err
}

type writeStep struct {
	n   int
	err error
}

type scriptWriter struct {
	steps  []writeStep
	calls  int
	output bytes.Buffer
}

func (w *scriptWriter) Write(p []byte) (int, error) {
	index := w.calls
	w.calls++
	if index >= len(w.steps) {
		return 0, errors.New("unexpected writer call")
	}
	step := w.steps[index]
	w.output.Write(p[:step.n])
	return step.n, step.err
}
