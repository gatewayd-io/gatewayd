// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.1
// 	protoc        (unknown)
// source: raft/proto/raft.proto

package proto

import (
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type ForwardApplyRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Data          []byte                 `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	TimeoutMs     int64                  `protobuf:"varint,2,opt,name=timeout_ms,json=timeoutMs,proto3" json:"timeout_ms,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ForwardApplyRequest) Reset() {
	*x = ForwardApplyRequest{}
	mi := &file_raft_proto_raft_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ForwardApplyRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ForwardApplyRequest) ProtoMessage() {}

func (x *ForwardApplyRequest) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ForwardApplyRequest.ProtoReflect.Descriptor instead.
func (*ForwardApplyRequest) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{0}
}

func (x *ForwardApplyRequest) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *ForwardApplyRequest) GetTimeoutMs() int64 {
	if x != nil {
		return x.TimeoutMs
	}
	return 0
}

type ForwardApplyResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ForwardApplyResponse) Reset() {
	*x = ForwardApplyResponse{}
	mi := &file_raft_proto_raft_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ForwardApplyResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ForwardApplyResponse) ProtoMessage() {}

func (x *ForwardApplyResponse) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ForwardApplyResponse.ProtoReflect.Descriptor instead.
func (*ForwardApplyResponse) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{1}
}

func (x *ForwardApplyResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *ForwardApplyResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type AddPeerRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	PeerId        string                 `protobuf:"bytes,1,opt,name=peer_id,json=peerId,proto3" json:"peer_id,omitempty"`
	PeerAddress   string                 `protobuf:"bytes,2,opt,name=peer_address,json=peerAddress,proto3" json:"peer_address,omitempty"`
	GrpcAddress   string                 `protobuf:"bytes,3,opt,name=grpc_address,json=grpcAddress,proto3" json:"grpc_address,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddPeerRequest) Reset() {
	*x = AddPeerRequest{}
	mi := &file_raft_proto_raft_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddPeerRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddPeerRequest) ProtoMessage() {}

func (x *AddPeerRequest) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddPeerRequest.ProtoReflect.Descriptor instead.
func (*AddPeerRequest) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{2}
}

func (x *AddPeerRequest) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

func (x *AddPeerRequest) GetPeerAddress() string {
	if x != nil {
		return x.PeerAddress
	}
	return ""
}

func (x *AddPeerRequest) GetGrpcAddress() string {
	if x != nil {
		return x.GrpcAddress
	}
	return ""
}

type AddPeerResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AddPeerResponse) Reset() {
	*x = AddPeerResponse{}
	mi := &file_raft_proto_raft_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AddPeerResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddPeerResponse) ProtoMessage() {}

func (x *AddPeerResponse) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddPeerResponse.ProtoReflect.Descriptor instead.
func (*AddPeerResponse) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{3}
}

func (x *AddPeerResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *AddPeerResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type RemovePeerRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	PeerId        string                 `protobuf:"bytes,1,opt,name=peer_id,json=peerId,proto3" json:"peer_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RemovePeerRequest) Reset() {
	*x = RemovePeerRequest{}
	mi := &file_raft_proto_raft_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RemovePeerRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemovePeerRequest) ProtoMessage() {}

func (x *RemovePeerRequest) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemovePeerRequest.ProtoReflect.Descriptor instead.
func (*RemovePeerRequest) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{4}
}

func (x *RemovePeerRequest) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type RemovePeerResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Success       bool                   `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Error         string                 `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *RemovePeerResponse) Reset() {
	*x = RemovePeerResponse{}
	mi := &file_raft_proto_raft_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *RemovePeerResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RemovePeerResponse) ProtoMessage() {}

func (x *RemovePeerResponse) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RemovePeerResponse.ProtoReflect.Descriptor instead.
func (*RemovePeerResponse) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{5}
}

func (x *RemovePeerResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *RemovePeerResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

type GetPeerInfoRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	PeerId        string                 `protobuf:"bytes,1,opt,name=peer_id,json=peerId,proto3" json:"peer_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetPeerInfoRequest) Reset() {
	*x = GetPeerInfoRequest{}
	mi := &file_raft_proto_raft_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetPeerInfoRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPeerInfoRequest) ProtoMessage() {}

func (x *GetPeerInfoRequest) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPeerInfoRequest.ProtoReflect.Descriptor instead.
func (*GetPeerInfoRequest) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{6}
}

func (x *GetPeerInfoRequest) GetPeerId() string {
	if x != nil {
		return x.PeerId
	}
	return ""
}

type GetPeerInfoResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Exists        bool                   `protobuf:"varint,1,opt,name=exists,proto3" json:"exists,omitempty"`
	GrpcAddress   string                 `protobuf:"bytes,2,opt,name=grpc_address,json=grpcAddress,proto3" json:"grpc_address,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetPeerInfoResponse) Reset() {
	*x = GetPeerInfoResponse{}
	mi := &file_raft_proto_raft_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetPeerInfoResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetPeerInfoResponse) ProtoMessage() {}

