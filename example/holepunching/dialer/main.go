package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/multiformats/go-multiaddr"
)

var DefaultDialer = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "holepunch_dailer",
	Port:           "",
	Mode:           dht.ModeClient,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func main() {
	config.LoadFromCliArgs(&DefaultDialer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  DefaultDialer.Name,
		Types: DefaultDialer.NodeTypes,
	}

	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &DefaultDialer, func(h host.Host) {
		log.Println("Finding data node...")
		nodes := n.FindNodes(config.DATA_NODE)
		for len(nodes) == 0 {
			time.Sleep(5 * time.Second)
			nodes = n.FindNodes(config.DATA_NODE)
		}

		log.Printf("Finding %s...", nodes[0])
		addr, err := n.FindPeerAddresses(ctx, nodes[0])
		for err != nil {
			time.Sleep(30 * time.Second)
			addr, err = n.FindPeerAddresses(ctx, nodes[0])
		}

		log.Printf("Connecting to %v...", addr.Addrs)
		if err := h.Connect(ctx, *addr); err != nil {
			log.Fatalf("Failed to connect to peer: %s\n", err)
		}

		log.Println("Connected to peer")

		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*3)
		resCh := ping.Ping(timeoutCtx, h, addr.ID)
		for res := range resCh {
			if res.Error != nil {
				log.Printf("Failed to ping peer: %s\n", res.Error)
				continue
			}
			log.Println("Pinged peer")
		}
		cancel()

		// to retry hole punching in case of `protocols not supported: [/libp2p/dcutr]` error
		for _, c := range h.Network().(*swarm.Swarm).ConnsToPeer(nodes[0]) {
			// close the connection if it is a relay connection to prove that hole punching works
			if _, err := c.RemoteMultiaddr().ValueForProtocol(multiaddr.P_CIRCUIT); err == nil {
				log.Printf("Closing relay connection to %s\n", c.RemotePeer())
				c.Close()
			}
		}

		time.Sleep(10 * time.Second)

		// This should connect directly to the peer or start hole punching
		if err := h.Connect(ctx, *addr); err != nil {
			log.Fatalf("Failed to connect to peer: %s\n", err)
		}

		timeoutCtx, cancel = context.WithTimeout(ctx, time.Second*30)
		defer cancel()
		resCh = ping.Ping(timeoutCtx, h, addr.ID)
		for res := range resCh {
			if res.Error != nil {
				log.Printf("Failed to ping peer: %s\n", res.Error)
				continue
			}
			log.Println("Pinged peer")
		}
	})

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	os.Exit(0)
}
