package main

import (
	"context"
	"flag"
	"log"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	flatbuffers "github.com/google/flatbuffers/go"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var DiscoveryNodeCfg = config.Config{
	NodeTypes:      []string{config.DISCOVERY_NODE},
	Name:           "dds_1",
	Port:           "18804",
	Mode:           dht.ModeClient,
	BootstrapPeers: []string{},
}

var domainList = map[string]*Libposemesh.Domain{}
var portalList = map[string]*Libposemesh.Portal{}

func createPortalStreamHandler(s network.Stream) {
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
	log.Printf("Portal %s created\n", portal.ShortId())
}

func createDomainStreamHandler(s network.Stream) {
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
	domainList[string(domain.Id())] = domain
	log.Printf("Domain %s created\n", domain.Id())
}

func main() {
	var name = flag.String("name", "discovery 1", "discovery service name")
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
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &DiscoveryNodeCfg, func(h host.Host) {
		h.SetStreamHandler(node.CREATE_DOMAIN_PROTOCOL_ID, createDomainStreamHandler)
		h.SetStreamHandler(node.CREATE_PORTAL_PROTOCOL_ID, createPortalStreamHandler)
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

				Libposemesh.GetSizePrefixedRootAsDomain(builder.FinishedBytes(), 0)

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

// portalTopic, err := ps.Join(PortalTopic)
// 		if err != nil {
// 			panic(err)
// 		}
// 		defer portalTopic.Close()
// 		go PublishPortals(ctx, portalTopic, h2)
// 		portalSub, err := portalTopic.Subscribe()
// 		if err != nil {
// 			panic(err)
// 		}
// go func() {
// 	for {
// 		msg, err := portalSub.Next(ctx)
// 		if err != nil {
// 			log.Println("Failed to read message:", err)
// 			continue
// 		}
// 		if msg.GetFrom() != h2.ID() {
// 			// pub, err := msg.GetFrom().ExtractPublicKey()
// 			// if err != nil {
// 			// 	log.Printf("Failed to extract public key: %s\n", err)
// 			// 	continue
// 			// }
// 			// sMsg := signedMsg{}
// 			var portal portal
// 			if err := json.Unmarshal(msg.Data, &portal); err != nil {
// 				log.Printf("invalid message: %s\n", msg.Data)
// 				continue
// 			}
// 			// verified, err := pub.Verify(sMsg.Msg, sMsg.Sig)
// 			// if err != nil {
// 			// 	log.Printf("Failed to verify message: %s\n", err)
// 			// 	continue
// 			// }
// 			// if verified {
// 			// 	portal := portal{}
// 			// 	if err := json.Unmarshal(sMsg.Msg, &portal); err != nil {
// 			// 		log.Printf("Failed to unmarshal portal: %s\n", err)
// 			// 		continue
// 			// 	}
// 			// 	log.Printf("Received portal: %s\n", portal)
// 			if _, ok := portalList[portal.ShortID]; !ok {
// 				portalList[portal.ShortID] = portal
// 				if err := os.MkdirAll(basePath+"/portals", os.ModePerm); err != nil {
// 					log.Printf("Failed to create directory: %s\n", err)
// 					continue
// 				}
// 				f, err := os.Create(basePath + "/portals/" + portal.ShortID)
// 				if err != nil {
// 					log.Printf("Failed to create file: %s\n", err)
// 					continue
// 				}
// 				defer f.Close()
// 				if err := json.NewEncoder(f).Encode(portal); err != nil {
// 					log.Printf("Failed to encode portal: %s\n", err)
// 					continue
// 				}
// 			}
// 			// }
// 		}
// 	}
// }()

// domainTopic, err := ps.Join(DomainTopic)
// if err != nil {
// 	panic(err)
// }
// defer domainTopic.Close()
// go PublishDomains(ctx, domainTopic, h2)
// domainSub, err := domainTopic.Subscribe()
// if err != nil {
// 	panic(err)
// }

// go func() {
// 	for {
// 		msg, err := domainSub.Next(ctx)
// 		if err != nil {
// 			log.Println("Failed to read message:", err)
// 			continue
// 		}
// 		if msg.GetFrom() != h2.ID() {
// 			domain := domain{}
// 			if err := json.Unmarshal(msg.Data, &domain); err != nil {
// 				log.Printf("invalid message: %s\n", msg.Data)
// 				continue
// 			}
// 			domainList[domain.PublicKey] = domain
// 			if err := os.MkdirAll(basePath+"/domains", os.ModePerm); err != nil {
// 				log.Printf("Failed to create directory: %s\n", err)
// 				continue
// 			}
// 			f, err := os.Create(basePath + "/domains/" + domain.PublicKey)
// 			if err != nil {
// 				log.Printf("Failed to create file: %s\n", err)
// 				continue
// 			}
// 			defer f.Close()
// 			if err := json.NewEncoder(f).Encode(domain); err != nil {
// 				log.Printf("Failed to encode portal: %s\n", err)
// 				continue
// 			}
// 		}
// 	}
// }()

// h2.SetStreamHandler(protocol.FETCH_DOMAINS_PROTOCOL_ID, LoadDomainsStreamHandler)
// h2.SetStreamHandler(protocol.CREATE_DOMAIN_PROTOCOL_ID, CreateDomainStreamHandler(ctx, domainTopic, h2))
// h2.SetStreamHandler(protocol.CREATE_PORTAL_PROTOCOL_ID, CreatePortalStreamHandler(ctx, portalTopic, h2))
