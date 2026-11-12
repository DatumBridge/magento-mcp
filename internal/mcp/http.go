package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
)

const (
	ProtocolVersion = "2024-11-05"
	SessionHeader   = "Mcp-Session-Id"
	MaxBodySize     = 1 << 20 // 1 MiB
	ServerName      = "magento-mcp"
	ServerVersion   = "1.0.0"
)

// ToolHandler executes a named MCP tool and returns result content.
type ToolHandler func(args json.RawMessage) (result map[string]interface{})

// ToolDesc is an MCP tools/list descriptor.
type ToolDesc struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Server is the Streamable HTTP MCP tool-server.
type Server struct {
	Sessions *SessionStore
	Tools    []ToolDesc
	Handlers map[string]ToolHandler
}

type rpcEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

type callParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// HandleStreamableHTTP implements initialize, tools/list, tools/call.
func (s *Server) HandleStreamableHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}
	_ = r.Body.Close()

	var env rpcEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		WriteRPCError(w, nil, -32700, "parse error")
		return
	}
	if env.JSONRPC != "2.0" {
		WriteRPCError(w, env.ID, -32600, "invalid request")
		return
	}

	switch env.Method {
	case "initialize":
		s.handleInitialize(w, env.ID)
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		s.handleToolsList(w, r, env.ID)
	case "tools/call":
		s.handleToolsCall(w, r, env)
	default:
		WriteRPCError(w, env.ID, -32601, "method not found: "+env.Method)
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, id json.RawMessage) {
	sid := s.Sessions.Create()
	w.Header().Set(SessionHeader, sid)
	w.Header().Set("Content-Type", "application/json")
	result := map[string]interface{}{
		"protocolVersion": ProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]string{
			"name":    ServerName,
			"version": ServerVersion,
		},
		"sessionId": sid,
	}
	WriteRPCResult(w, id, result)
	preview := sid
	if len(preview) > 8 {
		preview = preview[:8] + "…"
	}
	log.Debug().Str("mcp_session", preview).Msg("mcp initialize")
}

func (s *Server) getSessionID(r *http.Request, rawParams json.RawMessage) string {
	if sid := strings.TrimSpace(r.Header.Get(SessionHeader)); sid != "" {
		return sid
	}
	if len(rawParams) > 0 && string(rawParams) != "null" {
		var p struct {
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(rawParams, &p); err == nil && strings.TrimSpace(p.SessionID) != "" {
			return strings.TrimSpace(p.SessionID)
		}
	}
	return ""
}

func (s *Server) handleToolsList(w http.ResponseWriter, r *http.Request, id json.RawMessage) {
	sid := s.getSessionID(r, nil)
	if !s.Sessions.Valid(sid) {
		WriteRPCError(w, id, -32000, "invalid or missing Mcp-Session-Id (call initialize first)")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	WriteRPCResult(w, id, map[string]interface{}{"tools": s.Tools})
}

func (s *Server) handleToolsCall(w http.ResponseWriter, r *http.Request, env rpcEnvelope) {
	sid := s.getSessionID(r, env.Params)
	if !s.Sessions.Valid(sid) {
		WriteRPCError(w, env.ID, -32000, "invalid or missing Mcp-Session-Id (call initialize first)")
		return
	}

	var params callParams
	if len(env.Params) > 0 && string(env.Params) != "null" {
		if err := json.Unmarshal(env.Params, &params); err != nil {
			WriteRPCError(w, env.ID, -32602, "invalid params")
			return
		}
	}
	if params.Arguments == nil {
		params.Arguments = []byte("{}")
	}

	w.Header().Set("Content-Type", "application/json")
	h, ok := s.Handlers[params.Name]
	if !ok {
		WriteRPCError(w, env.ID, -32601, "unknown tool: "+params.Name)
		return
	}
	WriteRPCResult(w, env.ID, h(params.Arguments))
}

// ToolResultText builds a successful MCP tool result.
func ToolResultText(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": text}},
		"isError": false,
	}
}

// ToolResultError builds an MCP tool domain error (isError=true).
func ToolResultError(text string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]string{{"type": "text", "text": text}},
		"isError": true,
	}
}

type rpcSuccess struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result"`
}

type rpcFailure struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Error   rpcErrorBody    `json:"error"`
}

type rpcErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// jsonRPCID accepts a JSON-RPC id that is a string or a number.
// Objects and arrays are refused so the id is never decoded into an open type.
func jsonRPCID(id json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(id)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || !json.Valid(trimmed) {
		return nil
	}
	switch trimmed[0] {
	case '"':
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return nil
		}
		out, err := json.Marshal(s)
		if err != nil {
			return nil
		}
		return out
	default:
		if trimmed[0] != '-' && !unicode.IsDigit(rune(trimmed[0])) {
			return nil
		}
		dec := json.NewDecoder(bytes.NewReader(trimmed))
		dec.UseNumber()
		var n json.Number
		if err := dec.Decode(&n); err != nil || !json.Valid([]byte(n.String())) {
			return nil
		}
		var trail json.RawMessage
		if err := dec.Decode(&trail); err != io.EOF {
			return nil
		}
		return append(json.RawMessage(nil), n.String()...)
	}
}

// WriteRPCResult writes a JSON-RPC success response.
func WriteRPCResult(w http.ResponseWriter, id json.RawMessage, result interface{}) {
	body, err := json.Marshal(result)
	if err != nil {
		WriteRPCError(w, id, -32603, "internal error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcSuccess{
		JSONRPC: "2.0",
		ID:      jsonRPCID(id),
		Result:  body,
	})
}

// WriteRPCError writes a JSON-RPC error response.
func WriteRPCError(w http.ResponseWriter, id json.RawMessage, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rpcFailure{
		JSONRPC: "2.0",
		ID:      jsonRPCID(id),
		Error: rpcErrorBody{
			Code:    code,
			Message: message,
		},
	})
}
