package server

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qianlima-666/nginxpulse/internal/config"
)

func attachAppConfig(router *gin.Engine) {
	handler := func(c *gin.Context) {
		cfg, err := config.ReadRawConfig()
		if err != nil {
			c.Header("Content-Type", "application/javascript; charset=utf-8")
			c.Header("Cache-Control", "no-store")
			c.String(
				http.StatusOK,
				"window.__NGINXPULSE_BASE_PATH__ = \"\";\nwindow.__NGINXPULSE_SERVER_STATUS_ENABLED__ = false;",
			)
			return
		}
		base := config.NormalizeWebBasePath(cfg.System.WebBasePath)
		prefix := ""
		if base != "" {
			prefix = "/" + base
		}
		payload, _ := json.Marshal(prefix)
		serverStatusEnabled, _ := json.Marshal(cfg.System.ServerStatus.Enabled)
		c.Header("Content-Type", "application/javascript; charset=utf-8")
		c.Header("Cache-Control", "no-store")
		c.String(
			http.StatusOK,
			"window.__NGINXPULSE_BASE_PATH__ = %s;\nwindow.__NGINXPULSE_SERVER_STATUS_ENABLED__ = %s;",
			payload,
			serverStatusEnabled,
		)
	}
	router.GET("/app-config.js", handler)
	router.GET("/m/app-config.js", handler)
}
