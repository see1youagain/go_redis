# go_redis 开发笔记

本文档记录了 `go_redis` 项目从底层 TCP 服务器构建到实现 RESP 协议解析的开发过程和核心组件说明。

## 1. TCP 服务器实现

TCP 服务器是整个项目的基础，负责监听端口、接收客户端连接并进行管理。

### 1.1 启动流程 (`tcp/server.go`)

服务器的启动和生命周期管理在 `tcp/server.go` 中实现。

- **`ListenAndServeWithSignal(cfg *Config, handler tcp.Handler) error`**
  - **作用**: 这是服务器的启动入口函数，集成了优雅关闭功能。
  - **步骤**:
    1.  通过 `net.Listen("tcp", cfg.Address)` 创建一个 TCP 监听器。
    2.  创建一个 `sigChan` 用于接收操作系统的中断信号（如 `SIGINT`, `SIGTERM`）。
    3.  启动一个 goroutine 监听 `sigChan`，当接收到指定信号时，向 `closeChan` 发送一个空结构体，以触发关闭流程。
    4.  调用 `ListenAndServe` 函数执行核心的连接接收和处理逻辑。

- **`ListenAndServe(listener net.Listener, handler tcp.Handler, closeChan <-chan struct{})`**
  - **作用**: 负责循环接收连接并分发给 `Handler` 处理。
  - **步骤**:
    1.  启动一个 goroutine 监听 `closeChan`。一旦接收到关闭信号，它会依次关闭 `listener` 和 `handler`，实现资源的优雅释放。
    2.  主 goroutine 进入一个无限循环，调用 `listener.Accept()` 阻塞等待新的客户端连接。
    3.  每当有新连接 `conn` 建立时，启动一个新的 goroutine，调用 `handler.Handler(ctx, conn)` 来处理该连接的业务逻辑。
    4.  使用 `sync.WaitGroup` 确保在服务器退出前，所有已建立的连接都得到妥善处理。

### 1.2 TCP Handler 接口 (`interface/tcp/handler.go`)

为了解耦 TCP 服务器框架和具体的业务逻辑，我们定义了 `Handler` 接口。

- **`Handler(ctx context.Context, conn net.Conn)`**: 处理单个客户端连接的业务逻辑。
- **`Close() error`**: 关闭 `Handler`，释放其持有的所有资源（例如，关闭所有活跃的客户端连接）。

### 1.3 Echo 测试服务器 (`tcp/echo.go`)

`EchoHandler` 是 `tcp.Handler` 的一个简单实现，用于测试 TCP 服务器的基本功能。它会将客户端发送的任何消息原样返回。

- **`EchoHandler`**:
  - `activeConn sync.Map`: 使用并发安全的 Map 存储所有活跃的客户端连接。
  - `closing atomic.Boolean`: 一个原子布尔值，用于标记服务器是否正在关闭。在关闭过程中，不再接受新连接。
  - `Handler(...)`: 当新连接到来时，将其包装为 `EchoClient` 并存入 `activeConn`。然后在一个循环中读取客户端发送的数据（以 `\n` 分隔），并写回给客户端。
  - `Close()`: 将 `closing` 标记设置为 `true`，并遍历 `activeConn`，关闭所有客户端连接。

- **`EchoClient`**:
  - `Waiting wait.Wait`: 一个带超时的 `WaitGroup`，用于实现优雅关闭。在向客户端写入数据前 `Add(1)`，写入后 `Done()`，确保数据发送完毕前连接不会被强制关闭。

## 2. RESP 协议层

在 TCP 层之上，我们构建了 RESP（REdis Serialization Protocol）协议层来处理 Redis 命令。

### 2.1 连接封装 (`resp/connection/conn.go`)

`connection.Connection` 对底层的 `net.Conn` 进行了封装，提供了更丰富的功能。

- **`Connection` 结构体**:
  - `conn net.Conn`: 底层 TCP 连接。
  - `waitingReply wait.Wait`: 保证并发写操作的原子性和完整性。
  - `selectedDB int`: 当前连接选择的数据库编号。
- **主要方法**:
  - `Write([]byte)`: 线程安全地向客户端写入数据。
  - `SelectDB(int)` / `GetDBIndex()`: 切换或获取当前数据库。
  - `Close()`: 优雅地关闭连接，会等待正在进行的写操作完成。

### 2.2 协议解析器 (`resp/parser/parser.go`)

