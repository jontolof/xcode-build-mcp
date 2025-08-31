#!/usr/bin/env python3
"""
Simple test script for MCP server runtime tools.
Tests the Phase 5 runtime tools: list_simulators, simulator_control, install_app, launch_app
"""

import json
import subprocess
import sys
import time

def send_mcp_request(method, params=None):
    """Send an MCP request to the server and return the response."""
    request = {
        "jsonrpc": "2.0",
        "method": method,
        "id": 1
    }
    if params:
        request["params"] = params
    
    # Start the server process
    proc = subprocess.Popen(
        ["./xcode-build-mcp"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    try:
        # Send request
        request_json = json.dumps(request) + "\n"
        proc.stdin.write(request_json)
        proc.stdin.flush()
        
        # Read response
        response_line = proc.stdout.readline()
        if response_line:
            return json.loads(response_line.strip())
        else:
            stderr_output = proc.stderr.read()
            print(f"No response, stderr: {stderr_output}", file=sys.stderr)
            return None
    finally:
        proc.terminate()
        proc.wait()

def test_initialize():
    """Test MCP server initialization."""
    print("🔧 Testing MCP server initialization...")
    response = send_mcp_request("initialize", {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "test-client", "version": "1.0.0"}
    })
    
    if response and response.get("result"):
        print("✅ Server initialized successfully")
        return True
    else:
        print(f"❌ Server initialization failed: {response}")
        return False

def test_list_tools():
    """Test listing available tools."""
    print("🔧 Testing tools list...")
    response = send_mcp_request("tools/list")
    
    if response and response.get("result", {}).get("tools"):
        tools = response["result"]["tools"]
        tool_names = [tool["name"] for tool in tools]
        print(f"✅ Found {len(tools)} tools: {', '.join(tool_names)}")
        
        # Check for Phase 5 runtime tools
        phase5_tools = ["list_simulators", "simulator_control", "install_app", "launch_app"]
        missing_tools = [tool for tool in phase5_tools if tool not in tool_names]
        
        if not missing_tools:
            print("✅ All Phase 5 runtime tools are available")
            return True
        else:
            print(f"❌ Missing Phase 5 tools: {missing_tools}")
            return False
    else:
        print(f"❌ Failed to list tools: {response}")
        return False

def test_list_simulators():
    """Test the list_simulators tool."""
    print("🔧 Testing list_simulators tool...")
    response = send_mcp_request("tools/call", {
        "name": "list_simulators",
        "arguments": {}
    })
    
    if response and response.get("result"):
        result = response["result"]
        if isinstance(result.get("content"), list) and len(result["content"]) > 0:
            content = result["content"][0]
            if content.get("type") == "text":
                try:
                    simulators_data = json.loads(content["text"])
                    simulator_count = len(simulators_data.get("simulators", []))
                    print(f"✅ Found {simulator_count} simulators")
                    return True
                except json.JSONDecodeError:
                    print(f"❌ Invalid JSON in response: {content['text']}")
                    return False
            else:
                print(f"❌ Unexpected content type: {content.get('type')}")
                return False
        else:
            print(f"❌ Invalid response structure: {result}")
            return False
    else:
        print(f"❌ list_simulators failed: {response}")
        return False

def test_simulator_control():
    """Test the simulator_control tool (basic validation only)."""
    print("🔧 Testing simulator_control tool (validation only)...")
    # Test with invalid parameters to check error handling
    response = send_mcp_request("tools/call", {
        "name": "simulator_control",
        "arguments": {
            "action": "invalid_action",
            "simulator_id": "invalid_id"
        }
    })
    
    if response:
        if response.get("error"):
            print("✅ simulator_control correctly rejected invalid parameters")
            return True
        elif response.get("result"):
            print("❌ simulator_control should have rejected invalid parameters")
            return False
    
    print(f"❌ simulator_control test inconclusive: {response}")
    return False

def main():
    """Run all tests."""
    print("🚀 Starting Phase 5 Runtime Tools Testing\n")
    
    tests = [
        ("Initialize", test_initialize),
        ("List Tools", test_list_tools),
        ("List Simulators", test_list_simulators),
        ("Simulator Control", test_simulator_control),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        print(f"\n{'='*50}")
        print(f"Running: {test_name}")
        print(f"{'='*50}")
        
        try:
            if test_func():
                passed += 1
                print(f"✅ {test_name} PASSED")
            else:
                print(f"❌ {test_name} FAILED")
        except Exception as e:
            print(f"💥 {test_name} CRASHED: {e}")
        
        time.sleep(0.5)  # Brief pause between tests
    
    print(f"\n{'='*50}")
    print(f"RESULTS: {passed}/{total} tests passed")
    print(f"{'='*50}")
    
    return passed == total

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)