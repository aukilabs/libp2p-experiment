package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n1Cfg := &config.Config{
		NodeTypes:      []string{config.DATA_NODE},
		Name:           "mdns_example_n1",
		Mode:           dht.ModeClient,
		BootstrapPeers: []string{},
	}
	n1, err := node.NewNode(node.NodeInfo{
		Name:  n1Cfg.Name,
		Types: n1Cfg.NodeTypes,
	}, "volume")
	if err != nil {
		panic(err)
	}
	n1.Start(ctx, n1Cfg, func(h host.Host) {})

	n2Cfg := &config.Config{
		NodeTypes:      []string{config.DATA_NODE},
		Name:           "mdns_example_n2",
		Mode:           dht.ModeClient,
		BootstrapPeers: []string{},
	}
	n2, err := node.NewNode(node.NodeInfo{
		Name:  n2Cfg.Name,
		Types: n2Cfg.NodeTypes,
	}, "volume")
	if err != nil {
		panic(err)
	}
	n2.Start(ctx, n2Cfg, func(h host.Host) {
		nodes := n2.FindNodes(config.DATA_NODE)
		for len(nodes) == 0 {
			time.Sleep(5 * time.Second)
			nodes = n2.FindNodes(config.DATA_NODE)
		}
		resCh := ping.Ping(ctx, h, nodes[0])
		res := <-resCh
		if res.Error != nil {
			log.Fatal(res.Error)
		}
		fmt.Printf("Pinged %s in %s", nodes[0], res.RTT)
	})

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	os.Exit(0)
}
