package models

import (
	"github.com/aukilabs/go-libp2p-experiment/node"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

type DomainCluster struct {
	Secret  []byte          `json:"secret"`
	Writer  node.NodeInfo   `json:"writer"`
	Readers []peer.ID       `json:"readers"`
	Others  []node.NodeInfo `json:"others"`
	Topic   *pubsub.Topic   `json:"-"`
}

type Domain struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Cluster DomainCluster `json:"cluster"`
}
