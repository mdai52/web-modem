package router

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

// BasicAuthMiddleware 实现 Basic Auth 中间件
func BasicAuthMiddleware(next http.Handler) http.Handler {
	username := os.Getenv("BASIC_AUTH_USER")
	password := os.Getenv("BASIC_AUTH_PASSWORD")

	// 如果未配置认证信息，直接放行
	if username == "" || password == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取 Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			unauthorized(w)
			return
		}

		// 解析 Basic Auth
		if !strings.HasPrefix(auth, "Basic ") {
			unauthorized(w)
			return
		}

		// 解码认证信息
		encoded := strings.TrimPrefix(auth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			unauthorized(w)
			return
		}

		// 验证用户名密码
		credentials := string(decoded)
		expectedCredentials := username + ":" + password
		if credentials != expectedCredentials {
			unauthorized(w)
			return
		}

		// 认证成功，继续处理请求
		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error": "Unauthorized"}`))
}