func (x *GetPeerInfoResponse) ProtoReflect() protoreflect.Message {
	mi := &file_raft_proto_raft_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetPeerInfoResponse.ProtoReflect.Descriptor instead.
func (*GetPeerInfoResponse) Descriptor() ([]byte, []int) {
	return file_raft_proto_raft_proto_rawDescGZIP(), []int{7}
}

func (x *GetPeerInfoResponse) GetExists() bool {
	if x != nil {
		return x.Exists
	}
	return false
}

func (x *GetPeerInfoResponse) GetGrpcAddress() string {
	if x != nil {
		return x.GrpcAddress
	}
	return ""
}

var File_raft_proto_raft_proto protoreflect.FileDescriptor

var file_raft_proto_raft_proto_rawDesc = []byte{
	0x0a, 0x15, 0x72, 0x61, 0x66, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x72, 0x61, 0x66,
	0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x72, 0x61, 0x66, 0x74, 0x1a, 0x1c, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x48, 0x0a, 0x13, 0x46,
	0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x41, 0x70, 0x70, 0x6c, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x12, 0x1d, 0x0a, 0x0a, 0x74, 0x69, 0x6d, 0x65, 0x6f, 0x75,
	0x74, 0x5f, 0x6d, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x09, 0x74, 0x69, 0x6d, 0x65,
	0x6f, 0x75, 0x74, 0x4d, 0x73, 0x22, 0x46, 0x0a, 0x14, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64,
	0x41, 0x70, 0x70, 0x6c, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a,
	0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07,
	0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22, 0x6f, 0x0a,
	0x0e, 0x41, 0x64, 0x64, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12,
	0x17, 0x0a, 0x07, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x65, 0x65, 0x72,
	0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b,
	0x70, 0x65, 0x65, 0x72, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x67,
	0x72, 0x70, 0x63, 0x5f, 0x61, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x0b, 0x67, 0x72, 0x70, 0x63, 0x41, 0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x22, 0x41,
	0x0a, 0x0f, 0x41, 0x64, 0x64, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x65,
	0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x22, 0x2c, 0x0a, 0x11, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x50, 0x65, 0x65, 0x72, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x70, 0x65, 0x65, 0x72, 0x5f, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65, 0x65, 0x72, 0x49, 0x64, 0x22,
	0x44, 0x0a, 0x12, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12,
	0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x65, 0x72, 0x72, 0x6f, 0x72, 0x22, 0x2d, 0x0a, 0x12, 0x47, 0x65, 0x74, 0x50, 0x65, 0x65, 0x72,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x17, 0x0a, 0x07, 0x70,
	0x65, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x70, 0x65,
	0x65, 0x72, 0x49, 0x64, 0x22, 0x50, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x50, 0x65, 0x65, 0x72, 0x49,
	0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x65,
	0x78, 0x69, 0x73, 0x74, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x65, 0x78, 0x69,
	0x73, 0x74, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x67, 0x72, 0x70, 0x63, 0x5f, 0x61, 0x64, 0x64, 0x72,
	0x65, 0x73, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x67, 0x72, 0x70, 0x63, 0x41,
	0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x32, 0xf3, 0x02, 0x0a, 0x0b, 0x52, 0x61, 0x66, 0x74, 0x53,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x68, 0x0a, 0x0c, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72,
	0x64, 0x41, 0x70, 0x70, 0x6c, 0x79, 0x12, 0x19, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x46, 0x6f,
	0x72, 0x77, 0x61, 0x72, 0x64, 0x41, 0x70, 0x70, 0x6c, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1a, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64,
	0x41, 0x70, 0x70, 0x6c, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x21, 0x82,
	0xd3, 0xe4, 0x93, 0x02, 0x1b, 0x3a, 0x01, 0x2a, 0x22, 0x16, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x61,
	0x66, 0x74, 0x2f, 0x66, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64, 0x2d, 0x61, 0x70, 0x70, 0x6c, 0x79,
	0x12, 0x54, 0x0a, 0x07, 0x41, 0x64, 0x64, 0x50, 0x65, 0x65, 0x72, 0x12, 0x14, 0x2e, 0x72, 0x61,
	0x66, 0x74, 0x2e, 0x41, 0x64, 0x64, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x15, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x41, 0x64, 0x64, 0x50, 0x65, 0x65, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1c, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16,
	0x3a, 0x01, 0x2a, 0x22, 0x11, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x61, 0x66, 0x74, 0x2f, 0x61, 0x64,
	0x64, 0x2d, 0x70, 0x65, 0x65, 0x72, 0x12, 0x60, 0x0a, 0x0a, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65,
	0x50, 0x65, 0x65, 0x72, 0x12, 0x17, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x52, 0x65, 0x6d, 0x6f,
	0x76, 0x65, 0x50, 0x65, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e,
	0x72, 0x61, 0x66, 0x74, 0x2e, 0x52, 0x65, 0x6d, 0x6f, 0x76, 0x65, 0x50, 0x65, 0x65, 0x72, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x1f, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x19, 0x3a,
	0x01, 0x2a, 0x22, 0x14, 0x2f, 0x76, 0x31, 0x2f, 0x72, 0x61, 0x66, 0x74, 0x2f, 0x72, 0x65, 0x6d,
	0x6f, 0x76, 0x65, 0x2d, 0x70, 0x65, 0x65, 0x72, 0x12, 0x42, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x50,
	0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x12, 0x18, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x47,
	0x65, 0x74, 0x50, 0x65, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x19, 0x2e, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x47, 0x65, 0x74, 0x50, 0x65, 0x65, 0x72,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x2c, 0x5a, 0x2a,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77,
	0x61, 0x79, 0x64, 0x2d, 0x69, 0x6f, 0x2f, 0x67, 0x61, 0x74, 0x65, 0x77, 0x61, 0x79, 0x64, 0x2f,
	0x72, 0x61, 0x66, 0x74, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_raft_proto_raft_proto_rawDescOnce sync.Once
	file_raft_proto_raft_proto_rawDescData = file_raft_proto_raft_proto_rawDesc
)

func file_raft_proto_raft_proto_rawDescGZIP() []byte {
	file_raft_proto_raft_proto_rawDescOnce.Do(func() {
		file_raft_proto_raft_proto_rawDescData = protoimpl.X.CompressGZIP(file_raft_proto_raft_proto_rawDescData)
	})
	return file_raft_proto_raft_proto_rawDescData
}

var file_raft_proto_raft_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_raft_proto_raft_proto_goTypes = []any{
	(*ForwardApplyRequest)(nil),  // 0: raft.ForwardApplyRequest
	(*ForwardApplyResponse)(nil), // 1: raft.ForwardApplyResponse
	(*AddPeerRequest)(nil),       // 2: raft.AddPeerRequest
	(*AddPeerResponse)(nil),      // 3: raft.AddPeerResponse
	(*RemovePeerRequest)(nil),    // 4: raft.RemovePeerRequest
	(*RemovePeerResponse)(nil),   // 5: raft.RemovePeerResponse
	(*GetPeerInfoRequest)(nil),   // 6: raft.GetPeerInfoRequest
	(*GetPeerInfoResponse)(nil),  // 7: raft.GetPeerInfoResponse
}
var file_raft_proto_raft_proto_depIdxs = []int32{
	0, // 0: raft.RaftService.ForwardApply:input_type -> raft.ForwardApplyRequest
	2, // 1: raft.RaftService.AddPeer:input_type -> raft.AddPeerRequest
	4, // 2: raft.RaftService.RemovePeer:input_type -> raft.RemovePeerRequest
	6, // 3: raft.RaftService.GetPeerInfo:input_type -> raft.GetPeerInfoRequest
	1, // 4: raft.RaftService.ForwardApply:output_type -> raft.ForwardApplyResponse
	3, // 5: raft.RaftService.AddPeer:output_type -> raft.AddPeerResponse
	5, // 6: raft.RaftService.RemovePeer:output_type -> raft.RemovePeerResponse
	7, // 7: raft.RaftService.GetPeerInfo:output_type -> raft.GetPeerInfoResponse
	4, // [4:8] is the sub-list for method output_type
	0, // [0:4] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_raft_proto_raft_proto_init() }
func file_raft_proto_raft_proto_init() {
	if File_raft_proto_raft_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_raft_proto_raft_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_raft_proto_raft_proto_goTypes,
		DependencyIndexes: file_raft_proto_raft_proto_depIdxs,
		MessageInfos:      file_raft_proto_raft_proto_msgTypes,
	}.Build()
	File_raft_proto_raft_proto = out.File
	file_raft_proto_raft_proto_rawDesc = nil
	file_raft_proto_raft_proto_goTypes = nil
	file_raft_proto_raft_proto_depIdxs = nil
}
