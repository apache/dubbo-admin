# Dubbo Admin AI

Dubbo Admin AI 是面向 Dubbo Admin 场景的智能运维服务模块，用于通过 AI 对话辅助排障、分析与治理决策。

## 1. 简介

- 项目名：`Dubbo Admin AI`
- 一句话说明：为 Dubbo Admin 提供可对话的智能运维能力，帮助研发与运维人员更快定位问题并给出治理建议。

## 2. 核心特性（Features）

- ReAct 推理循环：支持 Think -> Act -> Observe -> Feedback 的多阶段推理与工具调用。
- RAG 检索增强：结合文档检索与重排序，提升回答准确度与可用性。
- 多模型支持：可配置接入 DashScope、Gemini、SiliconFlow 等模型提供商。
- SSE 流式输出：通过 Server-Sent Events 实时返回中间与最终响应。
- 会话管理能力：支持会话创建、查询、列举、删除与会话时效管理。

## 3. 快速开始（Quick Start）

### 环境要求

- Go `1.24.1+`
- 至少一个可用的 LLM API Key（参见 `.env.example`）

### 安装与启动

```bash
cd ai
cp .env.example .env
# 编辑 .env，填写至少一个 API Key

go run main.go --config ./config.yaml
```

默认访问地址：`http://localhost:8880`

## 4. 最小可用示例（Hello API）

### 4.1 创建会话请求

```bash
curl -X POST http://localhost:8880/api/v1/ai/sessions
```

成功后返回 JSON，`data.session_id` 可用于后续对话。

### 4.2 流式聊天请求（字段名使用 `sessionID`）

```bash
curl -N -X POST http://localhost:8880/api/v1/ai/chat/stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": "帮我分析这个服务的健康状态",
    "sessionID": "session_test"
  }'
```

### 4.3 预期返回/现象说明

- 接口以 SSE 形式持续输出事件流（`event: ...` / `data: ...`）。
- 可观察到分段文本增量，最终收到结束事件（如 `done`）。

## 5. API 概览

- `POST /api/v1/ai/chat/stream`
- `POST /api/v1/ai/sessions`
- `GET /api/v1/ai/sessions`
- `GET /api/v1/ai/sessions/:sessionId`
- `DELETE /api/v1/ai/sessions/:sessionId`
- `GET /health`

## 6. 配置说明入口

- 主配置：`config.yaml`
- 组件配置目录：`component/*/*.yaml`
- 环境变量说明：`.env.example`

## 7. 项目结构（精简版）

```text
ai/
├── main.go
├── runtime/
├── component/
├── config/
└── test/
```

## 8. 常见问题（FAQ）

### 端口冲突

- 默认端口为 `8880`（见 `component/server/server.yaml`）。
- 如端口被占用，请修改配置后重启服务。

### 缺少 API Key

- 若未配置任何模型 API Key，模型调用会失败。
- 请在 `.env` 中至少配置一个可用 Key，并确保配置文件引用了对应环境变量。

### 模型未注册/不可用

- 检查 `component/models/models.yaml` 中的 provider/model 配置是否正确。
- 检查网络可达性、API 配额与密钥权限。

### SSE 调用方式问题

- 使用 `curl -N` 或支持 SSE 的客户端。
- 请求头建议包含 `Accept: text/event-stream`。
- 流式接口请求体中会话字段为 `sessionID`（区分大小写）。

## 9. 开发者入口

- 架构文档：`architecture.md`
- 测试清单：`TEST_CHECKLIST.md`
- 贡献指南：`CONTRIBUTING.md`（如仓库提供）

## 10. License

Apache-2.0
