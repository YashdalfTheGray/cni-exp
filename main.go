package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/containernetworking/cni/pkg/invoke"
	"github.com/vishvananda/netns"
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
	"eniName": "eth5",
	"eniMACAddress": "02:81:33:7e:64:ed",
	"eniIPAddresses": ["10.0.0.32/24"],
	"vpcCIDRs": ["10.0.0.0/24"],
	"ipAddresses": ["10.0.0.193/28"],
	"gatewayIPAddress": "10.0.0.1",
	"bridgeType": "L3"
}
`

const ecsbridgeNetConfJson = `
{
	"cniVersion": "0.3.0",
	"type": "ecs-bridge",
	"bridge": "ecs-br0",
	"mtu": 9001,
	"ipam": {
		"type": "ecs-ipam",
		"id": "plz-work",
		"ipv4-subnet": "169.254.172.0/22",
		"ipv4-routes": [{ "dst": "169.254.170.2/32" }]
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
	dockerCommand := ""
	if givenAction == "add" {
		cniCommand = "ADD"
		dockerCommand = "run"
	} else if givenAction == "delete" {
		cniCommand = "DEL"
		dockerCommand = "stop"
	} else {
		fmt.Println("Couldn't map given action to CNI command")
		os.Exit(1)
	}
	fmt.Printf("Was asked to run the following command - %s\n", cniCommand)

	// find the current network namespace so that we can store a ref to it
	origins, err := netns.Get()
	assertNoError(err, "Something went wrong getting the current network namespace")
	defer origins.Close()
	fmt.Printf("Current network namespace: %+v\n", origins)

	// start docker pause container
	if dockerCommand == "run" {
		pauseContainerCommand := exec.Command("docker", "run", "--rm", "-d", "--name", "pretend-pause", "--net=none", "amazon/amazon-ecs-pause:0.1.0")
		err := pauseContainerCommand.Run()
		assertNoError(err, "Something went wrong starting the pause container")
	} else if dockerCommand == "stop" {
		nginxStopCommand := exec.Command("docker", "stop", "test-nginx")
		err := nginxStopCommand.Run()
		fmt.Println("Could not stop nginx container, it is likely that it never ran")
		fmt.Print(err)
	}

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
	ecsbridgePluginPath, err := invoke.FindInPath("ecs-bridge", []string{pluginsPath})
	assertNoError(err, fmt.Sprintf("Could not find the ecs-bridge plugin in path %s", pluginsPath))

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

	// switch to the ENI's network namespace
	eniNetNs, err := netns.GetFromName("seceni")
	assertNoError(err, "Something went wrong finding the ENI's network namespace")
	defer eniNetNs.Close()
	err = netns.Set(eniNetNs)
	assertNoError(err, "Something went wrong switching to the ENI's network namespace")

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

	// switch back because we need the bridge created in the root network namespace
	err = netns.Set(origins)
	assertNoError(err, "Something went wrong switching back to the root network namespace")

	// setup ecs-ipam args
	ecsBridgeExecInvokeArgs := &invoke.Args{
		ContainerID: "test-container",
		NetNS:       pauseContainerNetNamespace,
		IfName:      "en1",
		Path:        pluginsPath,
		Command:     cniCommand,
	}

	// execute ecs-ipam plugin
	ecsbridgeResult, err := invoke.ExecPluginWithResult(
		context.Background(),
		ecsbridgePluginPath,
		[]byte(ecsbridgeNetConfJson),
		ecsBridgeExecInvokeArgs,
		nil,
	)
	assertNoError(err, "Something went wrong with invoking the ecs-bridge plugin")
	fmt.Printf("%+v\n", ecsbridgeResult)

	if dockerCommand == "run" {
		// run nginx container within the pause container network namespace
		dockerRunCommand := exec.Command("docker", "run", "--rm", "-d", "--name", "test-nginx", fmt.Sprintf("--net=%s", newContainerNetworkId), "nginx")
		err = dockerRunCommand.Run()
		assertNoError(err, "Something went wrong starting the nginx container inside the same namespace as the pause container")
	} else if dockerCommand == "stop" {
		pauseStopCommand := exec.Command("docker", "stop", "pretend-pause")
		err := pauseStopCommand.Run()
		assertNoError(err, "Something went wrong stopping the pause container")
	}
}

func assertNoError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err)
		os.Exit(2)
	}
}
