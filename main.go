package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// 添加一个自定义的上下文键类型
type contextKey string

// 定义前缀的上下文键
const prefixKey contextKey = "prefix"
const sourceURLKey contextKey = "sourceURL"

var routeMap = map[string]string{
	"/web_search": "http://localhost:8080/sse",
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/overview", Overview)

	// 为每个前缀创建一个反向代理
	for prefix, target := range routeMap {
		targetURL, err := url.Parse(target)
		if err != nil {
			log.Fatalf("解析目标 URL %s 失败: %v", target, err)
		}

		targetURL.Path = ""

		// 为这个前缀创建代理
		proxy := createReverseProxy(targetURL)

		// 创建一个中间件来记录前缀
		prefixMiddleware := func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// 将前缀存储在请求上下文中
				ctx := context.WithValue(r.Context(), prefixKey, prefix)
				// 将源URL存储在请求上下文中
				ctx = context.WithValue(ctx, sourceURLKey, r.URL.String())
				// 使用新的上下文创建新的请求
				r = r.WithContext(ctx)
				// 继续处理请求
				handler.ServeHTTP(w, r)
			})
		}

		// 注册路由处理器，应用前缀中间件和CORS中间件
		mux.Handle(prefix+"/", prefixMiddleware(http.StripPrefix(prefix, corsMiddleware(proxy))))
		log.Printf("路由 %s/* 将转发到 %s", prefix, targetURL.String())
	}

	// 设置服务器
	server := &http.Server{
		Addr:         ":3000",
		Handler:      mux,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 0, // SSE 需要无限写入超时
	}

	// 启动服务器
	log.Println("反向代理服务器启动在 :3000")
	log.Fatal(server.ListenAndServe())
}
