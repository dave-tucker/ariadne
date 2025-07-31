#!/usr/bin/env python3
"""
Network Researcher using LangChain

This tool gathers network information using MCP tools from:
- Open Virtual Networking (OVN)
- Open vSwitch (OVS)
- TODO: iptables

The researcher uses OpenAI-compatible models via a custom endpoint for inference.
The RESEARCHER_MODEL_ENDPOINT_URL environment variable specifies the OpenAI-compatible server endpoint.

This researcher is designed to be used in a multi-agent environment.
It supports A2A (Agent-to-Agent) communication.
"""

import argparse
import asyncio
import logging
import os
import time
from typing import List, Any, Dict, Optional
from http.server import HTTPServer, BaseHTTPRequestHandler
import threading

from langchain_openai import ChatOpenAI
from langchain_mcp_adapters.client import MultiServerMCPClient
from langgraph.prebuilt import create_react_agent
from langgraph.checkpoint.memory import InMemorySaver

# A2A imports
from python_a2a import run_server, AgentCard, AgentSkill
from python_a2a.langchain import to_a2a_server

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class HealthCheckHandler(BaseHTTPRequestHandler):
    """Simple health check handler for Kubernetes probes"""

    def do_GET(self):
        if self.path == "/health":
            self.send_response(200)
            self.send_header("Content-type", "text/plain")
            self.end_headers()
            self.wfile.write(b"OK")
        else:
            self.send_response(404)
            self.end_headers()

    def log_message(self, format, *args):
        # Suppress logging for health checks
        pass


def extract_response_content(response):
    """Extract just the content from a LangGraph agent response"""
    if "messages" in response and response["messages"]:
        last_message = response["messages"][-1]
        if "content" in last_message:
            return last_message["content"]
    return str(response)


# MCP Server Configuration - configurable via environment variables
MCP_SERVERS = {
    "ovs-vswitchd": {
        "name": "Open vSwitch Database",
        "url": os.getenv("MCP_OVS_VSWITCHD_URL", "http://localhost:8080"),
        "description": "Manages Open vSwitch bridges, ports, interfaces, flow tables, and controllers",
    },
    "ovn-nb": {
        "name": "OVN Northbound Database",
        "url": os.getenv("MCP_OVN_NB_URL", "http://localhost:8081"),
        "description": "Manages logical switches, routers, load balancers, ACLs, and DHCP options",
    },
    "ovn-sb": {
        "name": "OVN Southbound Database",
        "url": os.getenv("MCP_OVN_SB_URL", "http://localhost:8082"),
        "description": "Manages chassis, datapaths, logical flows, port bindings, and MAC bindings",
    },
}

IC_MCP_SERVERS = {
    "ovn-ic-nb": {
        "name": "OVN IC Northbound Database",
        "url": os.getenv("MCP_OVN_IC_NB_URL", "http://localhost:8083"),
        "description": "Manages interconnection global config, transit switches, and SSL configs",
    },
    "ovn-ic-sb": {
        "name": "OVN IC Southbound Database",
        "url": os.getenv("MCP_OVN_IC_SB_URL", "http://localhost:8084"),
        "description": "Manages availability zones, gateways, and interconnection routing",
    },
}


def _start_health_server():
    """Start the health check HTTP server"""
    try:
        # Start health server on a different port to avoid conflicts with A2A server
        health_port = int(os.getenv("HEALTH_PORT", "8086"))
        health_server = HTTPServer(("0.0.0.0", health_port), HealthCheckHandler)

        def run_health_server():
            health_server.serve_forever()

        health_thread = threading.Thread(target=run_health_server, daemon=True)
        health_thread.start()

        logger.info(f"Health check server started on port {health_port}")
    except Exception as e:
        logger.error(f"Failed to start health check server: {e}")


