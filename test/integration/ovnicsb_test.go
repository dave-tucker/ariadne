package integration

import (
	"context"
	"testing"
	"time"

	"github.com/dave-tucker/ariadne/internal/mcp/ovnicsb"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"
)

func TestOVNICSBIntegration(t *testing.T) {
	suite.Run(t, new(OVNICSBIntegrationTestSuite))
}

type OVNICSBIntegrationTestSuite struct {
	suite.Suite
}

func (suite *OVNICSBIntegrationTestSuite) TestToolsList() {
	// Create a new OVN IC SB server directly
	server, err := ovnicsb.NewServer("localhost", 8089)
	suite.Require().NoError(err, "Failed to create OVN IC SB server")

	// Start the server on a specific port
	ctx := context.Background()
	err = server.Start(ctx, "localhost:8089")
	suite.Require().NoError(err, "Failed to start server")
	defer server.Stop(ctx)

	// Give the server a moment to start
	time.Sleep(1 * time.Second)

	// Create MCP client implementation
	impl := &mcp.Implementation{
		Name:    "ovsdb-mcp-test-client",
		Title:   "OVSDB MCP Test Client",
		Version: "1.0.0",
	}

	// Create MCP client
	mcpClient := mcp.NewClient(impl, nil)

	// Create Streamable HTTP transport to connect to the MCP server
	transport := mcp.NewStreamableClientTransport("http://localhost:8089/", nil)

	// Connect to the MCP server
	session, err := mcpClient.Connect(ctx, transport)
	suite.Require().NoError(err, "Failed to connect to MCP server")
	defer session.Close()

	// List tools using the MCP client
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	suite.Require().NoError(err, "Failed to list tools")

	// Assert that tools are returned
	suite.Assert().NotEmpty(toolsResult.Tools, "Expected tools to be returned")

	// Define expected tools for OVN IC SB MCP server
	expectedTools := []string{
		"list_availability_zones",
		"list_datapath_bindings",
		"list_port_bindings",
		"list_gateways",
		"list_routes",
		"list_encaps",
		"list_ic_sb_globals",
	}

	// Create a map of returned tool names for easy lookup
	returnedTools := make(map[string]bool)
	for _, tool := range toolsResult.Tools {
		returnedTools[tool.Name] = true
		suite.T().Logf("Found tool: %s - %s", tool.Name, tool.Description)
	}

	// Assert that all expected tools are present
	for _, expectedTool := range expectedTools {
		suite.Assert().True(returnedTools[expectedTool], "Expected tool %s to be present", expectedTool)
	}

	// Assert that we have the expected number of tools
	suite.Assert().Equal(len(expectedTools), len(toolsResult.Tools), "Expected %d tools, got %d", len(expectedTools), len(toolsResult.Tools))

	// Additional assertions for tool structure
	for _, tool := range toolsResult.Tools {
		suite.Assert().NotEmpty(tool.Name, "Tool name should not be empty")
		suite.Assert().NotEmpty(tool.Description, "Tool description should not be empty")
		suite.Assert().NotNil(tool.InputSchema, "Tool input schema should not be nil")
	}
}
