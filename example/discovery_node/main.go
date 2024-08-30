package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
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
	BootstrapPeers: config.DefaultBootstrapNodes,
}

var domainList = map[string]*Libposemesh.Domain{}
var portalList = map[string]*Libposemesh.Portal{}

func updateClusterSecret(domain *Libposemesh.Domain, clusterSecret string) *Libposemesh.Domain {
	builder := flatbuffers.NewBuilder(0)
	domainId := builder.CreateByteString(domain.Id())
	domainName := builder.CreateByteString(domain.Name())
	writer := builder.CreateByteString(domain.Writer())

	readerStrs := make([]flatbuffers.UOffsetT, domain.ReadersLength())
	for i := 0; i < domain.ReadersLength(); i++ {
		readerStrs[i] = builder.CreateByteString(domain.Readers(i))
	}
	Libposemesh.DomainStartReadersVector(builder, domain.ReadersLength())
	for i := domain.ReadersLength() - 1; i >= 0; i-- {
		builder.PrependUOffsetT(readerStrs[i])
	}
	readers := builder.EndVector(domain.ReadersLength())

	Libposemesh.DomainStart(builder)
	Libposemesh.DomainAddId(builder, domainId)
	Libposemesh.DomainAddName(builder, domainName)
	Libposemesh.DomainAddWriter(builder, writer)
	Libposemesh.DomainAddReaders(builder, readers)
	Libposemesh.DomainStartNodesVector(builder, domain.NodesLength())
	for i := domain.NodesLength() - 1; i >= 0; i-- {
		builder.PrependUOffsetT(builder.CreateByteString(domain.Nodes(i)))
	}
	nodes := builder.EndVector(domain.NodesLength())
	Libposemesh.DomainAddNodes(builder, nodes)
	clusterSecretOffset := builder.CreateString(clusterSecret)
	Libposemesh.DomainAddClusterSecret(builder, clusterSecretOffset)

	d := Libposemesh.DomainEnd(builder)
	builder.FinishSizePrefixed(d)

	return Libposemesh.GetSizePrefixedRootAsDomain(builder.FinishedBytes(), 0)
}

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

