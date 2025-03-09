package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// 创建反向代理的辅助函数
func createReverseProxy(targetURL *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定义传输层以支持 SSE
	defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	defaultTransport.ResponseHeaderTimeout = 0 // SSE 需要长连接
	defaultTransport.IdleConnTimeout = 0       // 防止空闲连接超时

	proxy.Transport = defaultTransport

	// 自定义代理的 Director 函数
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// 可以在这里修改请求头
		req.Header.Set("X-Proxy", "Go-Reverse-Proxy")

		// 从请求上下文中获取源URL
		if sourceURL, ok := req.Context().Value(sourceURLKey).(string); ok {
			log.Printf("代理请求: %s %s -> %s", req.Method, sourceURL, req.URL.String())
		} else {
			log.Printf("代理请求: %s %s -> %s", req.Method, req.URL.String(), req.URL.String())
		}
	}

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("代理错误: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("代理服务器错误"))
	}

	// 自定义修改响应
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 添加 CORS 头部
		resp.Header.Set("Access-Control-Allow-Origin", "*") // 或者指定域名
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		resp.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		resp.Header.Set("Access-Control-Allow-Credentials", "true")

		// 对于 SSE 响应，确保不会缓存并修改内容
		if resp.Header.Get("Content-Type") == "text/event-stream" {
			resp.Header.Set("Cache-Control", "no-cache")
			resp.Header.Set("Connection", "keep-alive")

			// 从请求上下文中获取前缀
			var requestPrefix string
			if prefixVal := resp.Request.Context().Value(prefixKey); prefixVal != nil {
				requestPrefix = prefixVal.(string)
			}

			// 拦截并修改 SSE 响应内容
			originalBody := resp.Body

			// 创建一个自定义的响应体读取器，传入前缀
			sseModifier := &sseResponseModifier{
				original: originalBody,
				inEvent:  false,
				event:    "",
				prefix:   requestPrefix, // 传递前缀到修改器
			}

			// 替换原始响应体
			resp.Body = sseModifier
		}
		return nil
	}

	return proxy
}
