// Package session 提供会话管理功能。
package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ============================================================================
// 基础类型
// ============================================================================

// SessionType 会话类型。
type SessionType string

const (
	SessionTypeUser      SessionType = "user"
	SessionTypeScheduled SessionType = "scheduled"
	SessionTypeSubAgent  SessionType = "sub_agent"
	SessionTypeHidden    SessionType = "hidden"
	SessionTypeTerminal  SessionType = "terminal"
	SessionTypeGateway   SessionType = "gateway"
	SessionTypeAcp       SessionType = "acp"
)

// Session 会话结构。
type Session struct {
	ID              string                 `json:"id"`
	WorkingDir      string                 `json:"working_dir"`
	Name            string                 `json:"name"`
	UserSetName     bool                   `json:"user_set_name"`
	SessionType     SessionType            `json:"session_type"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	ExtensionData   *ExtensionData         `json:"extension_data,omitempty"`
	TotalTokens     *int                   `json:"total_tokens,omitempty"`
	InputTokens     *int                   `json:"input_tokens,omitempty"`
	OutputTokens    *int                   `json:"output_tokens,omitempty"`
	ScheduleID      *string                `json:"schedule_id,omitempty"`
	Recipe          *Recipe                `json:"recipe,omitempty"`
	MessageCount    int                    `json:"message_count"`
	ProviderName    *string                `json:"provider_name,omitempty"`
	ModelConfig     *ModelConfig           `json:"model_config,omitempty"`
	GooseMode       string                 `json:"goose_mode"`
	ThreadID        *string                `json:"thread_id,omitempty"`
	mu              sync.RWMutex
}

// ModelConfig 模型配置（简化版）。
type ModelConfig struct {
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	ContextLimit int     `json:"context_limit"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
}

// Recipe 配方（简化版）。
type Recipe struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ExtensionData 扩展数据。
type ExtensionData struct {
	ExtensionStates map[string]json.RawMessage `json:"-"`
}

// MarshalJSON 实现 JSON 序列化。
func (e *ExtensionData) MarshalJSON() ([]byte, error) {
	if e.ExtensionStates == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(e.ExtensionStates)
}

// UnmarshalJSON 实现 JSON 反序列化。
func (e *ExtensionData) UnmarshalJSON(data []byte) error {
	if e.ExtensionStates == nil {
		e.ExtensionStates = make(map[string]json.RawMessage)
	}
	return json.Unmarshal(data, &e.ExtensionStates)
}

// NewExtensionData 创建新的扩展数据。
func NewExtensionData() *ExtensionData {
	return &ExtensionData{
		ExtensionStates: make(map[string]json.RawMessage),
	}
}

// GetExtensionState 获取扩展状态。
func (e *ExtensionData) GetExtensionState(name, version string) json.RawMessage {
	key := fmt.Sprintf("%s.%s", name, version)
	return e.ExtensionStates[key]
}

// SetExtensionState 设置扩展状态。
func (e *ExtensionData) SetExtensionState(name, version string, state json.RawMessage) {
	key := fmt.Sprintf("%s.%s", name, version)
	e.ExtensionStates[key] = state
}

// NewSession 创建新会话。
func NewSession(id, workingDir, name string) *Session {
	now := time.Now()
	return &Session{
		ID:           id,
		WorkingDir:   workingDir,
		Name:         name,
		SessionType:  SessionTypeUser,
		CreatedAt:    now,
		UpdatedAt:    now,
		ExtensionData: NewExtensionData(),
		MessageCount: 0,
		GooseMode:    "auto",
	}
}

// Update 更新会话。
func (s *Session) Update(name string, workingDir string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if name != "" {
		s.Name = name
		s.UserSetName = true
	}
	if workingDir != "" {
		s.WorkingDir = workingDir
	}
	s.UpdatedAt = time.Now()
}

// AddMessage 添加消息计数。
func (s *Session) AddMessage() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.MessageCount++
	s.UpdatedAt = time.Now()
}

// ============================================================================
// SessionManager 会话管理器
// ============================================================================

// SessionManager 会话管理器。
type SessionManager struct {
	db           *sql.DB
	dbPath       string
	mu           sync.RWMutex
	cache        map[string]*Session
	autoSave     bool
}

// SessionManagerConfig 会话管理器配置。
type SessionManagerConfig struct {
	DBPath   string `json:"db_path"`
	AutoSave bool   `json:"auto_save"`
}

