package config

import (
	"flag"
	"log"
	"strings"

	dht "github.com/libp2p/go-libp2p-kad-dht"
)

const ADAM_NODE = "ADAM"
const DATA_NODE = "DATA"
const BOOSTRAP_NODE = "BOOTSTRAP"
const DISCOVERY_NODE = "DOMAIN_SERVICE"
const SALVIA_NODE = "SALVIA"
const VISION_NODE = "VISION"

type Config struct {
	NodeTypes           []string    `cli:"node-types" usage:"The type of node to run" dft:"adam,domaindata"`
	Mode                dht.ModeOpt `cli:"mode" usage:"The mode of the node" dft:"server"`
	Port                string      `cli:"port" usage:"The port to listen on" dft:"4001"`
	BootstrapPeers      []string    `cli:"bootstrap-peers" usage:"The list of bootstrap peers to connect to" dft:"/ip4/12D3KooWQ3EFCYov3Lyi4YdFW2WvSqCixn9AUh3zBYLpggqS3yq9"`
	WalletPrivateKey    string      `cli:"wallet-private-key" usage:"The private key of the wallet" dft:""`
	Name                string      `cli:"name" usage:"The name of the node" dft:""`
	OwnerWallet         string      `cli:"owner-wallet" usage:"The wallet of the owner" dft:""`
	DomainClusterSecret string      `cli:"domain-cluster-secret" usage:"Secret key for domain cluster" dft:""`
	Psk                 string      `cli:"psk" usage:"The pre-shared key for the node" dft:""`
	EnableRelay         bool        `cli:"enable-relay" usage:"Enable relay" dft:"false"`
}

var DefaultBootstrapNodes = []string{
	"/ip4/13.52.221.114/udp/18804/quic-v1/p2p/12D3KooWSpGa2SQ3iz9KrJrgyoZE2jZUtSx8nKCfNaXYp8FY5irE",
	// "/ip4/127.0.0.1/udp/18804/quic-v1/p2p/12D3KooWBeuStnFAFdTcjn8HH8bu6VKiUXHy6fLHJ4bv5fDU2mi9",
}

func LoadFromCliArgs(cfg *Config) {
	var name = flag.String("name", "", "node name")
	port := flag.String("port", "", "port")
	bootstrapPeers := flag.String("bootstrap", "", "comma-separated list of bootstrap multiaddresses")
	enableRelay := flag.Bool("relay", false, "enable relay")
	mode := flag.String("mode", "", "mode")
	flag.Parse()
	if (name == nil || *name == "") && cfg.Name == "" {
		log.Fatal("name is required")
	}
	if *name != "" {
		cfg.Name = *name
	}
	if *port != "" {
		cfg.Port = *port
	}
	if *enableRelay != false {
		cfg.EnableRelay = *enableRelay
	}
	if *bootstrapPeers != "" {
		cfg.BootstrapPeers = strings.Split(*bootstrapPeers, ",")
	}
	if mode != nil && *mode != "" {
		if *mode == "client" {
			cfg.Mode = dht.ModeClient
		} else {
			cfg.Mode = dht.ModeServer
		}
	}
}
