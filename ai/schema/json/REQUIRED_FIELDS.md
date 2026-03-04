# Config Required Fields Matrix

This document defines required-field policy for configuration schemas.

## Main Config (`config.yaml`)

| Field | Required | Notes |
|---|---|---|
| `project` | yes | Non-empty string. |
| `version` | yes | Non-empty string. |
| `components` | yes | Object with at least one entry. |
| `components.<name>` | yes | Must be `string` or `array[string]`. |

## Logger Component (`component/logger/logger.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `logger`. |
| `spec` | yes | Object. |
| `spec.level` | no | Default: `info`. |

## Memory Component (`component/memory/memory.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `memory`. |
| `spec` | yes | Object. |
| `spec.history_key` | no | Default: `chat_history`. |
| `spec.max_turns` | no | Default: `100`, must be `>= 1` when set. |

## Models Component (`component/models/models.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `models`. |
| `spec` | yes | Object. |
| `spec.default_model` | yes | Non-empty string. |
| `spec.default_embedding` | yes | Non-empty string. |
| `spec.providers` | yes | Object with at least one provider. |
| `spec.providers.<provider>.base_url` | yes | Non-empty string. |
| `spec.providers.<provider>.api_key` | no | May be empty in some environments. |
| `spec.providers.<provider>.models[]` | no | Defaults to empty array. |
| `spec.providers.<provider>.embedders[]` | no | Defaults to empty array. |
| `spec.providers.<provider>.models[].name` | yes | Non-empty string. |
| `spec.providers.<provider>.models[].key` | yes | Non-empty string. |
| `spec.providers.<provider>.models[].type` | no | Default: `chat`. |
| `spec.providers.<provider>.embedders[].name` | yes | Non-empty string. |
| `spec.providers.<provider>.embedders[].key` | yes | Non-empty string. |
| `spec.providers.<provider>.embedders[].dimensions` | yes | Integer `>= 1`. |
| `spec.providers.<provider>.embedders[].type` | no | Default: `text`. |

## Tools Component (`component/tools/tools.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `tools`. |
| `spec` | yes | Object. |
| `spec.enable_mock_tools` | no | Default: `true`. |
| `spec.enable_internal_tools` | no | Default: `true`. |
| `spec.enable_mcp_tools` | no | Default: `true`. |
| `spec.mcp_host_name` | conditional | Required when `spec.enable_mcp_tools=true`. |
| `spec.mcp_timeout` | no | Default: `30`, integer `>= 1`. |
| `spec.mcp_max_retries` | no | Default: `3`, integer `>= 0`. |

## Server Component (`component/server/server.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `server`. |
| `spec` | yes | Object. |
| `spec.port` | no | Default: `8888`, range `1..65535`. |
| `spec.host` | no | Default: `0.0.0.0`. |
| `spec.debug` | no | Default: `false`. |
| `spec.cors_origins` | no | Default: `[*]`. |
| `spec.read_timeout` | no | Default: `30`, integer `>= 1`. |
| `spec.write_timeout` | no | Default: `30`, integer `>= 1`. |

## RAG Component (`component/rag/rag.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `rag`. |
| `spec` | yes | Object. |
| `spec.embedder` | yes | Object. |
| `spec.loader` | yes | Object. |
| `spec.splitter` | yes | Object. |
| `spec.indexer` | yes | Object. |
| `spec.retriever` | yes | Object. |
| `spec.reranker` | no | Optional object. |
| `spec.embedder.spec` | yes | Object. |
| `spec.embedder.spec.model` | yes | Non-empty string. |
| `spec.loader.spec` | yes | Object (may be empty). |
| `spec.splitter.spec` | yes | Object. |
| `spec.indexer.spec` | yes | Object. |
| `spec.retriever.spec` | yes | Object. |
| `spec.reranker.spec` | conditional | Required when `spec.reranker` exists. |
| `spec.reranker.spec.api_key` | conditional | Required when `spec.reranker.spec.enabled=true`. |

## Agent Component (`component/agent/agent.yaml`)

| Field | Required | Notes |
|---|---|---|
| `type` | yes | Must be `agent`. |
| `spec` | yes | Object. |
| `spec.model` | yes | Non-empty string. |
| `spec.prompt_base_path` | yes | Non-empty string. |
| `spec.stages` | yes | Array with at least one stage. |
| `spec.agent_type` | no | Default: `react` (current supported value). |
| `spec.max_iterations` | no | Default: `10`, integer `>= 1`. |
| `spec.stage_channel_buffer_size` | no | Default: `5`, integer `>= 1`. |
| `spec.mcp_host_name` | no | Default: `mcp_host`. |
| `spec.stages[].name` | yes | Non-empty string. |
| `spec.stages[].flow_type` | yes | Enum: `think|act|observe|feedback`. |
| `spec.stages[].prompt_file` | yes | Non-empty string. |
| `spec.stages[].temperature` | no | Default: `0.7`, `(0,2]`. |
| `spec.stages[].top_p` | no | Default: `0.9`, `(0,1]`. |
| `spec.stages[].max_tokens` | no | Default: `4096`, integer `>= 1`. |
| `spec.stages[].timeout` | no | Default: `30`, integer `>= 1`. |
| `spec.stages[].enable_tools` | no | Default: `false`. |
