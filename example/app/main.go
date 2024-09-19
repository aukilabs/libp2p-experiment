package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	"github.com/aukilabs/go-libp2p-experiment/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

var CameraAppCfg = config.Config{
	NodeTypes:      []string{"client"},
	Name:           "camera",
	Port:           "18888",
	Mode:           dht.ModeClient,
	BootstrapPeers: config.DefaultBootstrapNodes,
}

var domainList = map[string]*Libposemesh.Domain{}
var portalList = map[string]*Libposemesh.Portal{}

func sendDomainData(ctx context.Context, h host.Host, task *node.JobInfo, outputs []node.OutputInfo) error {
	for _, output := range outputs {
		dest := output.ID
		s, err := h.NewStream(ctx, dest, protocol.ID(output.ProtocolID))
		if err != nil {
			log.Printf("Failed to open stream to %s: %s\n", dest, err)
			return err
		}
		defer s.Close()
		builder := flatbuffers.NewBuilder(0)

		for i := 0; i < 100; i++ {
			time.Sleep(2 * time.Second)
			domainID := builder.CreateString(task.DomainPubKey)
			dataName := builder.CreateString(utils.RandomDomainDataName())
			content := builder.CreateByteVector(utils.RandomBitmap())

			Libposemesh.PartitionStart(builder)
			Libposemesh.PartitionAddData(builder, content)
			partition := Libposemesh.PartitionEnd(builder)

			Libposemesh.DomainDataStart(builder)
			Libposemesh.DomainDataAddDomainId(builder, domainID)
			Libposemesh.DomainDataAddName(builder, dataName)
			Libposemesh.DomainDataAddDataOffset(builder, uint32(builder.Offset()))
			Libposemesh.DomainDataAddDataType(builder, Libposemesh.AnyDomainDataPartition)
			Libposemesh.DomainDataAddData(builder, partition)
			domainData := Libposemesh.DomainDataEnd(builder)

			builder.FinishSizePrefixed(domainData)
			if _, err := s.Write(builder.FinishedBytes()); err != nil {
				log.Printf("Failed to write domain data: %s\n", err)
				return err
			}

			log.Printf("Sent %d domain data %d/%d\n", i, domainData, len(builder.FinishedBytes()))
			builder.Reset()
		}
	}
	return nil
}

type visual struct {
	mutex sync.RWMutex
	list  [][]int
}

var v = visual{list: make([][]int, 5)}

func (v *visual) AddRow(rowIndex int, row []int) {
	v.mutex.Lock()
	defer v.mutex.Unlock()
	v.list[rowIndex] = row
}
func (v *visual) Print() {
	v.mutex.RLock()
	defer v.mutex.RUnlock()
	cmd := exec.Command("clear") //Linux example, its tested
	cmd.Stdout = os.Stdout
	cmd.Run()
	for _, row := range v.list {
		if len(row) == 0 {
			row = make([]int, 100)
		}
		for _, c := range row {
			if c == 1 {
				fmt.Print("▪️")
			} else {
				fmt.Print("_")
			}
		}
		fmt.Print("\n")
	}
}

func findDomainDataNode(ctx context.Context, h host.Host, dds peer.ID, domainID string, permission string) ([]peer.ID, error) {
	s, err := h.NewStream(ctx, dds, node.FIND_DOMAIN_PROTOCOL_ID)
	if err != nil {
		return nil, err
	}
	errCh := make(chan error)
	peerCh := make(chan []peer.ID)
	go func() {
		sizebuf := make([]byte, 4)
		if _, err := s.Read(sizebuf); err != nil {
			log.Println(err)
			errCh <- err
			return
		}
		size := flatbuffers.GetSizePrefix(sizebuf, 0)
		buf := make([]byte, size)
		if _, err := s.Read(buf); err != nil {
			log.Println(err)
			errCh <- err
			return
		}
		domain := Libposemesh.GetRootAsDomain(buf, 0)
		if domain.ReadersLength() == 0 {
			log.Println("No readers found")
			errCh <- fmt.Errorf("no readers found")
			return
		}
		res := make([]peer.ID, 0)
		if permission == "read" {
			for i := 0; i < domain.ReadersLength(); i++ {
				reader := domain.Readers(i)
				peerID, err := peer.Decode(string(reader))
				if err != nil {
					log.Println(err)
					continue
				}
				if peerID == h.ID() {
					log.Println("can't download data from self")
					continue
				}
				res = append(res, peerID)
			}
		} else {
			writer := domain.Writer()
			peerID, err := peer.Decode(string(writer))
			if err != nil {
				log.Println(err)
				errCh <- err
				return
			}
			if peerID == h.ID() {
				log.Println("can't download data from self")
				errCh <- fmt.Errorf("can't download data from self")
				return
			}
			res = append(res, peerID)
		}
		peerCh <- res
	}()

	builder := flatbuffers.NewBuilder(0)
	domainIDOffset := builder.CreateString(domainID)
	Libposemesh.DownloadDomainDataReqStart(builder)
	Libposemesh.DownloadDomainDataReqAddDomainId(builder, domainIDOffset)
	req := Libposemesh.DownloadDomainDataReqEnd(builder)
	builder.FinishSizePrefixed(req)
	if _, err := s.Write(builder.FinishedBytes()); err != nil {
		return nil, err
	}

	select {
	case err := <-errCh:
		return nil, err
	case res := <-peerCh:
		return res, nil
	}
}

