package main

import (
	"context"
	"log"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

var DefaultDailer = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "holepunch_dailer",
	Port:           "",
	Mode:           dht.ModeServer,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func main() {
	config.LoadFromCliArgs(&DefaultDailer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  DefaultDailer.Name,
		Types: DefaultDailer.NodeTypes,
	}

	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &DefaultDailer, func(h host.Host) {
		log.Println("Finding data node...")
		nodes := n.FindNodes(config.DATA_NODE)
		for len(nodes) == 0 {
			time.Sleep(5 * time.Second)
			nodes = n.FindNodes(config.DATA_NODE)
		}

		log.Printf("Finding %s...", nodes[0])
		addr, err := n.FindPeerAddresses(ctx, nodes[0])
		for err != nil {
			log.Println("Failed to find peer address: ", err)
			time.Sleep(5 * time.Second)
			addr, err = n.FindPeerAddresses(ctx, nodes[0])
		}

		log.Printf("Connecting to %v...", addr.Addrs)
		if err := h.Connect(ctx, *addr); err != nil {
			log.Fatalf("Failed to connect to peer: %s\n", err)
		}

		log.Println("Connected to peer")
		resCh := ping.Ping(ctx, h, addr.ID)
		res := <-resCh
		if res.Error != nil {
			log.Fatalf("Failed to ping peer: %s\n", res.Error)
		}
		log.Println("Pinged peer")
	})
}
