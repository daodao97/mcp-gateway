package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create MCP server
	s := server.NewMCPServer(
		"get weather",
		"1.0.0",
	)

	// Add tool
	tool := mcp.NewTool("get_weather",
		mcp.WithDescription("Get the weather for a given city"),
		mcp.WithString("city",
			mcp.Required(),
			mcp.Description("Name of the city to get the weather for"),
		),
	)

	// Add tool handler
	s.AddTool(tool, getWeatherHandler)

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

func getWeatherHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	city, ok := request.Params.Arguments["city"].(string)
	if !ok {
		return mcp.NewToolResultError("city must be a string"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("The weather in %s is sunny", city)), nil
}
