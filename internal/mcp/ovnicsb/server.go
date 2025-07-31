package ovnicsb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dave-tucker/ariadne/internal/mcp"
	"github.com/dave-tucker/ariadne/internal/schema/ovnicsb"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

const defaultEndpoint = "unix:/var/run/ovn/ovn_ic_nb_db.sock"

type Server struct {
	*mcpsdk.Server
	dbModel    model.ClientDBModel
	httpServer *http.Server
}

type ListAvailabilityZonesArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the availability zone to filter by"`
}

type ListDatapathBindingsArgs struct {
	ZoneFilter string `json:"zone_filter" jsonschema:"the name of the availability zone to filter by"`
}

type ListPortBindingsArgs struct {
	DatapathFilter string `json:"datapath_filter" jsonschema:"the name of the datapath to filter by"`
}

type ListGatewaysArgs struct {
	ZoneFilter string `json:"zone_filter" jsonschema:"the name of the availability zone to filter by"`
}

type ListRoutesArgs struct {
	GatewayFilter string `json:"gateway_filter" jsonschema:"the name of the gateway to filter by"`
}

type ListEncapsArgs struct {
	GatewayFilter string `json:"gateway_filter" jsonschema:"the name of the gateway to filter by"`
}

type ListICSBGlobalsArgs struct {
}

