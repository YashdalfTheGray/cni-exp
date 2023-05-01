package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/containernetworking/cni/pkg/invoke"
)

// ipv6 only works when this executable is copied within the agent container
// const vpcBridgeNetConfJson = `
// {
//   "cniVersion": "0.3.1",
//   "name": "vpc",
//   "type": "vpc-bridge",
//   "eniName": "eth1",
//   "eniMACAddress": "12:34:56:78:9a:bc",
//   "eniIPAddresses": ["192.168.1.42/24", "fd03:1f14:070c:2d10:a458::18ba/80"],
//   "vpcCIDRs": ["192.168.0.0/24", "fd03:1f14:70c:2d10:a458::/80"],
//   "ipAddresses": ["192.168.1.65/28", "fd03:1f14:70c:2d10:a458:a0:0:43/100"],
//   "gatewayIPAddress": "192.168.1.1",
// 	"bridgeType": "L3"
// }
// `

const vpcBridgeNetConfJson = `
{
	"cniVersion": "0.3.1",
	"name": "vpc",
	"type": "vpc-bridge",
	"eniName": "eth4",
	"eniMACAddress": "02:d2:21:c1:9a:67",
	"eniIPAddresses": ["10.0.0.16/24"],
	"vpcCIDRs": ["10.0.0.0/24"],
	"ipAddresses": ["10.0.0.161/28"],
	"gatewayIPAddress": "10.0.0.1",
	"bridgeType": "L3"
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

	// start docker pause container
	pauseContainerCommand := exec.Command("docker", "run", "--rm", "-d", "--name", "pretend-pause", "--net=none", "amazon/amazon-ecs-pause:0.1.0")
	err := pauseContainerCommand.Run()
	assertNoError(err, "Something went wrong starting the pause container")

	// find pause container network namespace
	dockerInspectCommandPid := exec.Command("docker", "inspect", "-f", "{{.State.Pid}}", "pretend-pause")
	dockerInspectOutputPid, err := dockerInspectCommandPid.Output()
	assertNoError(err, "Something went wrong finding the pause container pid")
	pauseContainerNetNamespace := fmt.Sprintf("/proc/%s/ns/net", strings.TrimSpace(string(dockerInspectOutputPid)))

	// find the pause container full id
	dockerInspectCommandId := exec.Command("docker", "inspect", "-f", "{{.Id}}", "pretend-pause")
	dockerInspectOutputId, err := dockerInspectCommandId.Output()
	assertNoError(err, "Something went wrong finding the pause container id")
	newContainerNetworkId := fmt.Sprintf("container:%s", strings.TrimSpace(string(dockerInspectOutputId)))

	// parse plugin directory
	cwd, err := os.Getwd()
	assertNoError(err, "Could not get current working directory")
	pathSepStr := string(os.PathSeparator)
	pluginsPath := fmt.Sprintf("%s%splugins", cwd, pathSepStr)
	// parse plugin directory - within agent
	// pluginsPath := "/tmp"
	vpcBridgePluginPath, err := invoke.FindInPath("vpc-bridge", []string{pluginsPath})
	assertNoError(err, fmt.Sprintf("Could not find the vpc-bridge plugin in path %s", pluginsPath))

	// setup proper logging
	pluginLogsDir, err := os.MkdirTemp("", "vpc-bridge-exp-")
	assertNoError(err, "Unable to create directory for logs")
	err = os.Chmod(pluginLogsDir, 0755)
	assertNoError(err, "Unable to set permissions for logs directory")
	// setup proper logging - within agent
	// pluginLogsDir := "/log"
	os.Setenv("VPC_CNI_LOG_FILE", fmt.Sprintf("%s/vpc-bridge.log", pluginLogsDir))
	defer os.Unsetenv("VPC_CNI_LOG_FILE")
	os.Setenv("VPC_CNI_LOG_LEVEL", "debug")
	defer os.Unsetenv("VPC_CNI_LOG_LEVEL")

	// setup vpc-bridge args
	vpcBridgeExecInvokeArgs := &invoke.Args{
		ContainerID: "test-container",
		NetNS:       pauseContainerNetNamespace,
		IfName:      "en0",
		Path:        pluginsPath,
		Command:     cniCommand,
	}

	// execute vpc-bridge plugin
	vpcBridgeResult, err := invoke.ExecPluginWithResult(
		context.Background(),
		vpcBridgePluginPath,
		[]byte(vpcBridgeNetConfJson),
		vpcBridgeExecInvokeArgs,
		nil,
	)
	assertNoError(err, "Something went wrong with invoking the vpc-bridge plugin")
	fmt.Printf("%+v\n", vpcBridgeResult)

	// run nginx container within the pause container network namespace
	dockerRunCommand := exec.Command("docker", "run", "--rm", "-d", "--name", "test-nginx", fmt.Sprintf("--net=%s", newContainerNetworkId), "nginx")
	err = dockerRunCommand.Run()
	assertNoError(err, "Something went wrong starting the nginx container inside the same namespace as the pause container")
}

func assertNoError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err)
		os.Exit(2)
	}
}
