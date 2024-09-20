package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"sync"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/pnet"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
)

const PosemeshService = "posemesh"
const PosemeshRelayService = "posemesh/relay"
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
	priv := InitPeerIdentity(p)
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

func (n *Node) FindPeerAddresses(ctx context.Context, id peer.ID) (*peer.AddrInfo, error) {
	peer, err := n.kademliaDHT.FindPeer(ctx, id)
	if err != nil {
		log.Printf("Failed to find peer: %s\n", err)
		return nil, err
	}
	log.Printf("Found peer: %s\n", peer)
	return &peer, nil
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

func createDHT(ctx context.Context, host host.Host, mode dht.ModeOpt, bootstrapPeers ...peer.AddrInfo) (routing.Routing, error) {
	var opts []dht.Option

	opts = append(opts,
		dht.Concurrency(10),
		dht.Mode(mode),
		dht.BootstrapPeers(bootstrapPeers...),
		dht.ProtocolPrefix("/posemesh"),
	)

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
	port := cfg.Port
	if port == "" {
		port = "0"
	}
	var psk pnet.PSK
	var err error
	if cfg.Psk != "" {
		pskReader := bytes.NewReader([]byte(cfg.Psk))
		psk, err = pnet.DecodeV1PSK(pskReader)
		if err != nil {
			log.Fatal(err)
		}
	}
	addrs := make([]peer.AddrInfo, len(cfg.BootstrapPeers))
	for i, addr := range cfg.BootstrapPeers {
		ai, err := peer.AddrInfoFromString(addr)
		if err != nil {
			log.Println(err)
			continue
		}
		addrs[i] = *ai
	}
	var routingDiscovery *drouting.RoutingDiscovery
	opts := []libp2p.Option{
		libp2p.PrivateNetwork(psk),
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
		libp2p.DefaultTransports,
		// libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support noise connections
		// libp2p.Security(noise.ID, noise.New),
		// support any other default transports (TCP)
		// libp2p.DefaultTransports,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		// libp2p.ConnectionManager(connmgr),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err := createDHT(ctx, h, cfg.Mode, addrs...)
			node.kademliaDHT = idht
			return idht, err
		}),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
		libp2p.Ping(true),
	}
	if cfg.Mode == dht.ModeClient {
		opts = append(opts, libp2p.EnableRelay())
	} else {
		opts = append(opts, libp2p.EnableAutoRelayWithPeerSource(func(ctx context.Context, num int) <-chan peer.AddrInfo {
			for routingDiscovery == nil {
				time.Sleep(3 * time.Second)
			}
			relaysCh, err := routingDiscovery.FindPeers(ctx, PosemeshRelayService, discovery.Limit(num))
			if err != nil {
				log.Println(err)
				return nil
			}
			return relaysCh
		}, autorelay.WithMaxCandidates(10)))
	}
	if cfg.EnableRelay {
		opts = append(opts, libp2p.ForceReachabilityPublic())
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		panic(err)
	}
	node.Host = h

	routingDiscovery = drouting.NewRoutingDiscovery(node.kademliaDHT)

	if cfg.EnableRelay {
		if _, err := relay.New(h); err != nil {
			panic(err)
		}
		dutil.Advertise(ctx, routingDiscovery, PosemeshRelayService)
		log.Println("Relay enabled")
	}

	log.Println("Host created. We are:", h.ID())
	for _, addr := range h.Addrs() {
		log.Printf("Listening on: %s/p2p/%s\n", addr.String(), h.ID())
	}
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}
	node.PubSub = ps

	if err := node.setupMDNS(ctx, h); err != nil {
		panic(err)
	}

	topic, err := ps.Join(NodeInfoTopic)
	if err != nil {
		panic(err)
	}

	eh, err := topic.EventHandler()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			e, err := eh.NextPeerEvent(ctx)
			if err != nil {
				log.Println(err)
				continue
			}
			if e.Type == pubsub.PeerLeave {
				delete(node.neighbors, e.Peer)
			}
		}
	}()
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
			if neighber.ID != h.ID() {
				node.AddNode(&neighber)
			}
		}
	}()

	// h.Network().Notify(&network.NotifyBundle{
	// 	DisconnectedF: func(_ network.Network, conn network.Conn) {
	// 		fmt.Printf("Disconnected from %s\n", conn.RemoteMultiaddr())
	// 	},
	// })

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(30 * time.Second)
				connections := h.Network().Conns()
				for i, conn := range connections {
					if i == 0 {
						fmt.Print("###############CONNECTIONS################\n\n")
					}
					fmt.Printf("Connected to peer %s at %s\n", conn.RemotePeer(), conn.RemoteMultiaddr())
					if i == len(connections)-1 {
						fmt.Println("\n\n###############CONNECTIONS################")
					}
				}

			}
		}

	}()

	handlers(h)
}

type discoveryNotifee struct {
	PeerChan chan peer.AddrInfo
}

// interface to be called when new  peer is found
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.PeerChan <- pi
}

func (node *Node) setupMDNS(ctx context.Context, h host.Host) error {
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)
	ser := mdns.NewMdnsService(h, PosemeshService, n)
	if err := ser.Start(); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				ser.Close()
				return
			case pi := <-n.PeerChan:
				log.Printf("Found peer: %s in local network\n", pi)
				if err := h.Connect(context.Background(), pi); err != nil {
					log.Println("Failed to connect to peer:", err)
					continue
				}
			}
		}
	}()
	return nil
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
