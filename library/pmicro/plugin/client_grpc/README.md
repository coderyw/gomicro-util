# GRPC Client

改写自github.com/go-micro/plugins/v4/client/grpc,主要直接使用grpc通信。改写了通过service获取请求method的方法。由于原方法只能通过service和endpoint进行拼接，这就导致了设置service为服务名时候就会提示找不到方法错误，如果设置service为proto的包名时候，会导致服务发现报错，找不到服务。

所以这里对其进行修改，在创建客户端时候，传入的service，使用serviceName/package这样的形式，会在创建client时候进行拆分。服务发现阶段使用serviceName,接口调用阶段使用package组合endpoint。

## Overview

The client makes use of the [google.golang.org/grpc](google.golang.org/grpc) framework for the underlying communication mechanism.

## Usage

Specify the client to your micro service

```go
import (
	"github.com/micro/go-micro"
	"github.com/micro/go-plugins/client/grpc"
)

func main() {
	service := micro.NewService(
		micro.Name("greeter"),
		micro.Client(grpc.NewClient()),
	)
}
```
