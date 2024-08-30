package utils

import (
	"fmt"
	"io"
	"os"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/libp2p/go-libp2p/core/network"
)

func LoadFrameFromStream(reader network.Stream, basePath string, onImageReceivedFunc func(*Libposemesh.Image) error) error {
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
		image := Libposemesh.GetRootAsImage(frameBuf, 0)

		if err := os.MkdirAll(basePath+"/images", os.ModePerm); err != nil {
			return err
		}
		p := fmt.Sprintf("%s/images/%d", basePath, image.Timestamp())
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		for i := 0; i < image.FrameLength(); i++ {
			if _, err := f.Write([]byte{byte(image.Frame(i))}); err != nil {
				return err
			}
		}
		return onImageReceivedFunc(image)
	}
}
