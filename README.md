# cni-exp

## Getting started

First run the `./initial-setup.sh` script to pull down the code and build the plugin to start experimenting.

You'll also need a dummy link that we'll configure as part of the CNI plugin execution and also a network namespace to build everything in. You can do that by running the following commands. Note that you'll require elevated privileges to run these commands.

```
ip link add eth10 type dummy
ip netns add blue
```

Listing the links and network namespaces you have can be done using `ip link show` for links and `ip netns show` for network namespaces.

These aren't included as part of the `./initial-setup.sh` script because these resources are directly operated on by the plugin invocation and we might want to set these up/explore them after setup and teardown.

Finally, invoke the plugin by running `go run main.go <action>` with `action` being one of the following

| value    | consequence                                 |
| -------- | ------------------------------------------- |
| `add`    | invoke the plugin with the `ADD` command    |
| `delete` | invoke the plugin with the `DELETE` command |

You can also build the code by running `go build main.go` and then run `./main <action>`. It's probably better to do this because since we're setting up networks, we're going to want elevated privileges anyway.

## Cleanup

The `./initial-setup.sh` script is idempotent so running it again will just pull the code down. The `amazon-vpc-cni-plugins` directory can be safely deleted at any point as long as the main code file isn't running somewhere else.

You can also clean up the links and the network namespaces by running the following commands. Note that you'll require elevated privileges to run these commands as well.

```
ip link delete eth10
ip netns delete blue
```

Depending on which plugin you're running, you might also have other interfaces to clean up. Always take note of what interfaces existed before starting this experiment.

## What is actually happening here?

We're going to start at the highest level here, we're setting up a network namespace based around an interface on the host. The special thing about this setup is that this interface (likely the primary interface on the host) will be shared across multiple copies of this network setup.

The CNI plugin accepts configuration through `stdin` in JSON format, and an example is the following

```json
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
```

Additionally, it also takes environment variable arguments that are common for every CNI plugin which can be found [here](https://github.com/containernetworking/cni/blob/main/SPEC.md#parameters).

Some of the ones to pay attention to here are `CNI_NETNS` and `CNI_IFNAME`. The `CNI_NETNS` specifies the network namespace that we want to configure using the plugin and the `CNI_IFNAME` specifies the name of the interface that will exist in the network namespace eventually.

Once run, the plugin creates a bridge network called `vpcbr${interface_index}` where the `interface_index` is the index of the interface that is provided via the `eniName` option in the configuration. For example, assume that the `eth10` interface was at index 20 (can be checked by running `ip link show`), the bridge would be called `vpcbr20`.

The other link that is created is called `vpcbr${interface_name}dummy`. This is a dummy link that we create so that we can set the MTU for the bridge network. Usually it comes with a default of 1500 but AWS defaults to using jumbo frames, this helps us maintain the MTU at 9001 for the bridge network.

Then we create a veth pair (which is just a virtual ethernet cable), one end of the veth pair is connected to the host network namespace and the other end of the veth pair is moved to the network namespace (specified by `CNI_NETNS`) that we want the CNI plugin to use for configuration. Note that the name of this end of the veth pair will be renamed to the whatever the value of the `CNI_IFNAME` is specified to be since this is the interface in the network namespace.

Next, the `ipAddresses` values in the JSON configuration are assigned as addresses to the side of the veth pair created above that exists within the network namespace that we are operating on.

Once all the links are in place and the IP addresses are assigned, the plugin will create routes to make sure that traffic can move between the networks. The only route that we need to create in the host network namespace is a way for us to get from the internet to the list of `ipAddresses` provided in the JSON configuration. Since we only have one in our configuration, (there is provision for you to specify an IPv6 address also), and we've already established a bridge network that our veth pair is connected to, the traffic should go to the bridge network.

In the network namespace that we are operating on, we only need one route as well, anything that needs to exit the network namespace should go out via the side of the veth pair that exists within the network namespace.
