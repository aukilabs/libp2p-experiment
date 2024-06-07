package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime/multipart"
	"os"
	"sync"

	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

var cameraAppCfg = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "app",
	Port:           "18800",
	Mode:           dht.ModeClient,
	BootstrapPeers: defaultBootstrapNodes,
}

func SendDataForPoseRefinement(ctx context.Context, h host.Host, dest peer.ID, files []fs.DirEntry, task *taskInfo) error {
	log.Printf("Uploading data to %s..........................\n", dest)
	s, err := h.NewStream(ctx, dest, ADAM_PROTOCOL_ID)
	if err != nil {
		log.Printf("Failed to open stream to %s: %s\n", dest, err)
		return err
	}
	defer s.Close()

	mw := multipart.NewWriter(s)
	mw.SetBoundary("file")
	defer mw.Close()
	part, err := mw.CreateFormFile("task_info", "file")
	if err != nil {
		log.Printf("Failed to create form file: %s\n", err)
		return err
	}
	taskList[task.ID] = *task
	if err := json.NewEncoder(part).Encode(task); err != nil {
		log.Printf("Failed to encode task info: %s\n", err)
		return err
	}
	for _, file := range files {
		part, err = mw.CreateFormFile("file", file.Name())
		if err != nil {
			log.Printf("Failed to create form file: %s\n", err)
			return err
		}
		p := basePath + "/input/" + file.Name()
		f, err := os.Open(p)
		if err != nil {
			log.Printf("Failed to get file info: %s\n", err)
			return err
		}
		defer func() {
			f.Close()
			os.Remove(p)
		}()
		if _, err := io.Copy(part, f); err != nil {
			log.Printf("Failed to copy file: %s\n", err)
			return err
		}
	}
	log.Printf("Uploaded data to %s\n", dest)
	return nil
}

var lock sync.Mutex

func createCameraJob(ctx context.Context, node nodeInfo, h2 host.Host, msg *pubsub.Message) {
	if len(domainList) == 0 {
		for _, node := range nodeList {
			if node.Type == DISCOVERY_NODE {
				s, err := h2.NewStream(ctx, node.ID, FETCH_DOMAINS_PROTOCOL_ID)
				if err != nil {
					log.Println(err)
					return
				}
				defer s.Close()
				if err := json.NewEncoder(s).Encode(domainsQuery{}); err != nil {
					log.Println(err)
					return
				}
				if err := json.NewDecoder(s).Decode(&domainList); err != nil {
					log.Println(err)
					return
				}
			}
		}
		if len(domainList) == 0 {
			if node.Type != DATA_NODE {
				delete(nodeList, node.ID)
			}
			log.Println("Waiting on domain creation")
			return
		}
	}
	if node.Type == ADAM_NODE {
		lock.Lock()
		defer lock.Unlock()
		if err := os.MkdirAll(basePath+"/input", os.ModePerm); err != nil {
			log.Printf("Failed to create directory: %s\n", err)
			delete(nodeList, node.ID)
			return
		}
		files, err := os.ReadDir(basePath + "/input")
		if err != nil {
			delete(nodeList, node.ID)
			log.Printf("Failed to read directory: %s\n", err)
			return
		}
		if len(files) == 0 {
			delete(nodeList, node.ID)
			return
		}
		log.Printf("Found a %s node, connecting to it %s", node.Type, msg.GetFrom())
		outputs := make([]outputInfo, 0)

		var domain domain
		for _, domain = range domainList {
			for _, dataNodeID := range domain.DataNodes {
				outputs = append(outputs, outputInfo{ID: dataNodeID, ProtocolID: DOMAIN_DATA_PROTOCOL_ID})
			}
			break
		}

		jobInfo := jobInfo{
			ID:           uuid.NewString(),
			Name:         "domain setup",
			DomainPubKey: "test",
			State:        "running",
		}
		jobList[jobInfo.ID] = jobInfo
		taskInfo := taskInfo{
			ID:           uuid.NewString(),
			JobID:        jobInfo.ID,
			Name:         "upload video to ADAM",
			DomainPubKey: domain.PublicKey,
			Steps: []stepInfo{{
				Name:    "save domain data to domain data node and replication node",
				Outputs: outputs,
			}},
		}
		fmt.Println(outputs)
		if err := SendDataForPoseRefinement(ctx, h2, msg.GetFrom(), files, &taskInfo); err != nil {
			log.Println("Failed to send data to ADAM node:", err)
		}
		delete(nodeList, node.ID)
	}
}
