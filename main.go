package main

import (
	"context"
	"fmt"
	"os"

	"github.com/containernetworking/cni/pkg/invoke"
)

const vpcBridgeNetConfJson = `
{
  "cniVersion": "0.3.1",
  "name": "vpc",
  "type": "vpc-bridge",
  "eniName": "eth10",
  "eniMACAddress": "12:34:56:78:9a:bc",
  "eniIPAddresses": ["192.168.1.42/24"],
  "vpcCIDRs": ["192.168.0.0/16"],
  "bridgeNetNSPath": "",
  "ipAddresses": ["192.168.1.43/24"],
  "gatewayIPAddress": "192.168.1.1",
	"bridgeType": "L3"
}
`

const ipamNetConfJson = `
{
  "ipam": {
    "type": "ecs-ipam",
    "id": "12345",
    "ipv4-address": "10.0.0.2/24",
    "ipv4-gateway": "10.0.0.1",
    "ipv4-subnet": "10.0.0.0/24",
    "ipv4-routes": [
      { "dst": "169.254.170.2/32" },
      { "dst": "169.254.170.0/20", "gw": "10.0.0.1" }
    ]
  }
}
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Need to pass in an action, either add or delete")
		os.Exit(1)
	}

	// parse action
	givenAction := os.Args[1]
	cniCommand := ""
	if givenAction == "add" {
		cniCommand = "ADD"
	} else if givenAction == "delete" {
		cniCommand = "DEL"
	} else {
		fmt.Println("Couldn't map given action to CNI command")
		os.Exit(1)
	}
	fmt.Printf("Was asked to run the following command - %s\n", cniCommand)

	// parse plugin directory
	cwd, err := os.Getwd()
	assertNoError(err, "Could not get current working directory")
	pathSepStr := string(os.PathSeparator)
	pluginsPath := fmt.Sprintf(
		"%s%splugins%s",
		cwd,
		pathSepStr,
		pathSepStr,
	)
	pluginPath, err := invoke.FindInPath("vpc-bridge", []string{pluginsPath})
	assertNoError(err, fmt.Sprintf("Could not find the vpc-bridge plugin in path %s", pluginsPath))

	// setup proper logging
	pluginLogsDir, err := os.MkdirTemp("", "vpc-bridge-exp-")
	assertNoError(err, "Unable to create directory for logs")
	err = os.Chmod(pluginLogsDir, 0755)
	assertNoError(err, "Unable to set permissions for logs directory")
	os.Setenv("VPC_CNI_LOG_FILE", fmt.Sprintf("%s/vpc-bridge.log", pluginLogsDir))
	defer os.Unsetenv("VPC_CNI_LOG_FILE")
	os.Setenv("VPC_CNI_LOG_LEVEL", "debug")
	defer os.Unsetenv("VPC_CNI_LOG_LEVEL")

	// setup args
	execInvokeArgs := &invoke.Args{
		ContainerID: "test-container",
		NetNS:       "/var/run/netns/blue",
		IfName:      "blueveth",
		Path:        pluginsPath,
		Command:     cniCommand,
	}

	// execute plugin
	result, err := invoke.ExecPluginWithResult(
		context.Background(),
		pluginPath,
		[]byte(vpcBridgeNetConfJson),
		execInvokeArgs,
		nil,
	)
	assertNoError(err, "Something went wrong with invoking the plugin")
	fmt.Printf("%+v\n", result)
}

func assertNoError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err)
		os.Exit(2)
	}
}
