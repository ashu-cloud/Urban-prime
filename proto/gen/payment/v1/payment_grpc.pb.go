package paymentv1

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type PaymentServiceClient interface {
	AuthorizeHold(ctx context.Context, in *AuthorizeHoldRequest, opts ...grpc.CallOption) (*AuthorizeHoldResponse, error)
	ReleaseHold(ctx context.Context, in *ReleaseHoldRequest, opts ...grpc.CallOption) (*ReleaseHoldResponse, error)
	CapturePayment(ctx context.Context, in *CapturePaymentRequest, opts ...grpc.CallOption) (*CapturePaymentResponse, error)
}

type paymentServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPaymentServiceClient(cc grpc.ClientConnInterface) PaymentServiceClient {
	return &paymentServiceClient{cc}
}

func (c *paymentServiceClient) AuthorizeHold(ctx context.Context, in *AuthorizeHoldRequest, opts ...grpc.CallOption) (*AuthorizeHoldResponse, error) {
	out := new(AuthorizeHoldResponse)
	err := c.cc.Invoke(ctx, "/payment.v1.PaymentService/AuthorizeHold", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) ReleaseHold(ctx context.Context, in *ReleaseHoldRequest, opts ...grpc.CallOption) (*ReleaseHoldResponse, error) {
	out := new(ReleaseHoldResponse)
	err := c.cc.Invoke(ctx, "/payment.v1.PaymentService/ReleaseHold", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) CapturePayment(ctx context.Context, in *CapturePaymentRequest, opts ...grpc.CallOption) (*CapturePaymentResponse, error) {
	out := new(CapturePaymentResponse)
	err := c.cc.Invoke(ctx, "/payment.v1.PaymentService/CapturePayment", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type PaymentServiceServer interface {
	AuthorizeHold(context.Context, *AuthorizeHoldRequest) (*AuthorizeHoldResponse, error)
	ReleaseHold(context.Context, *ReleaseHoldRequest) (*ReleaseHoldResponse, error)
	CapturePayment(context.Context, *CapturePaymentRequest) (*CapturePaymentResponse, error)
	mustEmbedUnimplementedPaymentServiceServer()
}

type UnimplementedPaymentServiceServer struct{}

func (UnimplementedPaymentServiceServer) AuthorizeHold(context.Context, *AuthorizeHoldRequest) (*AuthorizeHoldResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AuthorizeHold not implemented")
}
func (UnimplementedPaymentServiceServer) ReleaseHold(context.Context, *ReleaseHoldRequest) (*ReleaseHoldResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReleaseHold not implemented")
}
func (UnimplementedPaymentServiceServer) CapturePayment(context.Context, *CapturePaymentRequest) (*CapturePaymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CapturePayment not implemented")
}
func (UnimplementedPaymentServiceServer) mustEmbedUnimplementedPaymentServiceServer() {}

func RegisterPaymentServiceServer(s grpc.ServiceRegistrar, srv PaymentServiceServer) {
	s.RegisterService(&PaymentService_ServiceDesc, srv)
}

var PaymentService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "payment.v1.PaymentService",
	HandlerType: (*PaymentServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "AuthorizeHold",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(AuthorizeHoldRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(PaymentServiceServer).AuthorizeHold(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/payment.v1.PaymentService/AuthorizeHold",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(PaymentServiceServer).AuthorizeHold(ctx, req.(*AuthorizeHoldRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "ReleaseHold",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(ReleaseHoldRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(PaymentServiceServer).ReleaseHold(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/payment.v1.PaymentService/ReleaseHold",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(PaymentServiceServer).ReleaseHold(ctx, req.(*ReleaseHoldRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
		{
			MethodName: "CapturePayment",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				in := new(CapturePaymentRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(PaymentServiceServer).CapturePayment(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/payment.v1.PaymentService/CapturePayment",
				}
				handler := func(ctx context.Context, req interface{}) (interface{}, error) {
					return srv.(PaymentServiceServer).CapturePayment(ctx, req.(*CapturePaymentRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/payment/v1/payment.proto",
}
