package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
)

const ADAM_NODE = "ADAM"
const DATA_NODE = "DATA"
const BOOSTRAP_NODE = "BOOTSTRAP"
const DISCOVERY_NODE = "DOMAIN_SERVICE"
const ADAM_PROTOCOL_ID = "/posemesh/adam/1.0.0"
const UPLOAD_DOMAIN_DATA_PROTOCOL_ID = "/posemesh/upload_domain/1.0.0"
const DOWNLOAD_DOMAIN_DATA_PROTOCOL_ID = "/posemesh/download_domain/1.0.0"
const DOMAIN_AUTH_PROTOCOL_ID = "/posemesh/domain_auth/1.0.0"
const ON_TASK_DONE_CALLBACK_PROTOCOL_ID = "/posemesh/callback/task_done/1.0.0"
const FETCH_DOMAINS_PROTOCOL_ID = "/posemesh/fetch_domains/1.0.0"
const CREATE_DOMAIN_PROTOCOL_ID = "/posemesh/create_domain/1.0.0"
const CREATE_PORTAL_PROTOCOL_ID = "/posemesh/create_portal/1.0.0"
const PosemeshService = "posemesh"
const NodeInfoTopic = "posemesh_nodes"
const PortalTopic = "portals"
const DomainTopic = "domains"

var cfgFlag = flag.String("config", "adamNodeCfg", "config")
var cfg config.Config
var basePath = "volume/" + cfg.Name

func main() {
	if cfgFlag == nil {
		log.Fatal("config flag is required")
	}
	flag.Parse()
	switch *cfgFlag {
	case "adam-1":
		cfg = adamNode1Cfg
	case "adam-2":
		cfg = adamNode2Cfg
	case "data-1":
		cfg = dataNodeCfg
	case "data-2":
		cfg = replicationNodeCfg
	case "dds-1":
		cfg = domainServiceNodeCfg
	case "dds-2":
		cfg = domainServiceNode2Cfg
	case "camera":
		cfg = cameraAppCfg
	case "dmt":
		cfg = dmtCfg
	}
	basePath = "volume/" + cfg.Name
	run()
}

type nodeInfo struct {
	Type string  `json:"node_type"`
	ID   peer.ID `json:"id"`
	Name string  `json:"name"`
}
type stepInfo struct {
	Name    string       `json:"name"`
	Outputs []outputInfo `json:"outputs"`
}
type taskInfo struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	DomainPubKey string     `json:"domain_pub_key"`
	Name         string     `json:"name"`
	State        string     `json:"-"`
	Steps        []stepInfo `json:"steps"`
}
type taskCallback struct {
	TaskID     string `json:"task_id"`
	Error      error  `json:"error"`
	ResultPath string `json:"result_path"`
}
type jobCallback struct {
	JobID string `json:"job_id"`
	Error error  `json:"error"`
}
type jobInfo struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	DomainPubKey string     `json:"domain_pub_key"`
	State        string     `json:"-"`
	Steps        []stepInfo `json:"steps"`
	Requester    peer.ID    `json:"requester"`
}
type outputInfo struct {
	ID         peer.ID `json:"id"`
	ProtocolID string  `json:"protocol_id"`
}
type portal struct {
	CreatedBy   peer.ID `json:"created_by"`
	ShortID     string  `json:"short_id"`
	Size        float32 `json:"size"`
	DefaultName string  `json:"default_name"`
	Hash        string  `json:"hash"`
}
type domain struct {
	PublicKey   string `json:"public_key"`
	OwnerWallet string `json:"owner_wallet"`
	Name        string `json:"name"`
	DataNodes   []peer.ID
}
type signedMsg struct {
	Msg []byte `json:"msg"`
	Sig []byte `json:"sig"`
}

var nodeList = map[peer.ID]nodeInfo{}
var taskList = map[string]taskInfo{}
var jobList = map[string]jobInfo{}
var portalList = map[string]portal{}
var domainList = map[string]domain{}

