# gogo

Go 语言版本的 goose AI 代理框架。

这是一个从 Rust 版本迁移过来的项目，保留了原有的架构和功能。

## 项目结构

```
gogo/
├── cmd/
│   ├── goose/          # CLI 入口
│   └── goosed/         # 服务器入口 ✅
├── internal/
│   ├── sdk/            # goose-sdk 迁移 ✅
│   ├── config/         # 配置管理 ✅
│   ├── conversation/   # 对话管理 ✅
│   ├── model/          # 模型配置 ✅
│   ├── providers/      # AI 提供商迁移 ✅
│   ├── mcp/            # MCP 扩展迁移 ✅
│   ├── acp/            # ACP 协议迁移 ✅
│   ├── server/         # HTTP 服务器 ✅
│   ├── session/        # 会话管理 🔄
│   └── agents/         # 代理逻辑 🔄
├── pkg/
│   └── api/            # 公共 API
└── scripts/            # 构建/工具脚本
```

## 迁移进度

### 已完成 ✅
- [x] **goose-sdk** - SDK 类型定义
- [x] **核心基础模块**:
  - [x] 配置管理 (`internal/config/base.go`, `goose_mode.go`)
  - [x] 对话管理 (`internal/conversation/message.go`)
  - [x] 模型配置 (`internal/model/model.go`)
  - [x] 提供商接口 (`internal/providers/base.go`, `registry.go`)
  - [x] OpenAI 提供商实现 (`internal/providers/openai.go`)
  - [x] Anthropic 提供商实现 (`internal/providers/anthropic.go`)
  - [x] Ollama 提供商实现 (`internal/providers/ollama.go`)
  - [x] Google 提供商实现 (`internal/providers/google.go`)
  - [x] Azure 提供商实现 (`internal/providers/azure.go`)
  - [x] OpenRouter 提供商实现 (`internal/providers/openrouter.go`)
- [x] **CLI/服务器骨架** (`cmd/goose/main.go`, `cmd/goosed/server.go`)
- [x] **goose-mcp** - MCP 扩展
  - [x] MCP 服务器基础类型
  - [x] 计算机控制服务器
  - [x] 文档处理工具 (PDF/DOCX/XLSX)
  - [x] MCP 服务器运行器
- [x] **goose-acp** - ACP 协议
  - [x] ACP 类型定义
  - [x] ACP 服务器
  - [x] HTTP/WebSocket 传输层
- [x] **goose-server** - HTTP 服务器 (2026-04-10)
  - [x] HTTP 服务器基础 (`internal/server/server.go`)
  - [x] 应用状态管理 (`internal/server/state/state.go`)
  - [x] HTTP 路由处理 (`internal/server/routes/router.go`)
- [x] **goose-session** - 会话持久化 (2026-04-10)
  - [x] 会话管理器 (`internal/session/session.go`)
  - [x] SQLite 存储
  - [x] 会话 CRUD 操作
  - [x] 会话类型（User, Scheduled, SubAgent, Hidden, Terminal, Gateway, Acp）
  - [x] 扩展数据管理
  - [x] 会话统计
- [x] **goose-agents** - 代理框架骨架 (2026-04-10)
  - [x] Agent 核心类型 (`internal/agents/agent.go`)
  - [x] 代理类型定义 (`internal/agents/types.go`)
  - [x] 重试管理器 (`internal/agents/retry.go`)
  - [x] 扩展管理器 (`internal/agents/extension.go`)
- [x] **goose-cli** - 命令行接口骨架 (2026-04-10)
  - [x] CLI 框架 (`internal/cli/cli.go`)
  - [x] configure 命令 (配置向导)
  - [x] session 命令 (列表、删除)
  - [x] recipe 命令 (列表、验证、解释)
  - [x] schedule 命令 (列表、添加、删除)
  - [x] term 命令 (运行、初始化)
  - [x] project 命令
  - [x] doctor 命令 (诊断)
  - [x] info 命令 (信息)
  - [x] chat 命令 (交互式对话) ✅ 新增

### 待迁移
- [ ] 更多 AI 提供商 (Anthropic, Google, Ollama 等)
- [ ] goose-cli 完整功能 (配方执行、会话运行模式)

## 构建

```bash
cd C:\git\gogo
go build ./...           # 构建所有
go build ./cmd/goose     # 构建 CLI
go build ./cmd/goosed    # 构建服务器
```

## 运行服务器

```bash
./cmd/goosed/goosed -host 127.0.0.1 -port 4040
```

## 测试

```bash
go test ./...
```

## 模块详情

### goose-server 模块

| 组件 | 描述 | 状态 |
|------|------|------|
| HTTP 服务器 | 基础 HTTP 服务、TLS 支持 | ✅ 完成 |
| 状态管理 | 会话、扩展、提供商状态 | ✅ 完成 |
| REST API | 会话/工具/配置/扩展端点 | ✅ 基础实现 |
| ACP 集成 | ACP 协议端点 | 🔄 待完善 |

### API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/health` | GET | 健康检查 |
| `/api/sessions` | GET/POST | 列出/创建会话 |
| `/api/sessions/{id}` | GET/PUT/DELETE | 获取/更新/删除会话 |
| `/api/tools` | GET | 列出工具 |
| `/api/provider` | GET/PUT | 获取/更新提供商 |
| `/api/config` | GET/PUT | 获取/更新配置 |
| `/api/extensions` | GET/POST/DELETE | 扩展管理 |
| `/acp/*` | ALL | ACP 协议端点 |

## 下一步

1. **goose-session** - 会话持久化（SQLite 存储）
2. **goose-agents** - 代理逻辑（工具调度、消息处理）
3. **goose-cli** - 完整 CLI（configure、session、recipe 等命令）
4. **更多 Provider** - Anthropic、Google、Ollama、Azure 等

## 统计

| 指标 | 数量 |
|------|------|
| Go 文件 | 35+ |
| 代码行数 | ~7500+ |
| 包 | 16 |

## 依赖

- Go 1.22+
- 无外部依赖（核心模块）

## 许可证

Apache 2.0 (与原始 goose 项目保持一致)
