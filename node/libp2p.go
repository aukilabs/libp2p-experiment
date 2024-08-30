package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	quicTransport "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
)

const PosemeshService = "posemesh"
const NodeInfoTopic = "posemesh_nodes"

type NodeInfo struct {
	Types []string `json:"node_types"`
	ID    peer.ID  `json:"id"`
	Name  string   `json:"name"`
}

type Node struct {
	NodeInfo
	host.Host
	kademliaDHT routing.Routing
	neighbors   map[peer.ID]*NodeInfo
	jobList     map[string]JobInfo
	mutex       sync.RWMutex
	BasePath    string
	identity    crypto.PrivKey
	PubSub      *pubsub.PubSub
}

func NewNode(info NodeInfo, basePath string) (node *Node, err error) {
	p := path.Join(basePath, info.Name)
	priv := initPeerIdentity(p)
	info.ID, err = peer.IDFromPublicKey(priv.GetPublic())
	if err != nil {
		return nil, err
	}
	node = &Node{
		NodeInfo:  info,
		neighbors: make(map[peer.ID]*NodeInfo),
		jobList:   make(map[string]JobInfo),
		mutex:     sync.RWMutex{},
		BasePath:  p,
		identity:  priv,
	}
	return node, nil
}

func (n *Node) AddNode(node *NodeInfo) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if _, ok := n.neighbors[node.ID]; !ok {
		n.neighbors[node.ID] = node
		log.Printf("Node %s joined the network\n", node.Name)
	}
}

func (n *Node) FindNodes(nodeType string) []peer.ID {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	nodes := make([]peer.ID, 0)
	for _, node := range n.neighbors {
		for _, nt := range node.Types {
			if nt == nodeType {
				nodes = append(nodes, node.ID)
			}
		}
	}
	return nodes
}

func (n *Node) FindPeerAddresses(ctx context.Context, id peer.ID) ([]multiaddr.Multiaddr, error) {
	peer, err := n.kademliaDHT.FindPeer(ctx, id)
	if err != nil {
		log.Printf("Failed to find peer: %s\n", err)
		return nil, err
	}
	log.Printf("Found peer: %s\n", peer)
	return peer.Addrs, nil
}

type StepInfo struct {
	Name    string       `json:"name"`
	Outputs []OutputInfo `json:"outputs"`
}
type JobInfo struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	DomainPubKey string     `json:"domain_pub_key"`
	State        string     `json:"-"`
	Steps        []StepInfo `json:"steps"`
	Requester    peer.ID    `json:"requester"`
}
type OutputInfo struct {
	ID         peer.ID `json:"id"`
	ProtocolID string  `json:"protocol_id"`
}

func createDHT(ctx context.Context, host host.Host, public bool, mode dht.ModeOpt, bootstrapPeers ...peer.AddrInfo) (routing.Routing, error) {
	var opts []dht.Option

	opts = append(opts,
		dht.Concurrency(10),
		dht.Mode(dht.ModeServer),
		dht.BootstrapPeers(bootstrapPeers...),
		dht.ProtocolPrefix("/posemesh"),
	)

	// dual.New(ctx, host, dual.DHTOption(opts...))
	kademliaDHT, err := dht.New(
		ctx, host, opts...,
	)
	if err != nil {
		return nil, err
	}

	if err := kademliaDHT.Bootstrap(ctx); err != nil {
		return nil, err
	}

	return kademliaDHT, nil
}

func (node *Node) Start(ctx context.Context, cfg *config.Config, handlers func(host.Host)) {
	// if err := addDomainProtocol(); err != nil {
	// 	log.Fatal(err)
	// }
	// connmgr, err := connmgr.NewConnManager(
	// 	100, // Lowwater
	// 	400, // HighWater,
	// 	connmgr.WithGracePeriod(time.Minute),
	// )
	// if err != nil {
	// 	panic(err)
	// }
	port := cfg.Port
	if port == "" {
		port = "0"
	}
	h2, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(node.identity),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/"+port,       // regular tcp connections
			"/ip4/0.0.0.0/tcp/"+port+"/ws", // websockets
			"/ip4/0.0.0.0/udp/"+port+"/quic-v1",
			"/ip6/::/udp/"+port+"/quic-v1",
			"/ip4/0.0.0.0/udp/"+port+"/quic-v1/webtransport",
			"/ip6/::/udp/"+port+"/quic-v1/webtransport",
		),
		libp2p.Transport(quicTransport.NewTransport),
		libp2p.Transport(webtransport.New),
		libp2p.Transport(ws.New),
		libp2p.Transport(tcp.NewTCPTransport),
		// libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support noise connections
		// libp2p.Security(noise.ID, noise.New),
		// support any other default transports (TCP)
		// libp2p.DefaultTransports,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		// libp2p.ConnectionManager(connmgr),
		// Attempt to open ports using uPNP for NATed hosts.
		// libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			addrs := make([]peer.AddrInfo, len(cfg.BootstrapPeers))
			for i, addr := range cfg.BootstrapPeers {
				ai, err := peer.AddrInfoFromString(addr)
				if err != nil {
					return nil, err
				}
				addrs[i] = *ai
			}
			idht, err := createDHT(ctx, h, true, dht.ModeServer, addrs...)
			return idht, err
		}),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
		libp2p.EnableRelay(),
	)
	if err != nil {
		panic(err)
	}
	node.Host = h2

	for _, addr := range h2.Addrs() {
		fmt.Printf("Listening on: %s/p2p/%s\n", addr.String(), h2.ID())
	}
	ps, err := pubsub.NewGossipSub(ctx, h2)
	if err != nil {
		panic(err)
	}
	node.PubSub = ps

	// if err := setupMDNS(h2); err != nil {
	// 	panic(err)
	// }

	topic, err := ps.Join(NodeInfoTopic)
	if err != nil {
		panic(err)
	}
	defer topic.Close()
	sub, err := topic.Subscribe()
	if err != nil {
		panic(err)
	}

	// publish our own node info every 10 seconds, cancel publishing when the context is done
	if cfg.NodeTypes[0] != "client" {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					data, err := json.Marshal(node.NodeInfo)
					if err != nil {
						panic(err)
					}
					if err := topic.Publish(ctx, data); err != nil {
						panic(err)
					}
				case <-ctx.Done():
					log.Println("Cancelled publishing")
					return
				}
			}
		}()
	}

	// read messages from the topic, and do something based on the node type, cancel reading when the context is done
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Println("Failed to read message:", err)
				continue
			}
			neighber := NodeInfo{}
			if err := json.Unmarshal(msg.Data, &neighber); err != nil {
				log.Printf("invalid message: %s\n", msg.Data)
				continue
			}
			if neighber.ID != h2.ID() {
				node.AddNode(&neighber)
			}
		}
	}()

	handlers(h2)

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	os.Exit(0)
}

// func (n *Node) JoinDomainCluster(ctx context.Context, domainPriKey string) error {
// 	domainPubKey, err := crypto.UnmarshalPublicKey([]byte(domainPriKey))
// 	if err != nil {
// 		return err
// 	}
// 	topic, err := n.PubSub.Join(domainPubKey)
// 	if err != nil {
// 		return err
// 	}
// 	go func() {
// 		for {
// 			msg, err := topic.Next(ctx)
// 			if err != nil {
// 				log.Println("Failed to read message:", err)
// 				continue
// 			}
// 			log.Printf("Received message: %s\n", msg.Data)
// 		}
// 	}()
// 	return nil
// }
