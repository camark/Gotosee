# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build CLI (Windows - disable CGO for static executable)
$env:CGO_ENABLED=0
go build -o goose.exe ./cmd/goose
go build -o goosed.exe ./cmd/goosed

# Build CLI (Linux/macOS)
CGO_ENABLED=0 go build -o goose ./cmd/goose
CGO_ENABLED=0 go build -o goosed ./cmd/goosed

# Run tests
go test ./...

# Run single test file
go test ./internal/agents/agent_integration_test.go -v

# Run with benchmark
go test ./internal/agents/compaction_test.go -bench=. -benchmem

# Run CLI directly
go run ./cmd/goose version
go run ./cmd/goose chat --agent
```

## Architecture Overview

**gogo** is an AI agent framework ported from Rust to Go, maintaining the same core architecture. It supports multiple AI providers, MCP (Model Context Protocol) extensions, and ACP (Agent Communication Protocol).

### Core Modules

| Module | Package | Responsibility |
|--------|---------|----------------|
| **CLI** | `cmd/goose` | Command-line interface entry point |
| **Server** | `cmd/goosed` | HTTP/gRPC server for remote agent access |
| **Agents** | `internal/agents` | Core agent loop, tool execution, LLM orchestration |
| **Providers** | `internal/providers` | AI model integrations (OpenAI, Anthropic, DeepSeek, etc.) |
| **MCP** | `internal/mcp` | Model Context Protocol servers (filesystem, git, memory, etc.) |
| **Session** | `internal/session` | SQLite-backed session persistence |
| **Conversation** | `internal/conversation` | Message types and conversation state |
| **Config** | `internal/config` | Configuration management |
| **Server** | `internal/server` | HTTP API server with REST endpoints |
| **ACP** | `internal/acp` | Agent Communication Protocol implementation |

### Key Architectural Patterns

**Provider Pattern**: All AI providers implement the `Provider` interface:
- `Complete(ctx, messages, config)` - Non-streaming completion
- `Stream(ctx, messages, config)` - Streaming response via channel
- `ListModels(ctx)` - Discover available models
- `Validate()` - Configuration validation

**MCP Server Pattern**: Tools are exposed via MCP servers registered in `internal/mcp/registry.go`:
- Each server implements `Server` interface with `ListTools()` and `CallTool()`
- Tools are auto-discovered and passed to LLM via `config.ExtraParams["tools"]`
- Built-in servers: filesystem, git, memory, fetch, time, environment, process, database, http-client, auto-visualiser, notion, tutorial, computer-controller

**Agent Loop** (`internal/agents/agent.go:Reply`):
1. Collect tools from all registered MCP servers
2. Call LLM with message history + available tools
3. Parse tool calls from response
4. Execute tools and append results to message history
5. Repeat until no tool calls or max turns reached

**Tool Calling Flow**:
- Tools are sent to LLM via `tools` parameter in API request
- LLM responses include `tool_calls` array with function name and arguments
- Tool results are sent back as `RoleUser` messages with `tool_result` content type
- System prompt (in English) encourages tool usage for file/system operations

### Supported AI Providers

All providers follow OpenAI-compatible API format for tool calling:
- OpenAI, Anthropic, Google, Azure
- Ollama (local models)
- OpenRouter (multi-model aggregation)
- DeepSeek, Kimi, MiniMax, Qwen (Chinese providers)

### Important Implementation Details

**Tool Calling Requirements** (for DeepSeek and similar providers):
- System prompt must be in English for better model response
- Tool descriptions should clearly indicate when to use the tool
- Tool parameters must be valid JSON Schema format
- `tool_choice: "auto"` is set to let model decide when to use tools

**Message Role for Tool Results**: Tool results must use `RoleUser` (not `RoleAssistant`) to match OpenAI API specification.

**CGO Warning**: Always build with `CGO_ENABLED=0` on Windows to avoid executable compatibility issues. The sqlite3 driver uses pure-Go mode when CGO is disabled.

### Configuration

Configuration is stored in user's home directory. Key settings:
- Provider type and API key
- Model selection
- Base URL for self-hosted models
- MCP server configurations

### Testing

- Unit tests use standard `testing` package
- Integration tests in `internal/agents/agent_integration_test.go` test full agent loop
- Benchmarks for performance-critical paths (Reply, Compaction)
- Tests require valid API keys for provider integration tests
