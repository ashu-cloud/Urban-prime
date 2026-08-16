package driverv1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type DriverServiceClient interface {
	MatchDriver(ctx context.Context, in *MatchDriverRequest, opts ...grpc.CallOption) (*MatchDriverResponse, error)
	UpdateDriverStatus(ctx context.Context, in *UpdateDriverStatusRequest, opts ...grpc.CallOption) (*UpdateDriverStatusResponse, error)
	RegisterDriver(ctx context.Context, in *RegisterDriverRequest, opts ...grpc.CallOption) (*RegisterDriverResponse, error)
	GetDriver(ctx context.Context, in *GetDriverRequest, opts ...grpc.CallOption) (*GetDriverResponse, error)
}

type driverServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDriverServiceClient(cc grpc.ClientConnInterface) DriverServiceClient {
	return &driverServiceClient{cc}
}

func (c *driverServiceClient) MatchDriver(ctx context.Context, in *MatchDriverRequest, opts ...grpc.CallOption) (*MatchDriverResponse, error) {
	out := new(MatchDriverResponse)
	err := c.cc.Invoke(ctx, "/driver.v1.DriverService/MatchDriver", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *driverServiceClient) UpdateDriverStatus(ctx context.Context, in *UpdateDriverStatusRequest, opts ...grpc.CallOption) (*UpdateDriverStatusResponse, error) {
	out := new(UpdateDriverStatusResponse)
	err := c.cc.Invoke(ctx, "/driver.v1.DriverService/UpdateDriverStatus", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *driverServiceClient) RegisterDriver(ctx context.Context, in *RegisterDriverRequest, opts ...grpc.CallOption) (*RegisterDriverResponse, error) {
	out := new(RegisterDriverResponse)
	err := c.cc.Invoke(ctx, "/driver.v1.DriverService/RegisterDriver", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *driverServiceClient) GetDriver(ctx context.Context, in *GetDriverRequest, opts ...grpc.CallOption) (*GetDriverResponse, error) {
	out := new(GetDriverResponse)
	err := c.cc.Invoke(ctx, "/driver.v1.DriverService/GetDriver", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type DriverServiceServer interface {
	MatchDriver(context.Context, *MatchDriverRequest) (*MatchDriverResponse, error)
	UpdateDriverStatus(context.Context, *UpdateDriverStatusRequest) (*UpdateDriverStatusResponse, error)
	RegisterDriver(context.Context, *RegisterDriverRequest) (*RegisterDriverResponse, error)
	GetDriver(context.Context, *GetDriverRequest) (*GetDriverResponse, error)
	mustEmbedUnimplementedDriverServiceServer()
}

type UnimplementedDriverServiceServer struct{}

func (UnimplementedDriverServiceServer) MatchDriver(context.Context, *MatchDriverRequest) (*MatchDriverResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MatchDriver not implemented")
}
func (UnimplementedDriverServiceServer) UpdateDriverStatus(context.Context, *UpdateDriverStatusRequest) (*UpdateDriverStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDriverStatus not implemented")
}
func (UnimplementedDriverServiceServer) RegisterDriver(context.Context, *RegisterDriverRequest) (*RegisterDriverResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RegisterDriver not implemented")
}
func (UnimplementedDriverServiceServer) GetDriver(context.Context, *GetDriverRequest) (*GetDriverResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetDriver not implemented")
}
func (UnimplementedDriverServiceServer) mustEmbedUnimplementedDriverServiceServer() {}

func RegisterDriverServiceServer(s grpc.ServiceRegistrar, srv DriverServiceServer) {
	s.RegisterService(&DriverService_ServiceDesc, srv)
}

func _DriverService_MatchDriver_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MatchDriverRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DriverServiceServer).MatchDriver(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/driver.v1.DriverService/MatchDriver",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DriverServiceServer).MatchDriver(ctx, req.(*MatchDriverRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DriverService_UpdateDriverStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateDriverStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DriverServiceServer).UpdateDriverStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/driver.v1.DriverService/UpdateDriverStatus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DriverServiceServer).UpdateDriverStatus(ctx, req.(*UpdateDriverStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DriverService_RegisterDriver_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterDriverRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DriverServiceServer).RegisterDriver(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/driver.v1.DriverService/RegisterDriver",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DriverServiceServer).RegisterDriver(ctx, req.(*RegisterDriverRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DriverService_GetDriver_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetDriverRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DriverServiceServer).GetDriver(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/driver.v1.DriverService/GetDriver",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DriverServiceServer).GetDriver(ctx, req.(*GetDriverRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var DriverService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "driver.v1.DriverService",
	HandlerType: (*DriverServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "MatchDriver",
			Handler:    _DriverService_MatchDriver_Handler,
		},
		{
			MethodName: "UpdateDriverStatus",
			Handler:    _DriverService_UpdateDriverStatus_Handler,
		},
		{
			MethodName: "RegisterDriver",
			Handler:    _DriverService_RegisterDriver_Handler,
		},
		{
			MethodName: "GetDriver",
			Handler:    _DriverService_GetDriver_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/driver/v1/driver.proto",
}
