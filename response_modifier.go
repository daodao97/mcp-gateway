package main

import (
	"bytes"
	"io"
	"log"
	"net/url"
	"path/filepath"
	"strings"
)

var currentServer = getEnv("CURRENT_SERVER", "http://localhost:3000")

// sseResponseModifier 用于修改 SSE 响应内容
type sseResponseModifier struct {
	original io.ReadCloser
	buffer   bytes.Buffer
	inEvent  bool
	event    string
	prefix   string
}

// Read 实现 io.Reader 接口，用于拦截和修改 SSE 数据
func (s *sseResponseModifier) Read(p []byte) (n int, err error) {
	// 如果缓冲区中有数据，先返回缓冲区中的数据
	if s.buffer.Len() > 0 {
		return s.buffer.Read(p)
	}

	// 从原始响应中读取数据
	n, err = s.original.Read(p)
	if n <= 0 {
		return n, err
	}

	// 处理读取到的数据
	data := string(p[:n])
	lines := strings.Split(data, "\n")
	var output bytes.Buffer

	for i, line := range lines {
		trimmedLine := strings.TrimRight(line, "\r")

		// 检测事件开始
		if strings.HasPrefix(trimmedLine, "event:") {
			s.event = strings.TrimSpace(strings.TrimPrefix(trimmedLine, "event:"))
			s.inEvent = true
			output.WriteString(trimmedLine + "\n")
		} else if strings.HasPrefix(trimmedLine, "data:") && s.event == "endpoint" {
			// 特殊处理 endpoint 事件的数据行
			originalURL := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "data:"))

			// 修改 URL，根据前缀进行替换
			var modifiedURL string
			if strings.HasPrefix(originalURL, "http") {
				_url, _ := url.Parse(originalURL)
				_url.Host = currentServer
				modifiedURL = filepath.Join(s.prefix, _url.Path) + "?" + _url.RawQuery
			} else {
				modifiedURL = s.prefix + originalURL
			}

			log.Printf("修改 endpoint URL: %s -> %s", originalURL, modifiedURL)
			output.WriteString("data: " + modifiedURL + "\n")
		} else {
			// 保持其他行不变
			output.WriteString(trimmedLine)
			// 只有不是最后一行时才添加换行符
			if i < len(lines)-1 || strings.HasSuffix(data, "\n") {
				output.WriteString("\n")
			}

			// 检测事件结束（空行）
			if trimmedLine == "" && s.inEvent {
				s.inEvent = false
				s.event = ""
			}
		}
	}

	// 将处理后的数据写入缓冲区
	processedData := output.Bytes()

	// 如果处理后的数据长度小于等于原始缓冲区长度，直接复制
	if len(processedData) <= len(p) {
		copy(p, processedData)
		return len(processedData), err
	}

	// 如果处理后的数据长度大于原始缓冲区，部分写入缓冲区，剩余部分保存
	copy(p, processedData[:len(p)])
	s.buffer.Write(processedData[len(p):])
	return len(p), err
}

// Close 实现 io.Closer 接口
func (s *sseResponseModifier) Close() error {
	return s.original.Close()
}
