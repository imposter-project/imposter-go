package external

import (
	"testing"
)

func TestFormatMsg_Plain(t *testing.T) {
	b := &bridgeLogger{}
	got := b.formatMsg("hello")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestFormatMsg_WithKeyValuePairs(t *testing.T) {
	b := &bridgeLogger{}
	got := b.formatMsg("starting plugin", "path", "/usr/bin/plugin", "pid", 1234)
	want := "starting plugin path=/usr/bin/plugin pid=1234"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatMsg_WithName(t *testing.T) {
	b := &bridgeLogger{name: "plugin.grpc"}
	got := b.formatMsg("ready")
	want := "plugin.grpc: ready"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatMsg_WithImpliedArgs(t *testing.T) {
	b := &bridgeLogger{impliedArgs: []interface{}{"component", "rpc"}}
	got := b.formatMsg("connected", "addr", "unix:///tmp/sock")
	want := "connected component=rpc addr=unix:///tmp/sock"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestNamed_Appends(t *testing.T) {
	b := &bridgeLogger{name: "plugin"}
	child := b.Named("grpc")
	if child.Name() != "plugin.grpc" {
		t.Errorf("got %q, want %q", child.Name(), "plugin.grpc")
	}
}

func TestNamed_FromEmpty(t *testing.T) {
	b := &bridgeLogger{}
	child := b.Named("plugin")
	if child.Name() != "plugin" {
		t.Errorf("got %q, want %q", child.Name(), "plugin")
	}
}

func TestWith_PreservesExisting(t *testing.T) {
	b := &bridgeLogger{impliedArgs: []interface{}{"a", 1}}
	child := b.With("b", 2).(*bridgeLogger)

	if len(child.impliedArgs) != 4 {
		t.Fatalf("got %d implied args, want 4", len(child.impliedArgs))
	}
	got := child.formatMsg("msg")
	want := "msg a=1 b=2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatMsg_OddArgs_IgnoresTrailing(t *testing.T) {
	b := &bridgeLogger{}
	got := b.formatMsg("msg", "key")
	// Odd trailing arg has no pair, so it's skipped
	if got != "msg" {
		t.Errorf("got %q, want %q", got, "msg")
	}
}
