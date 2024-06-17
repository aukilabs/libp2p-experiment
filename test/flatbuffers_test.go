package test

import (
	"io"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	"github.com/aukilabs/go-libp2p-experiment/utils"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestDomainDataFlatbuffers(t *testing.T) {
	_, pub, err := crypto.GenerateKeyPair(
		crypto.Ed25519, // Select your key type. Ed25519 are nice short
		-1,             // Select key length when possible (i.e. RSA).
	)
	require.NoError(t, err)
	key, err := peer.IDFromPublicKey(pub)
	require.NoError(t, err)
	domainID := key.String()

	t.Run("should parse with no problem", func(t *testing.T) {
		builder := flatbuffers.NewBuilder(0)
		domainIDOffset := builder.CreateString(domainID)
		dataName := builder.CreateString(randomDomainDataName())
		content := builder.CreateByteVector(randomBitmap())

		Libposemesh.PartitionStart(builder)
		Libposemesh.PartitionAddData(builder, content)
		partition := Libposemesh.PartitionEnd(builder)

		Libposemesh.DomainDataStart(builder)
		Libposemesh.DomainDataAddDomainId(builder, domainIDOffset)
		Libposemesh.DomainDataAddName(builder, dataName)
		Libposemesh.DomainDataAddDataOffset(builder, uint32(builder.Offset()))
		Libposemesh.DomainDataAddDataType(builder, Libposemesh.AnyDomainDataPartition)
		Libposemesh.DomainDataAddData(builder, partition)
		domainData := Libposemesh.DomainDataEnd(builder)

		builder.FinishSizePrefixed(domainData)

		dd := Libposemesh.GetSizePrefixedRootAsDomainData(builder.FinishedBytes(), 0)
		require.Equal(t, domainID, string(dd.DomainId()))

		unionData := new(flatbuffers.Table)
		require.True(t, dd.Data(unionData))
		require.Equal(t, Libposemesh.AnyDomainDataPartition, dd.DataType())
		require.NotPanics(t, func() {
			partition := new(Libposemesh.Partition)
			partition.Init(unionData.Bytes, unionData.Pos)
		})
	})

	t.Run("should parse size prefixed with no problem", func(t *testing.T) {
		r, w := io.Pipe()

		count := 2
		go func() {
			builder := flatbuffers.NewBuilder(0)
			defer w.Close()
			for i := 0; i < count; i++ {
				domainIDOffset := builder.CreateString(domainID)
				dataName := builder.CreateString(randomDomainDataName())
				content := builder.CreateByteVector(randomBitmap())
				Libposemesh.PartitionStart(builder)
				Libposemesh.PartitionAddData(builder, content)
				partition := Libposemesh.PartitionEnd(builder)

				Libposemesh.DomainDataStart(builder)
				Libposemesh.DomainDataAddDomainId(builder, domainIDOffset)
				Libposemesh.DomainDataAddName(builder, dataName)
				Libposemesh.DomainDataAddDataOffset(builder, uint32(builder.Offset()))
				Libposemesh.DomainDataAddDataType(builder, Libposemesh.AnyDomainDataPartition)
				Libposemesh.DomainDataAddData(builder, partition)
				domainData := Libposemesh.DomainDataEnd(builder)

				builder.FinishSizePrefixed(domainData)
				w.Write(builder.FinishedBytes())
				builder.Reset()
				time.Sleep(2 * time.Second)
			}
		}()
		wg := sync.WaitGroup{}
		for i := 0; i < count; i++ {
			sizeBuf := make([]byte, flatbuffers.SizeUOffsetT)
			_, err := io.ReadFull(r, sizeBuf)
			require.NoError(t, err)
			size := flatbuffers.GetSizePrefix(sizeBuf, 0)

			buf := make([]byte, size)
			_, err = io.ReadFull(r, buf)
			require.NoError(t, err)

			dd := Libposemesh.GetRootAsDomainData(buf, 0)
			require.Equal(t, domainID, string(dd.DomainId()))

			dd2 := Libposemesh.GetRootAsDomainData(dd.Table().Bytes, 0)
			require.Equal(t, domainID, string(dd2.DomainId()))
			wg.Add(1)
			go func() {
				defer wg.Done()
				manageDomainData(t, dd)
			}()
		}
		wg.Wait()
	})
}

func manageDomainData(t *testing.T, dd *Libposemesh.DomainData) {
	partition, err := utils.GetPartition(dd)
	require.NoError(t, err)
	f, err := os.CreateTemp("", "partition-*.bin")
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Write(dd.Table().Bytes)
	require.NoError(t, err)

	_, err = f.Seek(0, 0)
	require.NoError(t, err)
	buf, err := io.ReadAll(f)
	require.NoError(t, err)
	ddd := Libposemesh.GetRootAsDomainData(buf, 0)
	p, err := utils.GetPartition(ddd)
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, partition.DataLength(), p.DataLength())
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
