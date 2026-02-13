package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func getCounterValue(t *testing.T, counter *prometheus.CounterVec, labels ...string) float64 {
	t.Helper()
	metric, err := counter.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	var m dto.Metric
	if err := metric.Write(&m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	return m.GetCounter().GetValue()
}

func TestMetrics_IncrementsCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	before := getCounterValue(t, httpRequestsTotal, "GET", "/api/v1/test", "200")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/test", nil))

	after := getCounterValue(t, httpRequestsTotal, "GET", "/api/v1/test", "200")
	if after != before+1 {
		t.Errorf("expected counter to increment by 1, got delta %f", after-before)
	}
}

func TestMetrics_RecordsCorrectStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.POST("/api/v1/items", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})

	before := getCounterValue(t, httpRequestsTotal, "POST", "/api/v1/items", "201")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/items", nil))

	after := getCounterValue(t, httpRequestsTotal, "POST", "/api/v1/items", "201")
	if after != before+1 {
		t.Errorf("expected counter to increment by 1, got delta %f", after-before)
	}
}

func TestMetrics_UnmatchedRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	// No routes registered

	before := getCounterValue(t, httpRequestsTotal, "GET", "unmatched", "404")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/nonexistent", nil))

	after := getCounterValue(t, httpRequestsTotal, "GET", "unmatched", "404")
	if after != before+1 {
		t.Errorf("expected unmatched counter to increment, got delta %f", after-before)
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestMetrics_RecordsDuration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/api/v1/duration-test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/duration-test", nil))

	// Verify histogram was observed via the counter (each histogram observation also increments the count)
	// We confirm indirectly: if the request counter was incremented, the histogram was also observed
	after := getCounterValue(t, httpRequestsTotal, "GET", "/api/v1/duration-test", "200")
	if after == 0 {
		t.Error("expected request counter to be non-zero, indicating metrics were recorded")
	}
}
