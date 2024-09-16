package utils

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aukilabs/go-libp2p-experiment/models"
	"github.com/aukilabs/go-libp2p-experiment/node"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func InitializeDomainCluster(ctx context.Context, n *node.Node, s network.Stream) (*models.Domain, error) {
	var d models.Domain
	if err := json.NewDecoder(s).Decode(&d); err != nil {
		log.Println(err)
		return nil, err
	}
	pk := node.InitPeerIdentity(n.BasePath + "/domains/" + d.Name + "/secret")
	pid, err := peer.IDFromPrivateKey(pk)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	d.ID = pid.String()

	pkb, err := crypto.MarshalPrivateKey(pk)
	if err != nil {
		log.Println("failed to get private key")
		return nil, err
	}
	d.Cluster.Secret = pkb
	log.Println("DOMAIN CREATED", d.ID)
	paddrs := []peer.AddrInfo{}
	cluster := append(d.Cluster.Others, d.Cluster.Writer)
	for _, otherNode := range cluster {
		addr, err := peer.AddrInfoFromString("/p2p/" + otherNode.ID.String())
		if err != nil {
			log.Println(err)
			continue
		}
		paddrs = append(paddrs, *addr)
		s, err := n.Host.NewStream(ctx, addr.ID, node.JOIN_DOMAIN_CLUSTER_ID)
		if err != nil {
			log.Println("failed to connect", err)
			continue
		}
		if err := json.NewEncoder(s).Encode(d); err != nil {
			log.Println("failed to join cluster")
			continue
		}
	}
	return &d, nil
}

func EnableDomainCluster(ctx context.Context, n *node.Node, domainList map[string]models.Domain) {
	n.Host.SetStreamHandler(node.JOIN_DOMAIN_CLUSTER_ID, func(s network.Stream) {
		var d models.Domain
		if err := json.NewDecoder(s).Decode(&d); err != nil {
			log.Println(err)
			return
		}
		if err := os.MkdirAll(n.BasePath+"/domains/"+d.Name, os.ModePerm); err != nil {
			log.Printf("Failed to create directory: %s\n", err)
			return
		}
		f, err := os.Create(n.BasePath + "/domains/" + d.Name + "/secret")
		if err != nil {
			log.Printf("Failed to create file: %s\n", err)
			return
		}
		defer f.Close()

		pk, err := crypto.UnmarshalPrivateKey(d.Cluster.Secret)
		if err != nil {
			log.Println("failed to unmarshal pri key", err)
			return
		}

		if _, err := f.Write(d.Cluster.Secret); err != nil {
			log.Println("failed to save secret", err)
			return
		}

		pid, err := peer.IDFromPrivateKey(pk)
		if err != nil {
			log.Println("failed to get peer id")
			return
		}

		if err := n.Host.Peerstore().AddPrivKey(pid, pk); err != nil {
			log.Println(err)
			return
		}
		ps, err := pubsub.NewGossipSub(ctx, n.Host, pubsub.WithMessageAuthor(pid))
		if err != nil {
			log.Println(err)
			return
		}
		dTopic, err := ps.Join(d.ID)
		if err != nil {
			log.Println(err)
			return
		}
		d.Cluster.Topic = dTopic
		domainList[d.ID] = d
		if err := os.MkdirAll(n.BasePath+"/domains/"+d.Name, os.ModePerm); err != nil {
			log.Printf("Failed to create directory: %s\n", err)
			return
		}
		df, err := os.Create(n.BasePath + "/domains/" + d.Name + "/meta.json")
		if err != nil {
			log.Printf("Failed to create file: %s\n", err)
			return
		}
		defer df.Close()

		if err := json.NewEncoder(df).Encode(d); err != nil {
			log.Println(err)
			return
		}

		// sub, err := dTopic.Subscribe()
		// if err != nil {
		// 	log.Println(err)
		// 	return
		// }
		// go func() {
		// 	for {
		// 		msg, err := sub.Next(ctx)
		// 		if err != nil {
		// 			log.Println("Failed to read message:", err)
		// 			continue
		// 		}

		// 	}
		// }()

	})
}
