package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	"github.com/aukilabs/go-libp2p-experiment/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var visionCfg = config.Config{
	NodeTypes:      []string{config.VISION_NODE},
	Name:           "cbd",
	Port:           "",
	Mode:           dht.ModeServer,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func main() {
	name := flag.String("name", "cbd", "app name")
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: visionCfg.NodeTypes,
	}
	visionCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &visionCfg, func(h host.Host) {
		h.SetStreamHandler(node.UPLOAD_IMAGE_PROTOCOL_ID, func(s network.Stream) {
			if err := utils.LoadFrameFromStream(s, n.BasePath, func(image *Libposemesh.Image) error {
				builder := flatbuffers.NewBuilder(0)

				if _, err := s.Write(builder.FinishedBytes()); err != nil {
					log.Printf("failed to send %d pose back: %s", image.Timestamp(), err)
				}
				return nil
			}); err != nil {
				log.Println("failed to load frames: ", err)
			}
		})
		// h.SetStreamHandler(node.ADD_JOB_PROTOCOL_ID, func(s network.Stream) {
		// 	if err := utils.LoadJobFromStream(s, func(j *Libposemesh.Job) error {
		// 		log.Printf("Job %s received\n", j.ID())
		// 		return nil
		// 	}); err != nil {
		// 		log.Println("failed to load job: ", err)
		// 	}
		// })
	})
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	<-c

	log.Printf("\rExiting...\n")

	os.Exit(0)
}