协议解析器是 RESP 层的核心，它负责将从客户端 `io.Reader` 读取的原始字节流，转换为结构化的 `resp.Reply` 数据。解析器被设计为一个异步的、基于状态机的流式解析器。

- **核心组件**:
  - **`Payload` 结构体**:
    ```go
    type Payload struct {
        Data resp.Reply
        Err  error
    }
    ```
    作为解析结果的载体，通过 channel 在解析器和上层调用者之间传递。`Data` 字段包含成功解析出的 RESP 数据，`Err` 字段则用于传递解析过程中遇到的错误（如协议错误、IO 错误）。

  - **`readState` 状态机**:
    解析器的心脏。它记录了当前解析会话的上下文信息，以正确处理跨越多行、格式复杂的 RESP 命令（如 `*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n`）。
    - `readingMutiLine bool`: 标记是否正在解析一个多行命令（由 `*` 或 `$` 开头）。
    - `expectedArgsCount int`: 对于多行数组（`*`），记录期望的参数数量。
    - `msgType byte`: 记录当前命令的根类型（`*` 或 `$`）。
    - `args [][]byte`: 存储已解析出的多行命令的各个参数。
    - `bulkLen int64`: 对于定长字符串（`$`），记录其字节长度。

- **主要函数与解析流程**:
  1.  **`ParseStream(reader io.Reader) <-chan *Payload`**:
      - **作用**: 解析器的入口函数。它接收一个 `io.Reader`（通常是 `net.Conn`），并返回一个只读的 `*Payload` channel。
      - **实现**: 内部启动一个 `parse0` goroutine 来执行异步解析，使得上层业务逻辑可以以非阻塞的方式从 channel 中消费解析结果。

  2.  **`parse0(...)`**:
      - **作用**: 核心解析循环。
      - **流程**:
        a. 使用 `bufio.NewReader` 包装输入流以提高读取效率。
        b. 进入无限循环，调用 `readLine` 从流中读取一行或一个数据块。
        c. **如果不在多行模式 (`!state.readingMutiLine`)**:
           - 检查行首字节：
             - `*`: 是数组。调用 `parseMutiBulkHeader` 解析数组长度，设置 `expectedArgsCount`，并进入多行模式。
             - `$`: 是定长字符串。调用 `parseBulkHeader` 解析字符串长度，设置 `bulkLen`，并进入多行模式。
             - `+`, `-`, `:`:  是单行回复。调用 `parseSingleLineReply` 直接解析并发送 `Payload`。
             d. **如果在多行模式 (`state.readingMutiLine`)**:
           - 调用 `readBody` 处理当前行，将其作为参数追加到 `state.args` 中。
           - 检查 `state.finished()`（即 `len(state.args) == state.expectedArgsCount`）。如果为 `true`，说明所有参数都已集齐，则将 `state.args` 组装成一个 `reply.MultiBulkReply`，发送 `Payload`，并重置 `readState` 以准备解析下一条命令。

  3.  **`readLine(...)`**:
      - **作用**: 智能地从 `bufio.Reader` 读取数据。
      - **逻辑**:
        - 如果 `state.bulkLen == 0`（解析头部或简单回复），则按 `\r\n` 分隔符读取一行。
        - 如果 `state.bulkLen > 0`（解析定长字符串的主体），则精确读取 `bulkLen + 2` 个字节（包含末尾的 `\r\n`），这保证了二进制安全。

### 2.3 RESP 处理器 (`resp/handler/handler.go`)

`RespHandler` 是 `tcp.Handler` 接口的 RESP 协议实现，是连接数据库和协议解析的桥梁。

- **`Handler(...)`**:
  1.  将 `net.Conn` 封装为 `connection.Connection`。
  2.  调用 `parser.ParseStream` 创建一个解析通道 `ch`。
  3.  循环从 `ch` 中读取 `payload`。
  4.  如果 `payload` 包含错误，则处理错误（如连接关闭、协议错误）。
  5.  如果 `payload` 包含数据，则将其交给 `db.Exec` 执行，并将结果写回客户端。

## 3. 数据库接口 (`interface/database/database.go`)

定义了数据库核心模块必须实现的接口。

- **`Database` 接口**:
  - `Exec(client resp.Connection, args [][]byte) resp.Reply`: 执行命令的核心方法。
  - `Close()`: 关闭数据库，释放资源。
  - `AfterClientClose(client resp.Connection)`: 当客户端连接关闭后，执行一些清理工作（例如，在事务中）。

