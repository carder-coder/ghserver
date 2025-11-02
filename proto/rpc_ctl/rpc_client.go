package rpc_client

import (
	pb "ghserver/proto/pb"

	"github.com/dobyte/due/v2/transport"
	"google.golang.org/grpc"
)

// target 是RPC客户端连接的目标标识符
// discovery:// 前缀表示使用服务发现机制查找服务
// services 表示服务分组或命名空间，与配置文件中的registry.etcd.namespace保持一致
const target = "discovery://services"

// ServiceType 服务类型字符串常量
const (
	ServiceTypeLogin = "Login"
	ServiceTypeBag   = "Bag"
	ServiceTypeMail  = "Mail"
)

// NewRpcClient 根据服务类型创建对应的RPC客户端
func NewRpcClient(fn transport.NewMeshClient, serviceType string) (interface{}, error) {
	client, err := fn(target)
	if err != nil {
		return nil, err
	}

	conn := client.Client().(grpc.ClientConnInterface)

	switch serviceType {
	case ServiceTypeLogin:
		return pb.NewLoginServiceClient(conn), nil
	case ServiceTypeBag:
		return pb.NewBagServiceClient(conn), nil
	case ServiceTypeMail:
		return pb.NewMailServiceClient(conn), nil
	default:
		return nil, ErrInvalidServiceType
	}
}

// ErrInvalidServiceType 无效的服务类型错误
var ErrInvalidServiceType = &RpcError{Message: "invalid service type"}

// RpcError 自定义RPC错误类型
type RpcError struct {
	Message string
}

func (e *RpcError) Error() string {
	return e.Message
}
