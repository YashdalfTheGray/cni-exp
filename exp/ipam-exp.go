package main

import (
	"context"
	"fmt"
	"os"

	"github.com/containernetworking/cni/pkg/invoke"
)

const ipamNetConfJson = `
{
	"cniVersion": "0.3.0",
  "ipam": {
    "type": "ecs-ipam",
    "id": "12345",
    "ipv4-address": "192.168.1.43/24",
    "ipv4-gateway": "192.168.1.1",
    "ipv4-subnet": "192.168.1.0/24",
    "ipv4-routes": [
      { "dst": "169.254.170.2/32" },
      { "dst": "169.254.170.0/20" }
    ]
  }
}
`

func assertNoError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err)
		os.Exit(2)
	}
}

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

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	pathSepStr := string(os.PathSeparator)
	pluginsPath := fmt.Sprintf(
		"%s%splugins%s",
		cwd,
		pathSepStr,
		pathSepStr,
	)
	pluginPath, err := invoke.FindInPath("ecs-ipam", []string{pluginsPath})
	assertNoError(err, fmt.Sprintf("Could not find the ecs-ipam plugin in path %s", pluginsPath))

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
		[]byte(ipamNetConfJson),
		execInvokeArgs,
		nil,
	)
	assertNoError(err, "Something went wrong with invoking the plugin")
	fmt.Printf("%+v\n", result)
}
