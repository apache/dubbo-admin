# RAG 多路径检索重组改动总结

> Commit: `775d89f` - refactor(ai): reorganize RAG components with multi-path retrieval and merge
> Author: YuZhangLarry <yz064043@gmail.com>
> Date: 2026-03-24

---

## 改动概览

**改动原因**: 原 RAG 架构只支持单路径检索，召回效果有限。需要支持多路检索融合（Dense + Sparse）以提升召回率和准确率。

**代码统计**:
- 50 个文件变更
- +9,447 行新增
- -1,538 行删除

---

## 一、目录结构重组

### 重组前
```
ai/component/rag/
├── indexer.go          # 索引器接口
├── retriever.go        # 检索器接口
├── loader.go           # 文档加载器
├── parser.go           # 文档解析器
├── preprocessor.go     # 预处理器
├── splitter.go         # 文档分割器
├── reranker.go         # 重排序器
├── component.go        # 组件入口
├── config.go           # 配置
├── factory.go          # 工厂
├── options.go          # 选项
├── rag.go              # RAG 入口
├── rag.yaml            # 配置文件
└── test/
    ├── factory_test.go
    ├── rag_config_test.go
    └── workflow_test.go
```

### 重组后
```
ai/component/rag/
├── indexers/           # 索引器实现包 (新增)
│   ├── local.go        # 本地索引器
│   ├── milvus.go       # Milvus 索引器 (新增)
│   └── pinecone.go     # Pinecone 索引器 (新增)
├── retrievers/         # 检索器实现包 (新增)
│   ├── local.go        # 本地检索器 (新增)
│   ├── milvus.go       # Milvus 检索器 (新增)
│   └── pinecone.go     # Pinecone 检索器 (新增)
├── loaders/            # 文档加载器包 (新增)
│   ├── local.go        # 本地加载器 (从 loader.go 迁移)
│   ├── metadata.go     # 元数据加载器 (新增)
│   ├── parser.go       # 文档解析器 (迁移)
│   └── preprocessor.go # 预处理器 (迁移)
├── mergers/            # 结果融合器包 (新增)
│   ├── concat.go       # 连接融合
│   ├── dedup.go        # 去重器
│   ├── factory.go      # 融合器工厂
│   ├── merge.go        # 融合接口
│   ├── normalize.go    # 归一化策略
│   ├── rrf.go          # RRF 融合
│   └── weighted.go     # 加权融合
├── query/              # 查询处理层包 (新增)
│   ├── expansion.go    # 查询扩展
│   ├── factory.go      # 查询处理器工厂
│   ├── hyde.go         # HyDE (假设文档嵌入)
│   ├── intent.go       # 意图识别
│   ├── legacy.go       # 传统查询处理器
│   ├── processor.go    # 查询处理器接口
│   └── rewrite.go      # 查询重写
├── rerankers/          # 重排序器包
│   └── cohere.go       # Cohere Rerank (从 reranker.go 迁移)
├── test/               # 测试文件
│   ├── milvus_e2e_test.go         # Milvus E2E 测试 (新增)
│   ├── milvus_indexer_test.go     # 索引器测试 (新增)
│   ├── milvus_multipath_test.go   # 多路径测试 (新增)
│   ├── milvus_retriever_test.go   # 检索器测试 (新增)
│   ├── retrieval_test.go          # 检索测试 (新增)
│   ├── rag_config_test.go         # 配置测试 (修改)
│   └── workflow_test.go           # 工作流测试 (修改)
├── component.go        # 组件入口 (修改)
├── config.go           # 配置 (修改)
├── factory.go          # 工厂 (修改)
├── options.go          # 选项 (修改)
├── rag.go              # RAG 入口 (修改)
└── rag.yaml            # 配置文件 (修改)
```

---

## 二、多路径检索架构

```
                    用户查询
                       ↓
            ┌──────────────────────┐
            │   Query Layer        │
            │ (扩展/重写/意图/HyDE) │
            └──────────────────────┘
                       ↓
        ┌──────────────┼──────────────┐
        ↓              ↓              ↓
   ┌────────┐    ┌────────┐    ┌────────┐
   │ Dense  │    │ Sparse │    │ Hybrid │
   │ Path   │    │  Path  │    │  Path  │
   └────────┘    └────────┘    └────────┘
        │              │              │
        └──────────────┼──────────────┘
                       ↓
            ┌──────────────────────┐
            │    Merge Layer       │
            │ (去重 → 归一化 → 融合) │
            └──────────────────────┘
                       ↓
            ┌──────────────────────┐
            │    Reranker          │
            │   (Cohere 可选)      │
            └──────────────────────┘
                       ↓
                   最终结果
```

