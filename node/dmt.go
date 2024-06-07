package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var dmtCfg = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "dmt",
	Port:           "18808",
	Mode:           dht.ModeClient,
	BootstrapPeers: defaultBootstrapNodes,
}
var initialized = false

func createDMTJob(ctx context.Context, node nodeInfo, h2 host.Host) {
	if node.Type == DISCOVERY_NODE && !initialized {
		initialized = true
		CreateDomain(ctx, h2, node.ID)
		if err := CreatePortal(ctx, h2, node.ID); err != nil {
			log.Println(err)
		}
	}
}

func CreateDomain(ctx context.Context, h host.Host, domainService peer.ID) error {
	log.Printf("Creating domain to %s.............................\n", domainService.String())
	_, pub, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		return err
	}
	key, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return err
	}
	domain := domain{
		PublicKey:   key.String(),
		OwnerWallet: cfg.OwnerWallet,
		Name:        "domain",
	}
	// domainList[domain.PublicKey] = domain
	go func() {
		// retry to create domain every 10 seconds
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				for _, node := range nodeList {
					if node.Type == DATA_NODE {
						domain.DataNodes = append(domain.DataNodes, node.ID)
					}
				}
				if len(domain.DataNodes) == 0 {
					log.Println("No data nodes found")
					continue
				}

				s, err := h.NewStream(ctx, domainService, CREATE_DOMAIN_PROTOCOL_ID)
				if err != nil {
					log.Println(err)
					return
				}
				defer s.Close()
				if err := json.NewEncoder(s).Encode(domain); err != nil {
					log.Println(err)
					return
				}
				s.Close()
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func CreatePortal(ctx context.Context, h host.Host, domainService peer.ID) error {
	log.Printf("Creating portal to %s.............................\n", domainService.String())
	portal := portal{
		CreatedBy:   h.ID(),
		ShortID:     uuid.NewString(),
		DefaultName: "portal",
		Size:        1,
	}
	portalList[portal.ShortID] = portal
	s, err := h.NewStream(ctx, domainService, CREATE_PORTAL_PROTOCOL_ID)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := json.NewEncoder(s).Encode(portal); err != nil {
		return err
	}
	s.Close()
	log.Printf("Portal %s created\n", portal.ShortID)
	return nil
}
