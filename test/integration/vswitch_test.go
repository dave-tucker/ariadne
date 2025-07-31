package integration

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/dave-tucker/ariadne/internal/mcp/vswitch"
	vswitchSchema "github.com/dave-tucker/ariadne/internal/schema/vswitch"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestVSwitchIntegration(t *testing.T) {
	suite.Run(t, new(VSwitchIntegrationTestSuite))
}

type VSwitchIntegrationTestSuite struct {
	suite.Suite
}

// TestvswitchServerTools tests that the OVS vSwitchd MCP server returns the correct list of tools
func (suite *VSwitchIntegrationTestSuite) TestToolsList() {
	// Create a new OVS vSwitchd server directly
	server, err := vswitch.NewServer("localhost", 8086)
	suite.Require().NoError(err, "Failed to create OVS vSwitchd server")

	// Start the server on a specific port
	ctx := context.Background()
	err = server.Start(ctx, "localhost:8086")
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
	transport := mcp.NewStreamableClientTransport("http://localhost:8086/", nil)

	// Connect to the MCP server
	session, err := mcpClient.Connect(ctx, transport)
	suite.Require().NoError(err, "Failed to connect to MCP server")
	defer session.Close()

	// List tools using the MCP client
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	suite.Require().NoError(err, "Failed to list tools")

	// Assert that tools are returned
	suite.Assert().NotEmpty(toolsResult.Tools, "Expected tools to be returned")

	// Define expected tools for OVS vSwitchd MCP server
	expectedTools := []string{
		"list_bridges",
		"list_ports",
		"list_interfaces",
		"list_managers",
		"list_controllers",
		"list_flow_tables",
		"list_ssl_configs",
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

// TestOVSTools tests OVS tools against a real OVS container
func (suite *VSwitchIntegrationTestSuite) TestListBridges() {
	ctx := context.Background()

	// Create a new OVS vSwitchd server directly
	server, err := vswitch.NewServer("localhost", 8086)
	suite.Require().NoError(err, "Failed to create OVS vSwitchd server")

	// Start the server on a specific port
	err = server.Start(ctx, "localhost:8086")
	suite.Require().NoError(err, "Failed to start server")
	defer server.Stop(ctx)

	// Give the server a moment to start
	time.Sleep(1 * time.Second)

	// Start a container using the libovsdb/ovs:3.5.0 image, exposing port TCP 6640
	req := testcontainers.ContainerRequest{
		Image:        "libovsdb/ovs:3.5.0",
		ExposedPorts: []string{"6640/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("6640/tcp"),
			wait.ForLog("ovsdb-server --remote=punix:/usr/local/var/run/openvswitch/db.sock --remote=ptcp:6640 --pidfile=ovsdb-server.pid"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	suite.Require().NoError(err, "Failed to start OVS container")
	defer container.Terminate(ctx)

	dbModel, err := vswitchSchema.FullDatabaseModel()
	suite.Require().NoError(err, "Failed to create database model")

	port, err := container.MappedPort(ctx, "6640/tcp")
	suite.Require().NoError(err, "Failed to get port")
	endpoint := fmt.Sprintf("tcp:127.0.0.1:%s", port.Port())
	suite.T().Logf("Endpoint: %s", endpoint)

	ovs, err := client.NewOVSDBClient(dbModel, client.WithEndpoint(endpoint))
	suite.Require().NoError(err, "Failed to create OVS client")
	err = ovs.Connect(ctx)
	suite.Require().NoError(err, "Failed to connect to OVS")
	defer ovs.Disconnect()

	selectOps, queryID, selectErr := ovs.Where(&vswitchSchema.OpenvSwitch{}).Select()
	suite.Require().NoError(selectErr, "Failed to select OpenvSwitch")

	reply, err := ovs.Transact(ctx, selectOps...)
	suite.Require().NoError(err, "Failed to execute transaction")

	var results []vswitchSchema.OpenvSwitch
	err = ovs.GetSelectResults(selectOps, reply, map[string]interface{}{queryID: &results})
	suite.Require().NoError(err, "Failed to get select results")
	suite.Assert().Equal(1, len(results), "Expected 1 OpenvSwitch to be returned")
	fmt.Println(results[0])
	rootUUID := results[0].UUID

	createBridge(ovs, rootUUID, "br-test-listbr1")
	createBridge(ovs, rootUUID, "br-test-listbr2")
	createBridge(ovs, rootUUID, "br-test-listbr3")

	// Create MCP client implementation
	impl := &mcp.Implementation{
		Name:    "ovsdb-mcp-test-client",
		Title:   "OVSDB MCP Test Client",
		Version: "1.0.0",
	}

	// Create MCP client
	mcpClient := mcp.NewClient(impl, nil)

	// Create Streamable HTTP transport to connect to the MCP server
	transport := mcp.NewStreamableClientTransport("http://localhost:8086/", nil)

	// Connect to the MCP server
	session, err := mcpClient.Connect(ctx, transport)
	suite.Require().NoError(err, "Failed to connect to MCP server")
	defer session.Close()

	// List bridges using the MCP client
	bridgesResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_bridges",
		Arguments: map[string]interface{}{},
	})
	suite.Require().NoError(err, "Failed to list bridges")

	for _, bridge := range bridgesResult.Content {
		suite.Require().NoError(err, "Failed to marshal bridge")
		suite.T().Logf("Bridge: %s", bridge)
	}

	// TODO: Add assertions
}

func createBridge(ovs client.Client, rootUUID string, bridgeName string) {
	bridge := vswitchSchema.Bridge{
		UUID: "gopher",
		Name: bridgeName,
	}
	insertOp, err := ovs.Create(&bridge)
	if err != nil {
		log.Fatal(err)
	}

	ovsRow := vswitchSchema.OpenvSwitch{
		UUID: rootUUID,
	}
	mutateOps, err := ovs.Where(&ovsRow).Mutate(&ovsRow, model.Mutation{
		Field:   &ovsRow.Bridges,
		Mutator: "insert",
		Value:   []string{bridge.UUID},
	})
	if err != nil {
		log.Fatal(err)
	}

	operations := append(insertOp, mutateOps...)
	reply, err := ovs.Transact(context.TODO(), operations...)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := ovsdb.CheckOperationResults(reply, operations); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Bridge Addition Successful : ", reply[0].UUID.GoUUID)
}
