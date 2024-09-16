package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"os"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/models"
	"github.com/aukilabs/go-libp2p-experiment/node"
	"github.com/aukilabs/go-libp2p-experiment/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var DiscoveryNodeCfg = config.Config{
	NodeTypes:      []string{config.DISCOVERY_NODE},
	Name:           "dds_1",
	Port:           "18804",
	Mode:           dht.ModeClient,
	BootstrapPeers: []string{},
	EnableRelay:    true,
}

var domainList = map[string]models.Domain{}
var portalList = map[string]*Libposemesh.Portal{}

func createPortalStreamHandler(portalTopic *pubsub.Topic) func(s network.Stream) {
	return func(s network.Stream) {
		sizebuf := make([]byte, 4)
		if _, err := s.Read(sizebuf); err != nil {
			log.Println(err)
			return
		}
		size := flatbuffers.GetSizePrefix(sizebuf, 0)
		buf := make([]byte, size)
		if _, err := s.Read(buf); err != nil {
			log.Println(err)
			return
		}
		portal := Libposemesh.GetRootAsPortal(buf, 0)
		portalList[string(portal.ShortId())] = portal
		if err := publishPortal(context.Background(), portalTopic, portal); err != nil {
			log.Printf("Failed to publish portal %s: %s\n", portal.ShortId(), err)
		}
		log.Printf("Portal %s created\n", portal.ShortId())
	}
}

func main() {
	var name = flag.String("name", "discovery 1", "discovery service name")
	port := flag.String("port", "18804", "port")
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: DiscoveryNodeCfg.NodeTypes,
	}
	DiscoveryNodeCfg.Name = *name
	DiscoveryNodeCfg.Port = *port
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &DiscoveryNodeCfg, func(h host.Host) {
		// _, err := relay.New(h)
		// if err != nil {
		// 	log.Fatalf("Failed to create relay: %s\n", err)
		// }

		portalTopic, domainTopic, err := SyncDomainsAndPortals(ctx, h, n.BasePath, n.PubSub)
		if err != nil {
			log.Fatalf("Failed to sync domains and portals: %s\n", err)
		}

		// load domains meta file from disk
		domains, err := os.ReadDir(n.BasePath + "/domains")
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Fatalf("Failed to read domains directory: %s\n", err)
			}
		}
		for _, d := range domains {
			if d.IsDir() {
				meta, err := os.ReadFile(n.BasePath + "/domains/" + d.Name() + "/meta.json")
				if err != nil {
					log.Printf("Failed to read domain %s meta file: %s\n", d.Name(), err)
					continue
				}
				var domain models.Domain
				if err := json.Unmarshal(meta, &domain); err != nil {
					log.Printf("Failed to unmarshal domain %s meta file: %s\n", d.Name(), err)
					continue
				}
				domainList[domain.ID] = domain
				publishDomain(ctx, domainTopic, &domain)
			}
		}
		h.SetStreamHandler(node.CREATE_DOMAIN_PROTOCOL_ID, func(s network.Stream) {
			defer s.Close()
			createdDomain, err := utils.InitializeDomainCluster(ctx, n, s)
			if err != nil {
				log.Println(err)
				return
			}
			if err := publishDomain(ctx, domainTopic, createdDomain); err != nil {
				log.Printf("Failed to publish domain %s: %s\n", createdDomain.ID, err)
				return
			}
			domainList[createdDomain.ID] = *createdDomain

			if err := json.NewEncoder(s).Encode(createdDomain); err != nil {
				log.Println(err)
				return
			}
		})
		h.SetStreamHandler(node.CREATE_PORTAL_PROTOCOL_ID, createPortalStreamHandler(portalTopic))
		h.SetStreamHandler(node.FIND_DOMAIN_PROTOCOL_ID, func(s network.Stream) {
			defer s.Close()
			idByte, err := io.ReadAll(s)
			if err != nil {
				log.Println(err)
				return
			}
			domainId := string(idByte)
			if d, ok := domainList[domainId]; ok {
				if err := json.NewEncoder(s).Encode(d); err != nil {
					log.Println(err)
				}
			} else {
				log.Printf("models.Domain %s not found\n", domainId)
			}
		})
		h.SetStreamHandler(node.UPDATE_DOMAIN_PROTOCOL_ID, func(s network.Stream) {
			defer s.Close()
			var d models.Domain
			if err := json.NewDecoder(s).Decode(&d); err != nil {
				log.Println(err)
				return
			}
		})
		h.SetStreamHandler(node.FETCH_DOMAINS_PROTOCOL_ID, func(s network.Stream) {
			defer s.Close()
			if err := json.NewEncoder(s).Encode(domainList); err != nil {
				log.Println(err)
			}
		})
	})
}