func downloadDomainData(ctx context.Context, h host.Host, dds peer.ID, domainID, basePath string) error {
	log.Println("Downloading domain data from ", dds.String())
	s, err := h.NewStream(ctx, dds, node.DOWNLOAD_DOMAIN_DATA_PROTOCOL_ID)
	if err != nil {
		return err
	}
	v.Print()
	go func() {
		defer s.Close()

		if err := utils.ReceiveDomainData(ctx, s, basePath, func(ctx context.Context, s string, dd *Libposemesh.DomainData) error {
			index, _ := strconv.ParseInt(string(dd.Name()), 10, 64)
			partition, err := utils.GetPartition(dd)
			if err != nil {
				log.Println(err)
				return err
			}
			pd := make([]int, partition.DataLength())
			for i := 0; i < partition.DataLength(); i++ {
				if partition.Data(i) {
					pd[i] = 1
				} else {
					pd[i] = 0
				}
			}
			v.AddRow(int(index), pd)
			v.Print()
			return nil
		}); err != nil {
			log.Println(err)
			return
		}
	}()
	builder := flatbuffers.NewBuilder(0)
	domainIDOffset := builder.CreateString(domainID)
	Libposemesh.DownloadDomainDataReqStart(builder)
	Libposemesh.DownloadDomainDataReqAddDomainId(builder, domainIDOffset)
	req := Libposemesh.DownloadDomainDataReqEnd(builder)
	builder.FinishSizePrefixed(req)
	if _, err := s.Write(builder.FinishedBytes()); err != nil {
		return err
	}
	return nil
}

func createDomain(ctx context.Context, n *node.Node, domainService peer.ID) error {
	log.Printf("Creating domain to %s.............................\n", domainService.String())
	var dataNodes []peer.ID
	for len(dataNodes) == 0 {
		log.Printf("Waiting for data nodes.............................\n")
		dataNodes = n.FindNodes(config.DATA_NODE)
		if len(dataNodes) == 0 {
			time.Sleep(2 * time.Second)
		}
	}
	_, pub, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	if err != nil {
		return err
	}
	key, err := peer.IDFromPublicKey(pub)
	if err != nil {
		return err
	}
	builder := flatbuffers.NewBuilder(0)
	domainID := builder.CreateString(key.String())
	domainName := builder.CreateString("domain")
	writer := builder.CreateString(dataNodes[0].String())

	readerStrs := make([]flatbuffers.UOffsetT, len(dataNodes))
	for i, reader := range dataNodes {
		readerStrs[i] = builder.CreateString(reader.String())
	}
	Libposemesh.DomainStartReadersVector(builder, len(dataNodes))
	for i := len(dataNodes) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(readerStrs[i])
	}
	readers := builder.EndVector(len(dataNodes))

	Libposemesh.DomainStart(builder)
	Libposemesh.DomainAddId(builder, domainID)
	Libposemesh.DomainAddName(builder, domainName)
	Libposemesh.DomainAddWriter(builder, writer)
	Libposemesh.DomainAddReaders(builder, readers)
	d := Libposemesh.DomainEnd(builder)
	builder.FinishSizePrefixed(d)

	domain := Libposemesh.GetSizePrefixedRootAsDomain(builder.FinishedBytes(), 0)
	// decodedDomainID, err := peer.IDFromBytes(domain.Id())
	// if err != nil {

	// }
	domainList[key.String()] = domain

	s, err := n.Host.NewStream(ctx, domainService, node.CREATE_DOMAIN_PROTOCOL_ID)
	if _, err := s.Write(builder.FinishedBytes()); err != nil {
		return err
	}
	log.Println("Domain created", key.String())
	defer s.Close()
	return nil
}

