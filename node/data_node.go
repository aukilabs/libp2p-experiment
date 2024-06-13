package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/config"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/google/uuid"
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

type OnDomainDataReceived func(context.Context, *Libposemesh.DomainData) error

// TODO: Implement the function StreamDomainData
func StreamDomainData(ctx context.Context, s io.Reader, writer io.Writer) (*Libposemesh.DomainData, error) {
	sizeBuf := make([]byte, flatbuffers.SizeUint32)
	_, err := io.ReadFull(s, sizeBuf)
	if err != nil {
		log.Printf("Failed to read message size: %s\n", err)
		return nil, err
	}
	dataOffset := flatbuffers.GetUint32(sizeBuf)

	beforeData := make([]byte, dataOffset)
	_, err = io.ReadFull(s, beforeData)
	if err != nil {
		log.Printf("Failed to read message content: %s\n", err)
		return nil, err
	}
	return Libposemesh.GetRootAsDomainData(beforeData, 0), nil
}

func ReceiveDomainDataHandler(h host.Host, onDomainDataReceivedFunc OnDomainDataReceived) func(network.Stream) {
	return func(s network.Stream) {
		log.Println("received messages")

		for {
			sizeBuf := make([]byte, flatbuffers.SizeUOffsetT)
			_, err := io.ReadFull(s, sizeBuf)
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("Failed to read message size: %s\n", err)
				return
			}
			size := flatbuffers.GetSizePrefix(sizeBuf, 0)
			domainDataBuf := make([]byte, size)
			_, err = io.ReadFull(s, domainDataBuf)
			if err == io.EOF {
				return
			}
			if err != nil {
				log.Printf("Failed to read message content: %s\n", err)
				return
			}
			domainData := Libposemesh.GetRootAsDomainData(domainDataBuf, 0)
			log.Printf("received domain data - %s: %d", domainData.Name(), size)
			// if err := onDomainDataReceivedFunc(context.Background(), domainData); err != nil {
			// 	log.Printf("Failed to process domain data: %s\n", err)
			// 	return
			// }
			classifyDomainData(context.Background(), domainData)
		}
	}
}

func classifyDomainData(ctx context.Context, domainData *Libposemesh.DomainData) error {
	dirPath := path.Join(basePath, string(domainData.DomainId())+"_"+string(domainData.Name())+"_"+domainData.DataType().String())
	err := os.MkdirAll(dirPath, os.ModePerm)
	if err != nil {
		log.Printf("Failed to create directory: %s\n", err)
		return err
	}

	unionData := new(flatbuffers.Table)
	if ok := domainData.Data(unionData); ok {
		switch domainData.DataType() {
		case Libposemesh.AnyDomainDataPartition:
			fmt.Printf("Processing partition data-%d-%d\n", len(unionData.Bytes), unionData.Pos)
			partition := new(Libposemesh.Partition)
			partition.Init(unionData.Bytes, unionData.Pos)
			return processPartition(ctx, dirPath, partition)
		}
	}
	return nil
}

func processPartition(ctx context.Context, dirPath string, partition *Libposemesh.Partition) error {
	dir, err := os.ReadDir(dirPath)
	if err != nil {
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
		p := Libposemesh.GetRootAsPartition(buf.Bytes(), 0)
		// merge the partitions
		partition = mergePartitions(ctx, partition, p)
		os.Remove(path.Join(dirPath, fi.Name()))
	}
	// write the merged partition to a file
	merged, err := os.Create(path.Join(dirPath, uuid.NewString()+".bin"))
	if err != nil {
		log.Printf("Failed to create file: %s\n", err)
		return err
	}
	defer merged.Close()
	builder := flatbuffers.NewBuilder(0)
	data := make([]byte, partition.DataLength())
	for i := 0; i < partition.DataLength(); i++ {
		b := partition.Data(i)
		if b {
			data[i] = 1
		} else {
			data[i] = 0
		}
	}
	pd := builder.CreateByteVector(data)
	Libposemesh.PartitionStart(builder)
	Libposemesh.PartitionAddData(builder, pd)
	pdd := Libposemesh.PartitionEnd(builder)
	builder.Finish(pdd)
	if _, err := io.Copy(merged, bytes.NewReader(builder.FinishedBytes())); err != nil {
		log.Printf("Failed to write data: %s\n", err)
		return err
	}
	log.Printf("Save partition data written to %s\n", merged.Name())
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

func DownloadDomainData(ctx context.Context, h host.Host) func(s network.Stream) error {
	return func(s network.Stream) error {
		log.Println("polling for domain data")

		for {
			// these should be generated by the domain node
			sizeBuf := make([]byte, flatbuffers.SizeUint32)
			_, err := io.ReadFull(s, sizeBuf)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				log.Printf("Failed to read message size: %s\n", err)
				return err
			}
			size := flatbuffers.GetUint32(sizeBuf)
			dataOffsetBuf := make([]byte, size)
			_, err = io.ReadFull(s, dataOffsetBuf)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				log.Printf("Failed to read message content: %s\n", err)
				return err
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
		builder := flatbuffers.NewBuilder(0)

		for i := 0; i < 10; i++ {
			time.Sleep(2 * time.Second)
			domainID := builder.CreateString(task.DomainPubKey)
			dataName := builder.CreateString(randomDomainDataName())
			content := builder.CreateByteVector(randomBitmap())

			Libposemesh.PartitionStart(builder)
			Libposemesh.PartitionAddData(builder, content)
			partition := Libposemesh.PartitionEnd(builder)
			fmt.Printf("Sending partition data-%d-%d-%d\n", content, builder.Offset(), len(builder.Bytes))

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

func randomDomainDataName() string {
	// generate random number from 0 to 10
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate a random number from 0 to 10
	num := rand.Intn(11)

	// Convert the number to a string and return it
	return "partition" + strconv.Itoa(num)
}

func randomBitmap() []byte {
	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create a slice to hold the bitmap
	bitmap := make([]byte, 20)

	// Fill the bitmap with random 0s and 1s
	for i := range bitmap {
		// Generate a random number, 0 or 1
		bit := byte(rand.Intn(2))
		bitmap[i] = bit
	}

	return bitmap
}