---

## 三、核心功能改动

### 3.1 Milvus 多路径支持

| 类型 | 说明 | 字段 |
|------|------|------|
| **Dense** | 向量相似度检索 | `dense_vector` |
| **Sparse** | BM25 稀疏向量检索 | `sparse_vector` |
| **Hybrid** | Dense + Sparse 融合 | 两者结合 |

### 3.2 融合策略 (Mergers)

| 策略 | 说明 |
|------|------|
| **RRF** | 倒数排名融合，公式: `score(d) = Σ 1 / (k + rank_i(d))` |
| **Weighted** | 加权融合，支持多种归一化策略 |
| **Concat** | 简单连接多个路径的结果 |

### 3.3 归一化策略

- `MinMax`: Min-Max 归一化
- `ZScore`: Z-Score 标准化
- `Rank`: 基于排名的分数
- `Softmax`: Softmax 归一化
- `Sigmoid`: Sigmoid 转换
- `Log`: 对数转换

### 3.4 去重机制

1. 优先使用文档 `ID`
2. 无 ID 时使用元数据中的指定字段
3. 都无则使用内容哈希

### 3.5 查询处理层 (Query Layer)

| 处理器 | 功能 |
|--------|------|
| **QueryExpansion** | 查询扩展，生成多个相关查询 |
| **QueryRewrite** | 查询重写，优化原始查询 |
| **IntentDetection** | 意图识别，分类查询类型 |
| **HyDE** | 生成假设文档，用文档向量检索 |

---

## 四、RAG 核心结构改动

```go
// 新增多路径支持
type RAG struct {
    // 文档处理
    Loader   document.Loader
    Splitter document.Transformer
    Indexer  indexer.Indexer

    // 单路径检索 (向后兼容)
    Retriever retriever.Retriever

    // 多路径检索 (新增)
    RetrievalPaths []*RetrievalPath
    Merger         *mergers.MergeLayer

    // 查询理解 (新增)
    QueryLayer *query.Layer

    // 重排序
    Reranker rerankers.Reranker
}

// 检索路径定义 (新增)
type RetrievalPath struct {
    Label    string              // 路径标识 (如 "dense", "sparse")
    Retriever retriever.Retriever // 检索器
    TopK     int                 // 返回数量
    Weight   float64             // 加权融合权重
}
```

---

## 五、新增测试

| 文件 | 说明 |
|------|------|
| `milvus_e2e_test.go` | Milvus 端到端测试 |
| `milvus_indexer_test.go` | 索引器单元测试 |
| `milvus_multipath_test.go` | 多路径检索测试 |
| `milvus_retriever_test.go` | 检索器单元测试 |
| `retrieval_test.go` | 通用检索测试 |

---

## 六、依赖变更

### 新增 Go 依赖
```go
github.com/milvus-io/milvus-sdk-go/v2 v2.4.4+  // Milvus SDK
github.com/milvus-io/milvus/client/v2/...      // Milvus 客户端
```

### 修复
- 修复 Milvus SDK 依赖与 Go 1.25+ 的兼容性
- 添加 `ai/stub/etcd-server-v3/` 用于测试依赖隔离

---

## 七、使用示例

```go
// 创建 Dense 路径
densePath := &rag.RetrievalPath{
    Label:    "dense",
    Retriever: denseMilvusRetriever,
    TopK:     10,
    Weight:   0.7,
}

// 创建 Sparse 路径
sparsePath := &rag.RetrievalPath{
    Label:    "sparse",
    Retriever: sparseMilvusRetriever,
    TopK:     20,
    Weight:   0.3,
}

// 创建 RRF 融合器
merger, _ := mergers.NewRRFMerger(&mergers.RRFConfig{
    K:    60,
    TopK: 15,
})

// 创建 RAG
r := &rag.RAG{
    RetrievalPaths: []*rag.RetrievalPath{densePath, sparsePath},
    Merger:         merger,
}

// 执行检索
results, _ := r.RetrieveV2(ctx, &rag.RetrieveRequest{
    Query: "查询内容",
    TopK:  15,
})
```

---

## 八、改动机因总结

| 问题 | 解决方案 |
|------|----------|
| 单一路径召回率低 | 多路径并行检索 |
| 不同检索器分数不可比 | 分数归一化 |
| 重复文档混入结果 | 去重机制 |
| 融合策略单一 | RRF/Weighted/Concat |
| 查询理解能力弱 | Query Layer |
| 缺少 Milvus 支持 | Milvus Retriever/Indexer |
| 测试覆盖不足 | 新增 5 个测试文件 |

---

*文档生成时间: 2026-03-26*
