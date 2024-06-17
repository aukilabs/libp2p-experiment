package node

import (
	"io"
	"os"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func initPeerIdentity(basePath string) crypto.PrivKey {
	var priv crypto.PrivKey
	f, err := os.Open(basePath + "/peer.key")
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	defer f.Close()
	if os.IsNotExist(err) {
		priv, _, err = crypto.GenerateKeyPair(
			crypto.Ed25519, // Select your key type. Ed25519 are nice short
			-1,             // Select key length when possible (i.e. RSA).
		)
		if err != nil {
			panic(err)
		}
		if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
			panic(err)
		}
		// save priv to cfg.PeerPrivateKeyPath
		f, err := os.Create(basePath + "/peer.key")
		if err != nil {
			panic(err)
		}
		content, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			panic(err)
		}
		if _, err := f.Write(content); err != nil {
			panic(err)
		}
		pid, _ := peer.IDFromPublicKey(priv.GetPublic())
		f, err = os.Create(basePath + "/" + pid.String())
		if err != nil {
			panic(err)
		}
	} else {
		content, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}
		priv, err = crypto.UnmarshalPrivateKey(content)
		if err != nil {
			panic(err)
		}
	}
	return priv
}
