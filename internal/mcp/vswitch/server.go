package vswitch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dave-tucker/ariadne/internal/mcp"
	"github.com/dave-tucker/ariadne/internal/schema/vswitch"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/mapper"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

const defaultEndpoint = "unix:/var/run/openvswitch/db.sock"

type Server struct {
	*mcpsdk.Server
	dbModel    model.ClientDBModel
	httpServer *http.Server
}

type ListBridgesArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the bridge to filter by"`
}

type ListPortsArgs struct {
}

type ListInterfacesArgs struct {
	PortFilter string `json:"port_filter" jsonschema:"the name of the port to filter by"`
}

type ListManagersArgs struct {
}

type ListControllersArgs struct {
}

type ListFlowTablesArgs struct {
	BridgeFilter string `json:"bridge_filter" jsonschema:"the name of the bridge to filter by"`
}

type ListSSLConfigsArgs struct {
}

type ListResult struct {
	Data    map[string]any `json:"data"`
	Count   int            `json:"count"`
	Context string         `json:"context"`
}

func (s *Server) ListBridges(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListBridgesArgs]) (*mcpsdk.CallToolResultFor[ListResult], error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&vswitch.Bridge{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    nameFilter,
		})
	}

	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.Bridge{}, conditions...)
	if err != nil {
		return nil, err
	}

	m := mapper.NewMapper(vswitch.Schema())
	tableName := vswitch.BridgeTable
	tableSchema := vswitch.Schema().Table(tableName)

	var data []map[string]any

	for _, result := range results {
		info, err := mapper.NewInfo(tableName, tableSchema, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to create info: %w", err)
		}
		row, err := m.NewRow(info)
		if err != nil {
			return nil, fmt.Errorf("failed to create row: %w", err)
		}

		data = append(data, row)
	}

	var res mcpsdk.CallToolResultFor[ListResult]
	res.Content = []mcpsdk.Content{
		&mcpsdk.TextContent{
			Text: "success",
		},
	}
	res.StructuredContent = ListResult{
		Data:    map[string]any{"bridges": data},
		Count:   len(results),
		Context: "Bridges are the main configuration entities in Open vSwitch that contain ports and interfaces. Each bridge represents a virtual switch that can have multiple ports.",
	}

	return &res, nil
}

func (s *Server) ListPorts(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListPortsArgs]) (*mcpsdk.CallToolResultFor[map[string]any], error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.Port{})
	if err != nil {
		return nil, err
	}

	var data []map[string]any

	m := mapper.NewMapper(vswitch.Schema())
	tableName := vswitch.PortTable
	tableSchema := vswitch.Schema().Table(tableName)

	for _, result := range results {
		info, err := mapper.NewInfo(tableName, tableSchema, &result)
		if err != nil {
			return nil, fmt.Errorf("failed to create info: %w", err)
		}
		row, err := m.NewRow(info)
		if err != nil {
			return nil, fmt.Errorf("failed to create row: %w", err)
		}

		data = append(data, row)
	}

	var res mcpsdk.CallToolResultFor[map[string]any]
	res.Content = []mcpsdk.Content{
		&mcpsdk.TextContent{
			Text: "success",
		},
	}
	res.StructuredContent = map[string]any{
		"ports":   data,
		"count":   len(results),
		"context": "Ports are logical entities that group interfaces together within a bridge. Each port can have multiple interfaces and belongs to a specific bridge.",
	}
	return &res, nil
}

func (s *Server) ListInterfaces(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListInterfacesArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	portFilter := args.PortFilter
	var conditions []model.Condition
	if portFilter != "" {
		// First, get the port UUID
		var ports []vswitch.Port
		portCondition := model.Condition{
			Field:    &(&vswitch.Port{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    portFilter,
		}
		portSelectOps, portQueryID, portSelectErr := client.WhereAll(&vswitch.Port{}, portCondition).Select()
		if portSelectErr != nil {
			return nil, fmt.Errorf("failed to create port select operation: %w", portSelectErr)
		}

		portReply, err := client.Transact(ctx, portSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute port transaction: %w", err)
		}

		err = client.GetSelectResults(portSelectOps, portReply, map[string]interface{}{portQueryID: &ports})
		if err != nil {
			return nil, fmt.Errorf("failed to get port select results: %w", err)
		}

		if len(ports) == 0 {
			result := map[string]interface{}{
				"interfaces": []vswitch.Interface{},
				"count":      0,
				"context":    "No port found with the specified filter.",
			}
			json, err := json.Marshal(result)
			if err != nil {
				return nil, err
			}
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					&mcpsdk.TextContent{
						Text: string(json),
					},
				},
			}, nil
		}
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.Interface{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"interfaces": results,
		"count":      len(results),
		"context":    "Interfaces represent the actual network connections and can be physical or virtual. Each interface belongs to a port and can have various configuration options.",
	}

	json, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: string(json),
			},
		},
	}, nil
}

