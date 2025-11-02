package main

import (
	"context"
	server "ghserver/mode/lobby/service"

	"github.com/dobyte/due/locate/redis/v2"
	"github.com/dobyte/due/registry/etcd/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/log"
	ggrpc "google.golang.org/grpc"
)

func main() {
	// 创建容器
	container := due.NewContainer()
	// 创建用户定位器
	locator := redis.NewLocator()
	// 创建服务发现
	registry := etcd.NewRegistry()
	// 创建RPC传输器
	transporter := grpc.NewTransporter(grpc.WithClientDialOptions(ggrpc.WithChainUnaryInterceptor(clientInterceptor)))
	// 创建节点组件
	component := node.NewNode(
		node.WithLocator(locator),
		node.WithRegistry(registry),
		node.WithTransporter(transporter),
	)
	// 初始化应用
	initAPP(component.Proxy())
	// 添加节点组件
	container.Add(component)
	// 启动容器
	container.Serve()
}

// 客户端拦截器
func clientInterceptor(ctx context.Context, method string, req, reply any, cc *ggrpc.ClientConn, invoker ggrpc.UnaryInvoker, opts ...ggrpc.CallOption) error {
	log.Debug("client interceptor")

	return invoker(ctx, method, req, reply, cc, opts...)
}

// 服务接口，所有服务都需要实现这个接口
type Service interface {
	Init()
	Close() error
}

// 初始化应用
func initAPP(proxy *node.Proxy) {
	// 创建所有服务实例
	services := []Service{
		server.NewMailServer(proxy),
		server.NewRankingServer(proxy),
		server.NewShopServer(proxy),
		server.NewTaskServer(proxy),
	}

	// 初始化所有服务
	for _, s := range services {
		s.Init()
		log.Infof("Service initialized: %T", s)
	}

	// 注册关闭钩子，用于优雅关闭服务
	//proxy.OnShutdown(func() {
	//	for _, s := range services {
	//		if err := s.Close(); err != nil {
	//			log.Errorf("Failed to close service %T: %v", s, err)
	//		} else {
	//			log.Infof("Service closed: %T", s)
	//		}
	//	}
	//})
}
