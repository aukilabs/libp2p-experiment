// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package Libposemesh

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type BasicNode struct {
	_tab flatbuffers.Table
}

func GetRootAsBasicNode(buf []byte, offset flatbuffers.UOffsetT) *BasicNode {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &BasicNode{}
	x.Init(buf, n+offset)
	return x
}

func FinishBasicNodeBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.Finish(offset)
}

func GetSizePrefixedRootAsBasicNode(buf []byte, offset flatbuffers.UOffsetT) *BasicNode {
	n := flatbuffers.GetUOffsetT(buf[offset+flatbuffers.SizeUint32:])
	x := &BasicNode{}
	x.Init(buf, n+offset+flatbuffers.SizeUint32)
	return x
}

func FinishSizePrefixedBasicNodeBuffer(builder *flatbuffers.Builder, offset flatbuffers.UOffsetT) {
	builder.FinishSizePrefixed(offset)
}

func (rcv *BasicNode) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *BasicNode) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *BasicNode) Info(obj *NodeInfo) *NodeInfo {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		x := o + rcv._tab.Pos
		if obj == nil {
			obj = new(NodeInfo)
		}
		obj.Init(rcv._tab.Bytes, x)
		return obj
	}
	return nil
}

func BasicNodeStart(builder *flatbuffers.Builder) {
	builder.StartObject(1)
}
func BasicNodeAddInfo(builder *flatbuffers.Builder, info flatbuffers.UOffsetT) {
	builder.PrependStructSlot(0, flatbuffers.UOffsetT(info), 0)
}
func BasicNodeEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
