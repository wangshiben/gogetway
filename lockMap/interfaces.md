# 接口文档

## 1. `LockGroup` 接口

用于管理一组锁（`Lock`），支持动态创建、遍历、过滤及销毁。

### 方法列表

#### `GetLockedGroup(ctx context.Context, From string) (nextGroup LockGroup, isContinue bool, err error)`

- **功能**：获取下一个子锁组。
- **用途**：避免因频繁创建锁而阻塞整个锁系统。当当前锁组资源耗尽时，可调用此方法获取新的子锁组。
- **参数**：
    - `ctx`: 上下文，用于控制超时或取消。
    - `From`: 调用来源标识（如模块名、IP 等）。
- **返回**：
    - `nextGroup`: 下一个可用的 `LockGroup`。
    - `isContinue`: 是否还能继续获取更多锁组（`false` 表示已到末尾）。
    - `err`: 错误信息。

---

#### `GetLock(ctx context.Context, From string) (lock Lock, err error)`

- **功能**：从当前锁组中获取一个锁。
- **用途**：在获得新 `LockGroup` 后，优先尝试从此方法获取具体锁实例。
- **参数**：同上。
- **返回**：
    - `lock`: 获取到的锁；若无可用锁则返回 `nil`。
    - `err`: 错误信息。

---

#### `NewLockOrGroup(ctx context.Context, From string) (lock Lock, lockGroup LockGroup, err error)`

- **功能**：创建一个锁或一个锁组（二选一）。
- **建议**：通常只需创建其中之一即可满足需求。
- **返回**：
    - 若创建成功，`lock` 或 `lockGroup` 其中一个非空。
    - `err`: 错误信息。

---

#### `CreateLockGroup(ctx context.Context, From string) (lockGroup LockGroup, err error)`

- **功能**：显式创建一个新的锁组。
- **触发条件**：当 `GetLockedGroup` 返回 `isContinue = false` 且 `GetLock` 返回 `nil` 时，需决定是否创建新锁组。
- **返回**：新创建的 `LockGroup` 实例。

---

#### `CreateLock(ctx context.Context, From string) (lock Lock, err error)`

- **功能**：显式创建一个独立的锁。
- **使用场景**：不依赖锁组时直接创建单个锁。

---

#### `Destroy()`

- **功能**：销毁当前锁组。
- **注意**：仅在 `CheckLocks` 检测到该锁组未被使用时调用，**务必谨慎**，避免误删活跃锁组。

---

#### `CanDestroy() bool`

- **功能**：判断当前锁组是否可以安全销毁。
- **返回**：`true` 表示无活跃锁，可销毁。

---

#### `DefaultLock() RWLock`

- **功能**：获取一个默认的读写锁。
- **用途**：在调用 `NewLockOrGroup` 前提供一个基础锁实现。

---

#### `FilterChains() []FilterChain`

- **功能**：返回一组数据包过滤链。
- **触发时机**：当进入一个新的 `LockGroup` 且启用了数据包过滤时调用。
- **返回**：按顺序执行的 `FilterChain` 函数列表。

---

#### `CheckLocks(ctx context.Context)`

- **功能**：检查锁组中锁的使用状态。
- **调用时机**：
    1. 系统内存不足时；
    2. 定期执行（如每 1 分钟或 10 分钟，依配置而定）。
- **作用**：清理未使用的锁或锁组，释放资源。

---

## 2. `Lock` 接口

表示一个基本的互斥锁，支持状态查询、引用计数及元数据管理。

| 方法 | 说明 |
|------|------|
| `Lock()` | 获取独占锁（阻塞）。 |
| `Unlock()` | 释放锁。 |
| `Other() interface{}` | 获取附加的元数据对象。 |
| `UpdateOther(other interface{}) error` | 更新元数据，可能失败（如类型不兼容）。 |
| `IsLocked() bool` | 判断当前是否已被锁定。 |
| `GetIndex() uint64` | 获取锁的唯一索引（用于追踪）。 |
| `LastCalled() int64` | 返回最后一次被访问的时间戳（Unix 纳秒）。 |
| `Release(count uint)` | 减少引用计数（用于引用计数型锁）。 |
| `CanRelease() bool` | 判断是否可以安全释放（引用计数为 0 等）。 |
| `IncreaseGetIndex() uint64` | 原子递增并返回新的获取索引（用于调试/追踪并发获取次数）。 |
| `AddRelease()` | 增加一次“待释放”计数（配合 `Release` 使用）。 |

---

## 3. `RWLock` 接口

扩展 `Lock`，支持读写分离。

| 方法 | 说明 |
|------|------|
| `RLock()` | 获取共享读锁（允许多个读）。 |
| `RUnlock()` | 释放读锁。 |

> **继承关系**：`RWLock` 包含 `Lock` 的所有方法，因此也支持 `Lock()` / `Unlock()` 等独占操作。

---

## 4. `CtxWithValue` 接口

扩展标准 `context.Context`，支持运行时动态注入键值对。

| 方法 | 说明 |
|------|------|
| `Put(key string, value any)` | 向上下文中存入键值对（非线程安全，仅限当前协程使用）。 |
| `Clear()` | 清空所有自定义键值（谨慎使用）。 |

> **注意**：该接口主要用于内部上下文增强，不建议在公共 API 中暴露。

---

## 5. `FilterChain` 类型

```go
type FilterChain func(ctx context.Context, From string, bytes []byte) (isContinue bool, err error)