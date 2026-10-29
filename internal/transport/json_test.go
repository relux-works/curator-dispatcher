package transport

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// These cases prove the literal recognizers reject neighboring invalid byte
// shapes without parser normalization; they do not qualify socket authority.
func TestWireRecognition(t *testing.T) {
	for _, id := range []string{"a", strings.Repeat("a", 64), "0123456789abcdef0123456789abcdef"} {
		got, err := parseHello([]byte(hello(id)))
		if err != nil || got != id {
			t.Fatalf("hello: %q %v", got, err)
		}
		ok, err := parseOperation([]byte(capacity(id)), id)
		if err != nil || !ok {
			t.Fatal("capacity", err)
		}
		ok, err = parseOperation([]byte(`{"op":"start","request_id":"`+id+`"}`), id)
		if err != nil || ok {
			t.Fatal("closed unsupported recognizer", err)
		}
		for _, literal := range []string{hello(id), helloResult(id), capacity(id), hold(id), invalid(id)} {
			if !scanCanonical([]byte(literal), 2, true) {
				t.Fatalf("canonical positive: %s", literal)
			}
			if len(literal)+4 > 512 {
				t.Fatal("outbound wire budget")
			}
		}
	}
	for _, b := range []string{
		hello(""), hello(strings.Repeat("a", 65)), hello("a/b"), " " + hello("a"), hello("a") + "\n",
		`{"request_id":"a","versions":[2]}`, `{"request_id":"a","versions":[1,1]}`,
		`{"request_id":"a","request_id":"a","versions":[1]}`, `{"versions":[1],"request_id":"a"}`,
		`{"request_id":"a","versions":[1],"x":1}`, `{"request_id":"\u0061","versions":[1]}`, capacity("a"),
	} {
		if id, err := parseHello([]byte(b)); !errors.Is(err, ErrInvalid) || id != "" {
			t.Fatalf("accepted bad hello %q: %q %v", b, id, err)
		}
	}
	for _, b := range []string{hello("a"), capacity("b"), `{"op":"capacity","request_id":"a","uid":1000}`, `{"op":"\u0063apacity","request_id":"a"}`, `{"op":"capacity","op":"start","request_id":"a"}`, `{"op":"capacity","request_id":"a"} `} {
		if _, err := parseOperation([]byte(b), "a"); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted bad op %s: %v", b, err)
		}
	}
	for _, b := range []string{`{"a":null}`, `{"a":01}`, `{"a":0}`, `{"a":1.0}`, `{"a":2}`, `{"a":"\u0061"}`, `{"a":"é"}`, `{"a":{"b":[]}}`, `{"b":1,"a":1}`, `{"a":1,"a":1}`, "\xef\xbb\xbf{}"} {
		if scanCanonical([]byte(b), 2, true) {
			t.Fatalf("accepted bad canonical bytes %q", b)
		}
	}
}
func canonicalFixture(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
func configFixture(server bool) map[string]any {
	p := map[string]any{"dev": 1, "dir": "/home/reader", "ino": 2, "uid": 1000}
	c := map[string]any{"connect_gid": 901, "endpoint_dev": 1, "endpoint_ino": 3, "service_gid": 900, "service_uid": 900, "v": 1}
	if server {
		c["callers"] = []any{p}
	} else {
		c["caller"] = p
	}
	return c
}

// Inert numeric fixtures prove closed grammar/ranges only; none is installed,
// root-owned or a successful deployment or kernel principal enrollment.
func TestClosedConfiguration(t *testing.T) {
	for _, server := range []bool{false, true} {
		b := canonicalFixture(configFixture(server))
		if server {
			if _, err := parseServer(b); err != nil {
				t.Fatal(err)
			}
		} else {
			if _, err := parseClient(b); err != nil {
				t.Fatal(err)
			}
		}
		for _, mutate := range []func(map[string]any){
			func(c map[string]any) { delete(c, "v") }, func(c map[string]any) { c["V"] = 1; delete(c, "v") },
			func(c map[string]any) { c["unknown"] = 1 }, func(c map[string]any) { c["service_uid"] = 0 },
			func(c map[string]any) { c["service_uid"] = 30000 }, func(c map[string]any) { c["connect_gid"] = 900 },
			func(c map[string]any) { c["endpoint_ino"] = uint64(maxIdentity + 1) }, func(c map[string]any) { c["service_gid"] = uint64(4294967295) },
		} {
			c := configFixture(server)
			mutate(c)
			b = canonicalFixture(c)
			if server {
				got, err := parseServer(b)
				if !errors.Is(err, ErrConfig) || !reflect.DeepEqual(got, serverConfig{}) {
					t.Fatalf("bad server escaped: %s %v", b, err)
				}
			} else {
				got, err := parseClient(b)
				if !errors.Is(err, ErrConfig) || !reflect.DeepEqual(got, clientConfig{}) {
					t.Fatalf("bad client escaped: %s %v", b, err)
				}
			}
		}
	}
	for _, uid := range []uint32{0, 900, 30000, 59999, 4294967295} {
		c := configFixture(false)
		c["caller"].(map[string]any)["uid"] = uid
		if _, err := parseClient(canonicalFixture(c)); !errors.Is(err, ErrConfig) {
			t.Fatalf("bad principal UID %d", uid)
		}
	}
	for _, dir := range []string{"/", "/home//a", "/home/../a", "/home/a/", "relative", "/" + strings.Repeat("a", 512)} {
		c := configFixture(false)
		c["caller"].(map[string]any)["dir"] = dir
		if _, err := parseClient(canonicalFixture(c)); !errors.Is(err, ErrConfig) {
			t.Fatalf("bad dir %q", dir)
		}
	}
	for _, bad := range [][]byte{nil, []byte("{}\n"), []byte(`{"v":1,"v":1}`), bytes.Repeat([]byte("x"), serverConfigLimit+1), canonicalFixture(configFixture(false))} {
		if _, err := parseServer(bad); !errors.Is(err, ErrConfig) {
			t.Fatal("server accepted malformed/cross-role config", err)
		}
	}
	if _, err := parseClient(canonicalFixture(configFixture(true))); !errors.Is(err, ErrConfig) {
		t.Fatal("client accepted server config")
	}
}

// Entry count, ordering and byte maxima are independent limits. A rejection
// returns zero configuration, rather than a usable partially validated prefix.
func TestCallerAndSnapshotBounds(t *testing.T) {
	c := configFixture(true)
	entries := make([]any, 32)
	for i := range entries {
		entries[i] = map[string]any{"dev": 1, "dir": "/home/a", "ino": i + 1, "uid": 1000 + i}
	}
	c["callers"] = entries
	if _, err := parseServer(canonicalFixture(c)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []any{[]any{}, append(append([]any{}, entries...), entries[31]), []any{entries[0], entries[0]}, []any{entries[1], entries[0]}} {
		c["callers"] = bad
		if _, err := parseServer(canonicalFixture(c)); !errors.Is(err, ErrConfig) {
			t.Fatal("caller order/count", err)
		}
	}
	c["callers"] = entries
	for _, p := range entries {
		p.(map[string]any)["dir"] = "/" + strings.Repeat("a", 511)
	}
	if _, err := parseServer(canonicalFixture(c)); !errors.Is(err, ErrConfig) {
		t.Fatal("independent byte cap", err)
	}
	for _, size := range []int{1, clientConfigLimit} {
		b, err := readSnapshot(bytes.NewReader(bytes.Repeat([]byte("a"), size)), clientConfigLimit)
		if err != nil || len(b) != size {
			t.Fatal("snapshot bound", err)
		}
	}
	if b, err := readSnapshot(bytes.NewReader(bytes.Repeat([]byte("a"), clientConfigLimit+1)), clientConfigLimit); !errors.Is(err, ErrConfig) || b != nil {
		t.Fatal("oversize returned partial snapshot", err)
	}
}

// Principal boundaries and exact nested member names are independent of the
// root schema. Adjacent unmanaged UIDs remain accepted; managed/alias forms fail.
func TestPrincipalConfigurationNeighbors(t *testing.T) {
	for _, uid := range []uint32{1, 29999, 60000, 4294967294} {
		c := configFixture(false)
		p := c["caller"].(map[string]any)
		p["uid"] = uid
		p["dir"] = "/" + strings.Repeat("a", 511)
		if _, err := parseClient(canonicalFixture(c)); err != nil {
			t.Fatal("valid unmanaged boundary", uid, err)
		}
	}
	for _, mutate := range []func(map[string]any){
		func(p map[string]any) { p["dev"] = 0 }, func(p map[string]any) { p["ino"] = 0 },
		func(p map[string]any) { p["UID"] = p["uid"]; delete(p, "uid") },
		func(p map[string]any) { p["extra"] = true }, func(p map[string]any) { delete(p, "dir") },
		func(p map[string]any) { p["uid"] = "1000" },
	} {
		c := configFixture(false)
		mutate(c["caller"].(map[string]any))
		got, err := parseClient(canonicalFixture(c))
		if !errors.Is(err, ErrConfig) || !reflect.DeepEqual(got, clientConfig{}) {
			t.Fatal("invalid nested principal escaped", err)
		}
	}
}
