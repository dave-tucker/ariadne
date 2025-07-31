package ovnicnb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dave-tucker/ariadne/internal/mcp"
	"github.com/dave-tucker/ariadne/internal/schema/ovnicnb"
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

type ListTransitSwitchesArgs struct {
	NameFilter string `json:"name_filter" jsonschema:"the name of the transit switch to filter by"`
}

type ListICNBGlobalsArgs struct {
}

type ListConnectionsArgs struct {
}

type ListSSLConfigsArgs struct {
}

func (s *Server) ListTransitSwitches(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListTransitSwitchesArgs]) (*mcpsdk.CallToolResult, error) {
	args := params.Arguments

	nameFilter := args.NameFilter
	var conditions []model.Condition
	if nameFilter != "" {
		conditions = append(conditions, model.Condition{
			Field:    &(&ovnicnb.TransitSwitch{}).Name,
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicnb.TransitSwitch{}, conditions...)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"transit_switches": results,
		"count":            len(results),
		"context":          "Transit switches are logical switches that connect different availability zones in OVN Interconnection.",
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

func (s *Server) ListICNBGlobals(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListICNBGlobalsArgs]) (*mcpsdk.CallToolResult, error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicnb.ICNBGlobal{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"ic_nb_globals": results,
		"count":         len(results),
		"context":       "IC NB Globals contain global configuration settings for OVN Interconnection Northbound database.",
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

func (s *Server) ListConnections(ctx context.Context, ss *mcpsdk.ServerSession, params *mcpsdk.CallToolParamsFor[ListConnectionsArgs]) (*mcpsdk.CallToolResult, error) {
	client, err := client.NewOVSDBClient(s.dbModel, client.WithEndpoint(defaultEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	err = client.Connect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OVSDB: %w", err)
	}

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicnb.Connection{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"connections": results,
		"count":       len(results),
		"context":     "Connections define the network connections between different availability zones in OVN Interconnection.",
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

	results, err := mcp.ExecuteSelectQuery(ctx, client, ovnicnb.SSL{})
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{
		"ssl_configs": results,
		"count":       len(results),
		"context":     "SSL configurations define TLS settings for secure connections in OVN Interconnection.",
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

// NewServer creates a new OVN IC NB MCP server
func NewServer(host string, port int) (*Server, error) {

	// Create OVSDB client model using generated code
	dbModel, err := ovnicnb.FullDatabaseModel()
	if err != nil {
		return nil, fmt.Errorf("failed to create database model: %w", err)
	}

	server := mcpsdk.NewServer(&mcpsdk.Implementation{
		Name:    "ovn-ic-nb-mcp",
		Title:   "OVN IC NB MCP Server",
		Version: "0.1.0",
	}, nil)

	s := Server{
		Server:  server,
		dbModel: dbModel,
	}

	// Register tools inline
	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_transit_switches",
		Description: "List all transit switches in OVN IC NB database. Transit switches connect different availability zones.",
	}, s.ListTransitSwitches)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_ic_nb_globals",
		Description: "List all IC NB globals in OVN IC NB database. IC NB globals contain global configuration settings.",
	}, s.ListICNBGlobals)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_connections",
		Description: "List all connections in OVN IC NB database. Connections define network links between availability zones.",
	}, s.ListConnections)

	mcpsdk.AddTool(s.Server, &mcpsdk.Tool{
		Name:        "list_ssl_configs",
		Description: "List all SSL configurations in OVN IC NB database. SSL configs define TLS settings for secure connections.",
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
