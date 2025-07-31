#!/usr/bin/env python3
"""
A2A Client Example for Network Researcher

This example demonstrates how to connect to and interact with the Network Researcher
running as an A2A server.

Based on the python-a2a library: https://github.com/themanojdesai/python-a2a
"""

import asyncio
import json
from typing import Dict, Any

from python_a2a import A2AClient, Message, MessageRole, TextContent, FunctionCallContent


def main():
    """Main function demonstrating A2A client usage"""

    print("🔗 Network Researcher A2A Client Example")
    print("=" * 50)

    # Create client
    client = A2AClient(endpoint_url="http://localhost:8085/")

    try:
        print("\n📋 Getting agent capabilities...")
        capabilities = client.ask("What are your capabilities?")
        print(f"Capabilities:\n{capabilities}")

        print("\n🛠️ Getting available tools...")
        tools = client.ask("What tools do you have available?")
        print(f"Tools:\n{tools}")

        print("\n❓ Asking a specific question...")
        answer = client.ask("What bridges are currently configured in Open vSwitch?")
        print(f"A: {answer}")

    except Exception as e:
        print(f"❌ Error: {e}")


if __name__ == "__main__":
    main()
