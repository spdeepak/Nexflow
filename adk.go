package main

import (
	"context"
	"fmt"
	"log"

	agent2 "google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/full"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/mcptoolset"
)

func main() {
	ctx := context.Background()
	model, err := openaimodel.NewModel(ctx, "gemma4:31b-cloud", &openaimodel.ClientConfig{
		BaseURL: "http://localhost:11434/v1",
	})

	// Connect to the remote MCP server over HTTP (JSON-RPC), matching the
	// endpoint your curl example hits. Leaving Transport nil makes the
	// toolset build a StreamableClientTransport from Endpoint, so the SDK
	// talks HTTP to the remote server instead of launching a local subprocess.
	mcpToolset, err := mcptoolset.New(mcptoolset.Config{
		Endpoint:            "https://aisenseapi.com/mcp",
		RequireConfirmation: true,
		RequireConfirmationProvider: func(toolName string, toolInput any) bool {
			if toolName == "shorten_url" && toolInput == "https://aisense.no/free-public-mcp-server" {
				return true
			}
			return false
		},
	})
	mcpToolset = tool.FilterToolset(mcpToolset, tool.AllowedToolsPredicate([]string{"generate_uuid", "shorten_url"}))
	if err != nil {
		log.Fatal(err)
	}

	// The toolset auto-discovers available tools from the MCP server
	// (e.g. "generate_uuid") via tools/list, and exposes them to the agent
	// as callable tools without you writing per-tool wrappers.
	agent, err := llmagent.New(llmagent.Config{
		Name:        "mcp_demo_agent",
		Model:       model,
		Description: "Agent that can call tools exposed by the AISense MCP server.",
		Instruction: "You have access to tools from a connected MCP server, " +
			"including a all the tools in the MCP URL. Use them whenever the user's " +
			"request matches what a tool can do. Also, tell the user if you used the mcp to fulfill the request of the user",
		Toolsets: []tool.Toolset{mcpToolset},
	})
	if err != nil {
		log.Fatal(err)
	}
	config := &launcher.Config{
		AgentLoader: agent2.NewSingleLoader(agent),
	}
	l := full.NewLauncher()
	if err = l.Execute(context.Background(), config, nil); err != nil {
		fmt.Println(err)
	}

}
