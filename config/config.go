package config

import dht "github.com/libp2p/go-libp2p-kad-dht"

type Config struct {
	NodeTypes        []string    `cli:"node-types" usage:"The type of node to run" dft:"adam,domaindata"`
	Mode             dht.ModeOpt `cli:"mode" usage:"The mode of the node" dft:"server"`
	Port             string      `cli:"port" usage:"The port to listen on" dft:"4001"`
	BootstrapPeers   []string    `cli:"bootstrap-peers" usage:"The list of bootstrap peers to connect to" dft:"/ip4/12D3KooWQ3EFCYov3Lyi4YdFW2WvSqCixn9AUh3zBYLpggqS3yq9"`
	WalletPrivateKey string      `cli:"wallet-private-key" usage:"The private key of the wallet" dft:""`
	Name             string      `cli:"name" usage:"The name of the node" dft:""`
	OwnerWallet      string      `cli:"owner-wallet" usage:"The wallet of the owner" dft:""`
}
