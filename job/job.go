package job

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Peer struct {
	ID      peer.ID     `json:"id"`
	Handler protocol.ID `json:"handler"`
	Addr    string      `json:"addr"`
}

type Job struct {
	ID     string
	Name   string
	Input  Peer
	Steps  []Job
	Output []Peer
}