func (s *Server) ListManagers(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListManagersArgs]) (*mcpsdk.CallToolResult, error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.Manager{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"managers": results,
		"count":    len(results),
		"context":  "Managers define connections to OpenFlow controllers. Each manager specifies how Open vSwitch connects to external OpenFlow controllers for network control.",
	}

	json, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: string(json),
			},
		},
	}, nil
}

func (s *Server) ListControllers(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListControllersArgs]) (*mcpsdk.CallToolResult, error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.Controller{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"controllers": results,
		"count":       len(results),
		"context":     "Controllers define connections to OpenFlow controllers. Each controller specifies how Open vSwitch connects to external OpenFlow controllers for network control.",
	}

	json, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: string(json),
			},
		},
	}, nil
}

func (s *Server) ListFlowTables(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListFlowTablesArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	bridgeFilter := args.BridgeFilter
	var conditions []model.Condition
	if bridgeFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&vswitch.FlowTable{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"bridge": bridgeFilter},
		})
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.FlowTable{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"flow_tables": results,
		"count":       len(results),
		"context":     "Flow tables contain the forwarding rules for network traffic. Each flow table belongs to a bridge and contains multiple flow entries that define how packets should be processed.",
	}

	json, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: string(json),
			},
		},
	}, nil
}

func (s *Server) ListSSLConfigs(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListSSLConfigsArgs]) (*mcpsdk.CallToolResult, error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, vswitch.SSL{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"ssl_configs": results,
		"count":       len(results),
		"context":     "SSL configurations define TLS settings for secure connections. These configurations are used for secure communication with OpenFlow controllers and other external services.",
	}

	json, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: string(json),
			},
		},
	}, nil
}

// NewServer creates a new OVS vSwitchd MCP server instance
func NewServer(host string, port int) (*Server, error) {

	// Create OVSDB client model using generated code
	dbModel, err := vswitch.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("failed to create database model: %w", err)
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ovs-vswitch-mcp",
		Title:   "OVS vSwitch MCP Server",
		Version: "0.1.0",
	}, nil)

	s := Server{
		Server:  server,
		dbModel: dbModel,
	}

	// Register tools inline
	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_bridges",
		Description: "List all Open vSwitch bridges. Bridges are the main configuration entities in Open vSwitch that contain ports and interfaces.",
	}, s.ListBridges)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_ports",
		Description: "List all ports in Open vSwitch bridges. Ports are logical entities that group interfaces together within a bridge.",
	}, s.ListPorts)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_interfaces",
		Description: "List all interfaces in Open vSwitch. Interfaces represent the actual network connections and can be physical or virtual.",
	}, s.ListInterfaces)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_managers",
		Description: "List all OpenFlow managers in Open vSwitch. Managers define connections to OpenFlow controllers.",
	}, s.ListManagers)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_controllers",
		Description: "List all OpenFlow controllers in Open vSwitch. Controllers define connections to OpenFlow controllers.",
	}, s.ListControllers)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_flow_tables",
		Description: "List all flow tables in Open vSwitch. Flow tables contain the forwarding rules for network traffic.",
	}, s.ListFlowTables)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_ssl_configs",
		Description: "List all SSL configurations in Open vSwitch. SSL configurations define TLS settings for secure connections.",
	}, s.ListSSLConfigs)

	return &s, nil
}

// Start starts the MCP server on the specified address
func (s *Server) Start(ctx context.Context, addr string) error {
	// Create HTTP server using Streamable HTTP handler
	streamableHandler := mcpsdk.NewStreamableHTTPHandler(func(request *http.Request) *mcpsdk.Server {
		return s.Server
	}, nil)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: streamableHandler,
	}

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Log error if we had a logger
		}
	}()

	return nil
}

// Stop stops the MCP server
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
