package ovnnb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dave-tucker/ariadne/internal/mcp"
	"github.com/dave-tucker/ariadne/internal/schema/ovnnb"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovn-kubernetes/libovsdb/client"
	"github.com/ovn-kubernetes/libovsdb/model"
	"github.com/ovn-kubernetes/libovsdb/ovsdb"
)

const defaultEndpoint = "unix:/var/run/ovn/ovnnb_db.sock"

type Server struct {
	*mcpsdk.Server
	dbModel    model.ClientDBModel
	httpServer *http.Server
}

type ListLogicalSwitchesArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the logical switch to filter by"`
}

type ListLogicalSwitchPortsArgs struct {
	SwitchFilter string `json:"switch_filter" jsonschema:"the name of the logical switch to filter by"`
}

type ListLogicalRoutersArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the logical router to filter by"`
}

type ListACLsArgs struct {
	SwitchFilter string `json:"switch_filter" jsonschema:"the name of the logical switch to filter by"`
}

type ListLoadBalancersArgs struct {
	SwitchFilter string `json:"switch_filter" jsonschema:"the name of the logical switch to filter by"`
}

type ListNATRulesArgs struct {
	RouterFilter string `json:"router_filter" jsonschema:"the name of the logical router to filter by"`
}

type ListPortGroupsArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the port group to filter by"`
}

type ListAddressSetsArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the address set to filter by"`
}

type ListQoSRulesArgs struct {
	SwitchFilter string `json:"switch_filter" jsonschema:"the name of the logical switch to filter by"`
}

type ListMetersArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the meter to filter by"`
}

