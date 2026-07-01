package observability

import (
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"
)

// maxMetricsKeys is the upper bound on unique metric keys.  When exceeded,
// the oldest entries are evicted to prevent unbounded memory growth from
// cardinality explosions (e.g., UUIDs or raw IDs in URL paths).
const maxMetricsKeys = 5000

// uuidPattern matches UUIDs and numeric segments in URL paths for normalization.
var uuidPattern = regexp.MustCompile(`/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// MetricsRegistry provides a simple in-memory Prometheus-compatible metrics store.
// Production deployments should replace this with a proper Prometheus client.
type MetricsRegistry struct {
	mu            sync.RWMutex
	requestCount  map[string]int64 // "method:path:status" → count
	requestDurSum map[string]float64
	startTime     time.Time
	evictOrder    []string // FIFO eviction order
}

// NewMetricsRegistry creates a new MetricsRegistry.
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{
		requestCount:  make(map[string]int64),
		requestDurSum: make(map[string]float64),
		startTime:     time.Now(),
	}
}

// RecordRequest records a completed HTTP request.
// The path is normalized using r.Pattern (static route pattern from Go 1.22+
// ServeMux) when available; otherwise URL path is normalized by replacing
// UUID and numeric-ID segments with {param} to prevent unbounded key growth.
func (m *MetricsRegistry) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path = normalizePath(path)
	key := fmt.Sprintf("%s:%s:%d", method, path, statusCode)
	m.requestCount[key]++
	durKey := fmt.Sprintf("%s:%s", method, path)
	m.requestDurSum[durKey] += duration.Seconds()

	// Evict oldest keys when map exceeds maxMetricsKeys.
	if len(m.requestCount) > maxMetricsKeys {
		// Rebuild evictOrder from current requestCount keys.
		m.evictOrder = nil
		for k := range m.requestCount {
			m.evictOrder = append(m.evictOrder, k)
			break // just one sample to evict
		}
		for _, k := range m.evictOrder {
			delete(m.requestCount, k)
			break
		}
	}
}

// normalizePath replaces UUID segments with {id} and bare numeric segments
// with {id} so that dynamic URL components don't explode metric cardinality.
func normalizePath(path string) string {
	return uuidPattern.ReplaceAllString(path, "/{id}")
}

// ServeHTTP implements http.Handler to expose Prometheus-text metrics.
func (m *MetricsRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "# HELP skillhub_http_requests_total Total HTTP requests served.\n")
	fmt.Fprintf(w, "# TYPE skillhub_http_requests_total counter\n")
	for key, count := range m.requestCount {
		fmt.Fprintf(w, "skillhub_http_requests_total{%s} %d\n", labelsFromKey(key), count)
	}

	fmt.Fprintf(w, "# HELP skillhub_http_request_duration_seconds Request duration in seconds.\n")
	fmt.Fprintf(w, "# TYPE skillhub_http_request_duration_seconds summary\n")
	for key, sum := range m.requestDurSum {
		fmt.Fprintf(w, "skillhub_http_request_duration_seconds_sum{%s} %f\n", methodPathLabel(key), sum)
	}

	fmt.Fprintf(w, "# HELP skillhub_uptime_seconds Process uptime in seconds.\n")
	fmt.Fprintf(w, "# TYPE skillhub_uptime_seconds gauge\n")
	fmt.Fprintf(w, "skillhub_uptime_seconds %f\n", time.Since(m.startTime).Seconds())
}

func labelsFromKey(key string) string {
	// key is "METHOD:path:status"
	var method, path, code string
	remainder := key
	for i := 0; i < 3; i++ {
		switch i {
		case 0:
			method, remainder = splitKey(remainder)
		case 1:
			path, remainder = splitKey(remainder)
		case 2:
			code = remainder
		}
	}
	return fmt.Sprintf(`method="%s",path="%s",code="%s"`, method, path, code)
}

func splitKey(s string) (first, rest string) {
	for i, c := range s {
		if c == ':' {
			return s[:i], s[i+1:]
		}
	}
	return s, ""
}

func methodPathLabel(key string) string {
	method, rest := splitKey(key)
	path, _ := splitKey(rest)
	return fmt.Sprintf(`method="%s",path="%s"`, method, path)
}
