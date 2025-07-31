package ovnsb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dave-tucker/ariadne/internal/mcp"
	"github.com/dave-tucker/ariadne/internal/schema/ovnsb"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

const defaultEndpoint = "unix:/var/run/ovn/ovnsb_db.sock"

type Server struct {
	*mcpsdk.Server
	dbModel    model.ClientDBModel
	httpServer *http.Server
}

type ListDatapathBindingsArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the datapath to filter by"`
}

type ListPortBindingsArgs struct {
	DatapathFilter string `json:"datapath_filter" jsonschema:"the name of the datapath to filter by"`
}

type ListChassisArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the chassis to filter by"`
}

type ListLogicalFlowsArgs struct {
	DatapathFilter string `json:"datapath_filter" jsonschema:"the name of the datapath to filter by"`
}

type ListMACBindingsArgs struct {
	DatapathFilter string `json:"datapath_filter" jsonschema:"the name of the datapath to filter by"`
}

type ListEncapsArgs struct {
	ChassisFilter string `json:"chassis_filter" jsonschema:"the name of the chassis to filter by"`
}

type ListMetersArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the meter to filter by"`
}

type ListFDBEntriesArgs struct {
	DatapathFilter string `json:"datapath_filter" jsonschema:"the name of the datapath to filter by"`
}

