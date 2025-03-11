package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

// 添加一个自定义的上下文键类型
type contextKey string

// 定义前缀的上下文键
const prefixKey contextKey = "prefix"
const sourceURLKey contextKey = "sourceURL"

// 添加互斥锁以保护 routeMap
var (
	routeMap     = map[string]string{}
	routeMapLock = sync.RWMutex{}
	proxyMap     = map[string]http.Handler{}
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/overview", Overview)
	mux.HandleFunc("/register", Register)

	// 动态路由处理器
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 提取请求路径中的前缀
		path := r.URL.Path
		var prefix string
		for p := range getRoutes() {
			if len(p) > 0 && p != "/" && (path == p || path+"/" == p || path[:len(p)+1] == p+"/") {
				prefix = p
				break
			}
		}

		if prefix == "" {
			http.NotFound(w, r)
			return
		}

		// 获取或创建代理
		handler := getOrCreateProxy(prefix)
		if handler == nil {
			http.Error(w, "路由目标无效", http.StatusBadGateway)
			return
		}

		// 调用处理器
		handler.ServeHTTP(w, r)
	})

	port := getEnv("MCP_GATEWAY_PORT", "3121")

	// 设置服务器
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 0, // SSE 需要无限写入超时
	}

	// 启动服务器
	log.Printf("反向代理服务器启动在 :%s", port)
	log.Fatal(server.ListenAndServe())
}

// 获取当前路由映射的安全副本
func getRoutes() map[string]string {
	routeMapLock.RLock()
	defer routeMapLock.RUnlock()

	routes := make(map[string]string, len(routeMap))
	for k, v := range routeMap {
		routes[k] = v
	}
	return routes
}

// 获取或创建代理处理器
func getOrCreateProxy(prefix string) http.Handler {
	routeMapLock.RLock()
	handler, exists := proxyMap[prefix]
	target := routeMap[prefix]
	routeMapLock.RUnlock()

	if exists {
		return handler
	}

	// 如果处理器不存在，创建一个新的
	if target != "" {
		routeMapLock.Lock()
		defer routeMapLock.Unlock()

		// 再次检查，避免竞态条件
		if handler, exists := proxyMap[prefix]; exists {
			return handler
		}

		targetURL, err := url.Parse(target)
		if err != nil {
			log.Printf("解析目标 URL %s 失败: %v", target, err)
			return nil
		}

		targetURL.Path = ""

		// 为这个前缀创建代理
		proxy := createReverseProxy(targetURL)

		// 创建中间件来记录前缀
		handler := prefixMiddleware(prefix)(http.StripPrefix(prefix, corsMiddleware(proxy)))

		// 保存到代理映射
		proxyMap[prefix] = handler
		log.Printf("动态添加路由 %s/* 将转发到 %s", prefix, targetURL.String())

		return handler
	}

	return nil
}

// 前缀中间件现在接受前缀作为参数
func prefixMiddleware(prefix string) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
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
}

func Register(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	type RegisterReq struct {
		ServerName string `json:"server_name"`
		ServerURL  string `json:"server_url"`
	}

	var req RegisterReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Failed to unmarshal request body", http.StatusBadRequest)
		return
	}

	fmt.Printf("Register request: %+v\n", req)

	// 安全地更新路由映射
	routeMapLock.Lock()
	routeMap["/"+req.ServerName] = req.ServerURL
	// 删除现有的代理缓存，强制重新创建
	delete(proxyMap, req.ServerName)
	routeMapLock.Unlock()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Register request received"))
}
