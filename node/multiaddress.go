package node

import "github.com/multiformats/go-multiaddr"

func addDomainProtocol() error {
	domainProtocol := multiaddr.Protocol{
		Name:  "domain",
		Code:  999,
		VCode: multiaddr.CodeToVarint(999),
		Size:  42, // Ethereum address size
	}
	return multiaddr.AddProtocol(domainProtocol)
}
