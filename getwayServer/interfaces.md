## 1. `ConnectResource` 接口

表示一个与网络连接绑定的可写资源，封装了写入能力、写入队列及底层锁等信息。

### 方法列表

| 方法 | 说明 |
|------|------|
| `Writer() io.Writer` | 返回一个标准的 `io.Writer`，用于直接写入数据。 |
| `WriteFunc() WriteFunc` | 返回一个自定义的写函数（通常用于异步或批量写），其签名一般为 `func([]byte) error`。 |
| `WriteQueue() *WriteQueue` | 返回关联的写队列指针，用于管理待发送的数据缓冲。 |
| `GetLock() lockMap.Lock` | 获取该资源关联的底层锁（用于同步写操作或状态变更）。 |
| `WriteType() string` | 返回当前资源使用的写入类型标识（如 `"tcp"`, `"websocket"`, `"buffered"` 等），便于调试或路由策略。 |

> **注**：被注释掉的方法 `currentIndex()` 表明曾考虑暴露内部索引结构，但当前未启用。

---

## 2. `ResourceGroup` 接口

用于按上下文或来源动态创建和管理 `ConnectResource` 实例，实现资源池化或按需初始化。

### 方法列表

#### `GetResource(ctx context.Context, Connect net.Conn) (resource ConnectResource, err error)`

- **功能**：根据传入的网络连接 `net.Conn` 获取对应的 `ConnectResource`。
- **用途**：在新连接建立时，由框架调用以绑定资源。
- **参数**：
    - `ctx`: 请求上下文，可用于传递元数据或控制生命周期。
    - `Connect`: 底层网络连接（如 TCP 连接）。
- **返回**：
    - `resource`: 与该连接绑定的资源实例。
    - `err`: 初始化失败时的错误。

---

#### `NewResourceFunc(ctx context.Context, From string) NewResourceFunc`

- **功能**：返回一个用于创建新 `ConnectResource` 的工厂函数。
- **用途**：支持按来源（`From`）定制资源创建逻辑（如不同客户端使用不同写策略）。
- **参数**：
    - `ctx`: 上下文。
    - `From`: 调用来源标识（如服务名、IP、协议类型等）。
- **返回**：
    - `NewResourceFunc`: 函数类型，通常签名为 `func(net.Conn) (ConnectResource, error)`。

> **典型使用场景**：  
> 在代理或网关服务中，根据 `From`（如 `"internal-service"` vs `"external-client"`）返回不同的 `WriteFunc` 或 `WriteQueue` 配置。

---

## 补充说明

- **`WriteFunc` 与 `WriteQueue` 协同**：  
  通常 `WriteFunc` 会将数据推入 `WriteQueue`，由后台协程异步消费并调用 `Writer().Write()`，从而避免阻塞业务逻辑。

- **线程安全**：  
  所有对 `ConnectResource` 的并发写操作应通过 `GetLock()` 获取锁后进行，或依赖 `WriteQueue` 的内部同步机制。

- **生命周期**：  
  `ConnectResource` 的生命周期通常与 `net.Conn` 一致，连接关闭时应清理其 `WriteQueue` 并释放锁资源。

---