const PortalTopic = "portals"
const DomainTopic = "domains"

func publishPortal(ctx context.Context, portalTopic *pubsub.Topic, portal *Libposemesh.Portal) error {
	builder := flatbuffers.NewBuilder(0)
	defaultName := builder.CreateByteString(portal.DefaultName())
	shortId := builder.CreateByteString(portal.ShortId())

	Libposemesh.PortalStart(builder)
	Libposemesh.PortalAddShortId(builder, shortId)
	Libposemesh.PortalAddSize(builder, portal.Size())
	Libposemesh.PortalAddDefaultName(builder, defaultName)
	p := Libposemesh.PortalEnd(builder)
	builder.FinishSizePrefixed(p)

	return portalTopic.Publish(ctx, builder.FinishedBytes())
}

func publishPortals(ctx context.Context, portalTopic *pubsub.Topic) {
	for _, portal := range portalList {
		if err := publishPortal(ctx, portalTopic, portal); err != nil {
			log.Printf("Failed to publish portal %s: %s\n", portal.ShortId(), err)
		}
	}
}

func publishDomain(ctx context.Context, domainTopic *pubsub.Topic, domain *models.Domain) error {
	d, err := json.Marshal(domain)
	if err != nil {
		return err
	}
	return domainTopic.Publish(ctx, d)
}

func publishDomains(ctx context.Context, domainTopic *pubsub.Topic) {
	for _, domain := range domainList {
		if err := publishDomain(ctx, domainTopic, &domain); err != nil {
			log.Printf("Failed to publish domain %s: %s\n", domain.ID, err)
		}
	}
}

func receivePortals(ctx context.Context, portalSub *pubsub.Subscription, h host.Host, basePath string) {
	for {
		msg, err := portalSub.Next(ctx)
		if err != nil {
			log.Println("Failed to read message:", err)
			continue
		}
		if msg.GetFrom() != h.ID() {
			portal := Libposemesh.GetSizePrefixedRootAsPortal(msg.Data, 0)
			portalId := string(portal.ShortId())
			if _, ok := portalList[portalId]; !ok {
				portalList[portalId] = portal
				if err := os.MkdirAll(basePath+"/portals", os.ModePerm); err != nil {
					log.Printf("Failed to create directory: %s\n", err)
					continue
				}
				f, err := os.Create(basePath + "/portals/" + portalId)
				if err != nil {
					log.Printf("Failed to create file: %s\n", err)
					continue
				}
				defer f.Close()
				f.Write(msg.Data)
			}
		}
	}
}

func receiveDomains(ctx context.Context, domainSub *pubsub.Subscription, h host.Host, basePath string) {
	for {
		msg, err := domainSub.Next(ctx)
		if err != nil {
			log.Println("Failed to read message:", err)
			continue
		}
		if msg.GetFrom() != h.ID() {
			var d models.Domain
			if err := json.Unmarshal(msg.Data, &d); err != nil {
				log.Printf("invalid message: %s\n", msg.Data)
				continue
			}
			if err := os.MkdirAll(basePath+"/domains/"+d.ID, os.ModePerm); err != nil {
				log.Printf("Failed to create directory: %s\n", err)
				continue
			}
			f, err := os.Create(basePath + "/domains/" + d.ID + "/meta.json")
			if err != nil {
				log.Printf("Failed to create file: %s\n", err)
				continue
			}
			defer f.Close()
			f.Write(msg.Data)
		}
	}
}

func SyncDomainsAndPortals(ctx context.Context, h host.Host, basePath string, ps *pubsub.PubSub) (*pubsub.Topic, *pubsub.Topic, error) {
	portalTopic, err := ps.Join(PortalTopic)
	if err != nil {
		return nil, nil, err
	}
	defer portalTopic.Close()
	go publishPortals(ctx, portalTopic)
	portalSub, err := portalTopic.Subscribe()
	if err != nil {
		return nil, nil, err
	}
	go receivePortals(ctx, portalSub, h, basePath)

	domainTopic, err := ps.Join(DomainTopic)
	if err != nil {
		return nil, nil, err
	}
	defer domainTopic.Close()
	go publishDomains(ctx, domainTopic)
	domainSub, err := domainTopic.Subscribe()
	if err != nil {
		return nil, nil, err
	}

	go receiveDomains(ctx, domainSub, h, basePath)
	return portalTopic, domainTopic, nil
}
