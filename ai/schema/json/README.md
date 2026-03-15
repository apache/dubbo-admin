# JSON Schema Index

This directory contains JSON Schema definitions for YAML configs.

- `main.schema.json`: root [`config.yaml`](../../config.yaml)
- `logger.schema.json`: [`component/logger/logger.yaml`](../../component/logger/logger.yaml)
- `memory.schema.json`: [`component/memory/memory.yaml`](../../component/memory/memory.yaml)
- `models.schema.json`: [`component/models/models.yaml`](../../component/models/models.yaml)
- `tools.schema.json`: [`component/tools/tools.yaml`](../../component/tools/tools.yaml)
- `server.schema.json`: [`component/server/server.yaml`](../../component/server/server.yaml)
- `rag.schema.json`: [`component/rag/rag.yaml`](../../component/rag/rag.yaml)
- `agent.schema.json`: [`component/agent/agent.yaml`](../../component/agent/agent.yaml)

Notes:
- Schema draft: `2020-12`
- `additionalProperties: false` is used to enforce unknown-field errors at the structural layer.
- Loader is the only structural layer (`yaml.Unmarshal` -> schema defaults+validation -> strict decode with KnownFields).
- Defaults are declared in schema and injected only by Loader/schema engine.
- Required-field policy is documented in [`REQUIRED_FIELDS.md`](./REQUIRED_FIELDS.md).
