package grafana

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var LatencySummary = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "api",
		Name:       "latency_seconds",
		Help:       "Requests Latencies",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"method","path"},
)


func Latency(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		handler.ServeHTTP(w, r)
		elapsed := time.Since(start).Seconds()
		LatencySummary.WithLabelValues(
			r.Method,
			r.URL.Path,
		).Observe(float64(elapsed))
	})
}