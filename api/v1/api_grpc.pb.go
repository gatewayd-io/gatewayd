// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: api/v1/api.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	GatewayDAdminAPIService_Version_FullMethodName         = "/api.v1.GatewayDAdminAPIService/Version"
	GatewayDAdminAPIService_GetGlobalConfig_FullMethodName = "/api.v1.GatewayDAdminAPIService/GetGlobalConfig"
	GatewayDAdminAPIService_GetPluginConfig_FullMethodName = "/api.v1.GatewayDAdminAPIService/GetPluginConfig"
	GatewayDAdminAPIService_GetPlugins_FullMethodName      = "/api.v1.GatewayDAdminAPIService/GetPlugins"
	GatewayDAdminAPIService_GetPools_FullMethodName        = "/api.v1.GatewayDAdminAPIService/GetPools"
	GatewayDAdminAPIService_GetProxies_FullMethodName      = "/api.v1.GatewayDAdminAPIService/GetProxies"
	GatewayDAdminAPIService_GetServers_FullMethodName      = "/api.v1.GatewayDAdminAPIService/GetServers"
	GatewayDAdminAPIService_GetPeers_FullMethodName        = "/api.v1.GatewayDAdminAPIService/GetPeers"
	GatewayDAdminAPIService_AddPeer_FullMethodName         = "/api.v1.GatewayDAdminAPIService/AddPeer"
	GatewayDAdminAPIService_RemovePeer_FullMethodName      = "/api.v1.GatewayDAdminAPIService/RemovePeer"
)

// GatewayDAdminAPIServiceClient is the client API for GatewayDAdminAPIService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// GatewayDAdminAPIService is the administration API of GatewayD.
type GatewayDAdminAPIServiceClient interface {
	// Version returns the version of the GatewayD.
	Version(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*VersionResponse, error)
	// GetGlobalConfig returns the global configuration of the GatewayD.
	GetGlobalConfig(ctx context.Context, in *Group, opts ...grpc.CallOption) (*structpb.Struct, error)
	// GetPluginConfig returns the configuration of the specified plugin.
	GetPluginConfig(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error)
	// GetPlugins returns the list of plugins installed on the GatewayD.
	GetPlugins(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PluginConfigs, error)
	// GetPools returns the list of pools configured on the GatewayD.
	GetPools(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error)
	// GetProxies returns the list of proxies configured on the GatewayD.
	GetProxies(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error)
	// GetServers returns the list of servers configured on the GatewayD.
	GetServers(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error)
	// GetPeers returns information about all peers in the Raft cluster
	GetPeers(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error)
	// AddPeer adds a new peer to the Raft cluster
	AddPeer(ctx context.Context, in *AddPeerRequest, opts ...grpc.CallOption) (*AddPeerResponse, error)
	// RemovePeer removes an existing peer from the Raft cluster
	RemovePeer(ctx context.Context, in *RemovePeerRequest, opts ...grpc.CallOption) (*RemovePeerResponse, error)
}

type gatewayDAdminAPIServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewGatewayDAdminAPIServiceClient(cc grpc.ClientConnInterface) GatewayDAdminAPIServiceClient {
	return &gatewayDAdminAPIServiceClient{cc}
}

