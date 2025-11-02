package main

import (
	server "ghserver/mode/login_ws/service"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/registry/nacos/v2"
	"github.com/dobyte/due/transport/grpc/v2"
	"github.com/dobyte/due/v2"
	"github.com/dobyte/due/v2/log"
)

// @title 登录服API文档
// @version 1.0
// @host localhost:8080
// @BasePath /
func main() {
	// 创建容器
	container := due.NewContainer()
	// 创建服务注册发现
	registry := nacos.NewRegistry()
	// 创建RPC传输器
	transporter := grpc.NewTransporter()
	// 创建HTTP组件
	component := http.NewServer(
		http.WithRegistry(registry),
		http.WithTransporter(transporter),
	)
	// 初始化应用
	initAPP(component.Proxy())
	// 添加HTTP组件
	container.Add(component)
	// 启动容器
	container.Serve()
}

// 服务接口，所有服务都需要实现这个接口
type Service interface {
	Init()
	Close() error
}

// 初始化应用
func initAPP(proxy *http.Proxy) {
	// 创建所有服务实例
	services := []Service{
		server.NewLoginAPI(proxy),
	}

	// 初始化所有服务
	for _, s := range services {
		s.Init()
		log.Infof("Service initialized: %T", s)
	}

}
