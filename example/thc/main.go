package main

import (
	"context"
	"flag"
	"log"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

var CameraAppCfg = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "thc",
	Port:           "",
	Mode:           dht.ModeClient,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func main() {
	var name = flag.String("name", "dmt", "app name")
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: CameraAppCfg.NodeTypes,
	}
	CameraAppCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &CameraAppCfg, func(h host.Host) {

	})
}
