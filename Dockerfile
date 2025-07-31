# Build stage - builds all binaries
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS builder

RUN microdnf install -y git go-toolset

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN git config --global --add safe.directory /app

# Copy source code
COPY . .

# Build all binaries
RUN make build

# OVS vSwitchd MCP Server Runtime Stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS ovs-vswitch-mcp

# Install runtime dependencies
RUN microdnf install -y ca-certificates tzdata wget && microdnf clean all

# Create non-root user
RUN groupadd -g 1001 ovsdb && \
    useradd -u 1001 -g ovsdb -s /bin/bash -m ovsdb

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/ovs-vswitch-mcp .

# Change ownership to non-root user
RUN chown ovsdb:ovsdb /app/ovs-vswitch-mcp

# Switch to non-root user
USER ovsdb

# Expose default port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./ovs-vswitch-mcp"]
CMD ["-host", "0.0.0.0", "-port", "8080"]

# OVN Northbound MCP Server Runtime Stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS ovn-nbdb-mcp

# Install runtime dependencies
RUN microdnf install -y ca-certificates tzdata wget && microdnf clean all

# Create non-root user
RUN groupadd -g 1001 ovsdb && \
    useradd -u 1001 -g ovsdb -s /bin/bash -m ovsdb

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/ovn-nbdb-mcp .

# Change ownership to non-root user
RUN chown ovsdb:ovsdb /app/ovn-nbdb-mcp

# Switch to non-root user
USER ovsdb

# Expose default port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/health || exit 1

ENTRYPOINT ["./ovn-nbdb-mcp"]
CMD ["-host", "0.0.0.0", "-port", "8081"]

# OVN Southbound MCP Server Runtime Stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS ovn-sbdb-mcp

# Install runtime dependencies
RUN microdnf install -y ca-certificates tzdata wget && microdnf clean all

# Create non-root user
RUN groupadd -g 1001 ovsdb && \
    useradd -u 1001 -g ovsdb -s /bin/bash -m ovsdb

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/ovn-sbdb-mcp .

# Change ownership to non-root user
RUN chown ovsdb:ovsdb /app/ovn-sbdb-mcp

# Switch to non-root user
USER ovsdb

# Expose default port
EXPOSE 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8082/health || exit 1

ENTRYPOINT ["./ovn-sbdb-mcp"]
CMD ["-host", "0.0.0.0", "-port", "8082"]

# OVN IC Northbound MCP Server Runtime Stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS ovn-ic-nbdb-mcp

# Install runtime dependencies
RUN microdnf install -y ca-certificates tzdata wget && microdnf clean all

# Create non-root user
RUN groupadd -g 1001 ovsdb && \
    useradd -u 1001 -g ovsdb -s /bin/bash -m ovsdb

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/ovn-ic-nbdb-mcp .

# Change ownership to non-root user
RUN chown ovsdb:ovsdb /app/ovn-ic-nbdb-mcp

# Switch to non-root user
USER ovsdb

# Expose default port
EXPOSE 8083

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

ENTRYPOINT ["./ovn-ic-nbdb-mcp"]
CMD ["-host", "0.0.0.0", "-port", "8083"]

# OVN IC Southbound MCP Server Runtime Stage
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS ovn-ic-sbdb-mcp

# Install runtime dependencies
RUN microdnf install -y ca-certificates tzdata wget && microdnf clean all

# Create non-root user
RUN groupadd -g 1001 ovsdb && \
    useradd -u 1001 -g ovsdb -s /bin/bash -m ovsdb

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/ovn-ic-sbdb-mcp .

# Change ownership to non-root user
RUN chown ovsdb:ovsdb /app/ovn-ic-sbdb-mcp

# Switch to non-root user
USER ovsdb

# Expose default port
EXPOSE 8084

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8084/health || exit 1

ENTRYPOINT ["./ovn-ic-sbdb-mcp"]
CMD ["-host", "0.0.0.0", "-port", "8084"]

# Python Agent Runtime Stage
FROM ghcr.io/astral-sh/uv:bookworm-slim AS network-researcher

RUN apt-get update && apt-get install -y g++ gcc && apt-get clean && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy agent files
COPY . .

# Install Python dependencies using uv
RUN uv sync --frozen

# Expose ports for A2A server and health check
EXPOSE 8085 8086

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8086/health || exit 1

# Environment variables for configuration
ENV MCP_OVS_VSWITCHD_URL="http://localhost:8080"
ENV MCP_OVN_NB_URL="http://localhost:8081"
ENV MCP_OVN_SB_URL="http://localhost:8082"
ENV MCP_OVN_IC_NB_URL="http://localhost:8083"
ENV MCP_OVN_IC_SB_URL="http://localhost:8084"
ENV A2A_HOST="0.0.0.0"
ENV A2A_PORT="8085"
ENV HEALTH_PORT="8086"

# Run the agent (default to interactive mode, can be overridden)
ENTRYPOINT ["uv", "run", "network-researcher"] 
