package main

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	flatbuffers "github.com/google/flatbuffers/go"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var dataNodeCfg = config.Config{
	NodeTypes:      []string{DATA_NODE},
	Name:           "persistent_domain_data_node_1",
	Port:           "18803",
	Mode:           dht.ModeClient,
	BootstrapPeers: defaultBootstrapNodes,
}
var replicationNodeCfg = config.Config{
	NodeTypes:      []string{DATA_NODE},
	Name:           "replication_1",
	Port:           "18806",
	Mode:           dht.ModeClient,
	BootstrapPeers: defaultBootstrapNodes,
}

func ReceiveDomainDataHandler(h host.Host) func(network.Stream) {
	return func(s network.Stream) {
		mr := multipart.NewReader(s, "file")
		var task taskInfo
		var p string
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Failed to read part: %s\n", err)
				return
			}
			if part.FormName() == "task_info" {
				if err := json.NewDecoder(part).Decode(&task); err != nil {
					log.Printf("Failed to decode task info: %s\n", err)
					return
				}
				log.Printf("Receiving data from %s.............................", s.Conn().RemotePeer())
				p = basePath + "/output/domain_data/" + task.DomainPubKey
				if err := os.MkdirAll(p, os.ModePerm); err != nil {
					log.Printf("Failed to create directory: %s\n", err)
					return
				}
				continue
			}
			f, err := os.Create(path.Join(p, part.FileName()))
			if err != nil {
				log.Printf("Failed to create file: %s\n", err)
				return
			}
			if _, err := io.Copy(f, part); err != nil {
				log.Printf("Failed to copy file: %s\n", err)
				return
			}
		}
		log.Printf("Saved domain data at %s\n", p)
		delete(nodeList, s.Conn().RemotePeer())
		if len(task.Steps) > 0 {
			outputs := task.Steps[0].Outputs
			task.Steps = task.Steps[1:]
			if err := SendDomainData(context.Background(), h, &task, outputs); err != nil {
				log.Printf("Failed to send domain data: %s\n", err)
			}
		}
	}
}

func SendDomainData(ctx context.Context, h host.Host, task *taskInfo, outputs []outputInfo) error {
	for _, output := range outputs {
		dest := output.ID
		s, err := h.NewStream(ctx, dest, protocol.ID(output.ProtocolID))
		if err != nil {
			log.Printf("Failed to open stream to %s: %s\n", dest, err)
			return err
		}
		defer s.Close()

		builder := flatbuffers.NewBuilder(1024)
		Libposemesh.DomainDataStart(builder)

		mw := multipart.NewWriter(s)
		mw.SetBoundary("file")
		defer mw.Close()
		part, err := mw.CreateFormFile("task_info", "file")
		if err != nil {
			log.Printf("Failed to create form file: %s\n", err)
			return err
		}
		if err := json.NewEncoder(part).Encode(task); err != nil {
			log.Printf("Failed to encode task info: %s\n", err)
			return err
		}
		if err := filepath.WalkDir(basePath+"/output/domain_data/"+task.DomainPubKey, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			part, err := mw.CreateFormFile("file", d.Name())
			if err != nil {
				return err
			}
			if _, err := io.Copy(part, f); err != nil {
				return err
			}
			return nil
		}); err != nil {
			log.Printf("Failed to walk directory: %s\n", err)
			return err
		}
	}
	return nil
}
