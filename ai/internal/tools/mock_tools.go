package tools

import (
	"fmt"
	"log"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ================================================
// Prometheus Query Service Latency Tool
// ================================================
type ToolOutput interface {
	Tool() string
}

type PrometheusServiceLatencyInput struct {
	ServiceName      string  `json:"serviceName" jsonschema:"required,description=The name of the service to query"`
	TimeRangeMinutes int     `json:"timeRangeMinutes" jsonschema:"required,description=Time range in minutes"`
	Quantile         float64 `json:"quantile" jsonschema:"required,description=Quantile to query (e.g., 0.99 or 0.95)"`
}

type PrometheusServiceLatencyOutput struct {
	ToolName    string  `json:"toolName"`
	Quantile    float64 `json:"quantile"`
	ValueMillis int     `json:"valueMillis"`
	Summary     string  `json:"summary"`
}

func (o PrometheusServiceLatencyOutput) Tool() string {
	return o.ToolName
}

func prometheusQueryServiceLatency(ctx *ai.ToolContext, input PrometheusServiceLatencyInput) (PrometheusServiceLatencyOutput, error) {
	log.Printf("Tool 'prometheus_query_service_latency' called for service: %s", input.ServiceName)

	// Mock data based on the example in prompt
	valueMillis := 3500
	if input.ServiceName != "order-service" {
		valueMillis = 850
	}

	return PrometheusServiceLatencyOutput{
		ToolName:    "prometheus_query_service_latency",
		Quantile:    input.Quantile,
		ValueMillis: valueMillis,
		// Summary:     fmt.Sprintf("服务 %s 在过去%d分钟内的 P%.0f 延迟为 %dms", input.ServiceName, input.TimeRangeMinutes, input.Quantile*100, valueMillis),
	}, nil
}

// ================================================
// Prometheus Query Service Traffic Tool
// ================================================

type PrometheusServiceTrafficInput struct {
	ServiceName      string `json:"serviceName" jsonschema:"required,description=The name of the service to query"`
	TimeRangeMinutes int    `json:"timeRangeMinutes" jsonschema:"required,description=Time range in minutes"`
}

type PrometheusServiceTrafficOutput struct {
	ToolName            string  `json:"toolName"`
	RequestRateQPS      float64 `json:"requestRateQPS"`
	ErrorRatePercentage float64 `json:"errorRatePercentage"`
	Summary             string  `json:"summary"`
}

func (o PrometheusServiceTrafficOutput) Tool() string {
	return o.ToolName
}

func prometheusQueryServiceTraffic(ctx *ai.ToolContext, input PrometheusServiceTrafficInput) (PrometheusServiceTrafficOutput, error) {
	log.Printf("Tool 'prometheus_query_service_traffic' called for service: %s", input.ServiceName)

	return PrometheusServiceTrafficOutput{
		ToolName:            "prometheus_query_service_traffic",
		RequestRateQPS:      250.0,
		ErrorRatePercentage: 5.2,
		// Summary:             fmt.Sprintf("服务 %s 的 QPS 为 250, 错误率为 5.2%%", input.ServiceName),
	}, nil
}

// ================================================
// Query Timeseries Database Tool
// ================================================

type QueryTimeseriesDatabaseInput struct {
	PromqlQuery string `json:"promqlQuery" jsonschema:"required,description=PromQL query to execute"`
}

type TimeseriesMetric struct {
	Pod string `json:"pod"`
}

type TimeseriesValue struct {
	Timestamp int64  `json:"timestamp"`
	Value     string `json:"value"`
}

type TimeseriesResult struct {
	Metric TimeseriesMetric `json:"metric"`
	Value  TimeseriesValue  `json:"value"`
}

type QueryTimeseriesDatabaseOutput struct {
	ToolName string             `json:"toolName"`
	Query    string             `json:"query"`
	Results  []TimeseriesResult `json:"results"`
	Summary  string             `json:"summary"`
}

func (o QueryTimeseriesDatabaseOutput) Tool() string {
	return o.ToolName
}

func queryTimeseriesDatabase(ctx *ai.ToolContext, input QueryTimeseriesDatabaseInput) (QueryTimeseriesDatabaseOutput, error) {
	log.Printf("Tool 'query_timeseries_database' called with query: %s", input.PromqlQuery)

	return QueryTimeseriesDatabaseOutput{
		ToolName: "query_timeseries_database",
		Query:    input.PromqlQuery,
		Results: []TimeseriesResult{
			{
				Metric: TimeseriesMetric{Pod: "order-service-pod-1"},
				Value:  TimeseriesValue{Timestamp: 1692192000, Value: "3.5"},
			},
			{
				Metric: TimeseriesMetric{Pod: "order-service-pod-2"},
				Value:  TimeseriesValue{Timestamp: 1692192000, Value: "3.2"},
			},
		},
		// Summary: "查询返回了 2 个时间序列",
	}, nil
}

// ================================================
// Application Performance Profiling Tool
// ================================================

type ApplicationPerformanceProfilingInput struct {
	ServiceName     string `json:"serviceName" jsonschema:"required,description=The name of the service to profile"`
	PodName         string `json:"podName" jsonschema:"required,description=The specific pod name to profile"`
	DurationSeconds int    `json:"durationSeconds" jsonschema:"required,description=Duration of profiling in seconds"`
}

type PerformanceHotspot struct {
	CPUTimePercentage float64  `json:"cpuTimePercentage"`
	StackTrace        []string `json:"stackTrace"`
}

type ApplicationPerformanceProfilingOutput struct {
	ToolName     string               `json:"toolName"`
	Status       string               `json:"status"`
	TotalSamples int                  `json:"totalSamples"`
	Hotspots     []PerformanceHotspot `json:"hotspots"`
	Summary      string               `json:"summary"`
}

func (o ApplicationPerformanceProfilingOutput) Tool() string {
	return o.ToolName
}

func applicationPerformanceProfiling(ctx *ai.ToolContext, input ApplicationPerformanceProfilingInput) (ApplicationPerformanceProfilingOutput, error) {
	log.Printf("Tool 'application_performance_profiling' called for service: %s, pod: %s", input.ServiceName, input.PodName)

	return ApplicationPerformanceProfilingOutput{
		ToolName:     "application_performance_profiling",
		Status:       "completed",
		TotalSamples: 10000,
		Hotspots: []PerformanceHotspot{
			{
				CPUTimePercentage: 45.5,
				StackTrace: []string{
					"com.example.OrderService.processOrder()",
					"com.example.DatabaseClient.query()",
					"java.sql.PreparedStatement.executeQuery()",
				},
			},
		},
		// Summary: "性能分析显示，45.5%的CPU时间消耗在数据库查询调用链上",
	}, nil
}

// ================================================
// JVM Performance Analysis Tool
// ================================================

type JVMPerformanceAnalysisInput struct {
	ServiceName string `json:"serviceName" jsonschema:"required,description=The name of the Java service to analyze"`
	PodName     string `json:"podName" jsonschema:"required,description=The specific pod name to analyze"`
}

type JVMPerformanceAnalysisOutput struct {
	ToolName            string  `json:"toolName"`
	FullGcCountLastHour int     `json:"fullGcCountLastHour"`
	FullGcTimeAvgMillis int     `json:"fullGcTimeAvgMillis"`
	HeapUsagePercentage float64 `json:"heapUsagePercentage"`
	Summary             string  `json:"summary"`
}

func (o JVMPerformanceAnalysisOutput) Tool() string {
	return o.ToolName
}

func jvmPerformanceAnalysis(ctx *ai.ToolContext, input JVMPerformanceAnalysisInput) (JVMPerformanceAnalysisOutput, error) {
	log.Printf("Tool 'jvm_performance_analysis' called for service: %s, pod: %s", input.ServiceName, input.PodName)

	return JVMPerformanceAnalysisOutput{
		ToolName:            "jvm_performance_analysis",
		FullGcCountLastHour: 15,
		FullGcTimeAvgMillis: 1200,
		HeapUsagePercentage: 85.5,
		// Summary:             "GC activity is high, average Full GC time is 1200ms",
	}, nil
}

// ================================================
// Trace Dependency View Tool
// ================================================

type TraceDependencyViewInput struct {
	ServiceName string `json:"serviceName" jsonschema:"required,description=The name of the service to analyze dependencies"`
}

type TraceDependencyViewOutput struct {
	ToolName           string   `json:"toolName"`
	UpstreamServices   []string `json:"upstreamServices"`
	DownstreamServices []string `json:"downstreamServices"`
}

func (o TraceDependencyViewOutput) Tool() string {
	return o.ToolName
}

func traceDependencyView(ctx *ai.ToolContext, input TraceDependencyViewInput) (TraceDependencyViewOutput, error) {
	log.Printf("Tool 'trace_dependency_view' called for service: %s", input.ServiceName)

	return TraceDependencyViewOutput{
		ToolName:           "trace_dependency_view",
		UpstreamServices:   []string{"api-gateway", "user-service"},
		DownstreamServices: []string{"mysql-orders-db", "redis-cache", "payment-service"},
	}, nil
}

// ================================================
// Trace Latency Analysis Tool
// ================================================

type TraceLatencyAnalysisInput struct {
	ServiceName      string `json:"serviceName" jsonschema:"required,description=The name of the service to analyze"`
	TimeRangeMinutes int    `json:"timeRangeMinutes" jsonschema:"required,description=Time range in minutes"`
}

type LatencyBottleneck struct {
	DownstreamService      string  `json:"downstreamService"`
	LatencyAvgMillis       int     `json:"latencyAvgMillis"`
	ContributionPercentage float64 `json:"contributionPercentage"`
}

type TraceLatencyAnalysisOutput struct {
	ToolName              string              `json:"toolName"`
	TotalLatencyAvgMillis int                 `json:"totalLatencyAvgMillis"`
	Bottlenecks           []LatencyBottleneck `json:"bottlenecks"`
	Summary               string              `json:"summary"`
}

func (o TraceLatencyAnalysisOutput) Tool() string {
	return o.ToolName
}

func traceLatencyAnalysis(ctx *ai.ToolContext, input TraceLatencyAnalysisInput) (TraceLatencyAnalysisOutput, error) {
	log.Printf("Tool 'trace_latency_analysis' called for service: %s", input.ServiceName)

	return TraceLatencyAnalysisOutput{
		ToolName:              "trace_latency_analysis",
		TotalLatencyAvgMillis: 3200,
		Bottlenecks: []LatencyBottleneck{
			{
				DownstreamService:      "mysql-orders-db",
				LatencyAvgMillis:       3050,
				ContributionPercentage: 95.3,
			},
			{
				DownstreamService:      "user-service",
				LatencyAvgMillis:       150,
				ContributionPercentage: 4.7,
			},
		},
		// Summary: "平均总延迟为 3200ms。瓶颈已定位，95.3% 的延迟来自对下游 'mysql-orders-db' 的调用",
	}, nil
}

// ================================================
// Database Connection Pool Analysis Tool
// ================================================

type DatabaseConnectionPoolAnalysisInput struct {
	ServiceName string `json:"serviceName" jsonschema:"required,description=The name of the service to analyze"`
}

type DatabaseConnectionPoolAnalysisOutput struct {
	ToolName          string `json:"toolName"`
	MaxConnections    int    `json:"maxConnections"`
	ActiveConnections int    `json:"activeConnections"`
	IdleConnections   int    `json:"idleConnections"`
	PendingRequests   int    `json:"pendingRequests"`
	Summary           string `json:"summary"`
}

func (o DatabaseConnectionPoolAnalysisOutput) Tool() string {
	return o.ToolName
}

func databaseConnectionPoolAnalysis(ctx *ai.ToolContext, input DatabaseConnectionPoolAnalysisInput) (DatabaseConnectionPoolAnalysisOutput, error) {
	log.Printf("Tool 'database_connection_pool_analysis' called for service: %s", input.ServiceName)

	return DatabaseConnectionPoolAnalysisOutput{
		ToolName:          "database_connection_pool_analysis",
		MaxConnections:    100,
		ActiveConnections: 100,
		IdleConnections:   0,
		PendingRequests:   58,
		// Summary:           "数据库连接池已完全耗尽 (100/100)，当前有 58 个请求正在排队等待连接",
	}, nil
}

// ================================================
// Kubernetes Get Pod Resources Tool
// ================================================

type KubernetesGetPodResourcesInput struct {
	ServiceName string `json:"serviceName" jsonschema:"required,description=The name of the service"`
	Namespace   string `json:"namespace" jsonschema:"required,description=The namespace of the service"`
}

type PodResource struct {
	PodName         string  `json:"podName"`
	CPUUsageCores   float64 `json:"cpuUsageCores"`
	CPURequestCores float64 `json:"cpuRequestCores"`
	CPULimitCores   float64 `json:"cpuLimitCores"`
	MemoryUsageMi   int     `json:"memoryUsageMi"`
	MemoryRequestMi int     `json:"memoryRequestMi"`
	MemoryLimitMi   int     `json:"memoryLimitMi"`
}

type KubernetesGetPodResourcesOutput struct {
	ToolName string        `json:"toolName"`
	Pods     []PodResource `json:"pods"`
	Summary  string        `json:"summary"`
}

func (o KubernetesGetPodResourcesOutput) Tool() string {
	return o.ToolName
}

func kubernetesGetPodResources(ctx *ai.ToolContext, input KubernetesGetPodResourcesInput) (KubernetesGetPodResourcesOutput, error) {
	log.Printf("Tool 'kubernetes_get_pod_resources' called for service: %s in namespace: %s", input.ServiceName, input.Namespace)

	return KubernetesGetPodResourcesOutput{
		ToolName: "kubernetes_get_pod_resources",
		Pods: []PodResource{
			{
				PodName:         "order-service-pod-1",
				CPUUsageCores:   0.8,
				CPURequestCores: 0.5,
				CPULimitCores:   1.0,
				MemoryUsageMi:   1800,
				MemoryRequestMi: 1024,
				MemoryLimitMi:   2048,
			},
			{
				PodName:         "order-service-pod-2",
				CPUUsageCores:   0.9,
				CPURequestCores: 0.5,
				CPULimitCores:   1.0,
				MemoryUsageMi:   1950,
				MemoryRequestMi: 1024,
				MemoryLimitMi:   2048,
			},
		},
		// Summary: "2 out of 2 pods are near their memory limits",
	}, nil
}

// ================================================
// Dubbo Service Status Tool
// ================================================

type DubboServiceStatusInput struct {
	ServiceName string `json:"serviceName" jsonschema:"required,description=The name of the Dubbo service"`
}

type DubboProvider struct {
	IP     string `json:"ip"`
	Port   int    `json:"port"`
	Status string `json:"status"`
}

type DubboConsumer struct {
	IP          string `json:"ip"`
	Application string `json:"application"`
	Status      string `json:"status"`
}

type DubboServiceStatusOutput struct {
	ToolName  string          `json:"toolName"`
	Providers []DubboProvider `json:"providers"`
	Consumers []DubboConsumer `json:"consumers"`
}

func (o DubboServiceStatusOutput) Tool() string {
	return o.ToolName
}

func dubboServiceStatus(ctx *ai.ToolContext, input DubboServiceStatusInput) (DubboServiceStatusOutput, error) {
	log.Printf("Tool 'dubbo_service_status' called for service: %s", input.ServiceName)

	return DubboServiceStatusOutput{
		ToolName: "dubbo_service_status",
		Providers: []DubboProvider{
			{IP: "192.168.1.10", Port: 20880, Status: "healthy"},
			{IP: "192.168.1.11", Port: 20880, Status: "healthy"},
		},
		Consumers: []DubboConsumer{
			{IP: "192.168.1.20", Application: "web-frontend", Status: "connected"},
			{IP: "192.168.1.21", Application: "api-gateway", Status: "connected"},
		},
	}, nil
}

// ================================================
// Query Log Database Tool
// ================================================

type QueryLogDatabaseInput struct {
	ServiceName      string `json:"serviceName" jsonschema:"required,description=The name of the service"`
	Keyword          string `json:"keyword" jsonschema:"required,description=Keyword to search for"`
	TimeRangeMinutes int    `json:"timeRangeMinutes" jsonschema:"required,description=Time range in minutes"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type QueryLogDatabaseOutput struct {
	ToolName  string     `json:"toolName"`
	TotalHits int        `json:"totalHits"`
	Logs      []LogEntry `json:"logs"`
	Summary   string     `json:"summary"`
}

func (o QueryLogDatabaseOutput) Tool() string {
	return o.ToolName
}

func queryLogDatabase(ctx *ai.ToolContext, input QueryLogDatabaseInput) (QueryLogDatabaseOutput, error) {
	log.Printf("Tool 'query_log_database' called for service: %s, keyword: %s", input.ServiceName, input.Keyword)

	return QueryLogDatabaseOutput{
		ToolName:  "query_log_database",
		TotalHits: 152,
		Logs: []LogEntry{
			{
				Timestamp: "2025-08-16T15:32:05Z",
				Level:     "WARN",
				Message:   "Timeout waiting for idle object in database connection pool.",
			},
			{
				Timestamp: "2025-08-16T15:32:08Z",
				Level:     "WARN",
				Message:   "Timeout waiting for idle object in database connection pool.",
			},
		},
		// Summary: fmt.Sprintf("在过去%d分钟内，发现 152 条关于 '%s' 的日志条目", input.TimeRangeMinutes, input.Keyword),
	}, nil
}

// ================================================
// Search Archived Logs Tool
// ================================================

type SearchArchivedLogsInput struct {
	FilePathPattern string `json:"filePathPattern" jsonschema:"required,description=File path pattern to search"`
	GrepKeyword     string `json:"grepKeyword" jsonschema:"required,description=Keyword to grep for"`
}

type MatchingLine struct {
	FilePath    string `json:"filePath"`
	LineNumber  int    `json:"lineNumber"`
	LineContent string `json:"lineContent"`
}

type SearchArchivedLogsOutput struct {
	ToolName      string         `json:"toolName"`
	FilesSearched int            `json:"filesSearched"`
	MatchingLines []MatchingLine `json:"matchingLines"`
	Summary       string         `json:"summary"`
}

func (o SearchArchivedLogsOutput) Tool() string {
	return o.ToolName
}

func searchArchivedLogs(ctx *ai.ToolContext, input SearchArchivedLogsInput) (SearchArchivedLogsOutput, error) {
	log.Printf("Tool 'search_archived_logs' called with pattern: %s, keyword: %s", input.FilePathPattern, input.GrepKeyword)

	return SearchArchivedLogsOutput{
		ToolName:      "search_archived_logs",
		FilesSearched: 5,
		MatchingLines: []MatchingLine{
			{
				FilePath:    "/logs/mysql-orders-db/slow-query-2025-08-16.log.gz",
				LineNumber:  1024,
				LineContent: "Query_time: 25.3s | SELECT COUNT(id), SUM(price) FROM orders WHERE user_id = 'VIP_USER_123';",
			},
			{
				FilePath:    "/logs/mysql-orders-db/slow-query-2025-08-16.log.gz",
				LineNumber:  1029,
				LineContent: "Query_time: 28.1s | SELECT COUNT(id), SUM(price) FROM orders WHERE user_id = 'VIP_USER_456';",
			},
		},
		// Summary: "在归档的日志文件中，发现了多条查询，搜索了 5 个文件，找到了 2 条匹配行",
	}, nil
}

// ================================================
// Query Knowledge Base Tool
// ================================================

type QueryKnowledgeBaseInput struct {
	QueryText string `json:"queryText" jsonschema:"required,description=Query text to search for"`
}

type KnowledgeDocument struct {
	Source          string  `json:"source"`
	ContentSnippet  string  `json:"contentSnippet"`
	SimilarityScore float64 `json:"similarityScore"`
}

type QueryKnowledgeBaseOutput struct {
	ToolName  string              `json:"toolName"`
	Documents []KnowledgeDocument `json:"documents"`
}

func (o QueryKnowledgeBaseOutput) Tool() string {
	return o.ToolName
}

func queryKnowledgeBase(ctx *ai.ToolContext, input QueryKnowledgeBaseInput) (QueryKnowledgeBaseOutput, error) {
	log.Printf("Tool 'query_knowledge_base' called with query: %s", input.QueryText)

	return QueryKnowledgeBaseOutput{
		ToolName: "query_knowledge_base",
		Documents: []KnowledgeDocument{
			{
				Source:          "Project-VIP-Feature-Design-Doc.md",
				ContentSnippet:  "The 'Lifetime Achievement' badge requires calculating total user spending. Note: This may cause slow queries on the orders table if the user_id column is not properly indexed.",
				SimilarityScore: 0.92,
			},
		},
	}, nil
}

// ================================================
// Tool Registration Function
// ================================================

var registeredTools []ai.Tool = nil

// RegisterAllMockTools registers all mock diagnostic tools with the genkit instance
func RegisterAllMockTools(g *genkit.Genkit) {
	registeredTools = []ai.Tool{
		genkit.DefineTool(g, "prometheus_query_service_latency", "查询指定服务在特定时间范围内的 P95/P99 延迟指标", prometheusQueryServiceLatency),
		genkit.DefineTool(g, "prometheus_query_service_traffic", "查询指定服务在特定时间范围内的请求率 (QPS) 和错误率 (Error Rate)", prometheusQueryServiceTraffic),
		genkit.DefineTool(g, "query_timeseries_database", "执行一条完整的 PromQL 查询语句，用于进行普罗米修斯历史数据的深度或自定义分析", queryTimeseriesDatabase),
		genkit.DefineTool(g, "application_performance_profiling", "对指定服务的指定实例（Pod）进行性能剖析，以结构化文本格式返回消耗CPU最多的函数调用栈", applicationPerformanceProfiling),
		genkit.DefineTool(g, "jvm_performance_analysis", "检查指定Java服务的JVM状态，特别是GC（垃圾回收）活动", jvmPerformanceAnalysis),
		genkit.DefineTool(g, "trace_dependency_view", "基于链路追踪数据，查询指定服务的上下游依赖关系", traceDependencyView),
		genkit.DefineTool(g, "trace_latency_analysis", "分析指定服务在某时间范围内的链路追踪数据，定位延迟最高的下游调用", traceLatencyAnalysis),
		genkit.DefineTool(g, "database_connection_pool_analysis", "查询指定服务连接数据库的连接池状态", databaseConnectionPoolAnalysis),
		genkit.DefineTool(g, "kubernetes_get_pod_resources", "使用类似 kubectl 的功能，获取指定服务所有Pod的CPU和内存的静态配置（Limits/Requests）和动态使用情况", kubernetesGetPodResources),
		genkit.DefineTool(g, "dubbo_service_status", "使用类似 dubbo-admin 的命令，查询指定Dubbo服务的提供者和消费者列表及其状态", dubboServiceStatus),
		genkit.DefineTool(g, "query_log_database", "查询已索引的日志数据库（如Elasticsearch, Loki），用于实时或近实时的日志分析", queryLogDatabase),
		genkit.DefineTool(g, "search_archived_logs", "在归档的日志文件（如存储在S3或服务器文件系统的.log.gz文件）中进行文本搜索（类似grep）", searchArchivedLogs),
		genkit.DefineTool(g, "query_knowledge_base", "在向量数据库中查询与问题相关的历史故障报告或解决方案文档", queryKnowledgeBase),
	}

	log.Printf("Registered %d mock diagnostic tools", len(registeredTools))
}

func AllMockToolRef() (toolRef []ai.ToolRef, err error) {
	if registeredTools == nil {
		return nil, fmt.Errorf("no mock tools registered")
	}
	for _, tool := range registeredTools {
		toolRef = append(toolRef, tool)
	}
	return toolRef, nil
}

func AllMockToolNames() (toolNames map[string]struct{}, err error) {
	if registeredTools == nil {
		return nil, fmt.Errorf("no mock tools registered")
	}
	toolNames = make(map[string]struct{})
	for _, tool := range registeredTools {
		toolNames[tool.Name()] = struct{}{}
	}
	return toolNames, nil
}

func MockToolRefByNames(names []string) (toolRefs []ai.ToolRef) {
	for _, name := range names {
		for _, tool := range registeredTools {
			if tool.Name() == name {
				toolRefs = append(toolRefs, tool)
			}
		}
	}
	return toolRefs
}
