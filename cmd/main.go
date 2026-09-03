package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TorchPing/go-torch/pkg/ping"
	"github.com/TorchPing/go-torch/pkg/resolve"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	version = "dev"
)

func routePing(c *gin.Context) {
	host, err := validateHost(c.Param("host"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	port, err := parsePort(c.Param("port"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	target := ping.Target{
		Host:     host,
		Port:     port,
		Counter:  3,
		Interval: time.Second,
		Timeout:  time.Second * 3,
	}

	pinger := ping.NewPing()
	pinger.SetTarget(&target)

	pingerDone := pinger.StartContext(c.Request.Context())

	select {
	case <-pingerDone:
		break
	}
	result := pinger.Result()
	var resTime float64

	if result.SuccessCounter == 0 {
		resTime = 0
	} else {
		resTime = float64(result.TotalDuration) / float64(time.Millisecond) / float64(result.SuccessCounter)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": result.SuccessCounter > 0,
		"time":   resTime,
	})
}

func routeResolve(c *gin.Context) {
	host, err := validateHost(c.Param("host"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	target := resolve.Target{
		Host:     host,
		Counter:  3,
		Interval: time.Second,
		Timeout:  time.Second * 3,
	}

	resolver := resolve.NewResolve()
	resolver.SetTarget(&target)

	pingerDone := resolver.StartContext(c.Request.Context())

	select {
	case <-pingerDone:
		break
	}
	result := resolver.Result()
	var resTime float64

	if result.SuccessCounter == 0 {
		resTime = 0
	} else {
		resTime = float64(result.TotalDuration) / float64(time.Millisecond) / float64(result.SuccessCounter)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": result.SuccessCounter > 0,
		"time":   resTime,
		"result": result.Addrs,
	})
}

func validateHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" || len(host) > 253 {
		return "", errors.New("host must be between 1 and 253 characters")
	}
	if strings.ContainsAny(host, "\x00\r\n/\\") {
		return "", errors.New("host contains invalid characters")
	}
	return host, nil
}

func parsePort(port string) (uint16, error) {
	value, err := strconv.ParseUint(port, 10, 16)
	if err != nil || value == 0 {
		return 0, errors.New("port must be an integer between 1 and 65535")
	}
	return uint16(value), nil
}

func corsMiddleware() gin.HandlerFunc {
	origins := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGINS"))
	if origins == "" {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	allowedOrigins := make([]string, 0)
	for _, origin := range strings.Split(origins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}
	if len(allowedOrigins) == 0 {
		return func(c *gin.Context) {
			c.Next()
		}
	}
	return cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	})
}

func parseRateLimit(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 100, nil
	}
	rpm, err := strconv.Atoi(value)
	if err != nil || rpm < -1 {
		return 0, errors.New("RATE_LIMIT_RPM must be -1 or a non-negative integer")
	}
	return rpm, nil
}

func rateLimitMiddleware(rpm int) gin.HandlerFunc {
	if rpm == -1 {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	var mu sync.Mutex
	windowStart := time.Now()
	requests := 0
	return func(c *gin.Context) {
		mu.Lock()
		now := time.Now()
		if now.Sub(windowStart) >= time.Minute {
			windowStart = now
			requests = 0
		}
		if requests >= rpm {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusOK, gin.H{"data": "rate limit"})
			return
		}
		requests++
		mu.Unlock()
		c.Next()
	}
}

func main() {
	rateLimit, err := parseRateLimit(os.Getenv("RATE_LIMIT_RPM"))
	if err != nil {
		log.Fatalf("invalid rate limit: %v", err)
	}

	router := gin.Default()
	router.Use(rateLimitMiddleware(rateLimit))
	router.Use(corsMiddleware())
	if err := router.SetTrustedProxies(nil); err != nil {
		log.Fatalf("configure trusted proxies: %v", err)
	}

	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Meow~",
			"version": "Golang Edition",
			"ref":     version,
		})
	})

	router.GET("/ping/:host/:port", routePing)

	router.GET("/resolve/:host", routeResolve)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if _, err := parsePort(port); err != nil {
		log.Fatalf("invalid PORT: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
