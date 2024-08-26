package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/job"
	"github.com/aukilabs/go-libp2p-experiment/node"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"gocv.io/x/gocv"
)

var RecorderCfg = config.Config{
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
		Types: RecorderCfg.NodeTypes,
	}
	RecorderCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &RecorderCfg, func(h host.Host) {
		salviaNodes := n.FindNodes(config.SALVIA_NODE)
		for _, p := range salviaNodes {
			addrs, err := n.FindPeerAddresses(ctx, p)
			if err != nil {
				log.Printf("Failed to get peer addresses: %s\n", err)
				continue
			}
		}
	})
}
