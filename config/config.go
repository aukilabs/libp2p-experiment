package config

import dht "github.com/libp2p/go-libp2p-kad-dht"

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
}

var DefaultBootstrapNodes = []string{"/ip4/13.52.221.114/tcp/18804/p2p/12D3KooWKZXUGZCg982NtauvYKfCLLc7BuQXHd4sCUVqhTEFvGPA", "/ip4/127.0.0.1/tcp/18804/p2p/12D3KooWLarz8bTotktq3UXPxWGxtbWUb9SRxeCTnWppAMt9eXkr", "/ip4/127.0.0.1/tcp/18805/p2p/12D3KooWLQUrSJJ8PvZ3iWT5mw14qiiuux8eV6irZMZZNDc368xk"}
