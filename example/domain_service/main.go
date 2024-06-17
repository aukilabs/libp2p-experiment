package main

import (
	"bytes"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	"github.com/aukilabs/go-libp2p-experiment/node"
	"github.com/aukilabs/go-libp2p-experiment/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var domainServiceNodeCfg = config.Config{
	NodeTypes:      []string{config.DISCOVERY_NODE, config.DATA_NODE},
	Name:           "dds_1",
	Port:           "18804",
	Mode:           dht.ModeClient,
	BootstrapPeers: []string{},
}

var domainList = map[string]*Libposemesh.Domain{}
var portalList = map[string]*Libposemesh.Portal{}

func createPortalStreamHandler(s network.Stream) {
	sizebuf := make([]byte, 4)
	if _, err := s.Read(sizebuf); err != nil {
		log.Println(err)
		return
	}
	size := flatbuffers.GetSizePrefix(sizebuf, 0)
	buf := make([]byte, size)
	if _, err := s.Read(buf); err != nil {
		log.Println(err)
		return
	}
	portal := Libposemesh.GetSizePrefixedRootAsPortal(buf, 0)
	portalList[string(portal.ShortId())] = portal
	log.Printf("Portal %s created\n", portal.ShortId())
}

func createDomainStreamHandler(s network.Stream) {
	sizebuf := make([]byte, 4)
	if _, err := s.Read(sizebuf); err != nil {
		log.Println(err)
		return
	}
	size := flatbuffers.GetSizePrefix(sizebuf, 0)
	buf := make([]byte, size)
	if _, err := s.Read(buf); err != nil {
		log.Println(err)
		return
	}
	domain := Libposemesh.GetSizePrefixedRootAsDomain(buf, 0)
	domainList[string(domain.Id())] = domain
	log.Printf("Domain %s created\n", domain.Id())
}

var domainDataSubs = utils.NewDomainDataSubscribers()

func classifyDomainData(ctx context.Context, basePath string, domainData *Libposemesh.DomainData) error {
	dirPath := path.Join(basePath, "domaindata", string(domainData.DomainId()), string(domainData.Name())+"_"+domainData.DataType().String())
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		log.Printf("Failed to create directory: %s\n", err)
		return err
	}

	unionData := new(flatbuffers.Table)
	if ok := domainData.Data(unionData); ok {
		switch domainData.DataType() {
		case Libposemesh.AnyDomainDataPartition:
			return processPartition(ctx, dirPath, domainData)
		}
	}
	return nil
}

func processPartition(ctx context.Context, dirPath string, domainData *Libposemesh.DomainData) error {
	dir, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	partition, err := utils.GetPartition(domainData)
	if err != nil {
		log.Printf("Failed to get partition data: %s\n", err)
		return err
	}
	for _, fi := range dir {
		if fi.IsDir() {
			continue
		}
		f, err := os.Open(path.Join(dirPath, fi.Name()))
		if err != nil {
			log.Printf("Failed to open file: %s\n", err)
			return err
		}
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, f); err != nil {
			f.Close()
			log.Printf("Failed to read data: %s\n", err)
			return err
		}
		f.Close()
		dd := Libposemesh.GetSizePrefixedRootAsDomainData(buf.Bytes(), 0)
		p, err := utils.GetPartition(dd)
		if err != nil {
			log.Printf("Failed to get partition data from file: %s\n", err)
			return err
		}
		// merge the partitions
		partition = mergePartitions(ctx, partition, p)
		os.Remove(path.Join(dirPath, fi.Name()))
	}
	if len(dir) != 0 {
		builder := flatbuffers.NewBuilder(0)
		domainID := builder.CreateByteString(domainData.DomainId())
		name := builder.CreateByteString(domainData.Name())
		data := make([]byte, partition.DataLength())
		for i := 0; i < partition.DataLength(); i++ {
			b := partition.Data(i)
			if b {
				data[i] = 1
			} else {
				data[i] = 0
			}
		}
		// write the merged domain data to a file
		merged, err := os.CreateTemp(dirPath, "*.bin")
		if err != nil {
			log.Printf("Failed to create file: %s\n", err)
			return err
		}
		defer merged.Close()

		ver := uuid.NewString()
		version := builder.CreateString(ver)
		pd := builder.CreateByteVector(data)
		Libposemesh.PartitionStart(builder)
		Libposemesh.PartitionAddData(builder, pd)
		pdd := Libposemesh.PartitionEnd(builder)
		Libposemesh.DomainDataStart(builder)
		Libposemesh.DomainDataAddVersion(builder, version)
		Libposemesh.DomainDataAddDomainId(builder, domainID)
		Libposemesh.DomainDataAddName(builder, name)
		Libposemesh.DomainDataAddDataOffset(builder, uint32(builder.Offset()))
		Libposemesh.DomainDataAddDataType(builder, Libposemesh.AnyDomainDataPartition)
		Libposemesh.DomainDataAddData(builder, pdd)
		ddOffset := Libposemesh.DomainDataEnd(builder)
		builder.FinishSizePrefixed(ddOffset)

		if _, err := io.Copy(merged, bytes.NewReader(builder.FinishedBytes())); err != nil {
			log.Printf("Failed to write data: %s\n", err)
			return err
		}
		log.Printf("partition data written to %s\n", merged.Name())
		for _, s := range domainDataSubs.FindSubscribers(dirPath) {
			if s.Conn().IsClosed() {
				domainDataSubs.RemoveSubscriber(dirPath, s)
				continue
			}
			if _, err := s.Write(builder.FinishedBytes()); err != nil {
				log.Printf("Failed to write data to subscriber: %s\n", err)
				return err
			}
		}
	}
	return nil
}

