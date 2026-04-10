// Package routes 提供 HTTP 路由处理。
package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/camark/Gotosee/internal/server/state"
	"github.com/camark/Gotosee/internal/session"
)

// Router HTTP 路由器。
type Router struct {
	state       *state.AppState
	mux         *http.ServeMux
	middlewares []Middleware
}

// Middleware HTTP 中间件。
type Middleware func(http.Handler) http.Handler

// NewRouter 创建新的路由器。
func NewRouter(appState *state.AppState) *Router {
	r := &Router{
		state: appState,
		mux:   http.NewServeMux(),
	}
	r.setupRoutes()
	return r
}

// setupRoutes 设置路由。
func (r *Router) setupRoutes() {
	// API 路由
	r.mux.HandleFunc("/api/health", r.handleHealth)
	r.mux.HandleFunc("/api/sessions", r.handleSessions)
	r.mux.HandleFunc("/api/sessions/", r.handleSession)
	r.mux.HandleFunc("/api/tools", r.handleTools)
	r.mux.HandleFunc("/api/provider", r.handleProvider)
	r.mux.HandleFunc("/api/config", r.handleConfig)
	r.mux.HandleFunc("/api/extensions", r.handleExtensions)

	// ACP 端点
	r.mux.HandleFunc("/acp/", r.handleACP)

	// 静态文件
	r.mux.HandleFunc("/", r.handleIndex)
}

// ServeHTTP 实现 http.Handler。
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var handler http.Handler = r.mux

	// 应用中间件（从后往前）
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	handler.ServeHTTP(w, req)
}

// Use 添加中间件。
func (r *Router) Use(middleware Middleware) {
	r.middlewares = append(r.middlewares, middleware)
}

// handleHealth 健康检查。
func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"status": "ok",
		"version": "1.0.0",
	}
	writeJSON(w, http.StatusOK, response)
}

