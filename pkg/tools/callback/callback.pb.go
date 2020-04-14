// Code generated by protoc-gen-go. DO NOT EDIT.
// source: callback.proto

package callback

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type DataKind int32

const (
	DataKind_Data  DataKind = 0
	DataKind_Close DataKind = 1
)

var DataKind_name = map[int32]string{
	0: "Data",
	1: "Close",
}

var DataKind_value = map[string]int32{
	"Data":  0,
	"Close": 1,
}

func (x DataKind) String() string {
	return proto.EnumName(DataKind_name, int32(x))
}

func (DataKind) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_6cf7fe261a3a1c45, []int{0}
}

// Callback binary stream
type CallbackData struct {
	Data                 []byte   `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	Kind                 DataKind `protobuf:"varint,2,opt,name=kind,proto3,enum=callback.DataKind" json:"kind,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *CallbackData) Reset()         { *m = CallbackData{} }
func (m *CallbackData) String() string { return proto.CompactTextString(m) }
func (*CallbackData) ProtoMessage()    {}
func (*CallbackData) Descriptor() ([]byte, []int) {
	return fileDescriptor_6cf7fe261a3a1c45, []int{0}
}

func (m *CallbackData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_CallbackData.Unmarshal(m, b)
}
func (m *CallbackData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_CallbackData.Marshal(b, m, deterministic)
}
func (m *CallbackData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CallbackData.Merge(m, src)
}
func (m *CallbackData) XXX_Size() int {
	return xxx_messageInfo_CallbackData.Size(m)
}
func (m *CallbackData) XXX_DiscardUnknown() {
	xxx_messageInfo_CallbackData.DiscardUnknown(m)
}

var xxx_messageInfo_CallbackData proto.InternalMessageInfo

func (m *CallbackData) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func (m *CallbackData) GetKind() DataKind {
	if m != nil {
		return m.Kind
	}
	return DataKind_Data
}

func init() {
	proto.RegisterEnum("callback.DataKind", DataKind_name, DataKind_value)
	proto.RegisterType((*CallbackData)(nil), "callback.CallbackData")
}

func init() { proto.RegisterFile("callback.proto", fileDescriptor_6cf7fe261a3a1c45) }

var fileDescriptor_6cf7fe261a3a1c45 = []byte{
	// 169 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4b, 0x4e, 0xcc, 0xc9,
	0x49, 0x4a, 0x4c, 0xce, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x80, 0xf1, 0x95, 0xbc,
	0xb8, 0x78, 0x9c, 0xa1, 0x6c, 0x97, 0xc4, 0x92, 0x44, 0x21, 0x21, 0x2e, 0x96, 0x94, 0xc4, 0x92,
	0x44, 0x09, 0x46, 0x05, 0x46, 0x0d, 0x9e, 0x20, 0x30, 0x5b, 0x48, 0x8d, 0x8b, 0x25, 0x3b, 0x33,
	0x2f, 0x45, 0x82, 0x49, 0x81, 0x51, 0x83, 0xcf, 0x48, 0x48, 0x0f, 0x6e, 0x18, 0x48, 0x87, 0x77,
	0x66, 0x5e, 0x4a, 0x10, 0x58, 0x5e, 0x4b, 0x9e, 0x8b, 0x03, 0x26, 0x22, 0xc4, 0xc1, 0xc5, 0x02,
	0x62, 0x0b, 0x30, 0x08, 0x71, 0x72, 0xb1, 0x3a, 0xe7, 0xe4, 0x17, 0xa7, 0x0a, 0x30, 0x1a, 0x45,
	0x70, 0xf1, 0xc3, 0x2c, 0x0b, 0x4e, 0x2d, 0x2a, 0xcb, 0x4c, 0x4e, 0x15, 0x72, 0xe5, 0xe2, 0xf7,
	0x48, 0xcc, 0x4b, 0xc9, 0x49, 0x85, 0x49, 0x14, 0x0b, 0x89, 0x21, 0x2c, 0x40, 0x76, 0x9a, 0x14,
	0x0e, 0x71, 0x0d, 0x46, 0x03, 0xc6, 0x24, 0x36, 0xb0, 0xbf, 0x8c, 0x01, 0x01, 0x00, 0x00, 0xff,
	0xff, 0x5d, 0xf2, 0x29, 0x0f, 0xe9, 0x00, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// CallbackServiceClient is the client API for CallbackService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type CallbackServiceClient interface {
	HandleCallbacks(ctx context.Context, opts ...grpc.CallOption) (CallbackService_HandleCallbacksClient, error)
}

type callbackServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCallbackServiceClient(cc grpc.ClientConnInterface) CallbackServiceClient {
	return &callbackServiceClient{cc}
}

func (c *callbackServiceClient) HandleCallbacks(ctx context.Context, opts ...grpc.CallOption) (CallbackService_HandleCallbacksClient, error) {
	stream, err := c.cc.NewStream(ctx, &_CallbackService_serviceDesc.Streams[0], "/callback.CallbackService/HandleCallbacks", opts...)
	if err != nil {
		return nil, err
	}
	x := &callbackServiceHandleCallbacksClient{stream}
	return x, nil
}

type CallbackService_HandleCallbacksClient interface {
	Send(*CallbackData) error
	Recv() (*CallbackData, error)
	grpc.ClientStream
}

type callbackServiceHandleCallbacksClient struct {
	grpc.ClientStream
}

func (x *callbackServiceHandleCallbacksClient) Send(m *CallbackData) error {
	return x.ClientStream.SendMsg(m)
}

func (x *callbackServiceHandleCallbacksClient) Recv() (*CallbackData, error) {
	m := new(CallbackData)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// CallbackServiceServer is the server API for CallbackService service.
type CallbackServiceServer interface {
	HandleCallbacks(CallbackService_HandleCallbacksServer) error
}

// UnimplementedCallbackServiceServer can be embedded to have forward compatible implementations.
type UnimplementedCallbackServiceServer struct {
}

func (*UnimplementedCallbackServiceServer) HandleCallbacks(srv CallbackService_HandleCallbacksServer) error {
	return status.Errorf(codes.Unimplemented, "method HandleCallbacks not implemented")
}

func RegisterCallbackServiceServer(s *grpc.Server, srv CallbackServiceServer) {
	s.RegisterService(&_CallbackService_serviceDesc, srv)
}

func _CallbackService_HandleCallbacks_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(CallbackServiceServer).HandleCallbacks(&callbackServiceHandleCallbacksServer{stream})
}

type CallbackService_HandleCallbacksServer interface {
	Send(*CallbackData) error
	Recv() (*CallbackData, error)
	grpc.ServerStream
}

type callbackServiceHandleCallbacksServer struct {
	grpc.ServerStream
}

func (x *callbackServiceHandleCallbacksServer) Send(m *CallbackData) error {
	return x.ServerStream.SendMsg(m)
}

func (x *callbackServiceHandleCallbacksServer) Recv() (*CallbackData, error) {
	m := new(CallbackData)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _CallbackService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "callback.CallbackService",
	HandlerType: (*CallbackServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "HandleCallbacks",
			Handler:       _CallbackService_HandleCallbacks_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "callback.proto",
}