package grpc

import (
	"fmt"
	"strings"

	"go-micro.dev/v4/client"
	"go-micro.dev/v4/codec"
)

type grpcRequest struct {
	service     string
	pkg         string
	method      string
	contentType string
	request     interface{}
	opts        client.RequestOptions
	codec       codec.Codec
}

// service Struct.Method /service.Struct/Method
func methodToGRPC(req client.Request, method string) string {

	// no method or already grpc method
	if len(method) == 0 || method[0] == '/' {
		return method
	}

	// assume method is Foo.Bar
	mParts := strings.Split(method, ".")
	if len(mParts) != 2 {
		return method
	}
	pkg := req.Service()
	if val, ok := req.(*grpcRequest); ok {
		pkg = val.pkg
	}

	if len(pkg) == 0 {
		return fmt.Sprintf("/%s/%s", mParts[0], mParts[1])
	}

	// return /pkg.Foo/Bar
	return fmt.Sprintf("/%s.%s/%s", pkg, mParts[0], mParts[1])
}

// service：serviceName/pkg解
func newGRPCRequest(service, method string, request interface{}, contentType string, reqOpts ...client.RequestOption) client.Request {
	var opts client.RequestOptions
	for _, o := range reqOpts {
		o(&opts)
	}

	// set the content-type specified
	if len(opts.ContentType) > 0 {
		contentType = opts.ContentType
	}
	req := &grpcRequest{
		service:     service,
		method:      method,
		request:     request,
		contentType: contentType,
		opts:        opts,
	}
	arr := strings.Split(service, "/")
	if len(arr) == 2 {
		req.service = arr[0]
		req.pkg = arr[1]
	} else {
		req.pkg = req.service
	}

	return req
}

func (g *grpcRequest) ContentType() string {
	return g.contentType
}

func (g *grpcRequest) Service() string {
	return g.service
}

func (g *grpcRequest) Method() string {
	return g.method
}

func (g *grpcRequest) Endpoint() string {
	return g.method
}

func (g *grpcRequest) Codec() codec.Writer {
	return g.codec
}

func (g *grpcRequest) Body() interface{} {
	return g.request
}

func (g *grpcRequest) Stream() bool {
	return g.opts.Stream
}
