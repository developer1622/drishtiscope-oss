// Package protocol is the v1 WebSocket/HTTP envelope shared by the server.
package protocol

import (
	"encoding/json"
	"fmt"
	"time"
)

const SchemaVersion = 1

const (
	KindHello     = "hello"
	KindSnapshot  = "snapshot"
	KindEvent     = "event"
	KindHeartbeat = "heartbeat"
	KindError     = "error"
)

const (
	ModeEBPF = "ebpf"
	ModeReal = "real"
	ModeMock = "mock"
)

// Envelope is the wire format for every WebSocket frame and the snapshot REST wrapper.
type Envelope struct {
	V       int             `json:"v"`
	Kind    string          `json:"kind"`
	TS      string          `json:"ts"`
	Mode    string          `json:"mode"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Hello is sent once on WebSocket connect and as GET /api/meta.
type Hello struct {
	Schema         int    `json:"schema"`
	Hostname       string `json:"hostname"`
	Kernel         string `json:"kernel"`
	EbpfFailReason string `json:"ebpf_fail_reason,omitempty"`
	Product        string `json:"product"`
	Target         Target `json:"target"`
}

type Target struct {
	PID  int    `json:"pid"`
	Comm string `json:"comm"`
}

type ErrorPayload struct {
	Message string `json:"message"`
}

type HeartbeatPayload struct {
	UptimeS float64 `json:"uptime_s"`
}

// Marshal builds a v1 envelope. payload may be nil.
func Marshal(kind, mode string, payload any) ([]byte, error) {
	var raw json.RawMessage
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("protocol marshal payload: %w", err)
		}
		raw = b
	}
	return json.Marshal(Envelope{
		V:       SchemaVersion,
		Kind:    kind,
		TS:      time.Now().UTC().Format(time.RFC3339),
		Mode:    mode,
		Payload: raw,
	})
}

func ValidKind(kind string) bool {
	switch kind {
	case KindHello, KindSnapshot, KindEvent, KindHeartbeat, KindError:
		return true
	default:
		return false
	}
}

func ValidMode(mode string) bool {
	switch mode {
	case ModeEBPF, ModeReal, ModeMock, "auto":
		return true
	default:
		return false
	}
}