func (c *gatewayDAdminAPIServiceClient) Version(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*VersionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VersionResponse)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_Version_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetGlobalConfig(ctx context.Context, in *Group, opts ...grpc.CallOption) (*structpb.Struct, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(structpb.Struct)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetGlobalConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetPluginConfig(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(structpb.Struct)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetPluginConfig_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetPlugins(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*PluginConfigs, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PluginConfigs)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetPlugins_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetPools(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(structpb.Struct)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetPools_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetProxies(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(structpb.Struct)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetProxies_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetServers(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(structpb.Struct)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetServers_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) GetPeers(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*structpb.Struct, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(structpb.Struct)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_GetPeers_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) AddPeer(ctx context.Context, in *AddPeerRequest, opts ...grpc.CallOption) (*AddPeerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(AddPeerResponse)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_AddPeer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gatewayDAdminAPIServiceClient) RemovePeer(ctx context.Context, in *RemovePeerRequest, opts ...grpc.CallOption) (*RemovePeerResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RemovePeerResponse)
	err := c.cc.Invoke(ctx, GatewayDAdminAPIService_RemovePeer_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GatewayDAdminAPIServiceServer is the server API for GatewayDAdminAPIService service.
// All implementations must embed UnimplementedGatewayDAdminAPIServiceServer
// for forward compatibility.
//
// GatewayDAdminAPIService is the administration API of GatewayD.
type GatewayDAdminAPIServiceServer interface {
	// Version returns the version of the GatewayD.
	Version(context.Context, *emptypb.Empty) (*VersionResponse, error)
	// GetGlobalConfig returns the global configuration of the GatewayD.
	GetGlobalConfig(context.Context, *Group) (*structpb.Struct, error)
	// GetPluginConfig returns the configuration of the specified plugin.
	GetPluginConfig(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	// GetPlugins returns the list of plugins installed on the GatewayD.
	GetPlugins(context.Context, *emptypb.Empty) (*PluginConfigs, error)
	// GetPools returns the list of pools configured on the GatewayD.
	GetPools(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	// GetProxies returns the list of proxies configured on the GatewayD.
	GetProxies(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	// GetServers returns the list of servers configured on the GatewayD.
	GetServers(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	// GetPeers returns information about all peers in the Raft cluster
	GetPeers(context.Context, *emptypb.Empty) (*structpb.Struct, error)
	// AddPeer adds a new peer to the Raft cluster
	AddPeer(context.Context, *AddPeerRequest) (*AddPeerResponse, error)
	// RemovePeer removes an existing peer from the Raft cluster
	RemovePeer(context.Context, *RemovePeerRequest) (*RemovePeerResponse, error)
	mustEmbedUnimplementedGatewayDAdminAPIServiceServer()
}

// UnimplementedGatewayDAdminAPIServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGatewayDAdminAPIServiceServer struct{}

func (UnimplementedGatewayDAdminAPIServiceServer) Version(context.Context, *emptypb.Empty) (*VersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Version not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetGlobalConfig(context.Context, *Group) (*structpb.Struct, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGlobalConfig not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetPluginConfig(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPluginConfig not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetPlugins(context.Context, *emptypb.Empty) (*PluginConfigs, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPlugins not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetPools(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPools not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetProxies(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetProxies not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetServers(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetServers not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) GetPeers(context.Context, *emptypb.Empty) (*structpb.Struct, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPeers not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) AddPeer(context.Context, *AddPeerRequest) (*AddPeerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddPeer not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) RemovePeer(context.Context, *RemovePeerRequest) (*RemovePeerResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemovePeer not implemented")
}
func (UnimplementedGatewayDAdminAPIServiceServer) mustEmbedUnimplementedGatewayDAdminAPIServiceServer() {
}
func (UnimplementedGatewayDAdminAPIServiceServer) testEmbeddedByValue() {}

// UnsafeGatewayDAdminAPIServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GatewayDAdminAPIServiceServer will
// result in compilation errors.
type UnsafeGatewayDAdminAPIServiceServer interface {
	mustEmbedUnimplementedGatewayDAdminAPIServiceServer()
}

func RegisterGatewayDAdminAPIServiceServer(s grpc.ServiceRegistrar, srv GatewayDAdminAPIServiceServer) {
	// If the following call pancis, it indicates UnimplementedGatewayDAdminAPIServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&GatewayDAdminAPIService_ServiceDesc, srv)
}

func _GatewayDAdminAPIService_Version_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).Version(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_Version_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).Version(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetGlobalConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Group)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetGlobalConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetGlobalConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetGlobalConfig(ctx, req.(*Group))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetPluginConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetPluginConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetPluginConfig_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetPluginConfig(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetPlugins_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetPlugins(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetPlugins_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetPlugins(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetPools_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetPools(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetPools_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetPools(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetProxies_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetProxies(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetProxies_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetProxies(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetServers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetServers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetServers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetServers(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_GetPeers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).GetPeers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_GetPeers_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).GetPeers(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_AddPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AddPeerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).AddPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_AddPeer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).AddPeer(ctx, req.(*AddPeerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _GatewayDAdminAPIService_RemovePeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemovePeerRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GatewayDAdminAPIServiceServer).RemovePeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: GatewayDAdminAPIService_RemovePeer_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GatewayDAdminAPIServiceServer).RemovePeer(ctx, req.(*RemovePeerRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// GatewayDAdminAPIService_ServiceDesc is the grpc.ServiceDesc for GatewayDAdminAPIService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var GatewayDAdminAPIService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.v1.GatewayDAdminAPIService",
	HandlerType: (*GatewayDAdminAPIServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Version",
			Handler:    _GatewayDAdminAPIService_Version_Handler,
		},
		{
			MethodName: "GetGlobalConfig",
			Handler:    _GatewayDAdminAPIService_GetGlobalConfig_Handler,
		},
		{
			MethodName: "GetPluginConfig",
			Handler:    _GatewayDAdminAPIService_GetPluginConfig_Handler,
		},
		{
			MethodName: "GetPlugins",
			Handler:    _GatewayDAdminAPIService_GetPlugins_Handler,
		},
		{
			MethodName: "GetPools",
			Handler:    _GatewayDAdminAPIService_GetPools_Handler,
		},
		{
			MethodName: "GetProxies",
			Handler:    _GatewayDAdminAPIService_GetProxies_Handler,
		},
		{
			MethodName: "GetServers",
			Handler:    _GatewayDAdminAPIService_GetServers_Handler,
		},
		{
			MethodName: "GetPeers",
			Handler:    _GatewayDAdminAPIService_GetPeers_Handler,
		},
		{
			MethodName: "AddPeer",
			Handler:    _GatewayDAdminAPIService_AddPeer_Handler,
		},
		{
			MethodName: "RemovePeer",
			Handler:    _GatewayDAdminAPIService_RemovePeer_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/api.proto",
}
