package utils

import (
	"fmt"
	"io"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/libp2p/go-libp2p/core/network"
)

func LoadJobFromStream(reader network.Stream, onJobReceivedFunc func(*Libposemesh.Job) error) error {
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
		frameBuf := make([]byte, size)
		_, err = io.ReadFull(reader, frameBuf)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read message content: %w", err)
		}
		job := Libposemesh.GetRootAsJob(frameBuf, 0)

		return onJobReceivedFunc(job)
	}
}
