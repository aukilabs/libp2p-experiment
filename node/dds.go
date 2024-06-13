package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	flatbuffers "github.com/google/flatbuffers/go"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var defaultBootstrapNodes = []string{"/ip4/127.0.0.1/tcp/18804/p2p/12D3KooWJ86dj5s4AQRcNRhG3hqWioSsN3uGR55LVY7nx3eSttik", "/ip4/127.0.0.1/tcp/18805/p2p/12D3KooWLQUrSJJ8PvZ3iWT5mw14qiiuux8eV6irZMZZNDc368xk"}

var domainServiceNodeCfg = config.Config{
	NodeTypes:      []string{DISCOVERY_NODE},
	Name:           "dds_1",
	Port:           "18804",
	Mode:           dht.ModeClient,
	BootstrapPeers: []string{},
}

var domainServiceNode2Cfg = config.Config{
	NodeTypes:      []string{DISCOVERY_NODE},
	Name:           "dds_2",
	Port:           "18805",
	Mode:           dht.ModeClient,
	BootstrapPeers: []string{},
}

func PublishDomains(ctx context.Context, domainTopic *pubsub.Topic, h host.Host) {
	// priv := h.Peerstore().PrivKey(h.ID())
	// publish all portals every 2 minutes
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, d := range domainList {
				domainBytes, err := json.Marshal(d)
				if err != nil {
					log.Printf("Failed to marshal domain: %s\n", err)
					continue
				}
				// sig, err := priv.Sign(domainBytes)
				// if err != nil {
				// 	log.Printf("Failed to sign domain: %s\n", err)
				// 	continue
				// }
				// msg, err := json.Marshal(signedMsg{Sig: sig, Msg: domainBytes})
				// if err != nil {
				// 	log.Printf("Failed to marshal signed domain: %s\n", err)
				// 	continue
				// }
				if err := domainTopic.Publish(ctx, domainBytes); err != nil {
					log.Printf("Failed to publish domain: %s\n", err)
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func PublishPortals(ctx context.Context, portalTopic *pubsub.Topic, h host.Host) {
	// publish all portals every 2 minutes
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for _, p := range portalList {
				portalBytes, err := json.Marshal(p)
				if err != nil {
					log.Printf("Failed to marshal portal: %s\n", err)
					continue
				}
				// sig, err := priv.Sign(portalBytes)
				// if err != nil {
				// 	log.Printf("Failed to sign portal: %s\n", err)
				// 	continue
				// }
				// msg, err := json.Marshal(signedMsg{Sig: sig, Msg: portalBytes})
				// if err != nil {
				// 	log.Printf("Failed to marshal signed portal: %s\n", err)
				// 	continue
				// }
				if err := portalTopic.Publish(ctx, portalBytes); err != nil {
					log.Printf("Failed to publish portal: %s\n", err)
					continue
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func CreatePortalStreamHandler(ctx context.Context, portalTopic *pubsub.Topic, h host.Host) func(s network.Stream) {
	return func(s network.Stream) {
		// priv := h.Peerstore().PrivKey(h.ID())
		// portal := portal{
		// 	CreatedBy:   h.ID(),
		// 	ShortID:     uuid.NewString(),
		// 	DefaultName: "portal",
		// 	Size:        1,
		// }
		// portalBytes, err := json.Marshal(portal)
		// // if err != nil {
		// // 	return err
		// // }
		// // sig, err := priv.Sign(portalBytes)
		// // if err != nil {
		// // 	return err
		// // }
		// // msg, err := json.Marshal(signedMsg{Sig: sig, Msg: portalBytes})
		// // if err != nil {
		// // 	return err
		// // }
		var portal portal
		if err := json.NewDecoder(s).Decode(&portal); err != nil {
			log.Println(err)
			return
		}
		pb, err := json.Marshal(portal)
		if err != nil {
			log.Println(err)
			return
		}
		portalList[portal.ShortID] = portal
		if err := os.MkdirAll(basePath+"/portals", os.ModePerm); err != nil {
			log.Printf("Failed to create directory: %s\n", err)
			return
		}
		f, err := os.Create(basePath + "/portals/" + portal.ShortID)
		if err != nil {
			log.Printf("Failed to create file: %s\n", err)
			return
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(portal); err != nil {
			log.Printf("Failed to encode portal: %s\n", err)
			return
		}
		if err := portalTopic.Publish(ctx, pb); err != nil {
			return
		}
		return
	}
}

func CreateDomainStreamHandler(ctx context.Context, topic *pubsub.Topic, h host.Host) func(s network.Stream) {
	return func(s network.Stream) {
		var domain domain
		if err := json.NewDecoder(s).Decode(&domain); err != nil {
			log.Println(err)
			return
		}
		domainList[domain.PublicKey] = domain
		log.Printf("Domain %s created\n", domain.PublicKey)
		if err := os.MkdirAll(basePath+"/domains", os.ModePerm); err != nil {
			log.Printf("Failed to create directory: %s\n", err)
			return
		}
		f, err := os.Create(basePath + "/domains/" + domain.PublicKey)
		if err != nil {
			log.Printf("Failed to create file: %s\n", err)
			return
		}
		defer f.Close()
		if err := json.NewEncoder(f).Encode(domain); err != nil {
			log.Printf("Failed to encode portal: %s\n", err)
			return
		}
		db, err := json.Marshal(domain)
		if err != nil {
			log.Println(err)
			return
		}
		if err := topic.Publish(ctx, db); err != nil {
			log.Println(err)
			return
		}
		return
	}
}

type domainsQuery struct {
	PortalID string `json:"portal_id"`
}

func LoadDomainsStreamHandler(s network.Stream) {
	var query domainsQuery
	if err := json.NewDecoder(s).Decode(&query); err != nil {
		log.Println(err)
		return
	}
	if err := json.NewEncoder(s).Encode(domainList); err != nil {
		log.Println(err)
		return
	}
}

func DomainAuthHandler(s network.Stream) {
	defer s.Close()
	buf, err := io.ReadAll(s)
	if err != nil {
		s.Write([]byte("failed to read data"))
		log.Println(err)
		return
	}
	payload := Libposemesh.GetSizePrefixedRootAsAuthDomainReq(buf, 0)
	for _, domain := range domainList {
		if domain.PublicKey == string(payload.DomainId()) {
			builder := flatbuffers.NewBuilder(0)
			domainID := builder.CreateString(domain.PublicKey)
			domainName := builder.CreateString(domain.Name)
			writer := builder.CreateString(string(domain.DataNodes[0]))

			Libposemesh.DomainStartReadersVector(builder, len(domain.DataNodes[1:]))
			for _, reader := range domain.DataNodes[1:] {
				r := builder.CreateString(string(reader))
				builder.PrependUOffsetT(r)
			}
			readers := builder.EndVector(len(domain.DataNodes[1:]))

			Libposemesh.DomainStart(builder)
			Libposemesh.DomainAddId(builder, domainID)
			Libposemesh.DomainAddName(builder, domainName)
			Libposemesh.DomainAddWriter(builder, writer)
			Libposemesh.DomainAddReaders(builder, readers)
			d := Libposemesh.DomainEnd(builder)
			builder.FinishSizePrefixed(d)
			if _, err := s.Write(builder.FinishedBytes()); err != nil {
				log.Println(err)
			}
			return
		}
	}
}
