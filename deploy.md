# TPS游戏服务器部署文档

## 1. 目录结构说明

项目采用了微服务架构，`cmd`目录包含了三个核心组件：

- **gate**: 网关服务，负责玩家连接管理和登录验证
- **node**: 节点服务，负责核心游戏逻辑（战斗、背包、任务等）
- **job**: 任务服务，负责异步任务处理和数据处理

### 1.1 cmd目录作用

| 组件 | 主要职责 | 文件位置 |
|-----|---------|----------|
| gate | 处理玩家连接、身份验证、消息转发 | cmd/gate/main.go |
| node | 核心业务逻辑处理（战斗、背包、任务等） | cmd/node/main.go |
| job | 异步任务处理、数据处理和分析 | cmd/job/main.go |

## 2. 环境准备（Debian 12）

### 2.1 安装依赖

```bash
# 更新系统
apt update && apt upgrade -y

# 安装必要工具
apt install -y git curl wget unzip

# 安装Go环境（1.20+）
wget https://go.dev/dl/go1.20.14.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.20.14.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# 安装MongoDB
apt install -y mongodb

# 安装Redis
apt install -y redis-server

# 安装etcd
wget https://github.com/etcd-io/etcd/releases/download/v3.5.13/etcd-v3.5.13-linux-amd64.tar.gz
tar -xzf etcd-v3.5.13-linux-amd64.tar.gz
cp etcd-v3.5.13-linux-amd64/etcd* /usr/local/bin/

# 安装Kafka
wget https://dlcdn.apache.org/kafka/3.6.1/kafka_2.13-3.6.1.tgz
tar -xzf kafka_2.13-3.6.1.tgz
mv kafka_2.13-3.6.1 /opt/kafka
echo 'export PATH=$PATH:/opt/kafka/bin' >> ~/.bashrc
source ~/.bashrc
```

### 2.2 启动依赖服务

```bash
# 启动MongoDB
systemctl start mongodb
systemctl enable mongodb

# 启动Redis
systemctl start redis-server
systemctl enable redis-server

# 启动etcd
nohup etcd --listen-client-urls=http://0.0.0.0:2379 --advertise-client-urls=http://0.0.0.0:2379 &

# 启动Kafka（先启动Zookeeper）
nohup /opt/kafka/bin/zookeeper-server-start.sh /opt/kafka/config/zookeeper.properties &
sleep 10
nohup /opt/kafka/bin/kafka-server-start.sh /opt/kafka/config/server.properties &

# 创建Kafka主题
/opt/kafka/bin/kafka-topics.sh --create --bootstrap-server localhost:9092 --replication-factor 1 --partitions 3 --topic battle_logs
```

## 3. 代码部署

### 3.1 克隆代码

```bash
mkdir -p /opt/ghserver
cd /opt/ghserver
git clone https://github.com/your-repo/ghserver.git .
```

### 3.2 修改配置文件

#### 3.2.1 修改gate配置（多实例准备）

```bash
# 创建gate配置目录
mkdir -p configs/gate

# 复制并修改gate配置文件（3个实例）
for i in {1..3}; do
  cp configs/gate.yaml configs/gate/gate-${i}.yaml
  sed -i "s/addr: \"0.0.0.0:8001\"/addr: \"0.0.0.0:800${i}\"/g" configs/gate/gate-${i}.yaml
  sed -i "s/id: \"gate-1\"/id: \"gate-${i}\"/g" configs/gate/gate-${i}.yaml
  sed -i "s/addr: \"127.0.0.1:9001\"/addr: \"127.0.0.1:900${i}\"/g" configs/gate/gate-${i}.yaml
  sed -i "s/gate.log/gate-${i}.log/g" configs/gate/gate-${i}.yaml
done
```

#### 3.2.2 修改node配置（多实例准备）

```bash
# 创建node配置目录
mkdir -p configs/node

# 复制并修改node配置文件（4个实例）
for i in {1..4}; do
  cp configs/node.yaml configs/node/node-${i}.yaml
  sed -i "s/id: \"node-1\"/id: \"node-${i}\"/g" configs/node/node-${i}.yaml
  sed -i "s/addr: \"127.0.0.1:9002\"/addr: \"127.0.0.1:901${i}\"/g" configs/node/node-${i}.yaml
  sed -i "s/node.log/node-${i}.log/g" configs/node/node-${i}.yaml
done
```

#### 3.2.3 修改job配置（多实例准备）

```bash
# 创建job配置目录
mkdir -p configs/job

# 复制并修改job配置文件（6个实例）
for i in {1..6}; do
  cp configs/job.yaml configs/job/job-${i}.yaml
  sed -i "s/id: \"job-1\"/id: \"job-${i}\"/g" configs/job/job-${i}.yaml
  sed -i "s/job.log/job-${i}.log/g" configs/job/job-${i}.yaml
done
```

### 3.3 编译构建

```bash
# 修改Go模块路径（修复路径问题）
sed -i 's|github.com/dobyte/due/v2|github.com/dobyte/due/v2|g' ./cmd/*/main.go

# 编译gate
mkdir -p output/gate
cd /opt/ghserver
for i in {1..3}; do
  go build -o output/gate/gate-${i} ./cmd/gate/main.go
done

# 编译node
mkdir -p output/node
for i in {1..4}; do
  go build -o output/node/node-${i} ./cmd/node/main.go
done

# 编译job
mkdir -p output/job
for i in {1..6}; do
  go build -o output/job/job-${i} ./cmd/job/main.go
done
```

