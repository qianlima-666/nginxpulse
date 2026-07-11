package server

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/qianlima-666/nginxpulse/internal/analytics"
	"github.com/qianlima-666/nginxpulse/internal/ingest"
	"github.com/qianlima-666/nginxpulse/internal/web"
	"github.com/sirupsen/logrus"
)

// StartHTTPServer configures and starts the HTTP server in a goroutine.
func StartHTTPServer(statsFactory *analytics.StatsFactory, logParser *ingest.LogParser, addr string) (*http.Server, error) {
	router := buildRouter(statsFactory, logParser)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("HTTP 服务器运行失败")
		}
	}()

	logrus.Infof("服务器已启动，监听地址: %s", addr)
	return server, nil
}

func buildRouter(statsFactory *analytics.StatsFactory, logParser *ingest.LogParser) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(requestLogger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", accessKeyHeader},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	router.Use(basePathMiddleware(router))
	router.Use(accessKeyMiddleware())

	router.GET("/healthz", gin.WrapF(HealthHandler))

	web.SetupRoutes(router, statsFactory, logParser)
	attachAppConfig(router)
	attachWebUI(router)

	return router
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()
		fields := logrus.Fields{
			"method":      c.Request.Method,
			"path":        path,
			"raw_query":   rawQuery,
			"client_ip":   c.ClientIP(),
			"status":      status,
			"duration_ms": duration.Milliseconds(),
		}
		if referer := c.Request.Referer(); referer != "" {
			fields["referer"] = referer
		}
		if ua := c.Request.UserAgent(); ua != "" {
			fields["user_agent"] = ua
		}
		if websiteID := c.Query("id"); websiteID != "" {
			fields["website_id"] = websiteID
		}

		if status >= 400 {
			logrus.WithFields(fields).Warn("HTTP 请求返回错误状态")
			return
		}

		if strings.HasPrefix(path, "/api/") && duration > 100*time.Millisecond {
			logrus.WithFields(fields).Warn("高延迟 API 请求")
		}
	}
}
