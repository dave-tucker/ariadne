package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/dave-tucker/ariadne/internal/mcp/vswitch"
)

var (
	port    = flag.Int("port", 8080, "MCP server port")
	host    = flag.String("host", "localhost", "MCP server host")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Setup logging
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	logger.Info("Starting ovs-vswitch-mcp server",
		"host", *host,
		"port", *port)

	// Create server using the new package
	server, err := vswitch.NewServer(*host, *port)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Start the MCP server
	addr := fmt.Sprintf("%s:%d", *host, *port)
	if err := server.Start(context.Background(), addr); err != nil {
		logger.Error("Failed to start MCP server", "error", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down...")

	// Stop the server gracefully
	if err := server.Stop(context.Background()); err != nil {
		logger.Error("Error stopping MCP server", "error", err)
	}
}
