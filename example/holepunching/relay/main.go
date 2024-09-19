package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

var DefaultRelayNode = config.Config{
	NodeTypes:      []string{config.DISCOVERY_NODE},
	Name:           "relay_1",
	Port:           "",
	Mode:           dht.ModeServer,
	BootstrapPeers: []string{},
	EnableRelay:    true,
}

func main() {
	config.LoadFromCliArgs(&DefaultRelayNode)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  DefaultRelayNode.Name,
		Types: DefaultRelayNode.NodeTypes,
	}

	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &DefaultRelayNode, func(h host.Host) {})

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	os.Exit(0)
}
