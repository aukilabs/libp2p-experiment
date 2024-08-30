package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/job"
	"github.com/aukilabs/go-libp2p-experiment/node"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"gocv.io/x/gocv"
)

var RecorderCfg = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "thc",
	Port:           "",
	Mode:           dht.ModeClient,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

func logIntoCactusBackend(cactusBackendURL, user, psw string) (string, error) {
	url := cactusBackendURL + "/api/collections/users/auth-with-password"
	data := fmt.Sprintf(`{
        "identity": "%s", 
        "password": "%s"
    }`, user, psw)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Failed to login: %v", resp.Status)
	}
	defer resp.Body.Close()
	return resp.Header.Get("Authorization"), nil
}

func main() {
	name := flag.String("name", "dmt", "app name")
	cbdURL := flag.String("cbd", "", "CBD URL, you can either provide CBD URL, or it will use a random vision node")
	cactusBackendURL := flag.String("cactus-url", "", "Cactus backend URL")
	cactusBackendUser := flag.String("cactus-user", "", "Cactus backend user")
	cactusBackendPassword := flag.String("cactus-password", "", "Cactus backend password")
	domainID := flag.String("domain", "", "Domain ID")
	bootstrapNodes := flag.String("bootstrap-nodes", "", "bootstrap nodes, separated by comma")
	bootstrapNodesList := []string{}
	if bootstrapNodes != nil && *bootstrapNodes != "" {
		bootstrapNodesList = append(bootstrapNodesList, *bootstrapNodes)
	}
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	if cactusBackendURL == nil || *cactusBackendURL == "" {
		log.Fatal("cactus backend URL is required")
	}
	if cactusBackendUser == nil || *cactusBackendUser == "" {
		log.Fatal("cactus backend user is required")
	}
	if cactusBackendPassword == nil || *cactusBackendPassword == "" {
		log.Fatal("cactus backend password is required")
	}
	if domainID == nil || *domainID == "" {
		log.Fatal("domain ID is required")
	}
	// token, err := logIntoCactusBackend(*cactusBackendURL, *cactusBackendUser, *cactusBackendPassword)
	// if err != nil {
	// 	log.Fatalf("Failed to login to cactus backend: %v", err)
	// }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: RecorderCfg.NodeTypes,
	}
	RecorderCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &RecorderCfg, func(h host.Host) {
		j := job.Job{
			ID:   uuid.NewString(),
			Name: "Recorder",
		}
		slamJob := job.Job{
			ID:   uuid.NewString(),
			Name: "SLAM",
			Input: job.Peer{
				ID:      h.ID(),
				Handler: node.UPLOAD_IMAGE_PROTOCOL_ID,
			},
		}
		dests := n.FindNodes(config.SALVIA_NODE)
		for len(dests) == 0 {
			log.Println("No salvia node found. Retrying in 5 seconds...")
			time.Sleep(5 * time.Second)
			dests = n.FindNodes(config.SALVIA_NODE)
		}
		for _, dest := range dests {
			slamJob.Output = append(slamJob.Output, job.Peer{
				ID: dest,
			})
			slamJob.Steps = append(slamJob.Steps, job.Job{
				ID:   uuid.NewString(),
				Name: "create pose",
				Input: job.Peer{
					ID: dest,
				},
				Output: []job.Peer{{
					Addr: *cactusBackendURL,
				}},
			})
		}
		j.Steps = append(j.Steps, slamJob)

		cbdJob := job.Job{
			ID:   uuid.NewString(),
			Name: "CBD",
			Input: job.Peer{
				ID:      h.ID(),
				Handler: node.UPLOAD_IMAGE_PROTOCOL_ID,
			},
		}
		if cbdURL != nil && *cbdURL != "" {
			multiAddr, err := multiaddr.NewMultiaddr(*cbdURL)
			if err != nil {
				log.Fatalf("Failed to parse CBD URL: %v", err)
			}
			peerInfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
			if err != nil {
				log.Fatalf("Failed to parse CBD URL into peer: %v", err)
			}
			cbdJob.Output = append(cbdJob.Output, job.Peer{
				ID: peerInfo.ID,
			})
			cbdJob.Steps = append(cbdJob.Steps, job.Job{
				ID:   uuid.NewString(),
				Name: "create task",
				Input: job.Peer{
					ID: peerInfo.ID,
				},
				Output: []job.Peer{{
					Addr: *cactusBackendURL,
				}},
			})
		} else {
			visionNodes := n.FindNodes(config.VISION_NODE)
			for len(visionNodes) == 0 {
				log.Println("No vision node found. Retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
				visionNodes = n.FindNodes(config.VISION_NODE)
			}
			cbdJob.Output = append(cbdJob.Output, job.Peer{
				ID: visionNodes[0],
			})
			cbdJob.Steps = append(cbdJob.Steps, job.Job{
				ID:   uuid.NewString(),
				Name: "create task",
				Input: job.Peer{
					ID: visionNodes[0],
				},
				Output: []job.Peer{{
					Addr: *cactusBackendURL,
				}},
			})
		}
		j.Steps = append(j.Steps, cbdJob)
		go streamCameraFeed(ctx, h, &j)
	})
}

func streamCameraFeed(ctx context.Context, h host.Host, j *job.Job) {
	streams := make([]*bufio.ReadWriter, len(j.Steps))
	for i, j := range j.Steps {
		for _, dest := range j.Output {
			s, err := h.NewStream(ctx, dest.ID, dest.Handler)
			if err != nil {
				log.Printf("Failed to open stream to %s: %s\n", dest, err)
				return
			}
			rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))
			streams[i] = rw
			if len(j.Steps) == 0 {
				go func() {
					// read message from rw

				}()
			}
		}
	}

	cameraInput(func(b []byte) {
		// // Get timestamp
		// timestamp := time.Now().UnixNano()

		// // Pack timestamp and frame together
		// var message bytes.Buffer
		// binary.Write(&message, binary.LittleEndian, timestamp)
		// message.Write(b)
		// message.Write([]byte("\n"))

		builder := flatbuffers.NewBuilder(0)
		frameOffset := builder.CreateByteVector(b)
		Libposemesh.ImageStart(builder)
		Libposemesh.ImageAddTimestamp(builder, time.Now().UnixNano())
		Libposemesh.ImageAddFrame(builder, frameOffset)
		req := Libposemesh.ImageEnd(builder)
		builder.FinishSizePrefixed(req)

		// Send data
		for _, stream := range streams {
			_, err := stream.Write(builder.FinishedBytes())
			if err != nil {
				log.Printf("Error sending frame: %v", err)
				break
			}
		}
	})
}

type sendImage func([]byte)

func cameraInput(out sendImage) {
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		log.Fatalf("Error opening video capture device: %v", err)
	}
	defer webcam.Close()

	// Prepare an image matrix to hold the frames
	img := gocv.NewMat()
	defer img.Close()

	frameNum := 0
	divider := 4

	for {
		// Read a frame from the camera
		if ok := webcam.Read(&img); !ok {
			log.Println("Error reading frame from camera")
			break
		}
		if img.Empty() {
			continue
		}

		frameNum++
		if frameNum%divider != 0 {
			continue
		}

		// Encode frame as JPEG
		buf, err := gocv.IMEncode(gocv.JPEGFileExt, img)
		if err != nil {
			log.Printf("Error encoding frame: %v", err)
			continue
		}

		// Send frame
		out(buf.GetBytes())
	}
}
