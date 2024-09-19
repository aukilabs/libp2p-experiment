package main

import (
	"context"
	"log"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

var DefaultTarget = config.Config{
	NodeTypes:      []string{config.DATA_NODE},
	Name:           "holepunch_target",
	Port:           "",
	Mode:           dht.ModeServer,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func main() {
	config.LoadFromCliArgs(&DefaultTarget)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  DefaultTarget.Name,
		Types: DefaultTarget.NodeTypes,
	}

	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &DefaultTarget, func(h host.Host) {})
}
