
# TcpPlayer 接口说明文档

## 概述

`TcpPlayer` 是一个用于 TCP 流量重放（Traffic Replay）的结构体，支持从记录的数据中读取并按原始通信顺序向客户端或目标服务端重放流量。它支持时间戳控制、自定义数据解析逻辑，并可用于模拟真实网络交互行为。

---

## 结构体定义

```go
type TcpPlayer struct {
    Target           string              // 目标地址（如 "192.168.1.10:80"）
    Client           string              // 客户端地址（如 "127.0.0.1:50000"）
    clientConn       net.Conn            // 与客户端的活跃连接
    targetConn       net.Conn            // 与目标服务的活跃连接
    replayTime       bool                // 是否启用基于时间戳的延迟重放
    clientTimeRecord lockMap.Lock        // 记录最后一次向客户端发送包的时间（键："packageTime", "lastCalled"）
    targetTimeRecord lockMap.Lock        // 记录最后一次向目标发送包的时间（键："packageTime", "lastCalled"）
}
```

> **常量说明**：
> - `PackageTime = "packageTime"`：存储包原始时间戳。
> - `LastCalled = "lastCalled"`：存储上一次实际发送时间。

---

## 类型定义

### `DataParser`

```go
type DataParser func(data *proto.Packet) (*proto.Packet, error)
```

- **用途**：在重放前对原始数据包进行解析或修改（例如替换时间戳、篡改字段等）。
- **输入**：原始 `*proto.Packet`。
- **输出**：
    - 修改后的 `*proto.Packet`；
    - 若处理失败，返回非 `nil` 的 `error`，该包将被跳过。

---

## 方法说明

### 1. `SendSinglePacket(reader io.Reader, to string, parser DataParser)`

#### 功能
从 `reader` 中读取序列化的流量数据，筛选出 **发往指定地址 `to`** 的单向流量包，并依次重放（仅写入对应连接），支持自定义解析逻辑。

#### 参数
| 参数 | 类型 | 说明 |
|------|------|------|
| `reader` | `io.Reader` | 包含序列化 `proto.Packet` 数据的读取器（通常来自文件或内存缓冲区） |
| `to` | `string` | 目标地址（如 `"192.168.1.10:80"`），仅重放 `packet.To == to` 的包 |
| `parser` | `DataParser` | 可选的包解析/修改函数；若为 `nil`，则直接使用原始包 |

#### 行为说明
- 使用 `proto.ReadProtoFromReader` 解析流式数据；
- 对每个包调用 `parser`（若提供）；
- **仅当 `packet.To == to` 时才处理该包**（但当前实现中未实际发送，需注意：此方法目前为空操作，可能为占位或待完善）；
- 解析或反序列化错误将被记录日志，但不会中断流程。

> ⚠️ **注意**：当前方法体中缺少实际的发送逻辑（`if packet != nil && packet.To == to { }` 为空），调用者需确认是否已补充实现，或该方法仅为框架预留。

---

### 2. `SendPacket(packet proto.Packet) error`

#### 功能
根据包的来源（`packet.From`），将数据写入对应的连接（客户端或目标服务），并可选地根据时间戳进行延迟控制。

#### 参数
| 参数 | 类型 | 说明 |
|------|------|------|
| `packet` | `proto.Packet` | 要重放的数据包，必须包含有效 `From` 和 `Data` 字段 |

#### 返回值
- 成功：`nil`
- 失败：`error`（如连接写入失败、未知来源地址等）

#### 行为说明
1. 若 `packet.From == t.Client`：
    - 调用 `t.waitingAndSend(true, packet.Timestamp())`（等待至合适时间）；
    - 通过 `t.clientConn.Write(packet.Data)` 发送给客户端。
2. 若 `packet.From == t.Target`：
    - 调用 `t.waitingAndSend(false, packet.Timestamp())`；
    - 通过 `t.targetConn.Write(packet.Data)` 发送给目标服务。
3. 否则返回错误：`"packet from error"`。

> ✅ **关键特性**：
> - 支持双向流量重放；
> - 自动路由到正确连接；
> - 时间控制由 `waitingAndSend` 实现（依赖 `replayTime` 字段及 `clientTimeRecord` / `targetTimeRecord`）。

---

## 使用示例（伪代码）

```go
// 初始化连接
clientConn, _ := net.Dial("tcp", "127.0.0.1:50000")
targetConn, _ := net.Dial("tcp", "192.168.1.10:80")

player := &TcpPlayer{
    Client:     "127.0.0.1:50000",
    Target:     "192.168.1.10:80",
    clientConn: clientConn,
    targetConn: targetConn,
    replayTime: true, // 启用时间同步重放
}

// 重放一个包
err := player.SendPacket(recordedPacket)
if err != nil {
    log.Fatal(err)
}

// 或从文件重放单向流量（注意：当前 SendSinglePacket 无实际发送逻辑）
file, _ := os.Open("traffic.bin")
defer file.Close()
player.SendSinglePacket(file, "192.168.1.10:80", myCustomParser)
```

---

## 注意事项

1. **连接管理**：调用者需确保 `clientConn` 和 `targetConn` 在调用期间保持有效。
2. **线程安全**：`lockMap.Lock` 用于保护时间记录，但 `TcpPlayer` 本身非完全线程安全，建议单 goroutine 使用或外部加锁。
3. **时间重放**：`replayTime` 为 `true` 时，`waitingAndSend` 会根据包间时间差进行 sleep，以模拟真实时序。
4. **SendSinglePacket 状态**：当前实现未实际发送数据，请确认是否为待完成功能。

--- 

> 文档版本：v1.0  
> 最后更新：2026-01-12
```