type domainValidator struct {
}

func (v *domainValidator) Validate(key string, value []byte) error {
	// if !strings.HasPrefix(key, "/domain/") {
	// 	return fmt.Errorf("invalid key: %s", key)
	// }
	// TODO: validate value
	log.Println("Validating key:", key)
	return nil
}
func (v *domainValidator) Select(key string, values [][]byte) (int, error) {
	return 0, nil
}

func createDHT(ctx context.Context, host host.Host, public bool, mode dht.ModeOpt, bootstrapPeers ...peer.AddrInfo) (routing.Routing, error) {
	var opts []dht.Option

	opts = append(opts,
		dht.Concurrency(10),
		dht.Mode(dht.ModeServer),
		dht.Validator(&domainValidator{}),
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

	go configureDHT(ctx, host, kademliaDHT)

	return kademliaDHT, nil
}

func run() {
	// The context governs the lifetime of the libp2p node.
	// Cancelling it will stop the host.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	priv := InitPeerIdentity(basePath)

	// connmgr, err := connmgr.NewConnManager(
	// 	100, // Lowwater
	// 	400, // HighWater,
	// 	connmgr.WithGracePeriod(time.Minute),
	// )
	// if err != nil {
	// 	panic(err)
	// }
	h2, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/"+cfg.Port, // regular tcp connections
		),
		// support TLS connections
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
		// libp2p.EnableNATService(),
	)
	if err != nil {
		panic(err)
	}
	defer h2.Close()

	log.Printf("Peer %s at %s\n", h2.ID(), h2.Addrs())
	ps, err := pubsub.NewGossipSub(ctx, h2)
	if err != nil {
		panic(err)
	}

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
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, node := range cfg.NodeTypes {
					node := nodeInfo{
						Type: node,
						ID:   h2.ID(),
						Name: cfg.Name,
					}
					data, err := json.Marshal(node)
					if err != nil {
						panic(err)
					}
					if err := topic.Publish(ctx, data); err != nil {
						panic(err)
					}
				}
			case <-ctx.Done():
				log.Println("Cancelled publishing")
				return
			}
		}
	}()

	for _, node := range cfg.NodeTypes {
		// h2.SetStreamHandler(ON_TASK_DONE_CALLBACK_PROTOCOL_ID, OnTaskDoneStreamHandler(h2))
		if node == DATA_NODE {
			h2.SetStreamHandler(UPLOAD_DOMAIN_DATA_PROTOCOL_ID, ReceiveDomainDataHandler(h2, func(ctx context.Context, dd *Libposemesh.DomainData) error {
				go classifyDomainData(ctx, dd)
				return nil
			}))
		}
		if node == ADAM_NODE {
			h2.SetStreamHandler(ADAM_PROTOCOL_ID, ReceiveVideoForPoseRefinement(h2))
		}
		if node == DISCOVERY_NODE {
			h2.SetStreamHandler(DOMAIN_AUTH_PROTOCOL_ID, DomainAuthHandler)
			portalTopic, err := ps.Join(PortalTopic)
			if err != nil {
				panic(err)
			}
			defer portalTopic.Close()
			go PublishPortals(ctx, portalTopic, h2)
			portalSub, err := portalTopic.Subscribe()
			if err != nil {
				panic(err)
			}
			go func() {
				for {
					msg, err := portalSub.Next(ctx)
					if err != nil {
						log.Println("Failed to read message:", err)
						continue
					}
					if msg.GetFrom() != h2.ID() {
						// pub, err := msg.GetFrom().ExtractPublicKey()
						// if err != nil {
						// 	log.Printf("Failed to extract public key: %s\n", err)
						// 	continue
						// }
						// sMsg := signedMsg{}
						var portal portal
						if err := json.Unmarshal(msg.Data, &portal); err != nil {
							log.Printf("invalid message: %s\n", msg.Data)
							continue
						}
						// verified, err := pub.Verify(sMsg.Msg, sMsg.Sig)
						// if err != nil {
						// 	log.Printf("Failed to verify message: %s\n", err)
						// 	continue
						// }
						// if verified {
						// 	portal := portal{}
						// 	if err := json.Unmarshal(sMsg.Msg, &portal); err != nil {
						// 		log.Printf("Failed to unmarshal portal: %s\n", err)
						// 		continue
						// 	}
						// 	log.Printf("Received portal: %s\n", portal)
						if _, ok := portalList[portal.ShortID]; !ok {
							portalList[portal.ShortID] = portal
							if err := os.MkdirAll(basePath+"/portals", os.ModePerm); err != nil {
								log.Printf("Failed to create directory: %s\n", err)
								continue
							}
							f, err := os.Create(basePath + "/portals/" + portal.ShortID)
							if err != nil {
								log.Printf("Failed to create file: %s\n", err)
								continue
							}
							defer f.Close()
							if err := json.NewEncoder(f).Encode(portal); err != nil {
								log.Printf("Failed to encode portal: %s\n", err)
								continue
							}
						}
						// }
					}
				}
			}()

			domainTopic, err := ps.Join(DomainTopic)
			if err != nil {
				panic(err)
			}
			defer domainTopic.Close()
			go PublishDomains(ctx, domainTopic, h2)
			domainSub, err := domainTopic.Subscribe()
			if err != nil {
				panic(err)
			}

			go func() {
				for {
					msg, err := domainSub.Next(ctx)
					if err != nil {
						log.Println("Failed to read message:", err)
						continue
					}
					if msg.GetFrom() != h2.ID() {
						domain := domain{}
						if err := json.Unmarshal(msg.Data, &domain); err != nil {
							log.Printf("invalid message: %s\n", msg.Data)
							continue
						}
						domainList[domain.PublicKey] = domain
						if err := os.MkdirAll(basePath+"/domains", os.ModePerm); err != nil {
							log.Printf("Failed to create directory: %s\n", err)
							continue
						}
						f, err := os.Create(basePath + "/domains/" + domain.PublicKey)
						if err != nil {
							log.Printf("Failed to create file: %s\n", err)
							continue
						}
						defer f.Close()
						if err := json.NewEncoder(f).Encode(domain); err != nil {
							log.Printf("Failed to encode portal: %s\n", err)
							continue
						}
					}
				}
			}()

			h2.SetStreamHandler(FETCH_DOMAINS_PROTOCOL_ID, LoadDomainsStreamHandler)
			h2.SetStreamHandler(CREATE_DOMAIN_PROTOCOL_ID, CreateDomainStreamHandler(ctx, domainTopic, h2))
			h2.SetStreamHandler(CREATE_PORTAL_PROTOCOL_ID, CreatePortalStreamHandler(ctx, portalTopic, h2))
		}
	}

	// read messages from the topic, and do something based on the node type, cancel reading when the context is done
	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				log.Println("Failed to read message:", err)
				continue
			}
			node := nodeInfo{}
			if err := json.Unmarshal(msg.Data, &node); err != nil {
				log.Printf("invalid message: %s\n", msg.Data)
				continue
			}
			if node.ID != h2.ID() {
				if _, ok := nodeList[node.ID]; !ok {
					log.Printf("Node %s joined the network\n", node.Name)
					nodeList[node.ID] = node
					if cfg.Name == "dmt" {
						createDMTJob(ctx, node, h2)
						createCameraJob(ctx, node, h2, msg)
						continue
					}
					if cfg.Name == "app" {
						createCameraJob(ctx, node, h2, msg)
						continue
					}
				}
			}
		}
	}()

	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	os.Exit(0)
}

func configureDHT(ctx context.Context, host host.Host, kademliaDHT *dht.IpfsDHT) {
	// log.Println("Announcing ourselves...")
	// routingDiscovery := routing.NewRoutingDiscovery(kademliaDHT)
	// dutil.Advertise(ctx, routingDiscovery, PosemeshService)
}
