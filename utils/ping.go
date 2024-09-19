package utils

import (
	"bufio"
	"context"
	"io"
	"log"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/node"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

func Ping(ctx context.Context, h host.Host, dest peer.ID) error {
	s, err := h.NewStream(ctx, dest, node.PING_PROTOCOL_ID)
	if err != nil {
		return err
	}

	if _, err := s.Write([]byte("ping\n")); err != nil {
		return err
	}
	log.Println("##################PING################,", dest)

	reader := bufio.NewReader(s)
	chunk, err := reader.ReadBytes('\n')
	if err == io.EOF {
		return nil
	}
	log.Printf("##################Received: %s##################\n", chunk)

	return nil
}

func PingStreamHandler(s network.Stream) {
	reader := bufio.NewReader(s)
	for {
		chunk, err := reader.ReadBytes('\n')
		if err == io.EOF {
			return
		}
		log.Printf("Handler Received: %s\n", chunk)

		if _, err := s.Write([]byte("pong\n")); err != nil {
			log.Fatal(err)
		}
		time.Sleep(2 * time.Second)
	}
}
