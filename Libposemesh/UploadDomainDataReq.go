// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package Libposemesh

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type UploadDomainDataReq struct {
	_tab flatbuffers.Table
}

func GetRootAsUploadDomainDataReq(buf []byte, offset flatbuffers.UOffsetT) *UploadDomainDataReq {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &UploadDomainDataReq{}
	x.Init(buf, n+offset)
	return x
}

func FinishUploadDomainDataReqBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsUploadDomainDataReq(buf []byte, offset flatbuffers.UOffsetT) *UploadDomainDataReq {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &UploadDomainDataReq{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedUploadDomainDataReqBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *UploadDomainDataReq) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *UploadDomainDataReq) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *UploadDomainDataReq) Data(obj *DomainData, j int) bool {
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

func (rcv *UploadDomainDataReq) DataLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func UploadDomainDataReqStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func UploadDomainDataReqAddData(builder *flatbuffers.Builder, data flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(data), 0)
}
func UploadDomainDataReqStartDataVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(4, numElems, 4)
}
func UploadDomainDataReqEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