func (s *Server) ListLogicalSwitches(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListLogicalSwitchesArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnnb.LogicalSwitch{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.LogicalSwitch{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"logical_switches": results,
		"count":            len(results),
		"context":          "Logical switches are the primary networking entities in OVN that connect logical ports. They represent virtual Layer 2 networks.",
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

func (s *Server) ListLogicalSwitchPorts(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListLogicalSwitchPortsArgs]) (*mcpsdk.CallToolResult, error) {
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

	switchFilter := args.SwitchFilter
	var conditions []model.Condition
	if switchFilter != "" {
		// First, get the logical switch UUID
		var switches []ovnnb.LogicalSwitch
		switchCondition := model.Condition{
			Field:    &(&ovnnb.LogicalSwitch{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    switchFilter,
		}
		switchSelectOps, switchQueryID, switchSelectErr := client.WhereAll(&ovnnb.LogicalSwitch{}, switchCondition).Select()
		if switchSelectErr != nil {
			return nil, fmt.Errorf("failed to create logical switch select operation: %w", switchSelectErr)
		}

		switchReply, err := client.Transact(ctx, switchSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute logical switch transaction: %w", err)
		}

		err = client.GetSelectResults(switchSelectOps, switchReply, map[string]interface{}{switchQueryID: &switches})
		if err != nil {
			return nil, fmt.Errorf("failed to get logical switch select results: %w", err)
		}

		if len(switches) == 0 {
			result := map[string]interface{}{
				"logical_switch_ports": []ovnnb.LogicalSwitchPort{},
				"count":                0,
				"context":              "No logical switch found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.LogicalSwitchPort{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"logical_switch_ports": results,
		"count":                len(results),
		"context":              "Logical switch ports connect to logical switches and represent network endpoints. Each port belongs to a logical switch and can have various configuration options.",
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

func (s *Server) ListLogicalRouters(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListLogicalRoutersArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnnb.LogicalRouter{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.LogicalRouter{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"logical_routers": results,
		"count":           len(results),
		"context":         "Logical routers provide Layer 3 routing between logical switches. They handle routing decisions and can have multiple logical router ports.",
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

func (s *Server) ListACLs(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListACLsArgs]) (*mcpsdk.CallToolResult, error) {
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

	switchFilter := args.SwitchFilter
	var conditions []model.Condition
	if switchFilter != "" {
		// First, get the logical switch UUID
		var switches []ovnnb.LogicalSwitch
		switchCondition := model.Condition{
			Field:    &(&ovnnb.LogicalSwitch{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    switchFilter,
		}
		switchSelectOps, switchQueryID, switchSelectErr := client.WhereAll(&ovnnb.LogicalSwitch{}, switchCondition).Select()
		if switchSelectErr != nil {
			return nil, fmt.Errorf("failed to create logical switch select operation: %w", switchSelectErr)
		}

		switchReply, err := client.Transact(ctx, switchSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute logical switch transaction: %w", err)
		}

		err = client.GetSelectResults(switchSelectOps, switchReply, map[string]interface{}{switchQueryID: &switches})
		if err != nil {
			return nil, fmt.Errorf("failed to get logical switch select results: %w", err)
		}

		if len(switches) == 0 {
			result := map[string]interface{}{
				"acls":    []ovnnb.ACL{},
				"count":   0,
				"context": "No logical switch found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.ACL{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"acls":    results,
		"count":   len(results),
		"context": "ACLs (Access Control Lists) define security policies for logical switches. They control which traffic is allowed or denied based on various criteria.",
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

func (s *Server) ListLoadBalancers(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListLoadBalancersArgs]) (*mcpsdk.CallToolResult, error) {
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

	switchFilter := args.SwitchFilter
	var conditions []model.Condition
	if switchFilter != "" {
		// First, get the logical switch UUID
		var switches []ovnnb.LogicalSwitch
		switchCondition := model.Condition{
			Field:    &(&ovnnb.LogicalSwitch{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    switchFilter,
		}
		switchSelectOps, switchQueryID, switchSelectErr := client.WhereAll(&ovnnb.LogicalSwitch{}, switchCondition).Select()
		if switchSelectErr != nil {
			return nil, fmt.Errorf("failed to create logical switch select operation: %w", switchSelectErr)
		}

		switchReply, err := client.Transact(ctx, switchSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute logical switch transaction: %w", err)
		}

		err = client.GetSelectResults(switchSelectOps, switchReply, map[string]interface{}{switchQueryID: &switches})
		if err != nil {
			return nil, fmt.Errorf("failed to get logical switch select results: %w", err)
		}

		if len(switches) == 0 {
			result := map[string]interface{}{
				"load_balancers": []ovnnb.LoadBalancer{},
				"count":          0,
				"context":        "No logical switch found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.LoadBalancer{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"load_balancers": results,
		"count":          len(results),
		"context":        "Load balancers distribute incoming traffic across multiple backend servers. They provide high availability and scalability for services.",
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

func (s *Server) ListNATRules(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListNATRulesArgs]) (*mcpsdk.CallToolResult, error) {
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

	routerFilter := args.RouterFilter
	var conditions []model.Condition
	if routerFilter != "" {
		// First, get the logical router UUID
		var routers []ovnnb.LogicalRouter
		routerCondition := model.Condition{
			Field:    &(&ovnnb.LogicalRouter{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    routerFilter,
		}
		routerSelectOps, routerQueryID, routerSelectErr := client.WhereAll(&ovnnb.LogicalRouter{}, routerCondition).Select()
		if routerSelectErr != nil {
			return nil, fmt.Errorf("failed to create logical router select operation: %w", routerSelectErr)
		}

		routerReply, err := client.Transact(ctx, routerSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute logical router transaction: %w", err)
		}

		err = client.GetSelectResults(routerSelectOps, routerReply, map[string]interface{}{routerQueryID: &routers})
		if err != nil {
			return nil, fmt.Errorf("failed to get logical router select results: %w", err)
		}

		if len(routers) == 0 {
			result := map[string]interface{}{
				"nat_rules": []ovnnb.NAT{},
				"count":     0,
				"context":   "No logical router found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.NAT{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"nat_rules": results,
		"count":     len(results),
		"context":   "NAT (Network Address Translation) rules modify packet headers to change source or destination addresses. They are used for network address translation.",
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

func (s *Server) ListPortGroups(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListPortGroupsArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnnb.PortGroup{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.PortGroup{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"port_groups": results,
		"count":       len(results),
		"context":     "Port groups are collections of logical switch ports that can be referenced together for ACLs and other policies.",
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

func (s *Server) ListAddressSets(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListAddressSetsArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnnb.AddressSet{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.AddressSet{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"address_sets": results,
		"count":        len(results),
		"context":      "Address sets are collections of IP addresses that can be referenced together in ACLs and other policies.",
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

func (s *Server) ListQoSRules(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListQoSRulesArgs]) (*mcpsdk.CallToolResult, error) {
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

	switchFilter := args.SwitchFilter
	var conditions []model.Condition
	if switchFilter != "" {
		// First, get the logical switch UUID
		var switches []ovnnb.LogicalSwitch
		switchCondition := model.Condition{
			Field:    &(&ovnnb.LogicalSwitch{}).Name,
			Function: ovsdb.ConditionEqual,
			Value:    switchFilter,
		}
		switchSelectOps, switchQueryID, switchSelectErr := client.WhereAll(&ovnnb.LogicalSwitch{}, switchCondition).Select()
		if switchSelectErr != nil {
			return nil, fmt.Errorf("failed to create logical switch select operation: %w", switchSelectErr)
		}

		switchReply, err := client.Transact(ctx, switchSelectOps...)
		if err != nil {
			return nil, fmt.Errorf("failed to execute logical switch transaction: %w", err)
		}

		err = client.GetSelectResults(switchSelectOps, switchReply, map[string]interface{}{switchQueryID: &switches})
		if err != nil {
			return nil, fmt.Errorf("failed to get logical switch select results: %w", err)
		}

		if len(switches) == 0 {
			result := map[string]interface{}{
				"qos_rules": []ovnnb.QoS{},
				"count":     0,
				"context":   "No logical switch found with the specified filter.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.QoS{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"qos_rules": results,
		"count":     len(results),
		"context":   "QoS (Quality of Service) rules define bandwidth and traffic shaping policies for logical switch ports.",
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
			Field:    &(&ovnnb.Meter{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnnb.Meter{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"meters":  results,
		"count":   len(results),
		"context": "Meters provide rate limiting and policing capabilities for traffic flows. They can be used to enforce bandwidth limits.",
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

// NewServer creates a new OVN NB MCP server
func NewServer(host string, port int) (*Server, error) {

	// Create OVSDB client model using generated code
	dbModel, err := ovnnb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("failed to create database model: %w", err)
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ovn-nb-mcp",
		Title:   "OVN NB MCP Server",
		Version: "0.1.0",
	}, nil)

	s := Server{
		Server:  server,
		dbModel: dbModel,
	}

	// Register tools inline
	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_logical_switches",
		Description: "List all logical switches in OVN NB database. Logical switches are the primary networking entities that connect logical ports.",
	}, s.ListLogicalSwitches)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_logical_switch_ports",
		Description: "List all logical switch ports in OVN NB database. Logical switch ports connect to logical switches and represent network endpoints.",
	}, s.ListLogicalSwitchPorts)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_logical_routers",
		Description: "List all logical routers in OVN NB database. Logical routers provide Layer 3 routing between logical switches.",
	}, s.ListLogicalRouters)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_acls",
		Description: "List all ACLs in OVN NB database. ACLs define security policies for logical switches.",
	}, s.ListACLs)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_load_balancers",
		Description: "List all load balancers in OVN NB database. Load balancers distribute incoming traffic across multiple backend servers.",
	}, s.ListLoadBalancers)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_nat_rules",
		Description: "List all NAT rules in OVN NB database. NAT rules modify packet headers to change source or destination addresses.",
	}, s.ListNATRules)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_port_groups",
		Description: "List all port groups in OVN NB database. Port groups are collections of logical switch ports.",
	}, s.ListPortGroups)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_address_sets",
		Description: "List all address sets in OVN NB database. Address sets are collections of IP addresses.",
	}, s.ListAddressSets)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_qos_rules",
		Description: "List all QoS rules in OVN NB database. QoS rules define bandwidth and traffic shaping policies.",
	}, s.ListQoSRules)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_meters",
		Description: "List all meters in OVN NB database. Meters provide rate limiting and policing capabilities.",
	}, s.ListMeters)

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
