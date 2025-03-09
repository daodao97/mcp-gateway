package main

import "net/http"

// CORS 中间件
func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 对于 OPTIONS 请求，直接返回 CORS 头部
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*") // 或者指定域名
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400") // 24小时
			w.WriteHeader(http.StatusOK)
			return
		}

		// 对于非 OPTIONS 请求，继续处理
		handler.ServeHTTP(w, r)
	})
}
