package main

import (
	"context"
	"fmt"
	"os"

	"github.com/containernetworking/cni/pkg/invoke"
)

const ipamNetConfJson = `
{
	"cniVersion": "0.3.1",
	"name": "ipam-host-local",
	"ipam": {
		"type": "host-local",
		"ranges": [
			[{ "subnet": "10.0.0.224/28" }],
			[{ "subnet": "2600:1f14:70c:2d10:a458:0:0:0/80" }]
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
	pluginPath, err := invoke.FindInPath("host-local", []string{pluginsPath})
	assertNoError(err, fmt.Sprintf("Could not find the host-local plugin in path %s", pluginsPath))

	// setup args
	execInvokeArgs := &invoke.Args{
		ContainerID: "test-container-1",
		NetNS:       "/proc/19372/ns/net",
		IfName:      "dummy0",
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
