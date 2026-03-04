# 模型可用性测试指南

本指南说明如何测试 `config/models.yaml` 中配置的所有模型是否可用。

## 测试方法

### 方法1：使用现有测试（推荐）

AI模块已经有一个简单的文本生成测试，可以快速验证默认模型：

```bash
cd /Users/liwener/programming/ospp/dubbo-admin/ai

# 设置API密钥
export DASHSCOPE_API_KEY="your_qwen_api_key"

# 运行测试
go test -v ./test/ -run TestTextGeneration
```

### 方法2：手动测试单个模型

使用简单的Go程序测试特定模型：

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/firebase/genkit/go/ai"
    "github.com/firebase/genkit/go/genkit"
    "github.com/firebase/genkit/go/plugins/compat_oai"
    "github.com/openai/openai-go/option"
)

func main() {
    ctx := context.Background()

    // 初始化 genkit
    g := genkit.Init(ctx)

    // 注册模型提供商（以Dashscope为例）
    _ = compat_oai.Init(g, "dashscope",
        compat_oai.WithConfig(openai.BaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1")),
        compat_oai.WithAPIKey(os.Getenv("DASHSCOPE_API_KEY")),
    )

    // 测试模型
    resp, err := genkit.GenerateText(ctx, g,
        ai.WithPrompt("Hello, who are you?"),
    )
    if err != nil {
        log.Fatalf("Failed to generate: %v", err)
    }

    fmt.Printf("Response: %s\n", resp)
}
```

### 方法3：使用服务器API测试

启动AI服务器后，通过HTTP API测试：

```bash
# 1. 启动服务器
cd /Users/liwener/programming/ospp/dubbo-admin/ai
go run main.go --config config.yaml

# 2. 在另一个终端测试
curl -X POST http://localhost:8888/api/v1/ai/chat/stream \
  -H "Content-Type: application/json" \
  -d '{
    "sessionId": "test-session",
    "message": "Hello, who are you?",
    "stream": false
  }'
```

## 配置的模型列表

根据 `config/models.yaml`，当前配置了以下模型：

### Dashscope（通义千问）
- `qwen-max` - 聊天模型（默认）
- `qwen-plus` - 聊天模型
- `qwen-flash` - 快速聊天模型
- `qwen3-coder` - 代码生成模型
- `qwen3-embedding` - 文本嵌入模型

### Gemini（Google）
- `gemini-pro` - 聊天模型
- `gemini-pro-vision` - 多模态模型
- `text-embedding-004` - 文本嵌入模型

### SiliconFlow
- `gpt-3.5-turbo` - 聊天模型
- `gpt-4` - 聊天模型
- `text-embedding-ada-002` - 文本嵌入模型

## 测试所有模型

要测试所有配置的模型，可以：

1. **设置所有API密钥**：
```bash
export DASHSCOPE_API_KEY="your_qwen_api_key"
export GEMINI_API_KEY="your_gemini_api_key"
export SILICONFLOW_API_KEY="your_siliconflow_key"
```

2. **修改测试代码**，遍历所有模型

3. **或者使用脚本**：
```bash
for provider in dashscope gemini siliconflow; do
    for model in qwen-max gemini-pro gpt-3.5-turbo; do
        echo "Testing $provider/$model..."
        # 调用API测试
    done
done
```

## 常见问题

### Q: 模型调用失败怎么办？
A: 检查：
1. API密钥是否正确设置
2. 网络连接是否正常
3. API是否有效（额度和限制）
4. 模型名称是否正确

### Q: 如何添加新模型？
A: 编辑 `config/models.yaml`，按照现有格式添加。

### Q: 测试太慢怎么办？
A: 使用 `go test -short` 跳过耗时测试，或者只测试默认模型。

## 相关文件

- `config/models.yaml` - 模型配置文件
- `ai/test/llm_test.go` - 简单模型测试
- `ai/main.go` - 服务器入口
- `ai/component/models/component.go` - Models组件实现
