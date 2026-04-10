// Package main goosed 服务器入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/camark/Gotosee/internal/acp"
	"github.com/camark/Gotosee/internal/server"
	"github.com/camark/Gotosee/internal/server/routes"
	"github.com/camark/Gotosee/internal/server/state"
)

func main() {
	// 命令行参数
	host := flag.String("host", "127.0.0.1", "服务器监听地址")
	port := flag.Int("port", 4040, "服务器端口")
	tls := flag.Bool("tls", false, "启用 TLS")
	certFile := flag.String("cert", "", "TLS 证书文件")
	keyFile := flag.String("key", "", "TLS 私钥文件")
	flag.Parse()

	// 创建应用状态
	appState := state.NewAppState()

	// 创建 ACP 服务器
	acpServer := acp.NewACPServer()
	appState.SetACPServer(acpServer)

	// 创建路由器
	router := routes.NewRouter(appState)

	// 创建 HTTP 服务器
	config := &server.Config{
		Host:     *host,
		Port:     *port,
		TLS:      *tls,
		CertFile: *certFile,
		KeyFile:  *keyFile,
	}

	httpServer := server.NewServer(config, router)

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器
	go func() {
		fmt.Printf("Starting gogo server at %s\n", httpServer.URL())
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 等待退出信号
	sig := <-sigChan
	fmt.Printf("\nReceived signal %v, shutting down...\n", sig)

	// 关闭应用状态
	appState.Close()

	// 优雅关闭
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	fmt.Println("Server stopped")
}