func (s *Server) ListDatapathBindings(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListDatapathBindingsArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnsb.DatapathBinding{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"name": nameFilter},
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.DatapathBinding{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"datapath_bindings": results,
		"count":             len(results),
		"context":           "Datapath bindings represent the physical or virtual switches that implement logical switches and routers.",
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

func (s *Server) ListPortBindings(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListPortBindingsArgs]) (*mcpsdk.CallToolResult, error) {
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

	datapathFilter := args.DatapathFilter
	var conditions []model.Condition
	if datapathFilter != "" {
		// First, get the datapath UUID
		var datapaths []ovnsb.DatapathBinding
		datapathCondition := model.Condition{
			Field:    &(&ovnsb.DatapathBinding{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"name": datapathFilter},
		}
		datapathSelectOps, datapathQueryID, datapathSelectErr := client.WhereAll(&ovnsb.DatapathBinding{}, datapathCondition).Select()
		if datapathSelectErr != nil {
			return nil, fmt.Errorf("failed to create datapath select operation: %w", datapathSelectErr)
		}

		datapathReply, err := client.Transact(ctx, datapathSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute datapath transaction: %w", err)
		}

		err = client.GetSelectResults(datapathSelectOps, datapathReply, map[string]interface{}{datapathQueryID: &datapaths})
		if err != nil {
			return nil, fmt.Errorf("failed to get datapath select results: %w", err)
		}

		if len(datapaths) == 0 {
			result := map[string]interface{}{
				"port_bindings": []ovnsb.PortBinding{},
				"count":         0,
				"context":       "No datapath found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.PortBinding{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"port_bindings": results,
		"count":         len(results),
		"context":       "Port bindings map logical ports to physical ports on datapaths. They represent the actual network connections.",
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

func (s *Server) ListChassis(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListChassisArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnsb.Chassis{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.Chassis{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"chassis": results,
		"count":   len(results),
		"context": "Chassis represent physical or virtual machines that host OVN components and can run datapaths.",
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

func (s *Server) ListLogicalFlows(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListLogicalFlowsArgs]) (*mcpsdk.CallToolResult, error) {
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

	datapathFilter := args.DatapathFilter
	var conditions []model.Condition
	if datapathFilter != "" {
		// First, get the datapath UUID
		var datapaths []ovnsb.DatapathBinding
		datapathCondition := model.Condition{
			Field:    &(&ovnsb.DatapathBinding{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"name": datapathFilter},
		}
		datapathSelectOps, datapathQueryID, datapathSelectErr := client.WhereAll(&ovnsb.DatapathBinding{}, datapathCondition).Select()
		if datapathSelectErr != nil {
			return nil, fmt.Errorf("failed to create datapath select operation: %w", datapathSelectErr)
		}

		datapathReply, err := client.Transact(ctx, datapathSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute datapath transaction: %w", err)
		}

		err = client.GetSelectResults(datapathSelectOps, datapathReply, map[string]interface{}{datapathQueryID: &datapaths})
		if err != nil {
			return nil, fmt.Errorf("failed to get datapath select results: %w", err)
		}

		if len(datapaths) == 0 {
			result := map[string]interface{}{
				"logical_flows": []ovnsb.LogicalFlow{},
				"count":         0,
				"context":       "No datapath found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.LogicalFlow{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"logical_flows": results,
		"count":         len(results),
		"context":       "Logical flows represent the forwarding rules that are translated into OpenFlow flows on datapaths.",
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

func (s *Server) ListMACBindings(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListMACBindingsArgs]) (*mcpsdk.CallToolResult, error) {
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

	datapathFilter := args.DatapathFilter
	var conditions []model.Condition
	if datapathFilter != "" {
		// First, get the datapath UUID
		var datapaths []ovnsb.DatapathBinding
		datapathCondition := model.Condition{
			Field:    &(&ovnsb.DatapathBinding{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"name": datapathFilter},
		}
		datapathSelectOps, datapathQueryID, datapathSelectErr := client.WhereAll(&ovnsb.DatapathBinding{}, datapathCondition).Select()
		if datapathSelectErr != nil {
			return nil, fmt.Errorf("failed to create datapath select operation: %w", datapathSelectErr)
		}

		datapathReply, err := client.Transact(ctx, datapathSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute datapath transaction: %w", err)
		}

		err = client.GetSelectResults(datapathSelectOps, datapathReply, map[string]interface{}{datapathQueryID: &datapaths})
		if err != nil {
			return nil, fmt.Errorf("failed to get datapath select results: %w", err)
		}

		if len(datapaths) == 0 {
			result := map[string]interface{}{
				"mac_bindings": []ovnsb.MACBinding{},
				"count":        0,
				"context":      "No datapath found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.MACBinding{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"mac_bindings": results,
		"count":        len(results),
		"context":      "MAC bindings map MAC addresses to logical ports and IP addresses. They are used for ARP resolution.",
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

func (s *Server) ListEncaps(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListEncapsArgs]) (*mcpsdk.CallToolResult, error) {
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

	chassisFilter := args.ChassisFilter
	var conditions []model.Condition
	if chassisFilter != "" {
		// First, get the chassis UUID
		var chassis []ovnsb.Chassis
		chassisCondition := model.Condition{
			Field:    &(&ovnsb.Chassis{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    chassisFilter,
		}
		chassisSelectOps, chassisQueryID, chassisSelectErr := client.WhereAll(&ovnsb.Chassis{}, chassisCondition).Select()
		if chassisSelectErr != nil {
			return nil, fmt.Errorf("failed to create chassis select operation: %w", chassisSelectErr)
		}

		chassisReply, err := client.Transact(ctx, chassisSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute chassis transaction: %w", err)
		}

		err = client.GetSelectResults(chassisSelectOps, chassisReply, map[string]interface{}{chassisQueryID: &chassis})
		if err != nil {
			return nil, fmt.Errorf("failed to get chassis select results: %w", err)
		}

		if len(chassis) == 0 {
			result := map[string]interface{}{
				"encaps":  []ovnsb.Encap{},
				"count":   0,
				"context": "No chassis found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.Encap{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"encaps":  results,
		"count":   len(results),
		"context": "Encapsulations define the tunneling protocols used to connect chassis in an OVN deployment.",
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

func (s *Server) ListMeters(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListMetersArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnsb.Meter{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.Meter{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"meters":  results,
		"count":   len(results),
		"context": "Meters provide rate limiting and policing capabilities for traffic flows on datapaths.",
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

func (s *Server) ListFDBEntries(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListFDBEntriesArgs]) (*mcpsdk.CallToolResult, error) {
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

	datapathFilter := args.DatapathFilter
	var conditions []model.Condition
	if datapathFilter != "" {
		// First, get the datapath UUID
		var datapaths []ovnsb.DatapathBinding
		datapathCondition := model.Condition{
			Field:    &(&ovnsb.DatapathBinding{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"name": datapathFilter},
		}
		datapathSelectOps, datapathQueryID, datapathSelectErr := client.WhereAll(&ovnsb.DatapathBinding{}, datapathCondition).Select()
		if datapathSelectErr != nil {
			return nil, fmt.Errorf("failed to create datapath select operation: %w", datapathSelectErr)
		}

		datapathReply, err := client.Transact(ctx, datapathSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute datapath transaction: %w", err)
		}

		err = client.GetSelectResults(datapathSelectOps, datapathReply, map[string]interface{}{datapathQueryID: &datapaths})
		if err != nil {
			return nil, fmt.Errorf("failed to get datapath select results: %w", err)
		}

		if len(datapaths) == 0 {
			result := map[string]interface{}{
				"fdb_entries": []ovnsb.FDB{},
				"count":       0,
				"context":     "No datapath found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnsb.FDB{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"fdb_entries": results,
		"count":       len(results),
		"context":     "FDB (Forwarding Database) entries map MAC addresses to ports on datapaths for Layer 2 forwarding.",
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

// NewServer creates a new OVN SB MCP server
func NewServer(host string, port int) (*Server, error) {

	// Create OVSDB client model using generated code
	dbModel, err := ovnsb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("failed to create database model: %w", err)
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ovn-sb-mcp",
		Title:   "OVN SB MCP Server",
		Version: "0.1.0",
	}, nil)

	s := Server{
		Server:  server,
		dbModel: dbModel,
	}

	// Register tools inline
	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_datapath_bindings",
		Description: "List all datapath bindings in OVN SB database. Datapath bindings represent physical or virtual switches.",
	}, s.ListDatapathBindings)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_port_bindings",
		Description: "List all port bindings in OVN SB database. Port bindings map logical ports to physical ports.",
	}, s.ListPortBindings)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_chassis",
		Description: "List all chassis in OVN SB database. Chassis represent physical or virtual machines that host OVN components.",
	}, s.ListChassis)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_logical_flows",
		Description: "List all logical flows in OVN SB database. Logical flows represent forwarding rules translated to OpenFlow flows.",
	}, s.ListLogicalFlows)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_mac_bindings",
		Description: "List all MAC bindings in OVN SB database. MAC bindings map MAC addresses to logical ports and IP addresses.",
	}, s.ListMACBindings)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_encaps",
		Description: "List all encapsulations in OVN SB database. Encapsulations define tunneling protocols for chassis connections.",
	}, s.ListEncaps)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_meters",
		Description: "List all meters in OVN SB database. Meters provide rate limiting and policing capabilities.",
	}, s.ListMeters)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_fdb_entries",
		Description: "List all FDB entries in OVN SB database. FDB entries map MAC addresses to ports for Layer 2 forwarding.",
	}, s.ListFDBEntries)

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
