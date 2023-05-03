package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Need to pass in an action, either move or unmove")
		os.Exit(1)
	}
	givenAction := os.Args[1]

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if givenAction == "move" {
		// get current network namespace
		origins, err := netns.Get()
		assertNoError(err, "Something went wrong getting the current network namespace")
		defer origins.Close()
		fmt.Printf("Current network namespace: %+v\n", origins)

		// get details of the link to move
		linkToMove, err := netlink.LinkByName("eth5")
		assertNoError(err, "Something went wrong finding the eth5 interface")
		fmt.Printf("Link to move: %+v\n", linkToMove)

		// get the ip addresses assigned to the link
		ipAddrs, err := netlink.AddrList(linkToMove, netlink.FAMILY_ALL)
		assertNoError(err, "Something went wrong finding the eth5 interface addresses")
		for _, addr := range ipAddrs {
			fmt.Printf("Address: %+v\n", strings.Split(addr.String(), " ")[0])
		}

		// get the routes assigned to the link
		routes, err := netlink.RouteList(linkToMove, netlink.FAMILY_ALL)
		assertNoError(err, "Something went wrong finding the eth4 interface routes")
		for _, route := range routes {
			fmt.Printf("Route: %+v\n", route)
		}

		// create a new network namespace
		eniNetNs, err := netns.NewNamed("seceni")
		assertNoError(err, "Something went wrong creating the new network namespace")
		defer eniNetNs.Close()

		// switch back to the og namespace so that we can move the device
		netns.Set(origins)

		err = netlink.LinkSetNsFd(linkToMove, int(eniNetNs))
		assertNoError(err, "Something went wrong setting the link in the new network namespace")

		// switch into the new network namespace that we created
		netns.Set(eniNetNs)

		// set the link up
		err = netlink.LinkSetUp(linkToMove)
		assertNoError(err, "Something went wrong setting the link up")

		// assign all the addresses that we found earlier
		for _, addr := range ipAddrs {
			err = netlink.AddrAdd(linkToMove, &addr)
			assertNoError(err, "Something went wrong adding the address to the link")
		}

		// create all the routes that we found earlier
		for _, route := range routes {
			err = netlink.RouteAdd(&route)
			if err != nil {
				fmt.Println("Something went wrong adding the the following route")
				fmt.Println(err)
			}
		}

		// get a list of all the links in the new network namespace
		linkList, err := netlink.LinkList()
		assertNoError(err, "Something went wrong listing the interfaces inside new network namespace")
		for _, link := range linkList {
			fmt.Printf("Link: %+v\n", link)
		}

		netns.Set(origins)
	} else if givenAction == "unmove" {
		fmt.Println("Not implemented yet")
	}
}

func assertNoError(err error, message string) {
	if err != nil {
		fmt.Println(message)
		fmt.Println(err)
		os.Exit(2)
	}
}
