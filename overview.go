package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/daodao97/xgo/xlog"
	_client "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type ServerInfo struct {
	Type      string                `json:"type"`
	Url       string                `json:"url"`
	Info      *mcp.InitializeResult `json:"info,omitempty"`
	Prompt    []mcp.PromptMessage   `json:"prompt,omitempty"`
	Tools     []mcp.Tool            `json:"tools,omitempty"`
	Resources []mcp.Resource        `json:"resources,omitempty"`
}

func getServerInfo(serverUrl string) (*ServerInfo, error) {
	client, err := _client.NewSSEMCPClient(serverUrl)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the client
	if err := client.Start(ctx); err != nil {
		xlog.Error("Failed to start client", xlog.String("serverUrl", serverUrl), xlog.Err(err))
		return nil, err
	}

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	result, err := client.Initialize(ctx, initRequest)
	if err != nil {
		xlog.Error("Failed to initialize", xlog.String("serverUrl", serverUrl), xlog.Err(err))
		return nil, err
	}

	// Test Ping
	if err := client.Ping(ctx); err != nil {
		xlog.Error("Ping failed", xlog.String("serverUrl", serverUrl), xlog.Err(err))
	}

	// Test ListTools
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := client.ListTools(ctx, toolsRequest)
	if err != nil {
		xlog.Error("ListTools failed", xlog.String("serverUrl", serverUrl), xlog.Err(err))
	}

	xlog.Info("ListTools result", xlog.Any("result", toolsResult))

	// Test ListResources
	resourcesRequest := mcp.ListResourcesRequest{}
	resourcesResult, err := client.ListResources(ctx, resourcesRequest)
	if err != nil {
		xlog.Error("ListResources failed", xlog.String("serverUrl", serverUrl), xlog.Err(err))
	}

	xlog.Info("ListResources result", xlog.Any("result", resourcesResult))

	// Test GetPrompt
	promptRequest := mcp.GetPromptRequest{}
	promptResult, err := client.GetPrompt(ctx, promptRequest)
	if err != nil {
		xlog.Error("GetPrompt failed", xlog.String("serverUrl", serverUrl), xlog.Err(err))
	}

	xlog.Info("GetPrompt result", xlog.Any("result", promptResult))

	info := &ServerInfo{
		Info: result,
	}
	if promptResult != nil {
		info.Prompt = promptResult.Messages
	}
	if toolsResult != nil {
		info.Tools = toolsResult.Tools
	}
	if resourcesResult != nil {
		info.Resources = resourcesResult.Resources
	}

	return info, nil
}

func Overview(w http.ResponseWriter, r *http.Request) {
	domain := os.Getenv("MCP_GATEWAY_DOMAIN")
	if domain == "" {
		domain = "http://localhost:3121"
	}

	_domain, err := url.Parse(domain)
	if err != nil {
		xlog.Error("Failed to parse domain", xlog.String("domain", domain), xlog.Err(err))
		return
	}

	for prefix, serveUrl := range routeMap {
		if _, ok := serverInfoMap[prefix]; ok {
			continue
		}
		serverInfo, err := getServerInfo(serveUrl)
		if err != nil {
			xlog.Error("Failed to get server info", xlog.String("serverUrl", serveUrl), xlog.Err(err))
			continue
		}

		_severUrl, err := url.Parse(serveUrl)
		if err != nil {
			xlog.Error("Failed to parse server url", xlog.String("serverUrl", serveUrl), xlog.Err(err))
			continue
		}

		_severUrl.Scheme = _domain.Scheme
		_severUrl.Host = _domain.Host
		_severUrl.Path = prefix + _severUrl.Path
		serverInfo.Type = "sse"
		serverInfo.Url = _severUrl.String()
		serverInfoMap[prefix] = serverInfo
	}

	// response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(serverInfoMap)
}
