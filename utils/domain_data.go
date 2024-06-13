package utils

import (
	"fmt"

	"github.com/aukilabs/go-libp2p-experiment/Libposemesh"
	flatbuffers "github.com/google/flatbuffers/go"
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