## 4. 启动服务

### 4.1 创建启动脚本

```bash
# 创建启动脚本目录
mkdir -p scripts

# 创建gate启动脚本
cat > scripts/start_gate.sh << 'EOF'
#!/bin/bash

for i in {1..3}; do
  echo "Starting gate instance $i..."
  cd /opt/ghserver
  nohup ./output/gate/gate-${i} -config ./configs/gate/gate-${i}.yaml > ./logs/gate-${i}-output.log 2>&1 &
  echo $! > ./logs/gate-${i}.pid
done

echo "All gate instances started."
EOF

# 创建node启动脚本
cat > scripts/start_node.sh << 'EOF'
#!/bin/bash

for i in {1..4}; do
  echo "Starting node instance $i..."
  cd /opt/ghserver
  nohup ./output/node/node-${i} -config ./configs/node/node-${i}.yaml > ./logs/node-${i}-output.log 2>&1 &
  echo $! > ./logs/node-${i}.pid
done

echo "All node instances started."
EOF

# 创建job启动脚本
cat > scripts/start_job.sh << 'EOF'
#!/bin/bash

for i in {1..6}; do
  echo "Starting job instance $i..."
  cd /opt/ghserver
  nohup ./output/job/job-${i} -config ./configs/job/job-${i}.yaml > ./logs/job-${i}-output.log 2>&1 &
  echo $! > ./logs/job-${i}.pid
done

echo "All job instances started."
EOF

# 创建停止脚本
cat > scripts/stop_all.sh << 'EOF'
#!/bin/bash

# Stop gate instances
for i in {1..3}; do
  if [ -f ./logs/gate-${i}.pid ]; then
    PID=$(cat ./logs/gate-${i}.pid)
    echo "Stopping gate instance $i (PID: $PID)"
    kill -15 $PID
    rm ./logs/gate-${i}.pid
  fi
done

# Stop node instances
for i in {1..4}; do
  if [ -f ./logs/node-${i}.pid ]; then
    PID=$(cat ./logs/node-${i}.pid)
    echo "Stopping node instance $i (PID: $PID)"
    kill -15 $PID
    rm ./logs/node-${i}.pid
  fi
done

# Stop job instances
for i in {1..6}; do
  if [ -f ./logs/job-${i}.pid ]; then
    PID=$(cat ./logs/job-${i}.pid)
    echo "Stopping job instance $i (PID: $PID)"
    kill -15 $PID
    rm ./logs/job-${i}.pid
  fi
done

echo "All services stopped."
EOF

# 创建重启脚本
cat > scripts/restart_all.sh << 'EOF'
#!/bin/bash

cd /opt/ghserver
./scripts/stop_all.sh
sleep 5
./scripts/start_gate.sh
./scripts/start_node.sh
./scripts/start_job.sh
EOF

# 设置脚本权限
chmod +x scripts/*.sh
```

### 4.2 启动所有服务

```bash
# 创建日志目录
mkdir -p logs

# 启动所有服务
cd /opt/ghserver
./scripts/start_gate.sh
./scripts/start_node.sh
./scripts/start_job.sh
```

## 5. 监控与维护

### 5.1 检查服务状态

```bash
# 检查gate服务
ps aux | grep gate-

# 检查node服务
ps aux | grep node-

# 检查job服务
ps aux | grep job-

# 查看进程ID
ls -la logs/*.pid
```

### 5.2 查看日志

```bash
# 查看gate日志
for i in {1..3}; do
  tail -f logs/gate-${i}.log
done

# 查看node日志
for i in {1..4}; do
  tail -f logs/node-${i}.log
done

# 查看job日志
for i in {1..6}; do
  tail -f logs/job-${i}.log
done
```

### 5.3 负载均衡配置（可选）

如果需要在多台服务器上部署，可以使用Nginx进行负载均衡：

```nginx
http {
    upstream game_gate {
        server 192.168.1.10:8001;
        server 192.168.1.10:8002;
        server 192.168.1.10:8003;
    }

    server {
        listen 80;
        server_name game.example.com;

        location / {
            proxy_pass http://game_gate;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection 'upgrade';
            proxy_set_header Host $host;
            proxy_cache_bypass $http_upgrade;
        }
    }
}
```

## 6. 故障排除

### 6.1 常见问题

1. **服务无法启动**
   - 检查端口是否被占用：`netstat -tulpn | grep 端口号`
   - 检查配置文件路径是否正确
   - 查看启动日志：`cat logs/*-output.log`

2. **数据库连接失败**
   - 检查MongoDB服务是否运行：`systemctl status mongodb`
   - 检查连接字符串是否正确
   - 检查防火墙设置

3. **etcd连接失败**
   - 检查etcd服务是否运行：`ps aux | grep etcd`
   - 检查etcd配置中的地址是否正确

4. **服务间通信异常**
   - 检查etcd注册是否正常：`etcdctl get / --prefix`
   - 检查网络配置和防火墙设置

## 7. 扩容说明

如需进一步扩容，只需：

1. 在新服务器上重复环境准备步骤
2. 修改配置文件中的IP地址和端口
3. 确保所有服务器可以访问共享的etcd、MongoDB、Redis和Kafka
4. 启动新增的服务实例

服务会自动通过etcd进行服务发现和负载均衡。