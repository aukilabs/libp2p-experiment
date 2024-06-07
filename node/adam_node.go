package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/config"

	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var adamNode2Cfg = config.Config{
	NodeTypes:      []string{ADAM_NODE},
	Name:           "adam_node_2",
	Port:           "18802",
	BootstrapPeers: defaultBootstrapNodes,
	Mode:           dht.ModeServer,
}

var adamNode1Cfg = config.Config{
	NodeTypes:      []string{ADAM_NODE},
	Name:           "adam_node_1",
	Port:           "18801",
	BootstrapPeers: defaultBootstrapNodes,
	Mode:           dht.ModeServer,
}

func ReceiveVideoForPoseRefinement(h host.Host) func(s network.Stream) {
	return func(s network.Stream) {
		log.Printf("Received a stream from %s\n", s.Conn().RemotePeer())
		mr := multipart.NewReader(s, "file")
		var task taskInfo
		var step stepInfo
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
				var curTask taskInfo
				if err := json.NewDecoder(part).Decode(&curTask); err != nil {
					log.Printf("Failed to decode task info: %s\n", err)
					return
				}
				if len(curTask.Steps) == 0 {
					log.Println("No steps found")
					return
				}
				step = curTask.Steps[0]
				if len(step.Outputs) == 0 {
					log.Println("Output is empty")
					return
				}
				task = taskInfo{
					ID:           uuid.NewString(),
					Name:         step.Name,
					DomainPubKey: curTask.DomainPubKey,
					JobID:        curTask.JobID,
					Steps:        curTask.Steps[1:],
					State:        "started",
				}
				taskList[task.ID] = task
				continue
			}
			p := basePath + "/input"
			if err := os.MkdirAll(p, os.ModePerm); err != nil {
				log.Printf("Failed to create directory: %s\n", err)
				return
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
		go func() {
			// open -n ./node/ADAM.app --args -batchmode -i../volume/adam_node_2/input -o../volume/adam_node_2/output/restult.zip -logfile ../volume/adam_node_2/output/log.txt
			cmd := exec.Command("open", "-n", "./node/ADAM.app", "--args", "-batchmode", "-i../"+basePath+"/input", "-o../"+basePath+"/output/output.zip", "-logfile", "../"+basePath+"/output/log.txt")
			var out bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err != nil {
				log.Println(fmt.Sprint(err) + ": " + stderr.String())
				return
			}
			log.Println("Processing input...................................")
			// look up for output.zip file in output folder every 2 minutes
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if _, err := os.Stat(basePath + "/output/output.zip"); err != nil {
						log.Println("Output not found")
						continue
					}
					task.State = "completed"
					taskList[task.ID] = task
					log.Println("Task complete, sending result...................................")
					if err := SendADAMResult(context.Background(), h, &task, step.Outputs); err != nil {
						log.Printf("Failed to send domain data: %s\n", err)
						return
					}
					return
				}
			}
			// time.Sleep(10 * time.Second)
			// task.State = "completed"
			// taskList[task.ID] = task
			// log.Println("Task complete, sending result...................................")
			// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			// defer cancel()
			// if err := SendADAMResult(ctx, h, &task, step.Outputs); err != nil {
			// 	log.Printf("Failed to send result: %s\n", err)
			// 	return
			// }
		}()
	}
}

func SendADAMResult(ctx context.Context, h host.Host, task *taskInfo, outputs []outputInfo) error {
	for _, output := range outputs {
		dest := output.ID
		s, err := h.NewStream(ctx, dest, protocol.ID(output.ProtocolID))
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
		if err := json.NewEncoder(part).Encode(task); err != nil {
			log.Printf("Failed to encode task info: %s\n", err)
			return err
		}
		part, err = mw.CreateFormFile("file", "adam_result.zip")
		if err != nil {
			log.Printf("Failed to create form file: %s\n", err)
			return err
		}
		f, err := os.Open(basePath + "/output/output.zip")
		if err != nil {
			log.Printf("Failed to get file info: %s\n", err)
			return err
		}
		defer func() {
			f.Close()
			os.Remove(basePath + "/output/output.zip")
		}()
		if _, err := io.Copy(part, f); err != nil {
			log.Printf("Failed to copy file: %s\n", err)
			return err
		}
	}
	return nil
}