// DefaultSessionManagerConfig 返回默认配置。
func DefaultSessionManagerConfig() *SessionManagerConfig {
	return &SessionManagerConfig{
		DBPath:   "sessions.db",
		AutoSave: true,
	}
}

// NewSessionManager 创建新的会话管理器。
func NewSessionManager(config *SessionManagerConfig) (*SessionManager, error) {
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	sm := &SessionManager{
		db:       db,
		dbPath:   config.DBPath,
		cache:    make(map[string]*Session),
		autoSave: config.AutoSave,
	}

	// 初始化数据库
	if err := sm.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return sm, nil
}

// initDB 初始化数据库表。
func (sm *SessionManager) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		working_dir TEXT NOT NULL,
		name TEXT NOT NULL,
		user_set_name INTEGER DEFAULT 0,
		session_type TEXT DEFAULT 'user',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		extension_data TEXT,
		total_tokens INTEGER,
		input_tokens INTEGER,
		output_tokens INTEGER,
		schedule_id TEXT,
		recipe TEXT,
		message_count INTEGER DEFAULT 0,
		provider_name TEXT,
		model_config TEXT,
		goose_mode TEXT DEFAULT 'auto',
		thread_id TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at);
	CREATE INDEX IF NOT EXISTS idx_sessions_type ON sessions(session_type);
	`

	_, err := sm.db.Exec(schema)
	return err
}

// Close 关闭会话管理器。
func (sm *SessionManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.db != nil {
		return sm.db.Close()
	}
	return nil
}

// CreateSession 创建新会话。
func (sm *SessionManager) CreateSession(ctx context.Context, session *Session) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	query := `
	INSERT INTO sessions (
		id, working_dir, name, user_set_name, session_type,
		created_at, updated_at, extension_data,
		message_count, goose_mode
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	extData, _ := json.Marshal(session.ExtensionData)

	_, err := sm.db.ExecContext(ctx, query,
		session.ID,
		session.WorkingDir,
		session.Name,
		session.UserSetName,
		session.SessionType,
		session.CreatedAt,
		session.UpdatedAt,
		extData,
		session.MessageCount,
		session.GooseMode,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// 缓存会话
	sm.cache[session.ID] = session
	return nil
}

// GetSession 获取会话。
func (sm *SessionManager) GetSession(ctx context.Context, id string) (*Session, error) {
	sm.mu.RLock()
	if cached, ok := sm.cache[id]; ok {
		sm.mu.RUnlock()
		return cached, nil
	}
	sm.mu.RUnlock()

	query := `
	SELECT id, working_dir, name, user_set_name, session_type,
		created_at, updated_at, extension_data,
		total_tokens, input_tokens, output_tokens,
		schedule_id, recipe, message_count,
		provider_name, model_config, goose_mode, thread_id
	FROM sessions WHERE id = ?
	`

	row := sm.db.QueryRowContext(ctx, query, id)
	session, err := sm.scanSession(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	// 缓存
	sm.mu.Lock()
	sm.cache[id] = session
	sm.mu.Unlock()

	return session, nil
}

// UpdateSession 更新会话。
func (sm *SessionManager) UpdateSession(ctx context.Context, session *Session) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	query := `
	UPDATE sessions SET
		working_dir = ?, name = ?, user_set_name = ?,
		updated_at = ?, extension_data = ?,
		total_tokens = ?, input_tokens = ?, output_tokens = ?,
		message_count = ?, provider_name = ?,
		model_config = ?, goose_mode = ?, thread_id = ?
	WHERE id = ?
	`

	extData, _ := json.Marshal(session.ExtensionData)

	_, err := sm.db.ExecContext(ctx, query,
		session.WorkingDir,
		session.Name,
		session.UserSetName,
		session.UpdatedAt,
		extData,
		session.TotalTokens,
		session.InputTokens,
		session.OutputTokens,
		session.MessageCount,
		session.ProviderName,
		session.GooseMode,
		session.ThreadID,
		session.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// 更新缓存
	sm.cache[session.ID] = session
	return nil
}

// DeleteSession 删除会话。
func (sm *SessionManager) DeleteSession(ctx context.Context, id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	query := `DELETE FROM sessions WHERE id = ?`
	_, err := sm.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// 清除缓存
	delete(sm.cache, id)
	return nil
}

// ListSessions 列出所有会话。
func (sm *SessionManager) ListSessions(ctx context.Context, limit, offset int) ([]*Session, error) {
	query := `
	SELECT id, working_dir, name, user_set_name, session_type,
		created_at, updated_at, extension_data,
		total_tokens, input_tokens, output_tokens,
		schedule_id, recipe, message_count,
		provider_name, model_config, goose_mode, thread_id
	FROM sessions
	ORDER BY updated_at DESC
	LIMIT ? OFFSET ?
	`

	rows, err := sm.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session, err := sm.scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// ListSessionsByType 按类型列出会话。
func (sm *SessionManager) ListSessionsByType(ctx context.Context, sessionType SessionType, limit, offset int) ([]*Session, error) {
	query := `
	SELECT id, working_dir, name, user_set_name, session_type,
		created_at, updated_at, extension_data,
		total_tokens, input_tokens, output_tokens,
		schedule_id, recipe, message_count,
		provider_name, model_config, goose_mode, thread_id
	FROM sessions
	WHERE session_type = ?
	ORDER BY updated_at DESC
	LIMIT ? OFFSET ?
	`

	rows, err := sm.db.QueryContext(ctx, query, sessionType, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session, err := sm.scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// SearchSessions 搜索会话。
func (sm *SessionManager) SearchSessions(ctx context.Context, query string, limit int) ([]*Session, error) {
	sqlQuery := `
	SELECT id, working_dir, name, user_set_name, session_type,
		created_at, updated_at, extension_data,
		total_tokens, input_tokens, output_tokens,
		schedule_id, recipe, message_count,
		provider_name, model_config, goose_mode, thread_id
	FROM sessions
	WHERE name LIKE ? OR working_dir LIKE ?
	ORDER BY updated_at DESC
	LIMIT ?
	`

	searchPattern := "%" + query + "%"
	rows, err := sm.db.QueryContext(ctx, sqlQuery, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session, err := sm.scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// scanSession 从数据库行扫描会话。
func (sm *SessionManager) scanSession(scanner interface{ Scan(...interface{}) error }) (*Session, error) {
	session := &Session{}
	var extData, recipeData, modelData []byte
	var totalTokens, inputTokens, outputTokens sql.NullInt64
	var scheduleID, threadID sql.NullString

	err := scanner.Scan(
		&session.ID,
		&session.WorkingDir,
		&session.Name,
		&session.UserSetName,
		&session.SessionType,
		&session.CreatedAt,
		&session.UpdatedAt,
		&extData,
		&totalTokens,
		&inputTokens,
		&outputTokens,
		&scheduleID,
		&recipeData,
		&session.MessageCount,
		&session.ProviderName,
		&modelData,
		&session.GooseMode,
		&threadID,
	)

	if err != nil {
		return nil, err
	}

	// 处理可选字段
	if totalTokens.Valid {
		v := int(totalTokens.Int64)
		session.TotalTokens = &v
	}
	if inputTokens.Valid {
		v := int(inputTokens.Int64)
		session.InputTokens = &v
	}
	if outputTokens.Valid {
		v := int(outputTokens.Int64)
		session.OutputTokens = &v
	}
	if scheduleID.Valid {
		session.ScheduleID = &scheduleID.String
	}
	if threadID.Valid {
		session.ThreadID = &threadID.String
	}

	// 解析扩展数据
	if extData != nil {
		session.ExtensionData = &ExtensionData{}
		json.Unmarshal(extData, session.ExtensionData)
	}

	// 解析配方
	if recipeData != nil {
		session.Recipe = &Recipe{}
		json.Unmarshal(recipeData, session.Recipe)
	}

	// 解析模型配置
	if modelData != nil {
		session.ModelConfig = &ModelConfig{}
		json.Unmarshal(modelData, session.ModelConfig)
	}

	return session, nil
}

// GetSessionCount 获取会话总数。
func (sm *SessionManager) GetSessionCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM sessions`
	var count int
	err := sm.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// GetSessionStats 获取会话统计。
func (sm *SessionManager) GetSessionStats(ctx context.Context) (*SessionStats, error) {
	query := `
	SELECT
		COUNT(*) as total,
		SUM(message_count) as total_messages,
		SUM(COALESCE(total_tokens, 0)) as total_tokens
	FROM sessions
	`

	stats := &SessionStats{}
	err := sm.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalSessions,
		&stats.TotalMessages,
		&stats.TotalTokens,
	)

	return stats, err
}

// SessionStats 会话统计。
type SessionStats struct {
	TotalSessions int `json:"total_sessions"`
	TotalMessages int `json:"total_messages"`
	TotalTokens   int `json:"total_tokens"`
}

// 错误定义
var (
	ErrSessionNotFound = fmt.Errorf("session not found")
)