func createDomainStreamHandler(domainTopic *pubsub.Topic) func(s network.Stream) {
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
		domain := Libposemesh.GetRootAsDomain(buf, 0)
		domain = updateClusterSecret(domain, "cluster_secret")
		domainList[string(domain.Id())] = domain
		if err := publishDomain(context.Background(), domainTopic, domain); err != nil {
			log.Printf("Failed to publish domain %s: %s\n", domain.Id(), err)
		}
		log.Printf("Domain %s created\n", domain.Id())
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
		portalTopic, domainTopic, err := SyncDomainsAndPortals(ctx, h, n.BasePath, n.PubSub)
		if err != nil {
			log.Fatalf("Failed to sync domains and portals: %s\n", err)
		}
		h.SetStreamHandler(node.CREATE_DOMAIN_PROTOCOL_ID, createDomainStreamHandler(domainTopic))
		h.SetStreamHandler(node.CREATE_PORTAL_PROTOCOL_ID, createPortalStreamHandler(portalTopic))
		h.SetStreamHandler(node.FIND_DOMAIN_PROTOCOL_ID, func(s network.Stream) {
			defer s.Close()
			sizeBuf := make([]byte, 4)
			_, err := s.Read(sizeBuf)
			if err != nil {
				log.Printf("Failed to read size: %s\n", err)
				return
			}
			buf := make([]byte, flatbuffers.GetSizePrefix(sizeBuf, 0))
			_, err = s.Read(buf)
			if err != nil {
				log.Printf("Failed to read from stream: %s\n", err)
				return
			}
			downloadReq := Libposemesh.GetRootAsDownloadDomainDataReq(buf, 0)
			domainId := string(downloadReq.DomainId())
			if domain, ok := domainList[domainId]; ok {
				builder := flatbuffers.NewBuilder(0)
				domainID := builder.CreateByteString(domain.Id())
				domainName := builder.CreateByteString(domain.Name())
				writer := builder.CreateByteString(domain.Writer())

				readerStrs := make([]flatbuffers.UOffsetT, domain.ReadersLength())
				for i := 0; i < domain.ReadersLength(); i++ {
					readerStrs[i] = builder.CreateByteString(domain.Readers(i))
				}
				Libposemesh.DomainStartReadersVector(builder, domain.ReadersLength())
				for i := domain.ReadersLength() - 1; i >= 0; i-- {
					builder.PrependUOffsetT(readerStrs[i])
				}
				readers := builder.EndVector(domain.ReadersLength())

				Libposemesh.DomainStart(builder)
				Libposemesh.DomainAddId(builder, domainID)
				Libposemesh.DomainAddName(builder, domainName)
				Libposemesh.DomainAddWriter(builder, writer)
				Libposemesh.DomainAddReaders(builder, readers)
				d := Libposemesh.DomainEnd(builder)
				builder.FinishSizePrefixed(d)

				if _, err := s.Write(builder.FinishedBytes()); err != nil {
					log.Printf("Failed to write to stream: %s\n", err)
					return
				}
				s.Close()
			} else {
				log.Printf("Domain %s not found\n", domainId)
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

func publishDomain(ctx context.Context, domainTopic *pubsub.Topic, domain *Libposemesh.Domain) error {
	builder := flatbuffers.NewBuilder(0)
	domainId := builder.CreateByteString(domain.Id())
	domainName := builder.CreateByteString(domain.Name())
	writer := builder.CreateByteString(domain.Writer())

	readerStrs := make([]flatbuffers.UOffsetT, domain.ReadersLength())
	for i := 0; i < domain.ReadersLength(); i++ {
		readerStrs[i] = builder.CreateByteString(domain.Readers(i))
	}
	Libposemesh.DomainStartReadersVector(builder, domain.ReadersLength())
	for i := domain.ReadersLength() - 1; i >= 0; i-- {
		builder.PrependUOffsetT(readerStrs[i])
	}
	readers := builder.EndVector(domain.ReadersLength())

	Libposemesh.DomainStart(builder)
	Libposemesh.DomainAddId(builder, domainId)
	Libposemesh.DomainAddName(builder, domainName)
	Libposemesh.DomainAddWriter(builder, writer)
	Libposemesh.DomainAddReaders(builder, readers)
	Libposemesh.DomainStartNodesVector(builder, domain.NodesLength())
	for i := domain.NodesLength() - 1; i >= 0; i-- {
		builder.PrependUOffsetT(builder.CreateByteString(domain.Nodes(i)))
	}
	nodes := builder.EndVector(domain.NodesLength())
	Libposemesh.DomainAddNodes(builder, nodes)
	clusterSecret := builder.CreateByteString(domain.ClusterSecret())
	Libposemesh.DomainAddClusterSecret(builder, clusterSecret)

	d := Libposemesh.DomainEnd(builder)
	builder.FinishSizePrefixed(d)

	return domainTopic.Publish(ctx, builder.FinishedBytes())
}

func publishDomains(ctx context.Context, domainTopic *pubsub.Topic) {
	for _, domain := range domainList {
		if err := publishDomain(ctx, domainTopic, domain); err != nil {
			log.Printf("Failed to publish domain %s: %s\n", domain.Id(), err)
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
			domain := Libposemesh.GetSizePrefixedRootAsDomain(msg.Data, 0)
			domainId := string(domain.Id())
			domainList[domainId] = domain
			if err := os.MkdirAll(basePath+"/domains", os.ModePerm); err != nil {
				log.Printf("Failed to create directory: %s\n", err)
				continue
			}
			f, err := os.Create(basePath + "/domains/" + domainId)
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
