// Package state 提供应用状态管理。
package state

import (
	"context"
	"sync"

	"github.com/camark/Gotosee/internal/acp"
	"github.com/camark/Gotosee/internal/mcp"
	"github.com/camark/Gotosee/internal/session"
)

// AppState 应用状态。
type AppState struct {
	mu                sync.RWMutex
	sessionManager    *session.SessionManager
	sessions          map[string]*SessionState
	acpServer         *acp.ACPServer
	mcpRunners        map[string]*mcp.MCPServerRunner
	workingDirs       map[string]string
	extensions        []ExtensionState
	mode              string
	currentProvider   string
	currentModel      string
}

// SessionState 会话状态。
type SessionState struct {
	ID           string
	Active       bool
	WorkingDir   string
	Provider     string
	Model        string
	Mode         string
	Extensions   []ExtensionState
	CancelToken  interface{} // 用于取消会话的 token
	mu           sync.RWMutex
}

// ExtensionState 扩展状态。
type ExtensionState struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Config   interface{} `json:"config,omitempty"`
	Attached bool   `json:"attached,omitempty"`
}

// NewAppState 创建新的应用状态。
func NewAppState() *AppState {
	// 创建会话管理器
	sessionConfig := session.DefaultSessionManagerConfig()
	sessionManager, _ := session.NewSessionManager(sessionConfig)

	return &AppState{
		sessionManager: sessionManager,
		sessions:     make(map[string]*SessionState),
		acpServer:    nil,
		mcpRunners:   make(map[string]*mcp.MCPServerRunner),
		workingDirs:  make(map[string]string),
		extensions:   make([]ExtensionState, 0),
		mode:         "auto",
		currentProvider: "",
		currentModel:    "",
	}
}

// GetSession 获取会话状态。
func (s *AppState) GetSession(sessionID string) (*SessionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// CreateSession 创建新会话。
func (s *AppState) CreateSession(sessionID string) *SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := &SessionState{
		ID:         sessionID,
		Active:     true,
		WorkingDir: "",
		Provider:   "",
		Model:      "",
		Mode:       s.mode,
		Extensions: make([]ExtensionState, 0),
	}

	s.sessions[sessionID] = session
	return session
}

// CloseSession 关闭会话。
func (s *AppState) CloseSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	session.mu.Lock()
	session.Active = false
	session.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// ListSessions 列出所有会话。
func (s *AppState) ListSessions() []*SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*SessionState, 0, len(s.sessions))
	for _, session := range s.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// ListPersistentSessions 列出持久化会话。
func (s *AppState) ListPersistentSessions(ctx context.Context, limit, offset int) ([]*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionManager.ListSessions(ctx, limit, offset)
}

// CreatePersistentSession 创建持久化会话。
func (s *AppState) CreatePersistentSession(ctx context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionManager.CreateSession(ctx, sess)
}

// GetPersistentSession 获取持久化会话。
func (s *AppState) GetPersistentSession(ctx context.Context, id string) (*session.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionManager.GetSession(ctx, id)
}

// UpdatePersistentSession 更新持久化会话。
func (s *AppState) UpdatePersistentSession(ctx context.Context, sess *session.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionManager.UpdateSession(ctx, sess)
}

// DeletePersistentSession 删除持久化会话。
func (s *AppState) DeletePersistentSession(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionManager.DeleteSession(ctx, id)
}

// SetWorkingDir 设置工作目录。
func (s *AppState) SetWorkingDir(sessionID string, dir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workingDirs[sessionID] = dir

	if session, ok := s.sessions[sessionID]; ok {
		session.mu.Lock()
		session.WorkingDir = dir
		session.mu.Unlock()
	}

	return nil
}

// GetWorkingDir 获取工作目录。
func (s *AppState) GetWorkingDir(sessionID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workingDirs[sessionID]
}

// SetACPServer 设置 ACP 服务器。
func (s *AppState) SetACPServer(server *acp.ACPServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acpServer = server
}

// GetACPServer 获取 ACP 服务器。
func (s *AppState) GetACPServer() *acp.ACPServer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.acpServer
}

// RegisterMCPRunner 注册 MCP 运行器。
func (s *AppState) RegisterMCPRunner(name string, runner *mcp.MCPServerRunner) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mcpRunners[name] = runner
}

// GetMCPRunner 获取 MCP 运行器。
func (s *AppState) GetMCPRunner(name string) *mcp.MCPServerRunner {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mcpRunners[name]
}

// SetMode 设置模式。
func (s *AppState) SetMode(mode string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

// GetMode 获取模式。
func (s *AppState) GetMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// SetProvider 设置当前提供商。
func (s *AppState) SetProvider(provider string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentProvider = provider
}

// GetProvider 获取当前提供商。
func (s *AppState) GetProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentProvider
}

// SetModel 设置当前模型。
func (s *AppState) SetModel(model string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentModel = model
}

// GetModel 获取当前模型。
func (s *AppState) GetModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentModel
}

// Close 关闭应用状态（关闭会话管理器）。
func (s *AppState) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessionManager != nil {
		return s.sessionManager.Close()
	}
	return nil
}

// 错误定义
var (
	ErrSessionNotFound = &APIError{
		Code:    "SESSION_NOT_FOUND",
		Message: "会话不存在",
		Status:  404,
	}
	ErrInvalidRequest = &APIError{
		Code:    "INVALID_REQUEST",
		Message: "无效的请求",
		Status:  400,
	}
)

// APIError API 错误。
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *APIError) Error() string {
	return e.Message
}
