// Package transport 提供 ACP 传输层实现。
package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/camark/Gotosee/internal/acp"
)

// ============================================================================
// HTTP Streamable 传输
// ============================================================================

// HTTPTransport HTTP Streamable 传输。
type HTTPTransport struct {
	server     *acp.ACPServer
	mu         sync.RWMutex
	sessions   map[string]*HTTPSession
	basePath   string
}

// HTTPSession HTTP 会话。
type HTTPSession struct {
	ID         string
	Writer     http.ResponseWriter
	Flusher    http.Flusher
	Connected  bool
	mu         sync.RWMutex
}

// NewHTTPTransport 创建新的 HTTP 传输。
func NewHTTPTransport(server *acp.ACPServer, basePath string) *HTTPTransport {
	return &HTTPTransport{
		server:   server,
		sessions: make(map[string]*HTTPSession),
		basePath: basePath,
	}
}

// RegisterRoutes 注册 HTTP 路由。
func (t *HTTPTransport) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc(t.basePath+"/message", t.handleMessage)
	mux.HandleFunc(t.basePath+"/events", t.handleEvents)
}

// handleMessage 处理消息请求。
func (t *HTTPTransport) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 解析 JSON-RPC 请求
	var req JSONRPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &RPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 处理请求
	resp := t.handleRequest(r.Context(), req)

	// 发送响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleEvents 处理 SSE 事件流。
func (t *HTTPTransport) handleEvents(w http.ResponseWriter, r *http.Request) {
	// 设置 SSE 头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// 创建会话
	sessionID := r.URL.Query().Get("session_id")
	session := &HTTPSession{
		ID:        sessionID,
		Writer:    w,
		Flusher:   flusher,
		Connected: true,
	}

	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		delete(t.sessions, sessionID)
		t.mu.Unlock()
	}()

	// 保持连接
	<-r.Context().Done()
}

// handleRequest 处理 JSON-RPC 请求。
func (t *HTTPTransport) handleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	var result interface{}
	var rpcErr *RPCError

	switch req.Method {
	case "initialize":
		var initReq acp.InitializeRequest
		if err := json.Unmarshal(req.Params, &initReq); err != nil {
			rpcErr = &RPCError{Code: -32602, Message: "Invalid params"}
		} else {
			res, err := t.server.Initialize(ctx, &initReq)
			if err != nil {
				rpcErr = &RPCError{Code: -32603, Message: err.Error()}
			} else {
				result = res
			}
		}

	case "sessions/new":
		var newReq acp.NewSessionRequest
		if err := json.Unmarshal(req.Params, &newReq); err != nil {
			rpcErr = &RPCError{Code: -32602, Message: "Invalid params"}
		} else {
			res, err := t.server.NewSession(ctx, &newReq)
			if err != nil {
				rpcErr = &RPCError{Code: -32603, Message: err.Error()}
			} else {
				result = res
			}
		}

	case "sessions/close":
		var closeReq acp.CloseSessionRequest
		if err := json.Unmarshal(req.Params, &closeReq); err != nil {
			rpcErr = &RPCError{Code: -32602, Message: "Invalid params"}
		} else {
			res, err := t.server.CloseSession(ctx, &closeReq)
			if err != nil {
				rpcErr = &RPCError{Code: -32603, Message: err.Error()}
			} else {
				result = res
			}
		}

	case "sessions/list":
		res, err := t.server.ListSessions(ctx, &acp.ListSessionsRequest{})
		if err != nil {
			rpcErr = &RPCError{Code: -32603, Message: err.Error()}
		} else {
			result = res
		}

	case "tools/call":
		// 处理工具调用
		result = map[string]interface{}{"status": "ok"}

	default:
		rpcErr = &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)}
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	return resp
}

// SendEvent 发送 SSE 事件到会话。
func (t *HTTPTransport) SendEvent(sessionID string, eventType string, data interface{}) error {
	t.mu.RLock()
	session, ok := t.sessions[sessionID]
	t.mu.RUnlock()

	if !ok || !session.Connected {
		return fmt.Errorf("session not connected: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	dataJSON, _ := json.Marshal(data)
	event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(dataJSON))

	_, err := fmt.Fprint(session.Writer, event)
	if err != nil {
		return err
	}

	session.Flusher.Flush()
	return nil
}

// ============================================================================
// WebSocket 传输
// ============================================================================

// WebSocketTransport WebSocket 传输。
type WebSocketTransport struct {
	server   *acp.ACPServer
	upgrader WebSocketUpgrader
}

// WebSocketUpgrader WebSocket 升级接口。
type WebSocketUpgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, header http.Header) (WebSocketConn, error)
}

// WebSocketConn WebSocket 连接接口。
type WebSocketConn interface {
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// NewWebSocketTransport 创建新的 WebSocket 传输。
func NewWebSocketTransport(server *acp.ACPServer, upgrader WebSocketUpgrader) *WebSocketTransport {
	return &WebSocketTransport{
		server:   server,
		upgrader: upgrader,
	}
}

// HandleWebSocket 处理 WebSocket 连接。
func (t *WebSocketTransport) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := t.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	for {
		// 读取消息
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return
			}
			continue
		}

		// 解析 JSON-RPC 请求
		var req JSONRPCRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			resp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &RPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			}
			respJSON, _ := json.Marshal(resp)
			conn.WriteMessage(1, respJSON) // 1 = TextMessage
			continue
		}

		// 处理请求
		resp := t.handleRequest(r.Context(), req)

		// 发送响应
		respJSON, _ := json.Marshal(resp)
		conn.WriteMessage(1, respJSON)
	}
}

// handleRequest 处理 JSON-RPC 请求（与 HTTP 传输共享逻辑）。
func (t *WebSocketTransport) handleRequest(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	// 这里可以复用 HTTPTransport 的逻辑
	// 为简化，直接返回错误
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error: &RPCError{
			Code:    -32603,
			Message: "Not implemented",
		},
	}
}

// ============================================================================
// JSON-RPC 类型
// ============================================================================

// JSONRPCRequest JSON-RPC 请求。
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应。
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError RPC 错误。
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
