package protocol

import (
	"encoding/json"
	"testing"
)

func TestMarshalRoundTrip(t *testing.T) {
	b, err := Marshal(KindHello, ModeMock, Hello{
		Schema:  SchemaVersion,
		Product: "DrishtiScope",
		Target:  Target{PID: 1, Comm: "agy"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	if env.V != SchemaVersion || env.Kind != KindHello || env.Mode != ModeMock {
		t.Fatalf("envelope %+v", env)
	}
	var h Hello
	if err := json.Unmarshal(env.Payload, &h); err != nil {
		t.Fatal(err)
	}
	if h.Target.Comm != "agy" || h.Product != "DrishtiScope" {
		t.Fatalf("hello %+v", h)
	}
}

func TestMarshalNilPayload(t *testing.T) {
	b, err := Marshal(KindHeartbeat, ModeEBPF, nil)
	if err != nil {
		t.Fatal(err)
	}
	var env Envelope
	if err := json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Payload) != 0 && string(env.Payload) != "null" {
		t.Fatalf("payload %s", env.Payload)
	}
}

func TestValidKindMode(t *testing.T) {
	if !ValidKind(KindSnapshot) || ValidKind("nope") {
		t.Fatal("kind")
	}
	if !ValidMode("auto") || !ValidMode(ModeMock) || ValidMode("x") {
		t.Fatal("mode")
	}
}
