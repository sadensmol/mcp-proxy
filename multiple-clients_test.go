package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
)

var userHeader = func(name string) string {
	return fmt.Sprintf(`{"user": {"id:""test-user%s", "name": "%s", "email": "%s@mail.com"}}`, name, name, name)
}

func TestServerWithConfigFile(t *testing.T) {
	config, err := LoadConfig("config.json")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startHTTPServer(ctx, config)

	time.Sleep(2 * time.Second) // Give the server a moment to start up

	println("starting 2 clients ...")
	in, out, err := runClient(ctx, "client1")
	if err != nil {
		t.Fatalf("Failed to run client: %v", err)
	}
	in2, out2, err := runClient(ctx, "client2")
	if err != nil {
		t.Fatalf("Failed to run client: %v", err)
	}

	in <- "what is the week day today?"
	fmt.Printf("client 1 response: %s\n", <-out)
	time.Sleep(3 * time.Second)
	in2 <- "what is the week day tomorrow?"
	fmt.Printf("client 2 response: %s\n", <-out2)
}

func runClient(ctx context.Context, name string) (inCh chan string, outCh chan string, err error) {
	cl, err := client.NewSSEMCPClient("http://localhost:9090/brave-search/sse", transport.WithHeaders((map[string]string{
		"Authorization": userHeader(name)})))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Start the client
	if err := cl.Start(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to start client: %v", err)
	}

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	result, err := cl.Initialize(ctx, initRequest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize: %w", err)
	}
	fmt.Printf("Client initialized %+v", result)

	// Test Ping
	if err := cl.Ping(ctx); err != nil {
		return nil, nil, fmt.Errorf("ping failed: %w", err)
	}

	tools, err := cl.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tools: %w", err)
	}
	for _, tool := range tools.Tools {
		println(tool.Name)
	}

	inCh = make(chan string)
	outCh = make(chan string)
	go func() {
		defer cl.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-inCh:
				println("calling tool...")
				request := mcp.CallToolRequest{}
				request.Params.Name = "brave_web_search"
				request.Params.Arguments = map[string]interface{}{
					"query": in,
					"count": 1,
				}

				res, err := cl.CallTool(ctx, request)
				if err != nil {
					fmt.Printf("failed to call tool: %v", err)
					return
				}
				outCh <- spew.Sdump(res)
			}
		}
	}()

	return inCh, outCh, nil
}
