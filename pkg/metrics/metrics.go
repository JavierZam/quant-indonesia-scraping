package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// HTTPRequestsTotal counts all HTTP requests by method, path, and status code.
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "quant",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status_code"},
	)

	// HTTPRequestDuration tracks HTTP request latency by method and path.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "quant",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// IngestionArticlesTotal counts articles processed by the ingestion pipeline.
	IngestionArticlesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "quant",
			Name:      "ingestion_articles_total",
			Help:      "Total articles processed by ingestion pipeline.",
		},
		[]string{"source", "status"},
	)

	// LLMAnalysisDuration tracks how long LLM analysis calls take.
	LLMAnalysisDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "quant",
			Name:      "llm_analysis_duration_seconds",
			Help:      "LLM analysis duration in seconds.",
			Buckets:   []float64{0.5, 1, 2, 5, 10, 30},
		},
	)

	// LLMAnalysisTotal counts LLM analysis requests by status.
	LLMAnalysisTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "quant",
			Name:      "llm_analysis_total",
			Help:      "Total LLM analysis requests.",
		},
		[]string{"status"},
	)
)

// Register registers all custom Prometheus metrics with the default registry.
func Register() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		IngestionArticlesTotal,
		LLMAnalysisDuration,
		LLMAnalysisTotal,
	)
}