async def main():
    parser = argparse.ArgumentParser(
        description="Network Researcher - Interactive CLI or A2A Server",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Run as A2A server (default)
  uv run network-researcher
  
  # Run as A2A server with custom host/port
  uv run network-researcher --host 127.0.0.1 --port 9090
  
  # Run in interactive CLI mode
  uv run network-researcher --interactive
  
  # Run in interactive CLI mode (alternative)
  uv run network-researcher -i
        """,
    )

    parser.add_argument(
        "-i",
        "--interactive",
        action="store_true",
        help="Run in interactive CLI mode instead of A2A server mode",
    )

    parser.add_argument(
        "--host",
        default=os.getenv("A2A_HOST", "0.0.0.0"),
        help="Host to bind A2A server to (default: 0.0.0.0)",
    )

    parser.add_argument(
        "--port",
        type=int,
        default=int(os.getenv("A2A_PORT", "8085")),
        help="Port to bind A2A server to (default: 8085)",
    )

    parser.add_argument(
        "--ic",
        action="store_true",
        help="Use IC MCP servers",
    )

    args = parser.parse_args()

    """Run the researcher in interactive CLI mode"""
    print("🔍 Network Researcher with OpenAI-compatible model")
    print("=" * 60)
    print("This researcher uses an OpenAI-compatible server for inference!")
    print("=" * 60)

    # Create and initialize the researcher
    model_endpoint_url = os.getenv(
        "RESEARCHER_MODEL_ENDPOINT_URL", "http://localhost:11434"
    )
    if not model_endpoint_url:
        raise ValueError(
            "RESEARCHER_MODEL_ENDPOINT_URL environment variable must be set"
        )

    # Get model name from environment variable
    model_name = os.getenv("RESEARCHER_MODEL_NAME", "default")

    model = ChatOpenAI(
        base_url=model_endpoint_url,
        api_key="dummy",  # Not used for self-hosted models
        model_name=model_name,
        temperature=0.1,
        max_tokens=4096,
        verbose=True,
    )

    logger.info("Initializing Network Researcher with OpenAI-compatible model...")
    logger.info(f"Using model endpoint: {model_endpoint_url}")
    logger.info(f"Using model: {model_name}")

    # Create MCP client configuration
    mcp_config = {}
    for server_name, server_config in MCP_SERVERS.items():
        mcp_config[server_name] = {
            "transport": "streamable_http",
            "url": server_config["url"],
        }
        logger.info(f"MCP Server {server_name}: {server_config['url']}")

    if args.ic:
        for server_name, server_config in IC_MCP_SERVERS.items():
            mcp_config[server_name] = {
                "transport": "streamable_http",
                "url": server_config["url"],
            }
            logger.info(f"MCP Server {server_name}: {server_config['url']}")

    # Initialize MultiServerMCPClient
    mcp_client = MultiServerMCPClient(mcp_config)

    tools = await mcp_client.get_tools()
    logger.info(f"Successfully loaded {len(tools)} tools from MCP servers")

    # Log tool names for debugging
    tool_names = [tool.name for tool in tools]
    logger.info(f"Available tools: {tool_names}")

    # Create MCP client configuration
    mcp_config = {}
    for server_name, server_config in MCP_SERVERS.items():
        mcp_config[server_name] = {
            "transport": "streamable_http",
            "url": server_config["url"],
        }
        logger.info(f"MCP Server {server_name}: {server_config['url']}")

    # Initialize MultiServerMCPClient
    mcp_client = MultiServerMCPClient(mcp_config)

    # Load tools from all MCP servers
    try:
        tools = await mcp_client.get_tools()
        logger.info(f"Successfully loaded {len(tools)} tools from MCP servers")

        # Log tool names for debugging
        tool_names = [tool.name for tool in tools]
        logger.info(f"Available tools: {tool_names}")

    except Exception as e:
        logger.error(f"Failed to load tools from MCP servers: {e}")
        raise

    # System prompt for Network Researcher - improved to be more explicit about tool usage
    prompt = """You are a network researcher that MUST use tools to gather information from OVS/OVN databases.
You are running in a Kubernetes cluster where OVN-Kubernetes is installed and providing the networking for the cluster.

IMPORTANT: You MUST use the available tools to gather information. Do not make assumptions or provide generic answers.

When users ask questions:
1. ALWAYS identify which tools are needed to gather the requested information
2. USE those tools to collect data from the databases
3. Return the raw information gathered from the tools

For example:
- If asked about bridges, use the list_bridges tool
- If asked about logical switches, use the list_logical_switches tool
- If asked about ACLs, use the list_acls tool
- If asked about network topology, use multiple tools to gather comprehensive information

You are NOT responsible for:
- Making configuration changes
- Performing debugging or troubleshooting
- Executing commands to fix issues
- Providing expert analysis or recommendations

Your job is to gather and return the requested information using the available tools."""

    # Create the agent
    checkpointer = InMemorySaver()
    agent = create_react_agent(
        model=model, tools=tools, prompt=prompt, checkpointer=checkpointer
    )

    # Validate agent creation
    logger.info(f"Agent created successfully with {len(tools)} tools")
    logger.info(f"Agent type: {type(agent).__name__}")

    print("✅ Researcher initialized successfully!")
    print("I can gather information about Open Virtual Networking and Open vSwitch.")
    print("I can help you with:")
    print("- Network configuration data")
    print("- OVS/OVN database queries")
    print("- Network topology information")
    print("- Current network state")
    print(f"\nConnected to {len(MCP_SERVERS)} MCP servers with {len(tools)} tools")

    if args.interactive:
        print("\nType 'quit' to exit")
        print("-" * 60)

        config = {"configurable": {"thread_id": "1"}}
        # Interactive chat loop
        while True:
            try:
                user_input = input("\nYou: ").strip()

                if user_input.lower() in ["quit", "exit", "q"]:
                    print("Goodbye! 👋")
                    break
                elif user_input.lower() == "test":
                    print("\n🔍 Running the usage test...")
                    response = await agent.ainvoke(
                        {
                            "messages": [
                                {
                                    "role": "user",
                                    "content": "Please summarize the logical network topology in OVN",
                                }
                            ]
                        },
                        config,
                    )
                    for message in response["messages"]:
                        message.pretty_print()
                else:
                    print("\n🔍 Researching...")
                    response = await agent.ainvoke(
                        {"messages": [{"role": "user", "content": user_input}]}, config
                    )
                    for message in response["messages"]:
                        message.pretty_print()

            except KeyboardInterrupt:
                print("\n\nGoodbye! 👋")
                break
            except Exception as e:
                print(f"\nError: {e}")
    else:
        # Convert to A2A server using the proper method
        a2a_server = to_a2a_server(agent)

        a2a_server.agent_card = AgentCard(
            name="Network Researcher",
            description="I can gather information about Open Virtual Networking and Open vSwitch to answer questions about the network.",
            skills=[
                AgentSkill(
                    name="Research Network Information",
                    description="Uses tools to gather information about the network.",
                    examples=[
                        "What is the OVN network topology?",
                        "How many logical switches are there?",
                        "Are there any ACLs configured?",
                        "What Open vSwitch bridges are there?",
                        "What are the current flows in Open vSwitch?",
                    ],
                ),
            ],
            capabilities={"streaming": True, "memory": True},
            url=f"http://{args.host}:{args.port}",
        )

        # Start health check server
        _start_health_server()

        print(f"✅ A2A Server initialized successfully!")
        print(
            f"I can gather information about Open Virtual Networking and Open vSwitch."
        )
        print(f"Connected to {len(MCP_SERVERS)} MCP servers")
        print(
            f"Using OpenAI-compatible model at: {os.getenv('RESEARCHER_MODEL_ENDPOINT_URL')}"
        )
        print(f"A2A Server listening on {args.host}:{args.port}")
        print("\nReady to accept A2A connections!")
        print("-" * 50)

        # Start the A2A server
        run_server(a2a_server, host=args.host, port=args.port)


def main_sync():
    """Synchronous entry point for command-line usage"""
    asyncio.run(main())


if __name__ == "__main__":
    main_sync()
