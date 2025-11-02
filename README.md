# TPS游戏服务器框架

本项目是一个基于Due框架实现的第三人称射击(TPS)游戏服务器框架，支持分布式部署，包含完整的游戏核心功能模块。

## 技术栈

- **框架**: Due v2.4.1
- **语言**: Go 1.23+
- **网络协议**: TCP + WebSocket + Protobuf + RSA加密
- **服务发现**: etcd
- **节点通信**: gRPC
- **配置管理**: YAML + TOML本地配置
- **缓存**: Redis v9
- **数据库**: MongoDB
- **消息队列**: Kafka
- **安全**: RSA加密保护

## 功能模块

### 1. 登录服务 (login)
- 账号密码登录认证
- RSA加密保护
- 登录排队系统
- 会话管理
- Token生成与验证

### 2. 网关服务 (gate)
- 客户端连接管理
- 消息转发
- 心跳检测
- 连接安全验证

### 3. 大厅服务 (lobby)
- 玩家信息管理
- 房间匹配
- 游戏状态同步
- 社交功能支持

### 4. 战斗服务 (battle)
- PVE/PVP战斗系统
- 战斗状态同步
- 伤害计算
- 战斗结算
- 战斗数据通过Kafka传输

### 5. 数据库管理服务 (dbmgr)
- 数据持久化
- 数据同步
- 数据备份与恢复
- 性能优化

## 项目结构

```
├── configs/                # 配置文件
│   ├── gate/               # 网关服务配置
│   ├── job/                # 任务服务配置
│   └── node/               # 节点服务配置
├── define/                 # 数据模型定义
├── mode/                   # 运行模式模块
│   ├── battle/             # 战斗服务
│   ├── dbmgr/              # 数据库管理服务
│   ├── gate/               # 网关服务
│   ├── lobby/              # 大厅服务
│   └── login/              # 登录服务
├── proto/                  # Protobuf定义
├── test/                   # 测试代码
│   └── client/             # 客户端测试
├── utils/                  # 工具类
├── go.mod                  # Go模块定义
├── go.sum                  # 依赖校验文件
├── deploy.md               # 部署文档
└── README.md               # 项目说明文档
```

## 安装与运行

### 前置条件

- Go 1.23+
- MongoDB
- Redis 7+
- Kafka
- etcd

### 安装依赖

```bash
# 设置Go代理（国内用户推荐）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=off

# 安装依赖
go mod tidy
go mod download
```

### 配置文件

在启动前，需要配置以下服务：

1. MongoDB连接
2. Redis连接
3. Kafka连接
4. etcd连接
5. RSA密钥配置

### 启动服务

项目采用微服务架构，各服务需要单独启动，并指定配置文件：

```bash
# 启动登录服务
cd mode/login
go run main.go

# 启动网关服务
cd mode/gate
go run main.go --etc=../../configs/gate.toml

# 启动大厅服务
cd mode/lobby
go run main.go --etc=../../configs/lobby.toml

# 启动战斗服务
cd mode/battle
go run main.go --etc=../../configs/node/node-1.yaml

# 启动数据库管理服务
cd mode/dbmgr
go run main.go --etc=../../configs/node/node-2.yaml
```

## 注意事项

1. 本框架默认单进程玩家数量上限可在对应服务的配置文件中调整
2. 各服务模块支持分布式部署，通过etcd进行服务发现
3. 所有敏感数据（如密码）通过RSA加密传输
4. 战斗数据实时通过Kafka进行异步处理
5. 配置文件中的RSA密钥需要提前生成并正确配置

## 开发指南

1. 新增功能模块请参考现有模块结构，在mode目录下创建对应的服务
2. 消息定义需在proto/pb目录下添加对应的proto文件
3. 数据模型定义在define目录下
4. 关键操作请添加日志记录
5. 性能瓶颈部分可考虑添加Redis缓存
6. 配置文件需根据实际环境进行调整

## 后续优化方向

1. 战斗同步算法优化
2. 服务器负载均衡策略改进
3. 数据分片存储
4. 添加监控告警系统
5. 自动化运维部署脚本
6. 服务熔断和降级机制
7. 更完善的错误处理和重试机制