func createPortal(ctx context.Context, h host.Host, domainService peer.ID) error {
	log.Printf("Creating portal to %s.............................\n", domainService.String())
	builder := flatbuffers.NewBuilder(0)
	id := uuid.NewString()
	portalShortId := builder.CreateString(id)
	name := builder.CreateString("portal")
	Libposemesh.PortalStart(builder)
	Libposemesh.PortalAddDefaultName(builder, name)
	Libposemesh.PortalAddShortId(builder, portalShortId)
	Libposemesh.PortalAddSize(builder, 5)
	portalOffset := Libposemesh.PortalEnd(builder)
	builder.FinishSizePrefixed(portalOffset)

	s, err := h.NewStream(ctx, domainService, node.CREATE_PORTAL_PROTOCOL_ID)
	if err != nil {
		return err
	}
	defer s.Close()
	if _, err := s.Write(builder.FinishedBytes()); err != nil {
		return err
	}
	portal := Libposemesh.GetSizePrefixedRootAsPortal(builder.FinishedBytes(), 0)
	portalList[id] = portal
	return nil
}

func main() {
	var domainId = flag.String("domainId", "", "domain id this app cares about")
	var mode = flag.String("mode", "", "sender or receiver or both or ping")
	var shouldCreateDomain = flag.Bool("createDomain", false, "create domain and portal")
	var name = flag.String("name", "dmt", "app name")
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: CameraAppCfg.NodeTypes,
	}
	CameraAppCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &CameraAppCfg, func(h host.Host) {
		discoveryNodes := n.FindNodes(config.DISCOVERY_NODE)
		for len(discoveryNodes) == 0 {
			fmt.Println("finding discovery nodes...")
			time.Sleep(2 * time.Second)
			discoveryNodes = n.FindNodes(config.DISCOVERY_NODE)
		}
		if *domainId != "" {
			if mode == nil || *mode != "receiver" {
				var dataNodes []peer.ID
				for i := 0; i < len(discoveryNodes); i++ {
					dataNodes, err = findDomainDataNode(ctx, h, discoveryNodes[0], *domainId, "write")
					if err != nil {
						log.Println(err)
					} else {
						break
					}
				}
				go func() {
					log.Println("sending domain data", len(dataNodes))
					for i := 0; i < len(dataNodes); i++ {
						if err := sendDomainData(ctx, h, &node.JobInfo{
							DomainPubKey: *domainId,
						}, []node.OutputInfo{{
							ID:         dataNodes[i],
							ProtocolID: node.UPLOAD_DOMAIN_DATA_PROTOCOL_ID,
						}}); err != nil {
							log.Println(err)
							continue
						}
					}
				}()
			}
			if mode == nil || *mode != "sender" {
				var dataNodes []peer.ID
				for i := 0; i < len(discoveryNodes); i++ {
					dataNodes, err = findDomainDataNode(ctx, h, discoveryNodes[0], *domainId, "read")
					if err != nil {
						log.Println(err)
					} else {
						break
					}
				}
				go func() {
					log.Println("downloading domain data")
					for i := 0; i < len(dataNodes); i++ {
						if err := downloadDomainData(ctx, h, dataNodes[i], *domainId, n.BasePath); err != nil {
							log.Println(err)
							continue
						}
					}
				}()
			}
		}
		if shouldCreateDomain != nil && *shouldCreateDomain {
			go func() {
				log.Println("creating domain and portal")
				for i := 0; i < len(discoveryNodes); i++ {
					if err := createDomain(ctx, n, discoveryNodes[i]); err != nil {
						continue
					}
					// } else if err := createPortal(ctx, h, discoveryNodes[i]); err != nil {
					// 	continue
					// } else {
					// 	break
					// }
				}
			}()
		}
		if mode == nil || *mode == "ping" {
			h.SetStreamHandler(node.PING_PROTOCOL_ID, utils.PingStreamHandler)
			addrStr := "/p2p/12D3KooWAS9CtrdimJqrwSvFzG868iRqFKs6xMrMPZcF2CEkixdC"
			relayAddrStr := "/ip4/13.52.221.114/udp/18804/quic-v1/p2p/12D3KooWKZXUGZCg982NtauvYKfCLLc7BuQXHd4sCUVqhTEFvGPA/p2p-circuit" + addrStr
			relayAddr, err := peer.AddrInfoFromString(relayAddrStr)
			if err != nil {
				log.Fatal("PARSE", err)
			}
			addr, err := peer.AddrInfoFromString(addrStr)
			if err != nil {
				log.Fatal("PARSE", err)
			}
			succ := false
			for !succ {
				if err := h.Connect(ctx, *relayAddr); err != nil {
					log.Println("failed to connect, ", addr.Addrs)
				} else {
					// // // succ = true
					// if err := utils.Ping(ctx, h, addr.ID); err != nil {
					// 	log.Println("PING", err)
					// } else {
					succ = true
					// }
				}
			}
		}
	})
}
