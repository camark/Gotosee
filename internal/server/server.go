// Package server 提供 HTTP 服务器实现。
package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Server HTTP 服务器。
type Server struct {
	host         string
	port         int
	tls          bool
	certFile     string
	keyFile      string
	handler      http.Handler
	server       *http.Server
	mu           sync.RWMutex
}

// Config 服务器配置。
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	TLS      bool   `json:"tls"`
	CertFile string `json:"cert_file,omitempty"`
	KeyFile  string `json:"key_file,omitempty"`
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		Host: "127.0.0.1",
		Port: 4040,
		TLS:  false,
	}
}

// NewServer 创建新的 HTTP 服务器。
func NewServer(config *Config, handler http.Handler) *Server {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	return &Server{
		host:    config.Host,
		port:    config.Port,
		tls:     config.TLS,
		certFile: config.CertFile,
		keyFile: config.KeyFile,
		handler: handler,
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}
}

// Start 启动服务器。
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return fmt.Errorf("server not initialized")
	}

	var err error
	if s.tls {
		if s.certFile == "" || s.keyFile == "" {
			return fmt.Errorf("TLS enabled but cert_file or key_file not provided")
		}
		err = s.server.ListenAndServeTLS(s.certFile, s.keyFile)
	} else {
		err = s.server.ListenAndServe()
	}

	return err
}

// StartWithContext 启动服务器（支持上下文取消）。
func (s *Server) StartWithContext(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return fmt.Errorf("server not initialized")
	}

	// 在 goroutine 中启动服务器
	go func() {
		var err error
		if s.tls {
			err = s.server.ListenAndServeTLS(s.certFile, s.keyFile)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
		}
	}()

	// 等待上下文取消
	<-ctx.Done()

	// 优雅关闭
	return s.Shutdown(ctx)
}

// Shutdown 优雅关闭服务器。
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	return s.server.Shutdown(ctx)
}

// Address 返回服务器地址。
func (s *Server) Address() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// URL 返回服务器 URL。
func (s *Server) URL() string {
	protocol := "http"
	if s.tls {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s", protocol, s.Address())
}

// IsRunning 检查服务器是否运行中。
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.server != nil
}
