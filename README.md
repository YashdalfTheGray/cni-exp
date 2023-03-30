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
