package main

import (
	"context"
	"flag"
	"log"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	"github.com/aukilabs/go-libp2p-experiment/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var SalviaCfg = config.Config{
	NodeTypes:      []string{config.SALVIA_NODE},
	Name:           "salvia1",
	Port:           "",
	Mode:           dht.ModeServer,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func main() {
	name := flag.String("name", "salvia1", "app name")
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: SalviaCfg.NodeTypes,
	}
	SalviaCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &SalviaCfg, func(h host.Host) {
		h.SetStreamHandler(node.UPLOAD_IMAGE_PROTOCOL_ID, func(s network.Stream) {
			if err := utils.LoadFrameFromStream(s, n.BasePath, func(image *Libposemesh.Image) error {
				// SKIP: generate pose

				builder := flatbuffers.NewBuilder(0)
				Libposemesh.PoseStart(builder)
				Libposemesh.PoseAddPx(builder, utils.RandomFloat32())
				Libposemesh.PoseAddPy(builder, utils.RandomFloat32())
				Libposemesh.PoseAddPz(builder, utils.RandomFloat32())
				Libposemesh.PoseAddRx(builder, utils.RandomFloat32())
				Libposemesh.PoseAddRy(builder, utils.RandomFloat32())
				Libposemesh.PoseAddRz(builder, utils.RandomFloat32())
				Libposemesh.PoseAddRw(builder, utils.RandomFloat32())
				end := Libposemesh.PoseEnd(builder)
				builder.FinishSizePrefixed(end)

				if _, err := s.Write(builder.FinishedBytes()); err != nil {
					log.Printf("failed to send %d pose back: %s", image.Timestamp(), err)
				}
				return nil
			}); err != nil {
				log.Println("failed to load frames: ", err)
			}
		})
	})
}
