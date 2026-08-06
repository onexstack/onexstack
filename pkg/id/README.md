# ID 生成（id / rid）

> OnexStack 项目中所有 ID 生成的统一规范，面向开发者与 AI 编码使用。

---

## 1. 总体架构

OnexStack 采用**三层 ID 体系**：

```
Layer 1: 唯一数字 ID（uint64）           →  由 Sonyflake 算法生成
Layer 2: 人类可读短码（string）          →  由 NewCode 将 uint64 编码为短字符串
Layer 3: 带前缀的资源 ID（ResourceID）   →  格式: {prefix}-{code}
```

**调用层次关系：**

```
Sonyflake.Id() → uint64
    ↓
NewCode(uint64) → string (如 "VHB4JX86")
    ↓
ResourceID.New(uint64) → string (如 "post-VHB4JX")
```

---

## 2. 包结构

| 包路径 | 职责 | 何时使用 |
|--------|------|----------|
| `pkg/id` | 底层 ID 生成（Sonyflake 雪花 ID + NewCode 短码） | 需要原始 uint64 唯一 ID、或将唯一数字编码为短字符串 |
| `pkg/rid` | 资源 ID 封装（前缀 + 短码） | 生成面向用户的资源标识符，如 `user-XXXXXX`, `post-XXXXXX` |

---

## 3. 组件详解

### 3.1 Sonyflake — 分布式唯一 ID

