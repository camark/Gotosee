// Package acp 提供 ACP 服务器实现。
package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ACPServer ACP 服务器。
type ACPServer struct {
	mu          sync.RWMutex
	sessions    map[SessionID]*ACPSession
	tools       map[string]ACPTool
	middleware  []Middleware
	serverInfo  ServerInfo
	capabilities ServerCapabilities
}

// ACPSession ACP 会话。
type ACPSession struct {
	ID           SessionID
	Mode         SessionMode
	Model        *ModelInfo
	Messages     []Message
	ToolCalls    map[ToolCallID]*ToolCall
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Status       SessionStatus
	Permissions  map[string]*PermissionDecision
	mu           sync.RWMutex
}

// ACPTool ACP 工具定义。
type ACPTool struct {
	Name        string
	Description string
	Handler     ACPToolHandler
	InputSchema json.RawMessage
}

// ACPToolHandler 工具处理函数。
type ACPToolHandler func(ctx context.Context, session *ACPSession, args json.RawMessage) (*ToolCallResult, error)

// Middleware ACP 中间件。
type Middleware func(next ACPToolHandler) ACPToolHandler

// NewACPServer 创建新的 ACP 服务器。
func NewACPServer() *ACPServer {
	return &ACPServer{
		sessions:   make(map[SessionID]*ACPSession),
		tools:      make(map[string]ACPTool),
		serverInfo: ServerInfo{
			Name:    "gogo-acp",
			Version: "1.0.0",
		},
		capabilities: ServerCapabilities{
			Sessions: &SessionCapabilities{ListChanged: true},
			Tools:    &ToolCapabilities{ListChanged: true},
		},
	}
}

// Initialize 初始化服务器。
func (s *ACPServer) Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResponse, error) {
	return &InitializeResponse{
		ProtocolVersion: "2024-11-05",
		ServerInfo:      s.serverInfo,
		Capabilities:    s.capabilities,
	}, nil
}

// NewSession 创建新会话。
func (s *ACPServer) NewSession(ctx context.Context, req *NewSessionRequest) (*NewSessionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := SessionID(generateSessionID())

	mode := ModeAuto
	if req.Mode != nil {
		mode = *req.Mode
	}

	session := &ACPSession{
		ID:          sessionID,
		Mode:        mode,
		Model:       req.Model,
		Messages:    make([]Message, 0),
		ToolCalls:   make(map[ToolCallID]*ToolCall),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      SessionStatusActive,
		Permissions: make(map[string]*PermissionDecision),
	}

	s.sessions[sessionID] = session

	return &NewSessionResponse{
		SessionID: sessionID,
	}, nil
}

// CloseSession 关闭会话。
func (s *ACPServer) CloseSession(ctx context.Context, req *CloseSessionRequest) (*CloseSessionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[req.SessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	session.mu.Lock()
	session.Status = SessionStatusClosed
	session.mu.Unlock()

	return &CloseSessionResponse{}, nil
}

// ListSessions 列出所有会话。
func (s *ACPServer) ListSessions(ctx context.Context, req *ListSessionsRequest) (*ListSessionsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]SessionInfo, 0, len(s.sessions))
	for _, session := range s.sessions {
		session.mu.RLock()
		sessions = append(sessions, SessionInfo{
			ID:        session.ID,
			Mode:      session.Mode,
			Model:     session.Model,
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
			Status:    session.Status,
		})
		session.mu.RUnlock()
	}

	return &ListSessionsResponse{
		Sessions: sessions,
	}, nil
}

// SetSessionMode 设置会话模式。
func (s *ACPServer) SetSessionMode(ctx context.Context, req *SetSessionModeRequest) (*SetSessionModeResponse, error) {
	s.mu.RLock()
	session, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	session.mu.Lock()
	session.Mode = req.Mode
	session.UpdatedAt = time.Now()
	session.mu.Unlock()

	return &SetSessionModeResponse{}, nil
}

