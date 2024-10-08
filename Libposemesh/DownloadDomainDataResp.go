// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package Libposemesh

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type DownloadDomainDataResp struct {
	_tab flatbuffers.Table
}

func GetRootAsDownloadDomainDataResp(buf []byte, offset flatbuffers.UOffsetT) *DownloadDomainDataResp {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &DownloadDomainDataResp{}
	x.Init(buf, n+offset)
	return x
}

func FinishDownloadDomainDataRespBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsDownloadDomainDataResp(buf []byte, offset flatbuffers.UOffsetT) *DownloadDomainDataResp {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &DownloadDomainDataResp{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedDownloadDomainDataRespBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *DownloadDomainDataResp) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *DownloadDomainDataResp) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *DownloadDomainDataResp) Data(obj *DomainData, j int) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		x := rcv._tab.Vector(o)
		x += flatbuffers.UOffsetT(j) * 4
		x = rcv._tab.Indirect(x)
		obj.Init(rcv._tab.Bytes, x)
		return true
	}
	return false
}

func (rcv *DownloadDomainDataResp) DataLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func DownloadDomainDataRespStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func DownloadDomainDataRespAddData(builder *flatbuffers.Builder, data flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(data), 0)
}
func DownloadDomainDataRespStartDataVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(4, numElems, 4)
}
func DownloadDomainDataRespEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
