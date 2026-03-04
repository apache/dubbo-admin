# JSON Schema Index

This directory contains JSON Schema definitions for YAML configs.

- `main.schema.json`: root [`config.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/config.yaml)
- `logger.schema.json`: [`component/logger/logger.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/logger/logger.yaml)
- `memory.schema.json`: [`component/memory/memory.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/memory/memory.yaml)
- `models.schema.json`: [`component/models/models.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/models/models.yaml)
- `tools.schema.json`: [`component/tools/tools.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/tools/tools.yaml)
- `server.schema.json`: [`component/server/server.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/server/server.yaml)
- `rag.schema.json`: [`component/rag/rag.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/rag/rag.yaml)
- `agent.schema.json`: [`component/agent/agent.yaml`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/component/agent/agent.yaml)

Notes:
- Schema draft: `2020-12`
- `additionalProperties: false` is used to enforce unknown-field errors at the structural layer.
- Loader is the only structural layer (`yaml.Unmarshal` -> schema defaults+validation -> strict decode with KnownFields).
- Defaults are declared in schema and injected only by Loader/schema engine.
- Required-field policy is documented in [`REQUIRED_FIELDS.md`](/Users/liwener/.codex/worktrees/acbb/dubbo-admin/ai/schema/json/REQUIRED_FIELDS.md).
