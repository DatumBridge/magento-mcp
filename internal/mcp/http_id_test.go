package mcp

import (
	"encoding/json"
	"testing"
)

func TestJSONRPCID_acceptsStringAndNumber(t *testing.T) {
	if got := string(jsonRPCID(json.RawMessage(`"call-1"`))); got != `"call-1"` {
		t.Fatalf("string id = %s", got)
	}
	if got := string(jsonRPCID(json.RawMessage(`42`))); got != "42" {
		t.Fatalf("number id = %s", got)
	}
}

func TestJSONRPCID_refusesObjectAndArray(t *testing.T) {
	for _, raw := range []string{`{"a":1}`, `[1]`, `null`, ``} {
		if jsonRPCID(json.RawMessage(raw)) != nil {
			t.Fatalf("id %s must be refused", raw)
		}
	}
}