func mergePartitions(ctx context.Context, p1 *Libposemesh.Partition, p2 *Libposemesh.Partition) *Libposemesh.Partition {
	log.Println("Merging partitions-", p1.DataLength(), p2.DataLength())
	if p1.DataLength() == p2.DataLength() {
		for i := 0; i < p1.DataLength(); i++ {
			p1.MutateData(i, p1.Data(i) || p2.Data(i))
		}
	}
	return p1
}

func onDownloadDomainDataReqReceived(ctx context.Context, basePath string) func(s network.Stream) {
	return func(s network.Stream) {
		defer s.Close()
		sizeBuf := make([]byte, 4)
		_, err := s.Read(sizeBuf)
		if err != nil {
			log.Printf("Failed to read size: %s\n", err)
			return
		}
		buf := make([]byte, flatbuffers.GetSizePrefix(sizeBuf, 0))
		_, err = s.Read(buf)
		if err != nil {
			log.Printf("Failed to read from stream: %s\n", err)
			return
		}
		downloadReq := Libposemesh.GetRootAsDownloadDomainDataReq(buf, 0)
		domainId := string(downloadReq.DomainId())
		basePath := path.Join(basePath, domainId)
		log.Println("polling for domain data ", basePath)
		name := string(downloadReq.Name())
		if name == "" {
			name = ".*"
		}
		dataType := Libposemesh.AnyDomainData(downloadReq.Type())
		dataTypeStr := ".*"
		if dataType != Libposemesh.AnyDomainDataNONE {
			dataTypeStr = dataType.String()
		}
		key := name + "_" + dataTypeStr

		domainDataSubs.AddSubscriber(key, s)
		de, err := os.ReadDir(basePath)
		if err != nil {
			log.Printf("Failed to read directory: %s\n", err)
			return
		}
		for _, fi := range de {
			if fi.IsDir() && utils.MatchDomainData(key, fi.Name()) {
				files, err := os.ReadDir(path.Join(basePath, fi.Name()))
				if err != nil {
					log.Printf("Failed to read directory: %s\n", err)
					return
				}
				if len(files) == 0 {
					continue
				}
				f, err := os.Open(path.Join(basePath, fi.Name(), files[0].Name()))
				if err != nil {
					log.Printf("Failed to open file: %s\n", err)
					continue
				}
				_, err = io.Copy(s, f)
				if err != nil {
					log.Printf("Failed to write data to stream: %s\n", err)
					return
				}
				f.Seek(0, 0)
				buf := make([]byte, 4)
				_, err = f.Read(buf)
				if err != nil {
					log.Printf("Failed to read size: %s\n", err)
					return
				}
				size := flatbuffers.GetSizePrefix(buf, 0)
				buf = make([]byte, size)
				_, err = f.Read(buf)
				if err != nil {
					log.Printf("Failed to read data: %s\n", err)
					return
				}
				log.Println("Data sent to client", f.Name(), size)
				f.Close()
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	}
}

func main() {
	var name = flag.String("name", "", "domain server name")
	flag.Parse()
	if name == nil || *name == "" {
		log.Fatal("name is required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	info := node.NodeInfo{
		Name:  *name,
		Types: domainServiceNodeCfg.NodeTypes,
	}
	domainServiceNodeCfg.Name = *name
	n, err := node.NewNode(info, "volume")
	if err != nil {
		log.Fatalf("Failed to create node: %s\n", err)
	}
	n.Start(ctx, &domainServiceNodeCfg, func(h host.Host) {
		h.SetStreamHandler(node.CREATE_DOMAIN_PROTOCOL_ID, createDomainStreamHandler)
		h.SetStreamHandler(node.CREATE_PORTAL_PROTOCOL_ID, createPortalStreamHandler)
		h.SetStreamHandler(node.UPLOAD_DOMAIN_DATA_PROTOCOL_ID, func(s network.Stream) {
			defer s.Close()
			if err := utils.ReceiveDomainData(ctx, s, n.BasePath, classifyDomainData); err != nil {
				log.Printf("Failed to receive domain data: %s\n", err)
				return
			}
		})
		h.SetStreamHandler(node.DOWNLOAD_DOMAIN_DATA_PROTOCOL_ID, onDownloadDomainDataReqReceived(ctx, path.Join(n.BasePath, "domaindata")))
	})
}

// portalTopic, err := ps.Join(PortalTopic)
// 		if err != nil {
// 			panic(err)
// 		}
// 		defer portalTopic.Close()
// 		go PublishPortals(ctx, portalTopic, h2)
// 		portalSub, err := portalTopic.Subscribe()
// 		if err != nil {
// 			panic(err)
// 		}
// go func() {
// 	for {
// 		msg, err := portalSub.Next(ctx)
// 		if err != nil {
// 			log.Println("Failed to read message:", err)
// 			continue
// 		}
// 		if msg.GetFrom() != h2.ID() {
// 			// pub, err := msg.GetFrom().ExtractPublicKey()
// 			// if err != nil {
// 			// 	log.Printf("Failed to extract public key: %s\n", err)
// 			// 	continue
// 			// }
// 			// sMsg := signedMsg{}
// 			var portal portal
// 			if err := json.Unmarshal(msg.Data, &portal); err != nil {
// 				log.Printf("invalid message: %s\n", msg.Data)
// 				continue
// 			}
// 			// verified, err := pub.Verify(sMsg.Msg, sMsg.Sig)
// 			// if err != nil {
// 			// 	log.Printf("Failed to verify message: %s\n", err)
// 			// 	continue
// 			// }
// 			// if verified {
// 			// 	portal := portal{}
// 			// 	if err := json.Unmarshal(sMsg.Msg, &portal); err != nil {
// 			// 		log.Printf("Failed to unmarshal portal: %s\n", err)
// 			// 		continue
// 			// 	}
// 			// 	log.Printf("Received portal: %s\n", portal)
// 			if _, ok := portalList[portal.ShortID]; !ok {
// 				portalList[portal.ShortID] = portal
// 				if err := os.MkdirAll(basePath+"/portals", os.ModePerm); err != nil {
// 					log.Printf("Failed to create directory: %s\n", err)
// 					continue
// 				}
// 				f, err := os.Create(basePath + "/portals/" + portal.ShortID)
// 				if err != nil {
// 					log.Printf("Failed to create file: %s\n", err)
// 					continue
// 				}
// 				defer f.Close()
// 				if err := json.NewEncoder(f).Encode(portal); err != nil {
// 					log.Printf("Failed to encode portal: %s\n", err)
// 					continue
// 				}
// 			}
// 			// }
// 		}
// 	}
// }()

// domainTopic, err := ps.Join(DomainTopic)
// if err != nil {
// 	panic(err)
// }
// defer domainTopic.Close()
// go PublishDomains(ctx, domainTopic, h2)
// domainSub, err := domainTopic.Subscribe()
// if err != nil {
// 	panic(err)
// }

// go func() {
// 	for {
// 		msg, err := domainSub.Next(ctx)
// 		if err != nil {
// 			log.Println("Failed to read message:", err)
// 			continue
// 		}
// 		if msg.GetFrom() != h2.ID() {
// 			domain := domain{}
// 			if err := json.Unmarshal(msg.Data, &domain); err != nil {
// 				log.Printf("invalid message: %s\n", msg.Data)
// 				continue
// 			}
// 			domainList[domain.PublicKey] = domain
// 			if err := os.MkdirAll(basePath+"/domains", os.ModePerm); err != nil {
// 				log.Printf("Failed to create directory: %s\n", err)
// 				continue
// 			}
// 			f, err := os.Create(basePath + "/domains/" + domain.PublicKey)
// 			if err != nil {
// 				log.Printf("Failed to create file: %s\n", err)
// 				continue
// 			}
// 			defer f.Close()
// 			if err := json.NewEncoder(f).Encode(domain); err != nil {
// 				log.Printf("Failed to encode portal: %s\n", err)
// 				continue
// 			}
// 		}
// 	}
// }()

// h2.SetStreamHandler(protocol.FETCH_DOMAINS_PROTOCOL_ID, LoadDomainsStreamHandler)
// h2.SetStreamHandler(protocol.CREATE_DOMAIN_PROTOCOL_ID, CreateDomainStreamHandler(ctx, domainTopic, h2))
// h2.SetStreamHandler(protocol.CREATE_PORTAL_PROTOCOL_ID, CreatePortalStreamHandler(ctx, portalTopic, h2))