## 4. 测试步骤

1.  **启动 Echo 服务器**:
    - 在 `main` 函数中，使用 `tcp.MakeHandler()` 创建一个 `EchoHandler` 实例。
    - 调用 `tcp.ListenAndServeWithSignal` 启动服务器。
2.  **使用 `telnet` 或 `nc` 测试**:
    - 打开终端，执行 `telnet 127.0.0.1 <your_port>` 或 `nc 127.0.0.1 <your_port>`。
    - 输入任意字符串并按回车，观察服务器是否返回同样的内容。
    - 打开多个终端进行连接，测试并发处理能力。
3.  **测试优雅关闭**:
    - 在服务器运行期间，在运行服务器的终端按下 `Ctrl+C`。
    - 观察服务器日志，应有 "shutting down..." 等信息，并且所有客户端连接被断开。
4.  **切换到 RESP Handler**:
    - 将 `main` 函数中的 `tcp.MakeHandler()` 替换为 `handler.MakeHandler()`（并初始化 `db`），即可将服务器切换为 RESP 模式。

## 5. 内存数据库实现 (database/db.go)

在完成了网络层和协议解析层之后，我们进入项目的核心——内存数据库的实现。它负责处理具体的 Redis 命令，并管理数据，现已替换了原有的 `EchoDatabase`。

### 5.1 目标与作用

- **目标**: 实现一个支持多数据库、能够执行基本 Redis 命令（如 PING, SELECT）的并发安全内存数据库。
- **作用**:
  1.  **提供真实业务逻辑**: 作为 `resp.Handler` 的后端，执行客户端发来的命令。
  2.  **管理数据**: 在内存中存储键值对，并处理多数据库的切换逻辑。

### 5.2 核心组件设计与实现

我们创建了 `Database` 结构体，它实现了 `database.Database` 接口。

- **`Database` 结构体**:
  ```go
  type Database struct {
      dbSet      []*DBEntity // 数据库集合，用数组/切片模拟 Redis 的多个数据库
      cmdMap     map[string]CmdFunc // 命令注册表，映射命令名到处理函数
  }
  
  type DBEntity struct {
      data *sync.Map // 每个数据库的核心存储，使用并发安全的 map
      // 未来可扩展 TTL 等
  }
  ```

- **命令处理函数 `CmdFunc`**:
  ```go
  type CmdFunc func(db *Database, client resp.Connection, args [][]byte) resp.Reply
  ```
  - 这是一个函数类型，所有具体的命令（如 `ping`, `select`）都实现了这个签名。

### 5.3 实现细节

1.  **`database/database.go`**:
    - 定义了 `Database`、`DBEntity` 和 `CmdFunc` 类型。
    - 实现了 `NewDatabase()` 构造函数，它初始化 `dbSet` 和 `cmdMap`，并通过 `registerCommands` 注册了 `ping` 和 `select` 命令。

2.  **`Exec` 方法**:
    - 从客户端命令 `args` 中提取命令名（如 `PING`），并转换为小写以实现不区分大小写的命令匹配。
    - 在 `cmdMap` 中查找对应的 `CmdFunc`。
    - 如果找到，则调用该函数并返回其 `resp.Reply` 结果。
    - 如果未找到，则返回一个 `ERR unknown command` 错误回复。

3.  **已实现的命令**:
    - **`ping`**: 检查参数数量，返回 `reply.MakePongReply()`。
    - **`select`**: 解析参数获取数据库索引，检查索引合法性，并调用 `client.SelectDB(dbIndex)` 切换客户端的数据库上下文，最后返回 `reply.MakeOkReply()`。

4.  **与 `RespHandler` 集成**:
    - 在 `resp/handler/handler.go` 的 `MakeRespHandler` 函数中，已将 `database.NewEchoDatabase()` 替换为 `database.NewDatabase()`，使服务器具备了真实的数据库处理能力。

## 6. Go的Redis持久化


### 概述

- **核心变更**: 使用自研的内存数据库模块 (`database/database.go`) 替换了测试用的 `EchoDatabase`。
- **功能实现**:
    - 实现了支持多数据库切换的 `Database` 结构。
    - 实现了命令注册与分发机制。
    - 实现了 `PING` 和 `SELECT` 两个基本命令。
- **状态**: 服务器现在可以响应 `PING` 命令并处理 `SELECT` 命令进行数据库切换，为后续实现更多数据操作命令打下了基础。

