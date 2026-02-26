# Go-Redis

[![Go Version](https://img.shields.io/badge/Go-1.24.5-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

一个用纯 Go 语言实现的轻量级 Redis 服务器，支持基本的 Redis 命令、集群模式和 AOF 持久化。

## ✨ 特性

- 🚀 **纯 Go 实现**：从零开始构建的 TCP 服务器和 RESP 协议解析器
- 🔧 **多模式支持**：支持单机模式（Standalone）和集群模式（Cluster）
- 💾 **AOF 持久化**：支持 AOF（Append Only File）持久化机制
- ⚡ **高性能**：使用并发安全的数据结构和连接池优化性能
- 🎯 **一致性哈希**：集群模式下使用一致性哈希算法进行数据分片
- 🛡️ **优雅关闭**：支持信号监听和优雅的服务器关闭
- 📝 **日志系统**：完整的日志记录系统，支持日志文件轮转

## 📋 支持的 Redis 命令

### 字符串操作
- `GET key` - 获取键的值
- `SET key value` - 设置键值对
- `SETNX key value` - 仅当键不存在时设置
- `GETSET key value` - 设置新值并返回旧值
- `STRLEN key` - 获取字符串长度

### 键操作
- `EXISTS key [key ...]` - 检查键是否存在
- `DEL key [key ...]` - 删除一个或多个键
- `KEYS pattern` - 查找匹配模式的键
- `FLUSHDB` - 清空当前数据库
- `TYPE key` - 返回键的数据类型
- `RENAME key newkey` - 重命名键
- `RENAMENX key newkey` - 仅当新键名不存在时重命名

### 数据库操作
- `SELECT index` - 切换数据库
- `PING` - 测试连接

## 🏗️ 项目结构

```
go_redis/
├── main.go              # 程序入口
├── go.mod               # Go 模块定义
├── redis.conf           # 配置文件
├── appendonly.aof       # AOF 持久化文件
├── aof/                 # AOF 持久化实现
│   └── aof.go
├── cluster/             # 集群模式实现
│   ├── cluster_database.go  # 集群数据库核心
│   ├── router.go        # 命令路由
│   ├── client_pool.go   # 客户端连接池
│   └── ...
├── database/            # 单机模式数据库实现
│   ├── standalone_database.go  # 单机数据库核心
│   ├── db.go            # 数据库实例
│   ├── string.go        # 字符串命令实现
│   ├── keys.go          # 键命令实现
│   └── ...
├── interface/           # 接口定义
│   ├── database/        # 数据库接口
│   ├── resp/            # RESP 协议接口
│   └── tcp/             # TCP 处理器接口
├── resp/                # RESP 协议实现
│   ├── handler/         # RESP 处理器
│   ├── parser/          # RESP 协议解析器
│   ├── connection/      # 连接管理
│   ├── reply/           # 回复类型
│   └── client/          # Redis 客户端
├── tcp/                 # TCP 服务器实现
│   └── server.go        # TCP 服务器核心
├── config/              # 配置管理
│   └── config.go
├── datastruct/          # 数据结构
│   └── dict/            # 字典实现
├── lib/                 # 工具库
│   ├── logger/          # 日志系统
│   ├── consistenthash/  # 一致性哈希
│   ├── sync/            # 同步工具
│   ├── utils/           # 工具函数
│   └── wildcard/        # 通配符匹配
└── logs/                # 日志文件目录
```

## 🚀 快速开始

### 前置要求

- Go 1.18 或更高版本
- Git

### 安装

```bash
# 克隆仓库
git clone https://github.com/see1youagain/go_redis.git
cd go_redis

# 安装依赖
go mod download
```

### 配置

编辑 `redis.conf` 配置文件：

```properties
# 服务器绑定地址
bind 0.0.0.0

# 服务器端口
port 8888

# 数据库数量（默认 16）
databases 16

# 启用 AOF 持久化
appendonly yes
appendfilename appendonly.aof

# 集群配置（可选）
self 127.0.0.1:8888
peers 127.0.0.1:8889
```

### 运行

```bash
# 单机模式
go run main.go

# 或者编译后运行
go build -o go_redis main.go
./go_redis
```

服务器将在配置的端口（默认 8888）上启动。

### 使用客户端连接

使用 Redis 官方客户端 `redis-cli` 或任何支持 RESP 协议的客户端连接：

```bash
# 使用 redis-cli 连接
redis-cli -p 8888

# 执行命令
127.0.0.1:8888> SET mykey "Hello World"
OK
127.0.0.1:8888> GET mykey
"Hello World"
127.0.0.1:8888> KEYS *
1) "mykey"
```

## 🔧 集群模式

启动集群需要配置多个节点：

**节点1 (redis1.conf):**
```properties
bind 0.0.0.0
port 8888
self 127.0.0.1:8888
peers 127.0.0.1:8889,127.0.0.1:8890
```

**节点2 (redis2.conf):**
```properties
bind 0.0.0.0
port 8889
self 127.0.0.1:8889
peers 127.0.0.1:8888,127.0.0.1:8890
```

**节点3 (redis3.conf):**
```properties
bind 0.0.0.0
port 8890
self 127.0.0.1:8890
peers 127.0.0.1:8888,127.0.0.1:8889
```

分别启动每个节点，集群会自动使用一致性哈希算法分配数据。

## 📚 核心实现

### TCP 服务器

自实现的 TCP 服务器支持：
- 并发连接处理
- 优雅关闭（监听系统信号）
- 连接状态管理
- 超时控制

### RESP 协议

完整实现 RESP（Redis Serialization Protocol）协议：
- 支持简单字符串、错误、整数、批量字符串和数组
- 流式解析，高效处理大文件
- 错误处理和边界检查

### AOF 持久化

AOF 持久化特性：
- 每次写操作后追加到文件
- 服务器重启时自动加载 AOF 文件
- 支持数据恢复

### 集群模式

集群实现要点：
- 一致性哈希算法保证数据分布均衡
- 连接池管理节点间通信
- 透明的数据路由和转发

## 🧪 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./database
go test ./resp/parser
```

## 📈 性能

项目使用以下技术优化性能：
- `sync.Map` 实现并发安全的字典
- 连接池减少连接开销
- 原子操作保证线程安全
- 高效的 RESP 协议解析器

## 🛣️ 开发路线图

- [x] TCP 服务器实现
- [x] RESP 协议解析
- [x] 基本字符串命令
- [x] 键操作命令
- [x] 单机模式
- [x] AOF 持久化
- [x] 集群模式
- [x] 一致性哈希
- [ ] RDB 持久化
- [ ] 列表数据类型
- [ ] 哈希数据类型
- [ ] 集合数据类型
- [ ] 有序集合数据类型
- [ ] 发布/订阅
- [ ] 事务支持
- [ ] Lua 脚本支持
- [ ] 过期键管理
- [ ] 主从复制
- [ ] 哨兵模式

## 🤝 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本仓库
2. 创建你的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交你的修改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 👨‍💻 作者

- GitHub: [@see1youagain](https://github.com/see1youagain)

## 🙏 致谢

- 感谢 Redis 项目提供的优秀设计思想
- 感谢 Go 社区的各种优秀开源项目
- 特别感谢 [go-commons-pool](https://github.com/jolestar/go-commons-pool) 项目提供的连接池实现

## 📞 联系方式

如有问题或建议，欢迎：
- 提交 [Issue](https://github.com/see1youagain/go_redis/issues)
- 发起 [Pull Request](https://github.com/see1youagain/go_redis/pulls)

---

⭐ 如果这个项目对你有帮助，请给一个 Star！