// ForkSession 复制会话。
func (s *ACPServer) ForkSession(ctx context.Context, req *ForkSessionRequest) (*ForkSessionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceSession, ok := s.sessions[req.SessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	newSessionID := SessionID(generateSessionID())

	sourceSession.mu.RLock()
	newSession := &ACPSession{
		ID:          newSessionID,
		Mode:        sourceSession.Mode,
		Model:       sourceSession.Model,
		Messages:    make([]Message, len(sourceSession.Messages)),
		ToolCalls:   make(map[ToolCallID]*ToolCall),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      SessionStatusActive,
		Permissions: make(map[string]*PermissionDecision),
	}
	copy(newSession.Messages, sourceSession.Messages)
	sourceSession.mu.RUnlock()

	s.sessions[newSessionID] = newSession

	return &ForkSessionResponse{
		SessionID: newSessionID,
	}, nil
}

// LoadSession 加载会话。
func (s *ACPServer) LoadSession(ctx context.Context, req *LoadSessionRequest) (*LoadSessionResponse, error) {
	s.mu.RLock()
	session, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	sessionInfo := SessionInfo{
		ID:        session.ID,
		Mode:      session.Mode,
		Model:     session.Model,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
		Status:    session.Status,
	}

	return &LoadSessionResponse{
		Session:  sessionInfo,
		Messages: session.Messages,
	}, nil
}

// RegisterTool 注册工具。
func (s *ACPServer) RegisterTool(tool ACPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
}

// CallTool 调用工具。
func (s *ACPServer) CallTool(ctx context.Context, sessionID SessionID, toolCall *ToolCall) (*ToolCallResult, error) {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	tool, toolOk := s.tools[toolCall.Name]
	s.mu.RUnlock()

	if !ok {
		return &ToolCallResult{
			ToolCallID: toolCall.ToolCallID,
			Status:     ToolCallStatusError,
			Error:      &ToolCallError{Code: "NOT_FOUND", Message: "session not found"},
		}, nil
	}

	if !toolOk {
		return &ToolCallResult{
			ToolCallID: toolCall.ToolCallID,
			Status:     ToolCallStatusError,
			Error:      &ToolCallError{Code: "NOT_FOUND", Message: fmt.Sprintf("tool not found: %s", toolCall.Name)},
		}, nil
	}

	// 检查权限
	if session.Mode == ModeChat {
		return &ToolCallResult{
			ToolCallID: toolCall.ToolCallID,
			Status:     ToolCallStatusCancelled,
			Error:      &ToolCallError{Code: "CANCELLED", Message: "tools disabled in chat mode"},
		}, nil
	}

	// 执行工具
	result, err := tool.Handler(ctx, session, toolCall.Arguments)
	if err != nil {
		return &ToolCallResult{
			ToolCallID: toolCall.ToolCallID,
			Status:     ToolCallStatusError,
			Error:      &ToolCallError{Code: "INTERNAL_ERROR", Message: err.Error()},
		}, nil
	}

	return result, nil
}

// RequestPermission 请求权限。
func (s *ACPServer) RequestPermission(ctx context.Context, req *RequestPermissionRequest) (*RequestPermissionResponse, error) {
	s.mu.RLock()
	session, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}

	// 检查是否已有权限决定
	session.mu.RLock()
	if decision, ok := session.Permissions[req.ToolName]; ok {
		session.mu.RUnlock()
		return &RequestPermissionResponse{
			Outcome: RequestPermissionOutcome{
				Decision: decision,
			},
		}, nil
	}
	session.mu.RUnlock()

	// 在实际应用中，这里需要等待用户输入
	// 这里返回默认拒绝
	return &RequestPermissionResponse{
		Outcome: RequestPermissionOutcome{
			Decision: &PermissionDecision{
				Approve: false,
			},
		},
	}, nil
}

// AddMessage 添加消息到会话。
func (s *ACPServer) AddMessage(sessionID SessionID, msg Message) error {
	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Messages = append(session.Messages, msg)
	session.UpdatedAt = time.Now()

	return nil
}

// GetSession 获取会话。
func (s *ACPServer) GetSession(sessionID SessionID) (*ACPSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// generateSessionID 生成会话 ID。
func generateSessionID() string {
	// 简单实现，实际应使用更安全的 UUID
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