// handleSessions 会话管理。
func (r *Router) handleSessions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.listSessions(w, req)
	case http.MethodPost:
		r.createSession(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSession 单个会话处理。
func (r *Router) handleSession(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.getSession(w, req)
	case http.MethodPut:
		r.updateSession(w, req)
	case http.MethodDelete:
		r.deleteSession(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTools 工具管理。
func (r *Router) handleTools(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.listTools(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProvider 提供商管理。
func (r *Router) handleProvider(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.getProvider(w, req)
	case http.MethodPut:
		r.updateProvider(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfig 配置管理。
func (r *Router) handleConfig(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.getConfig(w, req)
	case http.MethodPut:
		r.updateConfig(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleExtensions 扩展管理。
func (r *Router) handleExtensions(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		r.listExtensions(w, req)
	case http.MethodPost:
		r.addExtension(w, req)
	case http.MethodDelete:
		r.removeExtension(w, req)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleACP ACP 协议端点。
func (r *Router) handleACP(w http.ResponseWriter, req *http.Request) {
	// 转发到 ACP 服务器处理
	acpServer := r.state.GetACPServer()
	if acpServer == nil {
		writeError(w, http.StatusInternalServerError, "ACP server not initialized")
		return
	}

	// TODO: 实现 ACP 请求处理
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "not_implemented",
	})
}

// handleIndex 首页。
func (r *Router) handleIndex(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"name":    "gogo",
		"version": "1.0.0",
		"status":  "running",
	})
}

// listSessions 列出会话。
func (r *Router) listSessions(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	sessions, err := r.state.ListPersistentSessions(ctx, 100, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}

	response := map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	}
	writeJSON(w, http.StatusOK, response)
}

// createSession 创建会话。
func (r *Router) createSession(w http.ResponseWriter, req *http.Request) {
	var reqBody struct {
		ID         string `json:"id,omitempty"`
		WorkingDir string `json:"working_dir,omitempty"`
		Name       string `json:"name,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	sessionID := reqBody.ID
	if sessionID == "" {
		sessionID = generateID()
	}

	session := session.NewSession(sessionID, reqBody.WorkingDir, reqBody.Name)

	ctx := context.Background()
	if err := r.state.CreatePersistentSession(ctx, session); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	response := map[string]interface{}{
		"session_id": session.ID,
		"created":    true,
	}
	writeJSON(w, http.StatusCreated, response)
}

// getSession 获取会话。
func (r *Router) getSession(w http.ResponseWriter, req *http.Request) {
	sessionID := extractSessionID(req.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	ctx := context.Background()
	session, err := r.state.GetPersistentSession(ctx, sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Session not found")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// updateSession 更新会话。
func (r *Router) updateSession(w http.ResponseWriter, req *http.Request) {
	sessionID := extractSessionID(req.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	var reqBody struct {
		Name       string `json:"name,omitempty"`
		WorkingDir string `json:"working_dir,omitempty"`
		Mode       string `json:"mode,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx := context.Background()
	session, err := r.state.GetPersistentSession(ctx, sessionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Session not found")
		return
	}

	session.Update(reqBody.Name, reqBody.WorkingDir)

	if err := r.state.UpdatePersistentSession(ctx, session); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update session")
		return
	}

	writeJSON(w, http.StatusOK, session)
}

// deleteSession 删除会话。
func (r *Router) deleteSession(w http.ResponseWriter, req *http.Request) {
	sessionID := extractSessionID(req.URL.Path)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, "Invalid session ID")
		return
	}

	ctx := context.Background()
	if err := r.state.DeletePersistentSession(ctx, sessionID); err != nil {
		writeError(w, http.StatusNotFound, "Session not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// listTools 列出工具。
func (r *Router) listTools(w http.ResponseWriter, req *http.Request) {
	// TODO: 实现工具列表
	response := map[string]interface{}{
		"tools": []interface{}{},
	}
	writeJSON(w, http.StatusOK, response)
}

// getProvider 获取提供商。
func (r *Router) getProvider(w http.ResponseWriter, req *http.Request) {
	provider := r.state.GetProvider()
	model := r.state.GetModel()

	response := map[string]interface{}{
		"provider": provider,
		"model":    model,
	}
	writeJSON(w, http.StatusOK, response)
}

// updateProvider 更新提供商。
func (r *Router) updateProvider(w http.ResponseWriter, req *http.Request) {
	var reqBody struct {
		Provider string `json:"provider"`
		Model    string `json:"model,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	r.state.SetProvider(reqBody.Provider)
	if reqBody.Model != "" {
		r.state.SetModel(reqBody.Model)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "updated",
	})
}

// getConfig 获取配置。
func (r *Router) getConfig(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"mode": r.state.GetMode(),
	}
	writeJSON(w, http.StatusOK, response)
}

// updateConfig 更新配置。
func (r *Router) updateConfig(w http.ResponseWriter, req *http.Request) {
	var reqBody struct {
		Mode string `json:"mode,omitempty"`
	}
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if reqBody.Mode != "" {
		r.state.SetMode(reqBody.Mode)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "updated",
	})
}

// listExtensions 列出扩展。
func (r *Router) listExtensions(w http.ResponseWriter, req *http.Request) {
	response := map[string]interface{}{
		"extensions": []state.ExtensionState{},
	}
	writeJSON(w, http.StatusOK, response)
}

// addExtension 添加扩展。
func (r *Router) addExtension(w http.ResponseWriter, req *http.Request) {
	// TODO: 实现添加扩展
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "not_implemented",
	})
}

// removeExtension 移除扩展。
func (r *Router) removeExtension(w http.ResponseWriter, req *http.Request) {
	// TODO: 实现移除扩展
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "not_implemented",
	})
}

// 辅助函数

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func extractSessionID(path string) string {
	// /api/sessions/{sessionID}
	parts := splitPath(path)
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

func splitPath(path string) []string {
	result := make([]string, 0)
	start := 1 // 跳过开头的 /
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				result = append(result, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		result = append(result, path[start:])
	}
	return result
}

func generateID() string {
	// 简单实现，实际应使用 UUID
	return fmt.Sprintf("id_%d", time.Now().UnixNano())
}
