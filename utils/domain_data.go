package utils

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/libp2p/go-libp2p/core/network"
)

func GetPartition(domainData *Libposemesh.DomainData) (*Libposemesh.Partition, error) {
	unionData := new(flatbuffers.Table)
	if !domainData.Data(unionData) {
		return nil, fmt.Errorf("failed to get data from domain data")
	}
	if domainData.DataType() != Libposemesh.AnyDomainDataPartition {
		return nil, fmt.Errorf("domain data is not partition")
	}
	partition := new(Libposemesh.Partition)
	partition.Init(unionData.Bytes, unionData.Pos)
	return partition, nil
}

func MatchDomainData(regExp string, path string) bool {
	matched, _ := regexp.MatchString(regExp, path)
	return matched
}

type DomainDataSubscribers struct {
	mutex       sync.RWMutex
	subscribers map[string][]network.Stream
}

func (d *DomainDataSubscribers) AddSubscriber(regExp string, s network.Stream) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.subscribers[regExp] = append(d.subscribers[regExp], s)
}
func (d *DomainDataSubscribers) RemoveSubscriber(regExp string, s network.Stream) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	for i, stream := range d.subscribers[regExp] {
		if stream == s {
			d.subscribers[regExp] = append(d.subscribers[regExp][:i], d.subscribers[regExp][i+1:]...)
			break
		}
	}
}
func (d *DomainDataSubscribers) FindSubscribers(key string) []network.Stream {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	streams := make([]network.Stream, 0)
	for regExp, v := range d.subscribers {
		if MatchDomainData(regExp, key) {
			streams = append(streams, v...)
		}
	}
	return streams
}

func NewDomainDataSubscribers() *DomainDataSubscribers {
	return &DomainDataSubscribers{subscribers: map[string][]network.Stream{}}
}

type OnDomainDataReceived func(context.Context, string, *Libposemesh.DomainData) error

func ReceiveDomainData(ctx context.Context, reader io.Reader, basePath string, onDomainDataReceivedFunc OnDomainDataReceived) error {
	for {
		sizeBuf := make([]byte, flatbuffers.SizeUOffsetT)
		_, err := io.ReadFull(reader, sizeBuf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read message size: %w", err)
		}
		size := flatbuffers.GetSizePrefix(sizeBuf, 0)
		domainDataBuf := make([]byte, size)
		_, err = io.ReadFull(reader, domainDataBuf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read message content: %w", err)
		}
		domainData := Libposemesh.GetRootAsDomainData(domainDataBuf, 0)
		dirPath := path.Join(basePath, "domaindata", string(domainData.DomainId()), string(domainData.Name())+"_"+domainData.DataType().String())
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return err
		}
		f, err := os.CreateTemp(dirPath, "*.bin")
		if err != nil {
			return err
		}
		if _, err := f.Write(sizeBuf); err != nil {
			f.Close()
			return err
		}
		if _, err := f.Write(domainDataBuf); err != nil {
			f.Close()
			return err
		}
		if err := onDomainDataReceivedFunc(ctx, basePath, domainData); err != nil {
			return fmt.Errorf("failed to process domain data: %w", err)
		}
	}
}

func RandomDomainDataName() string {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	num := rand.Intn(5)
	// Convert the number to a string and return it
	return strconv.Itoa(num)
}

func RandomBitmap() []byte {
	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	// Create a slice to hold the bitmap
	bitmap := make([]byte, 100)

	// Fill the bitmap with random 0s and 1s
	for i := range bitmap {
		// Generate a random number, 0 or 1
		bit := byte(rand.Intn(2))
		bitmap[i] = bit
	}

	return bitmap
}

func RandomFloat32() float32 {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	return rand.Float32()
}