基于 [sonyflake](https://github.com/sony/sonyflake)，Snowflake 算法的变体。

- **输出**：`uint64`，全局唯一
- **特性**：
  - 默认 startTime 为 `2022-10-10T00:00:00Z`（**项目固定值，严禁在运行时修改**，否则会导致 ID 重复）
  - 默认 machineId 为 `1`（需根据实际部署配置）

#### 安装

```bash
go get -u github.com/onexstack/onexstack/pkg/id
```

#### 用法

```go
import (
    "context"
    "fmt"

    "github.com/onexstack/onexstack/pkg/id"
)

func main() {
    // 标准初始化（每个服务实例化一次，全局复用）
    sf := id.NewSonyflake(
        id.WithSonyflakeMachineId(1),   // 可选，生产环境必须配置唯一机器 ID
        // startTime 不要修改！使用项目默认值
    )
    if sf.Error != nil {
        fmt.Println(sf.Error)
        return
    }
    fmt.Println(sf.Id(context.Background()))
}
```

#### 配置选项

| 选项函数 | 类型 | 默认值 | 说明 |
|----------|------|--------|------|
| `WithSonyflakeMachineId(id)` | `uint16` | `1` | **生产环境必须配置不同值**，范围 0~65535 |
| `WithSonyflakeStartTime(t)` | `time.Time` | `2022-10-10` | **不要修改**，一旦修改历史 ID 全部失效 |

#### 使用规则

- Sonyflake 实例必须是**全局单例**，整个进程只创建一个
- 初始化失败（`sf.Error != nil`）应立即 panic，不可降级运行
- 调用 `Id()` 时内部已实现指数退避重试，接口层无需额外处理

---

### 3.2 NewCode — 唯一数字到短码的编码

将一个 `uint64` 唯一 ID 转换为**固定长度的非重复短字符串**。

- **算法**：扩散（diffusion）+ 混淆（P-box permutation），不可逆编码
- **输出**：默认 8 位字符串，由自定义字符集组成

#### 用法

```go
import (
    "fmt"
    "github.com/onexstack/onexstack/pkg/id"
)

func main() {
    fmt.Println(id.NewCode(1))  // "VHB4JX86"
    fmt.Println(id.NewCode(2))  // "VHB4JX87"
    fmt.Println(id.NewCode(3))  // "VHB4JX88"
    fmt.Println(id.NewCode(4))  // "VHB4JX89"

    // 带自定义选项
    fmt.Println(id.NewCode(
        1,
        id.WithCodeChars([]rune{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}),
        id.WithCodeN1(9),
        id.WithCodeN2(3),
        id.WithCodeL(5),
        id.WithCodeSalt(99999),
    ))  // "80773"
}
```

#### 配置选项

| 选项函数 | 类型 | 默认值 | 说明 |
|----------|------|--------|------|
| `WithCodeChars(arr)` | `[]rune` | 见下方默认字符集 | 自定义字符集，长度 >= 2，每个字符从此集合生成 |
| `WithCodeL(l)` | `int` | `8` | 输出字符串长度，> 0 |
| `WithCodeN1(n)` | `int` | `17` | 扩散系数，必须与字符集长度互质 |
| `WithCodeN2(n)` | `int` | `5` | 混淆系数，必须与输出长度互质 |
| `WithCodeSalt(salt)` | `uint64` | `123567369` | 盐值，不同盐值可生成完全不同的编码结果。相同 options 和相同 id 始终生成相同 code |

#### 默认字符集

```text
['2','3','4','5','6','7','8','9',
 'A','B','C','D','E','F','G','H',
 'J','K','L','M','N','P','Q','R',
 'S','T','V','W','X','Y']
```

**设计原则：**

- 排除了易混淆字符：`0, 1, I, O, U, Z`（避免用户误读）
- 共计 29 个字符，提供约 29^8 ≈ 500 万亿种组合

#### 使用规则

- 不可将 NewCode 用作安全令牌或密码（非加密算法，仅混淆）
- 不同的 `salt` + 相同的 `id` 会产生不同的 code —— 可用于定期轮换
- 相同的参数 + 相同的 id = 相同的结果（确定性算法）

---

### 3.3 ResourceID — 资源标识符

为每种资源类型生成带前缀的唯一标识符，位于 `pkg/rid` 包。

- **格式**：`{prefix}-{code}`，如 `post-VHB4JX`、`user-ABC123`

#### 用法

```go
import (
    "context"
    "fmt"

    "github.com/onexstack/onexstack/pkg/id"
    "github.com/onexstack/onexstack/pkg/rid"
)

func main() {
    // 定义资源类型（通常在领域层定义一次）
    var PostID = rid.NewResourceID("post")
    var UserID = rid.NewResourceID("user")
    var OrderID = rid.NewResourceID("order")

    sf := id.NewSonyflake(id.WithSonyflakeMachineId(1))

    // 生成资源 ID（需要传入一个 uint64 唯一 ID）
    postId := PostID.New(sf.Id(context.Background()))
    fmt.Println(postId)  // 输出: "post-VHB4JX"
}
```

#### 前缀命名规范

| 规则 | 正确示例 | 错误示例 |
|------|----------|----------|
| 全小写字母 | `user`, `order`, `blog_post` | `User`, `ORDER`, `blog-post` |
| 使用下划线分隔 | `blog_post`, `access_token` | `blog-post`, `accessToken` |
| 短且有语义 | `post`, `user` | `p`, `xyz` |
| 单数形式 | `user`, `order` | `users`, `orders` |

#### Salt 机制

`rid.New()` 内部自动调用 `rid.Salt()` 生成机器级盐值，盐值来源于：

1. `/etc/machine-id`（Linux）
2. `/sys/class/dmi/id/product_uuid`（Linux 备选）
3. `os.Hostname()`
4. 随机数（最终兜底）

同一台机器始终生成相同的盐值，确保编码结果可复现。

---

## 4. 使用模式（Patterns）

### 4.1 模式一：纯唯一 ID（用于数据库主键、内部系统）

**场景：** 数据库主键、内部关联、日志追踪

```go
// 初始化（main.go 或 wire 注入）
sf := id.NewSonyflake(id.WithSonyflakeMachineId(machineID))
if sf.Error != nil {
    panic(sf.Error)
}

// 使用
uid := sf.Id(context.Background())
// 直接入库: INSERT INTO users (id, name) VALUES (uid, "Alice")
```

### 4.2 模式二：短码（用于对外展示、邀请码、分享码）

**场景：** 用户可见的短标识、邀请码、激活码

```go
sf := id.NewSonyflake(id.WithSonyflakeMachineId(machineID))
code := id.NewCode(sf.Id(context.Background()))
// 输出: "VHB4JX86"
```

### 4.3 模式三：类型化资源 ID（REST API 资源标识）

**场景：** 面向最终用户的资源 ID，需在 URL / API / 前端中使用

```go
// --- 定义阶段（领域层，定义一次）---
var PostID = rid.NewResourceID("post")
var UserID = rid.NewResourceID("user")

// --- 使用阶段（业务逻辑）---
sf := id.NewSonyflake(id.WithSonyflakeMachineId(machineID))
postId := PostID.New(sf.Id(context.Background()))
// 输出: "post-3xk9am2d"
```

---

## 5. 编码规范

### 5.1 Sonyflake 实例化

```go
// ✅ 正确：全局单例，启动时创建一次
var globalSF *id.Sonyflake

func init() {
    machineID := getMachineIDFromConfig() // 从配置获取唯一机器 ID
    globalSF = id.NewSonyflake(
        id.WithSonyflakeMachineId(machineID),
    )
    if globalSF.Error != nil {
        panic(globalSF.Error)
    }
}

// ❌ 错误：每次调用都创建新实例
func createOrder() {
    sf := id.NewSonyflake()  // 每次都新建，浪费且可能 ID 重复
    orderID := sf.Id(context.Background())
}
```

### 5.2 ResourceID 定义

```go
// ✅ 正确：在领域包中集中定义
// internal/domain/identifier.go
package domain

var (
    UserResourceID  = rid.NewResourceID("user")
    PostResourceID  = rid.NewResourceID("post")
    OrderResourceID = rid.NewResourceID("order")
)

// ❌ 错误：在业务代码中散落定义
func createUser() {
    userRID := rid.NewResourceID("user") // 应该预先定义
}
```

### 5.3 完整示例：在服务中使用 ID 生成

```go
package service

import (
    "context"
    "github.com/onexstack/onexstack/pkg/id"
    "github.com/onexstack/onexstack/pkg/rid"
)

// PostService 博文服务
type PostService struct {
    sf *id.Sonyflake
}

// CreatePost 创建博文
func (s *PostService) CreatePost(ctx context.Context, req *CreatePostReq) (*Post, error) {
    // 1. 生成唯一 ID
    uid := s.sf.Id(ctx)

    // 2. 生成资源 ID（用户可见）
    resourceID := rid.NewResourceID("post").New(uid)

    post := &Post{
        ID:         uid,        // uint64，用作数据库主键
        ResourceID: resourceID, // string，"post-XXXXXX"，用于 API 返回
        Title:      req.Title,
        Content:    req.Content,
    }

    return post, s.repo.Create(ctx, post)
}
```

---

## 6. 不可修改的常量

以下默认配置是项目的**基础设施约定**，新增服务或模块时**必须遵守**：

| 配置项 | 值 | 修改后果 |
|--------|-----|----------|
| Sonyflake StartTime | `2022-10-10T00:00:00Z` | 导致所有历史 ID 失效，分布式环境 ID 冲突 |
| NewCode 默认字符集 | 排除 0/1/I/O/U/Z（29 字符） | 产生易混淆字符，用户体验下降 |
| NewCode 默认长度 | 8 | 编码空间缩小，可能产生碰撞 |
| Salt 来源 | 机器指纹（machine-id → hostname → 随机） | 降低编码分散性 |

---

## 7. AI 编码 Checklist

当 AI 要编写涉及 ID 生成的代码时，应按以下清单检查：

- [ ] **Sonyflake 实例**：是否只创建了一个全局实例？
- [ ] **StartTime**：是否使用了默认 StartTime（未传入 `WithSonyflakeStartTime`）？
- [ ] **MachineID 配置**：生产环境是否从配置中读取了唯一机器 ID？
- [ ] **错误处理**：Sonyflake 初始化失败是否 panic 了？
- [ ] **ResourceID 前缀**：是否符合小写 + 单数 + 下划线分隔规范？
- [ ] **ResourceID 定义位置**：是否集中定义在领域层而非散落在业务代码？
- [ ] **ID 用途**：
  - 数据库主键 → 使用 `uint64`（Sonyflake 原始值）
  - 对外展示 → 使用 `ResourceID.New()` 生成的带前缀短码
  - API 请求/响应 → 使用 `ResourceID` 字符串
- [ ] **不要自定义字符集**：除非有特殊业务需求，否则使用默认字符集
- [ ] **不要修改 Salt 来源逻辑**：`rid.Salt()` 的实现是基础设施代码，不要修改
