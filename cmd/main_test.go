package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{name: "minimum", input: "1", want: 1},
		{name: "maximum", input: "65535", want: 65535},
		{name: "zero", input: "0", wantErr: true},
		{name: "negative", input: "-1", wantErr: true},
		{name: "overflow", input: "65536", wantErr: true},
		{name: "not a number", input: "http", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePort(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parsePort(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateHost(t *testing.T) {
	for _, host := range []string{"example.com", "127.0.0.1", "::1"} {
		if _, err := validateHost(host); err != nil {
			t.Errorf("validateHost(%q) returned error: %v", host, err)
		}
	}

	for _, host := range []string{"", "a/b", "a\\b", "a\r\nb"} {
		if _, err := validateHost(host); err == nil {
			t.Errorf("validateHost(%q) accepted an invalid host", host)
		}
	}
}

func TestCorsDisabledByDefault(t *testing.T) {
	t.Setenv("CORS_ALLOW_ORIGINS", "")
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(corsMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://untrusted.example")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if got := res.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS header: %q", got)
	}
}

func TestParseRateLimit(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{name: "default", input: "", want: 100},
		{name: "configured", input: "25", want: 25},
		{name: "configured with spaces", input: " 25 ", want: 25},
		{name: "unlimited", input: "-1", want: -1},
		{name: "invalid negative", input: "-2", wantErr: true},
		{name: "invalid text", input: "fast", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRateLimit(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRateLimit(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parseRateLimit(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRateLimitReturnsOKJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimitMiddleware(1))
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"data": "ok"})
	})

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusCreated)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusOK {
		t.Fatalf("limited request status = %d, want %d", second.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(second.Body.String()); got != `{"data":"rate limit"}` {
		t.Fatalf("limited request body = %q, want rate-limit JSON", got)
	}
}

func TestRateLimitCanBeDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimitMiddleware(-1))
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	for i := 0; i < 2; i++ {
		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/", nil))
		if res.Code != http.StatusNoContent {
			t.Fatalf("unlimited request %d status = %d, want %d", i+1, res.Code, http.StatusNoContent)
		}
	}
}

func TestRateLimitIsGlobal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(rateLimitMiddleware(1))
	router.GET("/first", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/second", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	firstRequest := httptest.NewRequest(http.MethodGet, "/first", nil)
	firstRequest.RemoteAddr = "192.0.2.10:1234"
	first := httptest.NewRecorder()
	router.ServeHTTP(first, firstRequest)

	secondRequest := httptest.NewRequest(http.MethodGet, "/second", nil)
	secondRequest.RemoteAddr = "198.51.100.20:5678"
	second := httptest.NewRecorder()
	router.ServeHTTP(second, secondRequest)

	if first.Code != http.StatusNoContent {
		t.Fatalf("first request status = %d, want %d", first.Code, http.StatusNoContent)
	}
	if second.Code != http.StatusOK {
		t.Fatalf("second request status = %d, want %d", second.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(second.Body.String()); got != `{"data":"rate limit"}` {
		t.Fatalf("second request body = %q, want rate-limit JSON", got)
	}
}
