# 配方使用指南

配方（Recipe）是可定制的 AI 行为模式模板，用于定义 AI 助手的角色、任务和响应方式。

## 快速开始

### 列出可用配方

```bash
# 列出本地配方
goose recipe

# 列出全局配方
goose recipe -g

# 指定配方目录
goose recipe -d /path/to/recipes
```

### 查看配方详情

```bash
# 解释配方内容和设置
goose recipe explain examples/recipes/translator.json
```

### 运行配方

```bash
# 基础用法
goose recipe run examples/recipes/translator.json

# 指定提供商和模型
goose recipe run examples/recipes/translator.json -p openai -m gpt-4o

# 传递设置参数
goose recipe run examples/recipes/translator.json -s tone=casual
goose recipe run examples/recipes/code-reviewer.json -s language=Python -s focus=安全
```

## 配方格式

配方使用 JSON 格式，支持以下字段：

```json
{
  "name": "配方名称",
  "description": "配方描述",
  "version": "1.0.0",
  "author": {
    "name": "作者名",
    "email": "email@example.com",
    "url": "https://example.com"
  },
  "settings": [
    {
      "key": "设置键",
      "type": "string",
      "description": "设置描述",
      "default": "默认值",
      "required": false
    }
  ],
  "instructions": "AI 指令内容"
}
```

### 字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| `name` | 是 | 配方名称 |
| `description` | 否 | 配方描述 |
| `version` | 否 | 版本号 |
| `author` | 否 | 作者信息 |
| `settings` | 否 | 可配置参数列表 |
| `instructions` | 是 | AI 指令（系统提示） |

### Settings 字段

每个设置项包含：

| 字段 | 必填 | 说明 |
|------|------|------|
| `key` | 是 | 设置键名 |
| `type` | 是 | 类型（string, int, bool） |
| `description` | 否 | 描述 |
| `default` | 否 | 默认值 |
| `required` | 否 | 是否必需 |

## 示例配方

### 翻译助手

```json
{
  "name": "translator",
  "description": "英译中翻译助手",
  "version": "1.0.0",
  "settings": [
    {
      "key": "tone",
      "type": "string",
      "description": "翻译语气",
      "default": "正式"
    }
  ],
  "instructions": "你是专业的英译中翻译助手..."
}
```

运行方式：
```bash
goose recipe run translator.json -s tone=学术
```

### 代码审查

```json
{
  "name": "code-reviewer",
  "description": "代码审查助手",
  "settings": [
    {
      "key": "language",
      "type": "string",
      "description": "编程语言",
      "default": "Go"
    },
    {
      "key": "focus",
      "type": "string",
      "description": "审查重点",
      "default": "全部"
    }
  ],
  "instructions": "你是经验丰富的代码审查专家..."
}
```

运行方式：
```bash
goose recipe run code-reviewer.json -s language=Python -s focus=性能
```

## 验证配方

使用 validate 命令检查配方格式：

```bash
goose recipe validate examples/recipes/translator.json
```

输出：
```
验证配方：examples/recipes/translator.json
名称：translator
描述：将英文翻译成中文的翻译助手
版本：1.0.0
设置项：1
✓ 配方格式正确
```

## 配方目录结构

```
recipes/
├── translator.json      # 翻译助手
├── code-reviewer.json   # 代码审查
├── tech-writer.json     # 技术文档写作
└── ...
```

## 最佳实践

1. **指令清晰**：Instructions 应明确描述 AI 的角色和任务
2. **设置合理**：为常用参数提供默认值，减少用户输入
3. **版本管理**：使用 semver 版本号管理配方迭代
4. **测试验证**：创建配方后充分测试各种设置组合

## 分享配方

欢迎贡献配方！将你的配方放到 `examples/recipes/` 目录下。
