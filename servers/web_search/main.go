package main

import (
	"context"
	"fmt"
	"strings"

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

	_s := server.NewSSEServer(
		s,
		server.WithBaseURL("http://localhost:8080"),
		server.WithMessageEndpoint("/message"),
		server.WithSSEEndpoint("/sse"),
	)
	if err := _s.Start(":8080"); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}

}

func webSearchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := request.Params.Arguments["query"].(string)
	if !ok {
		return mcp.NewToolResultError("query must be a string"), nil
	}

	resp, err := TavilySearch(&TavilySearchReq{
		Query: query,
	})
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