func (s *Server) ListAvailabilityZones(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListAvailabilityZonesArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnicsb.AvailabilityZone{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.AvailabilityZone{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"availability_zones": results,
		"count":              len(results),
		"context":            "Availability zones represent different geographical or logical regions in OVN Interconnection.",
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

func (s *Server) ListDatapathBindings(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListDatapathBindingsArgs]) (*mcpsdk.CallToolResult, error) {
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

	zoneFilter := args.ZoneFilter
	var conditions []model.Condition
	if zoneFilter != "" {
		// First, get the availability zone UUID
		var zones []ovnicsb.AvailabilityZone
		zoneCondition := model.Condition{
			Field:    &(&ovnicsb.AvailabilityZone{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    zoneFilter,
		}
		zoneSelectOps, zoneQueryID, zoneSelectErr := client.WhereAll(&ovnicsb.AvailabilityZone{}, zoneCondition).Select()
		if zoneSelectErr != nil {
			return nil, fmt.Errorf("failed to create availability zone select operation: %w", zoneSelectErr)
		}

		zoneReply, err := client.Transact(ctx, zoneSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute availability zone transaction: %w", err)
		}

		err = client.GetSelectResults(zoneSelectOps, zoneReply, map[string]interface{}{zoneQueryID: &zones})
		if err != nil {
			return nil, fmt.Errorf("failed to get availability zone select results: %w", err)
		}

		if len(zones) == 0 {
			result := map[string]interface{}{
				"datapath_bindings": []ovnicsb.DatapathBinding{},
				"count":             0,
				"context":           "No availability zone found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.DatapathBinding{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"datapath_bindings": results,
		"count":             len(results),
		"context":           "Datapath bindings represent the physical or virtual switches that implement transit switches in OVN Interconnection.",
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
		var datapaths []ovnicsb.DatapathBinding
		datapathCondition := model.Condition{
			Field:    &(&ovnicsb.DatapathBinding{}).ExternalIDs,
			Function: ovsdb.ConditionEqual,
			Value:    map[string]string{"name": datapathFilter},
		}
		datapathSelectOps, datapathQueryID, datapathSelectErr := client.WhereAll(&ovnicsb.DatapathBinding{}, datapathCondition).Select()
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
				"port_bindings": []ovnicsb.PortBinding{},
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.PortBinding{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"port_bindings": results,
		"count":         len(results),
		"context":       "Port bindings map logical ports to physical ports on datapaths in OVN Interconnection.",
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

func (s *Server) ListGateways(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListGatewaysArgs]) (*mcpsdk.CallToolResult, error) {
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

	zoneFilter := args.ZoneFilter
	var conditions []model.Condition
	if zoneFilter != "" {
		// First, get the availability zone UUID
		var zones []ovnicsb.AvailabilityZone
		zoneCondition := model.Condition{
			Field:    &(&ovnicsb.AvailabilityZone{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    zoneFilter,
		}
		zoneSelectOps, zoneQueryID, zoneSelectErr := client.WhereAll(&ovnicsb.AvailabilityZone{}, zoneCondition).Select()
		if zoneSelectErr != nil {
			return nil, fmt.Errorf("failed to create availability zone select operation: %w", zoneSelectErr)
		}

		zoneReply, err := client.Transact(ctx, zoneSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute availability zone transaction: %w", err)
		}

		err = client.GetSelectResults(zoneSelectOps, zoneReply, map[string]interface{}{zoneQueryID: &zones})
		if err != nil {
			return nil, fmt.Errorf("failed to get availability zone select results: %w", err)
		}

		if len(zones) == 0 {
			result := map[string]interface{}{
				"gateways": []ovnicsb.Gateway{},
				"count":    0,
				"context":  "No availability zone found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.Gateway{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"gateways": results,
		"count":    len(results),
		"context":  "Gateways provide routing and connectivity between availability zones in OVN Interconnection.",
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

func (s *Server) ListRoutes(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListRoutesArgs]) (*mcpsdk.CallToolResult, error) {
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

	gatewayFilter := args.GatewayFilter
	var conditions []model.Condition
	if gatewayFilter != "" {
		// First, get the gateway UUID
		var gateways []ovnicsb.Gateway
		gatewayCondition := model.Condition{
			Field:    &(&ovnicsb.Gateway{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    gatewayFilter,
		}
		gatewaySelectOps, gatewayQueryID, gatewaySelectErr := client.WhereAll(&ovnicsb.Gateway{}, gatewayCondition).Select()
		if gatewaySelectErr != nil {
			return nil, fmt.Errorf("failed to create gateway select operation: %w", gatewaySelectErr)
		}

		gatewayReply, err := client.Transact(ctx, gatewaySelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute gateway transaction: %w", err)
		}

		err = client.GetSelectResults(gatewaySelectOps, gatewayReply, map[string]interface{}{gatewayQueryID: &gateways})
		if err != nil {
			return nil, fmt.Errorf("failed to get gateway select results: %w", err)
		}

		if len(gateways) == 0 {
			result := map[string]interface{}{
				"routes":  []ovnicsb.Route{},
				"count":   0,
				"context": "No gateway found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.Route{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"routes":  results,
		"count":   len(results),
		"context": "Routes define the network paths between availability zones in OVN Interconnection.",
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

	gatewayFilter := args.GatewayFilter
	var conditions []model.Condition
	if gatewayFilter != "" {
		// First, get the gateway UUID
		var gateways []ovnicsb.Gateway
		gatewayCondition := model.Condition{
			Field:    &(&ovnicsb.Gateway{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    gatewayFilter,
		}
		gatewaySelectOps, gatewayQueryID, gatewaySelectErr := client.WhereAll(&ovnicsb.Gateway{}, gatewayCondition).Select()
		if gatewaySelectErr != nil {
			return nil, fmt.Errorf("failed to create gateway select operation: %w", gatewaySelectErr)
		}

		gatewayReply, err := client.Transact(ctx, gatewaySelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute gateway transaction: %w", err)
		}

		err = client.GetSelectResults(gatewaySelectOps, gatewayReply, map[string]interface{}{gatewayQueryID: &gateways})
		if err != nil {
			return nil, fmt.Errorf("failed to get gateway select results: %w", err)
		}

		if len(gateways) == 0 {
			result := map[string]interface{}{
				"encaps":  []ovnicsb.Encap{},
				"count":   0,
				"context": "No gateway found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.Encap{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"encaps":  results,
		"count":   len(results),
		"context": "Encapsulations define the tunneling protocols used to connect gateways in OVN Interconnection.",
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

func (s *Server) ListICSBGlobals(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListICSBGlobalsArgs]) (*mcpsdk.CallToolResult, error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicsb.ICSBGlobal{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"ic_sb_globals": results,
		"count":         len(results),
		"context":       "IC SB Globals contain global configuration settings for OVN Interconnection Southbound database.",
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

// NewServer creates a new OVN IC SB MCP server
func NewServer(host string, port int) (*Server, error) {

	// Create OVSDB client model using generated code
	dbModel, err := ovnicsb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("failed to create database model: %w", err)
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ovn-ic-sb-mcp",
		Title:   "OVN IC SB MCP Server",
		Version: "0.1.0",
	}, nil)

	s := Server{
		Server:  server,
		dbModel: dbModel,
	}

	// Register tools inline
	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_availability_zones",
		Description: "List all availability zones in OVN IC SB database. Availability zones represent different regions.",
	}, s.ListAvailabilityZones)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_datapath_bindings",
		Description: "List all datapath bindings in OVN IC SB database. Datapath bindings represent physical or virtual switches.",
	}, s.ListDatapathBindings)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_port_bindings",
		Description: "List all port bindings in OVN IC SB database. Port bindings map logical ports to physical ports.",
	}, s.ListPortBindings)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_gateways",
		Description: "List all gateways in OVN IC SB database. Gateways provide routing between availability zones.",
	}, s.ListGateways)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_routes",
		Description: "List all routes in OVN IC SB database. Routes define network paths between availability zones.",
	}, s.ListRoutes)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_encaps",
		Description: "List all encapsulations in OVN IC SB database. Encapsulations define tunneling protocols for gateways.",
	}, s.ListEncaps)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_ic_sb_globals",
		Description: "List all IC SB globals in OVN IC SB database. IC SB globals contain global configuration settings.",
	}, s.ListICSBGlobals)

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
