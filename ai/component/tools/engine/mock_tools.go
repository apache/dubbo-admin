package engine

import (
	"dubbo-admin-ai/runtime"
	"fmt"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// ================================================
// Prometheus Query Service Latency Tool
// ================================================
type PrometheusServiceLatencyInput struct {
	ServiceName      string  `json:"service_name" jsonschema:"required,description=The name of the service to query"`
	TimeRangeMinutes int     `json:"time_range_minutes" jsonschema:"required,description=Time range in minutes"`
	Quantile         float64 `json:"quantile" jsonschema:"required,description=Quantile to query (e.g., 0.99 or 0.95)"`
}

func (i PrometheusServiceLatencyInput) String() string {
	return fmt.Sprintf("PrometheusServiceLatencyInput{ServiceName: %s, TimeRangeMinutes: %d, Quantile: %.2f}", i.ServiceName, i.TimeRangeMinutes, i.Quantile)
}

type PrometheusServiceLatencyOutput struct {
	Quantile    float64 `json:"quantile"`
	ValueMillis int     `json:"value_millis"`
}

func (o PrometheusServiceLatencyOutput) String() string {
	return fmt.Sprintf("PrometheusServiceLatencyOutput{Quantile: %.2f, ValueMillis: %d}", o.Quantile, o.ValueMillis)
}

func prometheusQueryServiceLatency(ctx *ai.ToolContext, input PrometheusServiceLatencyInput) (ToolOutput, error) {

	// Mock data based on the example in prompt
	valueMillis := 3500
	if input.ServiceName != "order-service" {
		valueMillis = 850
	}

	output := PrometheusServiceLatencyOutput{
		Quantile:    input.Quantile,
		ValueMillis: valueMillis,
	}

	return ToolOutput{
		ToolName: "prometheus_query_service_latency",
		Summary:  fmt.Sprintf("Service %s had a P%.0f latency of %dms over the past %d minutes", input.ServiceName, input.Quantile*100, output.ValueMillis, input.TimeRangeMinutes),
		Result:   output,
	}, nil
}

// ================================================
// Prometheus Query Service Traffic Tool
// ================================================

type PrometheusServiceTrafficInput struct {
	ServiceName      string `json:"service_name" jsonschema:"required,description=The name of the service to query"`
	TimeRangeMinutes int    `json:"time_range_minutes" jsonschema:"required,description=Time range in minutes"`
}

func (i PrometheusServiceTrafficInput) String() string {
	return fmt.Sprintf("PrometheusServiceTrafficInput{ServiceName: %s, TimeRangeMinutes: %d}", i.ServiceName, i.TimeRangeMinutes)
}

type PrometheusServiceTrafficOutput struct {
	RequestRateQPS      float64 `json:"request_rate_qps"`
	ErrorRatePercentage float64 `json:"error_rate_percentage"`
}

func (o PrometheusServiceTrafficOutput) String() string {
	return fmt.Sprintf("PrometheusServiceTrafficOutput{RequestRateQPS: %.1f, ErrorRatePercentage: %.1f}", o.RequestRateQPS, o.ErrorRatePercentage)
}

func prometheusQueryServiceTraffic(ctx *ai.ToolContext, input PrometheusServiceTrafficInput) (ToolOutput, error) {

	output := PrometheusServiceTrafficOutput{
		RequestRateQPS:      250.0,
		ErrorRatePercentage: 5.2,
	}

	return ToolOutput{
		ToolName: "prometheus_query_service_traffic",
		Summary:  fmt.Sprintf("Service %s has a QPS of %.1f and error rate of %.1f%%", input.ServiceName, output.RequestRateQPS, output.ErrorRatePercentage),
		Result:   output,
	}, nil
}

// ================================================
// Query Timeseries Database Tool
// ================================================

type QueryTimeseriesDatabaseInput struct {
	PromqlQuery string `json:"promql_query" jsonschema:"required,description=PromQL query to execute"`
}

func (i QueryTimeseriesDatabaseInput) String() string {
	return fmt.Sprintf("QueryTimeseriesDatabaseInput{PromqlQuery: %s}", i.PromqlQuery)
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
	Query   string             `json:"query"`
	Results []TimeseriesResult `json:"results"`
}

func (o QueryTimeseriesDatabaseOutput) String() string {
	return fmt.Sprintf("QueryTimeseriesDatabaseOutput{Query: %s, Results: %d items}", o.Query, len(o.Results))
}

func queryTimeseriesDatabase(ctx *ai.ToolContext, input QueryTimeseriesDatabaseInput) (ToolOutput, error) {

	output := QueryTimeseriesDatabaseOutput{
		Query: input.PromqlQuery,
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
	}

	return ToolOutput{
		ToolName: "query_timeseries_database",
		Summary:  "Query returned 2 time series",
		Result:   output,
	}, nil
}

// ================================================
// Application Performance Profiling Tool
// ================================================

type ApplicationPerformanceProfilingInput struct {
	ServiceName     string `json:"service_name" jsonschema:"required,description=The name of the service to profile"`
	PodName         string `json:"pod_name" jsonschema:"required,description=The specific pod name to profile"`
	DurationSeconds int    `json:"duration_seconds" jsonschema:"required,description=Duration of profiling in seconds"`
}

func (i ApplicationPerformanceProfilingInput) String() string {
	return fmt.Sprintf("ApplicationPerformanceProfilingInput{ServiceName: %s, PodName: %s, DurationSeconds: %d}", i.ServiceName, i.PodName, i.DurationSeconds)
}

type PerformanceHotspot struct {
	CPUTimePercentage float64  `json:"cpu_time_percentage"`
	StackTrace        []string `json:"stack_trace"`
}

type ApplicationPerformanceProfilingOutput struct {
	Status       string               `json:"status"`
	TotalSamples int                  `json:"total_samples"`
	Hotspots     []PerformanceHotspot `json:"hotspots"`
}

func (o ApplicationPerformanceProfilingOutput) String() string {
	return fmt.Sprintf("ApplicationPerformanceProfilingOutput{Status: %s, TotalSamples: %d, Hotspots: %d items}", o.Status, o.TotalSamples, len(o.Hotspots))
}

func applicationPerformanceProfiling(ctx *ai.ToolContext, input ApplicationPerformanceProfilingInput) (ToolOutput, error) {

	output := ApplicationPerformanceProfilingOutput{
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
	}

	return ToolOutput{
		ToolName: "application_performance_profiling",
		Summary:  "Performance analysis shows that 45.5% of CPU time is consumed by database query call chain",
		Result:   output,
	}, nil
}

// ================================================
// JVM Performance Analysis Tool
// ================================================

type JVMPerformanceAnalysisInput struct {
	ServiceName string `json:"service_name" jsonschema:"required,description=The name of the Java service to analyze"`
	PodName     string `json:"pod_name" jsonschema:"required,description=The specific pod name to analyze"`
}

func (i JVMPerformanceAnalysisInput) String() string {
	return fmt.Sprintf("JVMPerformanceAnalysisInput{ServiceName: %s, PodName: %s}", i.ServiceName, i.PodName)
}

type JVMPerformanceAnalysisOutput struct {
	FullGcCountLastHour int     `json:"full_gc_count_last_hour"`
	FullGcTimeAvgMillis int     `json:"full_gc_time_avg_millis"`
	HeapUsagePercentage float64 `json:"heap_usage_percentage"`
}

func (o JVMPerformanceAnalysisOutput) String() string {
	return fmt.Sprintf("JVMPerformanceAnalysisOutput{FullGcCountLastHour: %d, FullGcTimeAvgMillis: %d, HeapUsagePercentage: %.1f}", o.FullGcCountLastHour, o.FullGcTimeAvgMillis, o.HeapUsagePercentage)
}

func jvmPerformanceAnalysis(ctx *ai.ToolContext, input JVMPerformanceAnalysisInput) (ToolOutput, error) {

	output := JVMPerformanceAnalysisOutput{
		FullGcCountLastHour: 15,
		FullGcTimeAvgMillis: 1200,
		HeapUsagePercentage: 85.5,
	}

	return ToolOutput{
		ToolName: "jvm_performance_analysis",
		Summary:  "GC activity is high, average Full GC time is 1200ms",
		Result:   output,
	}, nil
}

// ================================================
// Trace Dependency View Tool
// ================================================

type TraceDependencyViewInput struct {
	ServiceName string `json:"service_name" jsonschema:"required,description=The name of the service to analyze dependencies"`
}

func (i TraceDependencyViewInput) String() string {
	return fmt.Sprintf("TraceDependencyViewInput{ServiceName: %s}", i.ServiceName)
}

type TraceDependencyViewOutput struct {
	UpstreamServices   []string `json:"upstream_services"`
	DownstreamServices []string `json:"downstream_services"`
}

func (o TraceDependencyViewOutput) String() string {
	return fmt.Sprintf("TraceDependencyViewOutput{UpstreamServices: %v, DownstreamServices: %v}", o.UpstreamServices, o.DownstreamServices)
}

func traceDependencyView(ctx *ai.ToolContext, input TraceDependencyViewInput) (ToolOutput, error) {

	output := TraceDependencyViewOutput{
		UpstreamServices:   []string{"api-gateway", "user-service"},
		DownstreamServices: []string{"mysql-orders-db", "redis-cache", "payment-service"},
	}

	return ToolOutput{
		ToolName: "trace_dependency_view",
		Summary:  fmt.Sprintf("Upstream and downstream dependency query for service %s completed", input.ServiceName),
		Result:   output,
	}, nil
}

// ================================================
// Trace Latency Analysis Tool
// ================================================

type TraceLatencyAnalysisInput struct {
	ServiceName      string `json:"service_name" jsonschema:"required,description=The name of the service to analyze"`
	TimeRangeMinutes int    `json:"time_range_minutes" jsonschema:"required,description=Time range in minutes"`
}

func (i TraceLatencyAnalysisInput) String() string {
	return fmt.Sprintf("TraceLatencyAnalysisInput{ServiceName: %s, TimeRangeMinutes: %d}", i.ServiceName, i.TimeRangeMinutes)
}

type LatencyBottleneck struct {
	DownstreamService      string  `json:"downstream_service"`
	LatencyAvgMillis       int     `json:"latency_avg_millis"`
	ContributionPercentage float64 `json:"contribution_percentage"`
}

type TraceLatencyAnalysisOutput struct {
	TotalLatencyAvgMillis int                 `json:"total_latency_avg_millis"`
	Bottlenecks           []LatencyBottleneck `json:"bottlenecks"`
}

func (o TraceLatencyAnalysisOutput) String() string {
	return fmt.Sprintf("TraceLatencyAnalysisOutput{TotalLatencyAvgMillis: %d, Bottlenecks: %d items}", o.TotalLatencyAvgMillis, len(o.Bottlenecks))
}

func traceLatencyAnalysis(ctx *ai.ToolContext, input TraceLatencyAnalysisInput) (ToolOutput, error) {

	output := TraceLatencyAnalysisOutput{
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
	}

	return ToolOutput{
		ToolName: "trace_latency_analysis",
		Summary:  "Average total latency is 3200ms. Bottleneck identified: 95.3% of latency comes from calls to downstream 'mysql-orders-db'",
		Result:   output,
	}, nil
}

// ================================================
// Database Connection Pool Analysis Tool
// ================================================

type DatabaseConnectionPoolAnalysisInput struct {
	ServiceName string `json:"service_name" jsonschema:"required,description=The name of the service to analyze"`
}

func (i DatabaseConnectionPoolAnalysisInput) String() string {
	return fmt.Sprintf("DatabaseConnectionPoolAnalysisInput{ServiceName: %s}", i.ServiceName)
}

type DatabaseConnectionPoolAnalysisOutput struct {
	MaxConnections    int `json:"max_connections"`
	ActiveConnections int `json:"active_connections"`
	IdleConnections   int `json:"idle_connections"`
	PendingRequests   int `json:"pending_requests"`
}

func (o DatabaseConnectionPoolAnalysisOutput) String() string {
	return fmt.Sprintf("DatabaseConnectionPoolAnalysisOutput{MaxConnections: %d, ActiveConnections: %d, IdleConnections: %d, PendingRequests: %d}", o.MaxConnections, o.ActiveConnections, o.IdleConnections, o.PendingRequests)
}

func databaseConnectionPoolAnalysis(ctx *ai.ToolContext, input DatabaseConnectionPoolAnalysisInput) (ToolOutput, error) {

	output := DatabaseConnectionPoolAnalysisOutput{
		MaxConnections:    100,
		ActiveConnections: 100,
		IdleConnections:   0,
		PendingRequests:   58,
	}

	return ToolOutput{
		ToolName: "database_connection_pool_analysis",
		Summary:  "Database connection pool is fully exhausted (100/100), currently 58 requests are queued waiting for connections",
		Result:   output,
	}, nil
}

// ================================================
// Kubernetes Get Pod Resources Tool
// ================================================

type KubernetesGetPodResourcesInput struct {
	ServiceName string `json:"service_name" jsonschema:"required,description=The name of the service"`
	Namespace   string `json:"namespace" jsonschema:"required,description=The namespace of the service"`
}

func (i KubernetesGetPodResourcesInput) String() string {
	return fmt.Sprintf("KubernetesGetPodResourcesInput{ServiceName: %s, Namespace: %s}", i.ServiceName, i.Namespace)
}

type PodResource struct {
	PodName         string  `json:"pod_name"`
	CPUUsageCores   float64 `json:"cpu_usage_cores"`
	CPURequestCores float64 `json:"cpu_request_cores"`
	CPULimitCores   float64 `json:"cpu_limit_cores"`
	MemoryUsageMi   int     `json:"memory_usage_mi"`
	MemoryRequestMi int     `json:"memory_request_mi"`
	MemoryLimitMi   int     `json:"memory_limit_mi"`
}

type KubernetesGetPodResourcesOutput struct {
	Pods []PodResource `json:"pods"`
}

func (o KubernetesGetPodResourcesOutput) String() string {
	return fmt.Sprintf("KubernetesGetPodResourcesOutput{Pods: %d items}", len(o.Pods))
}

func kubernetesGetPodResources(ctx *ai.ToolContext, input KubernetesGetPodResourcesInput) (ToolOutput, error) {

	output := KubernetesGetPodResourcesOutput{
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
	}

	return ToolOutput{
		ToolName: "kubernetes_get_pod_resources",
		Summary:  "2 out of 2 pods are near their memory limits",
		Result:   output,
	}, nil
}

// ================================================
// Dubbo Service Status Tool
// ================================================

type DubboServiceStatusInput struct {
	ServiceName string `json:"service_name" jsonschema:"required,description=The name of the Dubbo service"`
}

func (i DubboServiceStatusInput) String() string {
	return fmt.Sprintf("DubboServiceStatusInput{ServiceName: %s}", i.ServiceName)
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
	Providers []DubboProvider `json:"providers"`
	Consumers []DubboConsumer `json:"consumers"`
}

func (o DubboServiceStatusOutput) String() string {
	return fmt.Sprintf("DubboServiceStatusOutput{Providers: %d items, Consumers: %d items}", len(o.Providers), len(o.Consumers))
}

func dubboServiceStatus(ctx *ai.ToolContext, input DubboServiceStatusInput) (ToolOutput, error) {

	output := DubboServiceStatusOutput{
		Providers: []DubboProvider{
			{IP: "192.168.1.10", Port: 20880, Status: "healthy"},
			{IP: "192.168.1.11", Port: 20880, Status: "healthy"},
		},
		Consumers: []DubboConsumer{
			{IP: "192.168.1.20", Application: "web-frontend", Status: "connected"},
			{IP: "192.168.1.21", Application: "api-gateway", Status: "connected"},
		},
	}

	return ToolOutput{
		ToolName: "dubbo_service_status",
		Summary:  fmt.Sprintf("Provider and consumer status query for service %s completed", input.ServiceName),
		Result:   output,
	}, nil
}

// ================================================
// Query Log Database Tool
// ================================================

type QueryLogDatabaseInput struct {
	ServiceName      string `json:"service_name" jsonschema:"required,description=The name of the service"`
	Keyword          string `json:"keyword" jsonschema:"required,description=Keyword to search for"`
	TimeRangeMinutes int    `json:"time_range_minutes" jsonschema:"required,description=Time range in minutes"`
}

func (i QueryLogDatabaseInput) String() string {
	return fmt.Sprintf("QueryLogDatabaseInput{ServiceName: %s, Keyword: %s, TimeRangeMinutes: %d}", i.ServiceName, i.Keyword, i.TimeRangeMinutes)
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type QueryLogDatabaseOutput struct {
	TotalHits int        `json:"total_hits"`
	Logs      []LogEntry `json:"logs"`
}

func (o QueryLogDatabaseOutput) String() string {
	return fmt.Sprintf("QueryLogDatabaseOutput{TotalHits: %d, Logs: %d items}", o.TotalHits, len(o.Logs))
}

func queryLogDatabase(ctx *ai.ToolContext, input QueryLogDatabaseInput) (ToolOutput, error) {

	output := QueryLogDatabaseOutput{
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
	}

	return ToolOutput{
		ToolName: "query_log_database",
		Summary:  fmt.Sprintf("Found 152 log entries about '%s' in the past %d minutes", input.Keyword, input.TimeRangeMinutes),
		Result:   output,
	}, nil
}

// ================================================
// Search Archived Logs Tool
// ================================================

type SearchArchivedLogsInput struct {
	FilePathPattern string `json:"file_path_pattern" jsonschema:"required,description=File path pattern to search"`
	GrepKeyword     string `json:"grep_keyword" jsonschema:"required,description=Keyword to grep for"`
}

func (i SearchArchivedLogsInput) String() string {
	return fmt.Sprintf("SearchArchivedLogsInput{FilePathPattern: %s, GrepKeyword: %s}", i.FilePathPattern, i.GrepKeyword)
}

type MatchingLine struct {
	FilePath    string `json:"file_path"`
	LineNumber  int    `json:"line_number"`
	LineContent string `json:"line_content"`
}

type SearchArchivedLogsOutput struct {
	FilesSearched int            `json:"files_searched"`
	MatchingLines []MatchingLine `json:"matching_lines"`
}

func (o SearchArchivedLogsOutput) String() string {
	return fmt.Sprintf("SearchArchivedLogsOutput{FilesSearched: %d, MatchingLines: %d items}", o.FilesSearched, len(o.MatchingLines))
}

func searchArchivedLogs(ctx *ai.ToolContext, input SearchArchivedLogsInput) (ToolOutput, error) {

	output := SearchArchivedLogsOutput{
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
	}

	return ToolOutput{
		ToolName: "search_archived_logs",
		Summary:  "Found multiple queries in archived log files, searched 5 files and found 2 matching lines",
		Result:   output,
	}, nil
}

// ================================================
// Query Knowledge Base Tool
// ================================================

type QueryKnowledgeBaseInput struct {
	QueryText string `json:"query_text" jsonschema:"required,description=Query text to search for"`
}

func (i QueryKnowledgeBaseInput) String() string {
	return fmt.Sprintf("QueryKnowledgeBaseInput{QueryText: %s}", i.QueryText)
}

type KnowledgeDocument struct {
	Source          string  `json:"source"`
	ContentSnippet  string  `json:"content_snippet"`
	SimilarityScore float64 `json:"similarity_score"`
}

type QueryKnowledgeBaseOutput struct {
	Documents []KnowledgeDocument `json:"documents"`
}

func (o QueryKnowledgeBaseOutput) String() string {
	return fmt.Sprintf("QueryKnowledgeBaseOutput{Documents: %d items}", len(o.Documents))
}

// ================================================
// Tool Registration Function
// ================================================

type MockToolManager struct {
	registry *genkit.Genkit
	tools    []ai.Tool
}

func defineMockTools(rt *runtime.Runtime) []ai.Tool {
	g := rt.GetGenkitRegistry()
	return []ai.Tool{
		genkit.DefineTool(g, "prometheus_query_service_latency", "Query P95/P99 latency metrics for a specific service within a time range", prometheusQueryServiceLatency),
		genkit.DefineTool(g, "prometheus_query_service_traffic", "Query request rate (QPS) and error rate for a specific service within a time range", prometheusQueryServiceTraffic),
		genkit.DefineTool(g, "query_timeseries_database", "Execute a complete PromQL query for deep or custom analysis of Prometheus historical data", queryTimeseriesDatabase),
		genkit.DefineTool(g, "application_performance_profiling", "Perform performance profiling on a specific service instance (Pod) and return the function call stack consuming the most CPU in structured text format", applicationPerformanceProfiling),
		genkit.DefineTool(g, "jvm_performance_analysis", "Check the JVM status of a specific Java service, especially GC (garbage collection) activity", jvmPerformanceAnalysis),
		genkit.DefineTool(g, "trace_dependency_view", "Query upstream and downstream dependencies of a specific service based on tracing data", traceDependencyView),
		genkit.DefineTool(g, "trace_latency_analysis", "Analyze tracing data for a specific service within a time range to identify downstream calls with highest latency", traceLatencyAnalysis),
		genkit.DefineTool(g, "database_connection_pool_analysis", "Query the connection pool status of a specific service's database connection", databaseConnectionPoolAnalysis),
		genkit.DefineTool(g, "kubernetes_get_pod_resources", "Use kubectl-like functionality to get CPU and memory static configuration (limits/requests) and dynamic usage for all pods of a specific service", kubernetesGetPodResources),
		genkit.DefineTool(g, "dubbo_service_status", "Use dubbo-admin-like commands to query the provider and consumer lists and their status for a specific Dubbo service", dubboServiceStatus),
		genkit.DefineTool(g, "query_log_database", "Query indexed log databases (such as Elasticsearch, Loki) for real-time or near real-time log analysis", queryLogDatabase),
		genkit.DefineTool(g, "search_archived_logs", "Perform text search (similar to grep) in archived log files (such as .log.gz files stored in S3 or server file system)", searchArchivedLogs),
	}
}

func NewMockToolManager(rt *runtime.Runtime) *MockToolManager {
	g := rt.GetGenkitRegistry()
	return &MockToolManager{
		registry: g,
		tools:    defineMockTools(rt),
	}
}

func (mtm *MockToolManager) Register(tools ...ai.Tool) {
	mtm.tools = append(mtm.tools, tools...)
}

func (mtm *MockToolManager) ToolRefs() []ai.ToolRef {
	if mtm.tools == nil {
		return nil
	}
	var toolRef []ai.ToolRef
	for _, tool := range mtm.tools {
		toolRef = append(toolRef, tool)
	}
	return toolRef
}