### 6.1 定义落盘处理结构体`AofHandler`

```go
type AofHandler struct {
	database       database.Database // database接口，包含数据库执行等函数
	aofFile        *os.File // 操作文件的指针
	aofFileName    string // aof文件名
	currentDBIndex int // 当前操作的数据库索引
	aofChan        chan *payload // 使用独立的协程处理Aof写操作，读操作会直接采用LoadAof函数（先加载、再写）
}
```

并且构建三个函数`NewAofHandler`、`handleAof`、`LoadAof`。

`NewAofHandler`：创建新的AofHandler，并且判断配置设置、加载文件、创建chan并且开启处理协程

`handleAof`：处理Aof写操作，接收`chan payload`，将其写入aofFile中

`LoadAof`：用于初始化时加载aof的内容，并重放数据库操作

此间有一个问题，在我们执行数据库具体操作时，我们只能操作`db *DB`，而这个DB是不知道自己是哪一个分数据库的，因此我们需要在DB结构体中添加一个函数`addAof func(CmdLine)`，在初始化DB的时候，暂时将`addAof`设为空函数（此时不需要落盘，只需要读取）。在NewDatabase时，会调用NewAofHandler，之后再处理DB的aof函数：

```go
database.aofHandler = aofHandler
for i, db := range database.dbSet {
    dbIndex := i // 为了避免函数闭包问题，采用临时变量存储
    db.addAof = func(line CmdLine) {
        database.aofHandler.AddAof(dbIndex, line)
    }
}
```

此后，在`database/string.go`和`database/keys.go`中，在有关存储的函数中增加Aof落盘。

落盘选择沿用`resp/parser/parser.go/ParseStream`，由于` os.Open()`返回的变量实现了`io.Reader`的函数`Read(p []byte) (n int, err error)`，所以直接调用`parser.ParseStream`即可。此外，我们需要执行数据库的`Exec`操作，但是`Exec(client resp.Connection, args [][]byte) resp.Reply`需要一个`Connection`连接，因此我们创建一个虚假的连接，传入`Exec`函数中，完成了落盘文件内操作的重放。

在此之间，我们修改了`parser.go`中潜藏的`io.EOF`操作的报错问题，在此之前由于对于文件、连接的`EOF Error`的理解缺失，导致网络连接、文件读取的`EOF`报错都会被传入`chan *Payload`中，且并未关闭`channel`，导致死锁问题。对此进行了修改

## 7. Go实现Redis集群

### 7.1 一致性哈希（理论）

一致性哈希是一种分布式系统中常用的哈希算法，主要用于解决节点动态增减时数据重分布的问题。它将所有的节点和数据通过哈希函数映射到一个虚拟的环上，数据根据哈希值分配到最近的节点。当节点增加或减少时，只需要重新分配少量的数据，极大地提高了系统的扩展性和容错性。常用于分布式缓存、数据库分片等场景。

Redis 集群定义了 16384 个哈希槽（编号 0-16383）。

每个 key 通过 CRC16 算法计算哈希值，然后对 16384 取模，确定属于哪个槽。

集群中的每个节点负责一部分哈希槽，节点之间分配槽的范围。

当节点增加或减少时，只需要迁移对应槽的数据，减少了数据迁移量，实现了高效的扩展和容错。

### 7.2 定义一致性哈希`consistenthash.go`

首先需要定义一个抽象函数`hashFunc`，为了方便后续定制化hash函数，使用如下函数声明:

```go
type HashFunc func(data []byte) uint32
```

NodeMap的作用是，将所有的节点进行一致性hash，并且将其从小到大进行排序。需要具备几个函数，增加创建NodeMap结构体函数`NewNodeMap`，增加节点函数`AddNodes`以及通过key去判断发送给哪个节点的函数`PickNode`。

注意，我们实现的AddNodes没有管理数据一致性问题，尤其是动态节点的数据迁移问题，只是进行节点的hash和排序。

7.3 

### 7.4 定义ClusterDatabase

`ClusterDatabase`负责管理本地单机版`Database`且与其余`ClusterDatabase`进行连接，转发其他数据库的数据、存储与运用本地数据库。因此他需要几个结构：

```go
type ClusterDatabase struct {
    self string // 存储自身的ip:port地址，方便判别是否是属于本地的数据
    nodes []string // 存储现在的所有节点
    peerPicker *consistenthash.NodeMap // 用于消息判断的节点选择
}
```













