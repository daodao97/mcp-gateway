package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/daodao97/xgo/xrequest"
	"github.com/daodao97/xgo/xutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"web search",
		"1.0.0",
	)

	// Add tool
	tool := mcp.NewTool("web_search",
		mcp.WithDescription("Search the web for a given query"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Query to search the web for"),
		),
	)

	// Add tool handler
	s.AddTool(tool, webSearchHandler)

	port := getEnv("MCP_SERVER_PORT", "8080")

	_s := server.NewSSEServer(
		s,
		server.WithBaseURL("http://localhost:"+port),
		server.WithMessageEndpoint("/message"),
		server.WithSSEEndpoint("/sse"),
	)

	fmt.Printf("Server started on port %s\n", port)

	xutil.Go(context.Background(), func() {
		regMcpServerToGateway(port)
	})

	if err := _s.Start(":" + port); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func webSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	fmt.Printf("request: %+v\n", request)
	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return mcp.NewToolResultError("query must be a string"), nil
	}

	session := server.SessionStoreFromCtx(ctx)
	if session == nil {
		return mcp.NewToolResultError("sessionId not found"), nil
	}

	tavilyKey := session.Get("tavily_api_key")

	resp, err := TavilySearch(&TavilySearchReq{
		Query: query,
	}, tavilyKey)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var results []string
	for _, result := range resp.Results {
		// 格式化为Markdown: 标题作为二级标题，URL作为链接，内容作为引用块
		markdownResult := fmt.Sprintf("### %s\n[%s](%s)\n\n> %s\n",
			result.Title,
			result.URL,
			result.URL,
			strings.ReplaceAll(result.Content, "\n", "\n> "))
		results = append(results, markdownResult)
	}

	return mcp.NewToolResultText(strings.Join(results, "\n---\n")), nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func regMcpServerToGateway(port string) {
	gatewayUrl := getEnv("MCP_GATEWAY_DOMAIN", "http://localhost:3121")
	resp, err := xrequest.New().
		SetBody(map[string]any{
			"server_name": "web_search",
			"server_url":  "http://localhost:" + port + "/sse",
		}).
		SetRetry(30, 10*time.Second).
		Post(gatewayUrl + "/register")
	if err != nil {
		fmt.Printf("Failed to register MCP server to gateway: %v\n", err)
	}

	fmt.Printf("MCP server registered to gateway: %v\n", resp)
}
