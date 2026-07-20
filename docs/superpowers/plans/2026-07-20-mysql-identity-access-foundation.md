# MySQL 身份与访问控制底座实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 MCPJungle 改造成仅使用 MySQL 8.0 的内部 MCP Hub 后端底座，打通内部账户、服务端会话、权限组、个人设备令牌与 MCP 调用授权。

**架构：** 人员身份与 MCP 调用凭证彻底分离。Web 管理端使用服务端 Session Cookie；MCP 客户端只使用个人设备令牌。用户通过多个权限组获得 MCP 服务并集，受限设备令牌只能在用户权限内继续收窄，所有授权判断默认拒绝。

**技术栈：** Go 1.24、Gin、GORM、MySQL 8.0、bcrypt、React/Ant Design（本阶段只保证现有前端可编译，页面重构在独立计划中完成）

---

## 文件结构与职责

### 数据库与启动

- 修改 `internal/db/db.go`：只接受 MySQL URL 或原生 DSN，统一追加 `utf8mb4`、`parseTime=True`、`loc=UTC`，拒绝空 DSN和非 MySQL DSN。
- 重写 `internal/db/db_test.go`：覆盖 MySQL DSN 规范化、密码脱敏、非法协议拒绝，不再创建 SQLite 文件。
- 修改 `cmd/start.go`、`cmd/start_test.go`：删除 SQLite/PostgreSQL 环境变量和 CLI 参数，启动时只解析 `DATABASE_URL` 或 `MYSQL_*`。
- 修改 `pkg/testhelpers/testhelpers.go`：测试数据库仅连接 `MCPJUNGLE_TEST_DATABASE_URL` 指向的 MySQL 8.0，按测试创建独立表并清理。
- 创建 `docker-compose.test.yaml`：提供本地/CI 共用的 MySQL 8.0 测试实例。
- 修改 `go.mod`、`go.sum`：删除 SQLite、PostgreSQL 驱动及其传递依赖。

### 身份、会话与权限

- 修改 `internal/model/user.go`、`pkg/types/user.go`：定义四种角色、三种账户状态、首次改密字段，删除旧 AccessToken 和 AllowedServers。
- 创建 `internal/model/user_session.go`：持久化服务端会话摘要、到期时间、吊销时间和最近访问时间。
- 创建 `internal/model/permission_group.go`：定义权限组、用户成员关系和 MCP 服务关系。
- 创建 `internal/model/device_token.go`：定义设备令牌摘要、作用域模式和受限服务关系。
- 修改 `internal/migrations/migration.go`：只迁移新模型，不做旧字段或旧客户端数据回填。
- 创建 `internal/auth/token.go`：集中生成高熵随机令牌、SHA-256 摘要和可展示前缀。
- 创建 `internal/service/session/session.go`：登录会话创建、校验、续用、登出及整人吊销。
- 创建 `internal/service/permission/permission.go`：权限组管理与用户有效服务集合计算。
- 创建 `internal/service/devicetoken/device_token.go`：个人设备令牌创建、列出、撤销和鉴权。
- 重写 `internal/service/user/user.go`：账户创建、首次改密、禁用/启用和角色管理。

### HTTP 与 MCP 接入

- 创建 `internal/api/v1_auth.go`、`v1_users.go`、`v1_permission_groups.go`、`v1_device_tokens.go`：提供 `/api/v1` 接口。
- 修改 `internal/api/server.go`、`server_config.go`：注入新服务并注册 `/api/v1` 路由。
- 重写 `internal/api/middleware.go`：每次请求从数据库恢复活跃会话和用户状态，按角色检查管理权限。
- 修改 `internal/service/mcp/proxy_filter.go`、`proxy.go`：使用设备令牌的有效服务集合过滤工具并阻止直接调用。
- 删除 `internal/model/mcp_client.go`、`internal/service/mcpclient/`、`internal/api/mcp_clients.go` 及相应类型/客户端入口：彻底移除旧 McpClient 凭证模型。

## API 与领域契约

### 固定枚举

```go
type UserRole string
const (
    UserRoleSystemAdmin  UserRole = "system_admin"
    UserRoleServiceAdmin UserRole = "service_admin"
    UserRoleMember       UserRole = "member"
    UserRoleAuditor      UserRole = "auditor"
)

type UserStatus string
const (
    UserStatusPending  UserStatus = "pending"
    UserStatusActive   UserStatus = "active"
    UserStatusDisabled UserStatus = "disabled"
)

type DeviceTokenScope string
const (
    DeviceTokenScopeInheritAll DeviceTokenScope = "inherit_all"
    DeviceTokenScopeRestricted DeviceTokenScope = "restricted"
)
```

### 有效权限公式

```text
user_services = UNION(用户所有 active 权限组关联的 MCP 服务)

token.scope = inherit_all:
effective_services = user_services

token.scope = restricted:
effective_services = INTERSECTION(user_services, token.selected_services)
```

用户没有活跃权限组、权限组没有服务、设备令牌已撤销/过期、用户不是 active 状态时，结果都必须是拒绝全部。

### 第一阶段接口

```text
POST   /api/v1/auth/login
POST   /api/v1/auth/logout
GET    /api/v1/auth/me
PUT    /api/v1/auth/password

POST   /api/v1/users
GET    /api/v1/users
PATCH  /api/v1/users/:id
POST   /api/v1/users/:id/enable
POST   /api/v1/users/:id/disable

POST   /api/v1/permission-groups
GET    /api/v1/permission-groups
GET    /api/v1/permission-groups/:id
PATCH  /api/v1/permission-groups/:id
PUT    /api/v1/permission-groups/:id/users
PUT    /api/v1/permission-groups/:id/services
POST   /api/v1/permission-groups/:id/enable
POST   /api/v1/permission-groups/:id/disable

POST   /api/v1/device-tokens
GET    /api/v1/device-tokens
DELETE /api/v1/device-tokens/:id
```

登录成功写入 `mcpj_session` Cookie，属性固定为 `HttpOnly`、`SameSite=Lax`、8 小时有效；生产请求使用 `Secure`。设备令牌明文只在创建响应中返回一次，数据库只保存 SHA-256 摘要与短前缀。

---

### 任务 1：收敛为 MySQL 8.0 单数据库

**文件：**
- 修改：`internal/db/db.go`
- 重写：`internal/db/db_test.go`
- 修改：`cmd/start.go`
- 修改：`cmd/start_test.go`
- 修改：`pkg/testhelpers/testhelpers.go`
- 创建：`docker-compose.test.yaml`
- 修改：`go.mod`
- 修改：`go.sum`

- [ ] **步骤 1：先把数据库测试改成 MySQL 契约**

测试至少包含以下表格：

```go
func TestNormalizeMySQLDSN(t *testing.T) {
    tests := []struct{ name, input, want string }{
        {"URL", "mysql://user:pass@db:3306/hub", "user:pass@tcp(db:3306)/hub?charset=utf8mb4&parseTime=True&loc=UTC"},
        {"原生 DSN", "user:pass@tcp(db:3306)/hub", "user:pass@tcp(db:3306)/hub?charset=utf8mb4&parseTime=True&loc=UTC"},
        {"保留参数", "user:pass@tcp(db:3306)/hub?timeout=5s", "user:pass@tcp(db:3306)/hub?timeout=5s&charset=utf8mb4&parseTime=True&loc=UTC"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NormalizeMySQLDSN(tt.input)
            require.NoError(t, err)
            require.Equal(t, tt.want, got)
        })
    }
}

func TestNewDBConnectionRejectsEmptyDSN(t *testing.T) {
    db, err := NewDBConnection("")
    require.Nil(t, db)
    require.ErrorContains(t, err, "MySQL DSN is required")
}

func TestNewDBConnectionRejectsPostgresDSN(t *testing.T) {
    db, err := NewDBConnection("postgres://user:pass@db:5432/hub")
    require.Nil(t, db)
    require.ErrorContains(t, err, "only MySQL is supported")
}

func TestMaskDSNRedactsPassword(t *testing.T) {
    masked := maskDSN("user:secret@tcp(db:3306)/hub")
    require.Equal(t, "user:***@tcp(db:3306)/hub", masked)
    require.NotContains(t, masked, "secret")
}
```

- [ ] **步骤 2：运行红灯测试**

运行：`go test ./internal/db ./cmd -run 'TestNormalizeMySQLDSN|TestNewDBConnectionRejects|TestMaskDSN|TestGetMysqlDSN'`

预期：FAIL，原因是现有连接函数仍回退 SQLite、接受 PostgreSQL，且默认时区为 Local。

- [ ] **步骤 3：实现 MySQL 专用连接与启动配置**

将公开签名收敛为：

```go
func NewDBConnection(dsn string) (*gorm.DB, error)
func NormalizeMySQLDSN(raw string) (string, error)
func getMysqlDSN() (string, error)
```

`getMysqlDSN` 优先读取 `DATABASE_URL`，否则要求 `MYSQL_HOST`，并从 `MYSQL_PORT`、`MYSQL_USER[_FILE]`、`MYSQL_PASSWORD[_FILE]`、`MYSQL_DB[_FILE]` 组装 DSN。两条路径都进入 `NormalizeMySQLDSN`。删除 `SQLITE_DB_PATH`、全部 `POSTGRES_*` 常量和 `--sqlite-db-path`。

- [ ] **步骤 4：测试辅助函数只使用 MySQL**

将测试入口改为：

```go
func RequireTestDB(t *testing.T) *gorm.DB
```

它读取 `MCPJUNGLE_TEST_DATABASE_URL`；未配置时调用 `t.Skip`；配置后连接 MySQL，为当前测试使用唯一表前缀或独立数据库，并在 `t.Cleanup` 中清理。所有原 `CreateTestDB()` 调用改为 `RequireTestDB(t)`。

- [ ] **步骤 5：提供 MySQL 8.0 测试容器并删除驱动**

`docker-compose.test.yaml` 固定 `mysql:8.0`，数据库名 `mcpjungle_test`，健康检查使用 `mysqladmin ping`。运行 `go mod tidy` 后确认 `go.mod` 不再包含 `glebarez/sqlite`、`gorm.io/driver/sqlite`、`gorm.io/driver/postgres`。

- [ ] **步骤 6：运行绿灯验证**

运行：

```powershell
go test ./internal/db ./cmd
go test -run '^$' ./...
rg -n "sqlite|postgres|SQLITE_|POSTGRES_" --glob '!docs/superpowers/specs/**' --glob '!docs/superpowers/plans/**'
```

预期：测试和全仓编译通过；搜索结果只允许出现在历史变更日志，不允许出现在运行时、测试、Compose 或当前说明中。

- [ ] **步骤 7：提交任务 1**

提交意图：`refactor(数据库)!: 统一运行与测试环境为 MySQL 8.0`

---

### 任务 2：建立新身份与授权数据模型

**文件：**
- 修改：`internal/model/user.go`
- 创建：`internal/model/user_session.go`
- 创建：`internal/model/permission_group.go`
- 创建：`internal/model/device_token.go`
- 创建：`internal/model/access_control_test.go`
- 修改：`internal/migrations/migration.go`
- 修改：`pkg/types/user.go`

- [ ] **步骤 1：编写模型与迁移失败测试**

创建 MySQL 集成测试并断言：

```go
func TestAccessControlMigrationCreatesExpectedTables(t *testing.T) {
    db := testhelpers.RequireTestDB(t)
    require.NoError(t, migrations.Migrate(db))
    for _, table := range []string{
        "users", "user_sessions", "permission_groups",
        "permission_group_users", "permission_group_mcp_servers",
        "device_tokens", "device_token_services",
    } {
        require.True(t, db.Migrator().HasTable(table), table)
    }
    require.False(t, db.Migrator().HasTable("mcp_clients"))
}
```

再添加唯一约束测试：用户名、权限组名、Session 摘要和设备令牌摘要不可重复；两张关联表的联合主键不可重复。

- [ ] **步骤 2：运行测试验证因模型缺失而失败**

运行：`go test ./internal/model -run 'TestAccessControlMigration|TestAccessControlUniqueConstraints'`

- [ ] **步骤 3：实现模型**

`User` 保留 GORM 主键与时间戳，字段固定为：`Username`、`DisplayName`、`Role`、`Status`、`PasswordHash`、`MustChangePassword`、`LastLoginAt`。删除 `AccessToken`、`AllowedServers` 及其方法。

关联模型使用显式联合主键：

```go
type PermissionGroupUser struct {
    PermissionGroupID uint `gorm:"primaryKey"`
    UserID            uint `gorm:"primaryKey"`
}

type PermissionGroupMcpServer struct {
    PermissionGroupID uint `gorm:"primaryKey"`
    McpServerID       uint `gorm:"primaryKey"`
}

type DeviceTokenService struct {
    DeviceTokenID uint `gorm:"primaryKey"`
    McpServerID   uint `gorm:"primaryKey"`
}
```

- [ ] **步骤 4：迁移只登记新模型**

删除 `McpClient` 迁移与 `backfillClientUserIDs`。按外键依赖顺序迁移用户、服务、权限组、关联表、Session、设备令牌和设备令牌服务。

- [ ] **步骤 5：运行绿灯与全仓编译**

运行：`go test ./internal/model ./internal/migrations && go test -run '^$' ./...`

- [ ] **步骤 6：提交任务 2**

提交意图：`feat(身份模型)!: 建立账户权限组与设备令牌数据结构`

---

### 任务 3：实现账户生命周期与服务端会话

**文件：**
- 创建：`internal/auth/token.go`
- 创建：`internal/auth/token_test.go`
- 重写：`internal/service/user/user.go`
- 重写：`internal/service/user/user_test.go`
- 创建：`internal/service/session/session.go`
- 创建：`internal/service/session/session_test.go`
- 删除：`internal/auth/jwt.go`

- [ ] **步骤 1：为令牌工具写红灯测试**

断言 `Generate("mjs_", 32)` 连续生成不同令牌，`Hash` 输出 64 位小写十六进制，`Prefix` 不泄露完整令牌，`subtle.ConstantTimeCompare` 可校验摘要。

- [ ] **步骤 2：为账户生命周期写红灯测试**

覆盖以下行为：系统管理员创建 pending 用户并生成一次性初始密码；pending 用户登录后必须改密；改密后状态变 active；禁用用户时同时吊销其所有 Session 和设备令牌；禁止硬删除；最后一个 active 系统管理员不能被禁用或降权。

- [ ] **步骤 3：为 Session 写红灯测试**

覆盖创建 8 小时会话、按摘要查找、过期拒绝、已吊销拒绝、关联用户非 active 时拒绝、登出只吊销当前会话、管理员禁用用户后所有会话立即失效。

- [ ] **步骤 4：运行红灯**

运行：`go test ./internal/auth ./internal/service/user ./internal/service/session`

预期：FAIL，原因是新服务和契约尚未实现。

- [ ] **步骤 5：实现最少业务代码**

服务签名固定为：

```go
func (s *UserService) Create(input CreateUserInput) (*model.User, string, error)
func (s *UserService) VerifyPassword(username, password string) (*model.User, error)
func (s *UserService) ChangePassword(userID uint, current, next string) error
func (s *UserService) SetStatus(actorID, userID uint, status types.UserStatus) error
func (s *UserService) UpdateRole(actorID, userID uint, role types.UserRole) error

func (s *Service) Create(userID uint, ip, userAgent string) (plain string, session *model.UserSession, err error)
func (s *Service) Authenticate(plain string) (*model.User, *model.UserSession, error)
func (s *Service) Revoke(sessionID uint) error
func (s *Service) RevokeAllForUser(userID uint) error
```

密码最少 12 个字符；响应和日志均不得出现密码哈希、Session 摘要或完整令牌。

- [ ] **步骤 6：运行绿灯**

运行：`go test ./internal/auth ./internal/service/user ./internal/service/session`

- [ ] **步骤 7：提交任务 3**

提交意图：`feat(账户): 使用服务端会话落实账户全生命周期`

---

### 任务 4：接入 `/api/v1` 登录、账户与角色鉴权

**文件：**
- 创建：`internal/api/v1_auth.go`
- 创建：`internal/api/v1_auth_test.go`
- 创建：`internal/api/v1_users.go`
- 创建：`internal/api/v1_users_test.go`
- 修改：`internal/api/middleware.go`
- 修改：`internal/api/middleware_test.go`
- 修改：`internal/api/server.go`
- 修改：`internal/api/server_config.go`
- 修改：`cmd/start.go`

- [ ] **步骤 1：写 API 红灯测试**

使用 `httptest` 覆盖：登录成功设置 HttpOnly/SameSite Cookie；错误账号或密码统一返回 401；pending 登录返回 `must_change_password=true` 且只允许改密/登出；Cookie 无效、会话过期、用户禁用均返回 401；普通成员访问用户管理返回 403；系统管理员可创建/启停用户；列表响应不含任何哈希和令牌摘要。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/api -run 'TestV1Auth|TestV1Users|TestSessionMiddleware|TestRoleMiddleware'`

- [ ] **步骤 3：实现 Cookie 与中间件**

中间件从 `mcpj_session` Cookie 读取明文随机令牌，经 Session 服务查询数据库后把最新 `*model.User` 放入 Context；绝不信任客户端角色。角色矩阵：`system_admin` 管账户/权限组/全局配置，`service_admin` 预留服务管理，`member` 仅个人资源，`auditor` 预留只读审计。

- [ ] **步骤 4：注册 `/api/v1` 路由**

公开组只包含登录；其余路由依次经过初始化、Session 验证和角色中间件。现有 `/api/dashboard/auth/login` 不再签发 JWT，内部调用新登录逻辑，前端改造完成后再删除兼容入口。

- [ ] **步骤 5：运行绿灯与 API 包回归**

运行：`go test ./internal/api`

- [ ] **步骤 6：提交任务 4**

提交意图：`feat(API)!: 使用 Session Cookie 提供 v1 身份接口`

---

### 任务 5：实现权限组管理与有效服务集合

**文件：**
- 创建：`internal/service/permission/permission.go`
- 创建：`internal/service/permission/permission_test.go`
- 创建：`internal/api/v1_permission_groups.go`
- 创建：`internal/api/v1_permission_groups_test.go`
- 修改：`internal/api/server.go`

- [ ] **步骤 1：写权限计算红灯测试**

覆盖：多个 active 权限组取并集；disabled 权限组不生效；无组返回空集；重复服务去重；不存在的用户或服务返回明确错误；更新成员/服务使用事务全量替换，失败时不产生半套关系。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/service/permission`

- [ ] **步骤 3：实现权限服务**

固定接口：

```go
func (s *Service) EffectiveServiceIDs(userID uint) (map[uint]struct{}, error)
func (s *Service) ReplaceUsers(groupID uint, userIDs []uint) error
func (s *Service) ReplaceServices(groupID uint, serverIDs []uint) error
func (s *Service) SetEnabled(groupID uint, enabled bool) error
```

查询必须显式过滤 `permission_groups.enabled = true` 和关联用户 `users.status = 'active'`。

- [ ] **步骤 4：写并实现权限组 API**

系统管理员可以创建、查看、修改、启停并全量替换成员/服务。ID 数组去重；包含不存在 ID 时返回 400 且不写入；空数组表示清空关系，不表示全部授权。

- [ ] **步骤 5：运行绿灯**

运行：`go test ./internal/service/permission ./internal/api -run 'TestPermission|TestV1PermissionGroup'`

- [ ] **步骤 6：提交任务 5**

提交意图：`feat(权限): 以权限组计算用户可访问服务集合`

---

### 任务 6：实现个人设备令牌自助管理

**文件：**
- 创建：`internal/service/devicetoken/device_token.go`
- 创建：`internal/service/devicetoken/device_token_test.go`
- 创建：`internal/api/v1_device_tokens.go`
- 创建：`internal/api/v1_device_tokens_test.go`
- 修改：`internal/api/server.go`

- [ ] **步骤 1：写设备令牌红灯测试**

覆盖：active 用户可创建；pending/disabled 用户不可创建；每人最多 10 个 active 令牌；默认 90 天且不得提交超过 90 天的有效期；同名设备只在用户范围内唯一；明文只由创建方法返回；数据库只有摘要；撤销幂等；过期/撤销令牌鉴权失败；restricted 令牌提交的服务必须属于用户当前权限。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/service/devicetoken`

- [ ] **步骤 3：实现设备令牌服务**

固定接口：

```go
func (s *Service) Create(userID uint, input CreateInput) (*model.DeviceToken, string, error)
func (s *Service) ListForUser(userID uint) ([]model.DeviceToken, error)
func (s *Service) Revoke(userID, tokenID uint, systemAdmin bool) error
func (s *Service) Authenticate(plain, ip, clientInfo string) (*model.User, *model.DeviceToken, map[uint]struct{}, error)
```

创建时使用数据库事务检查上限并写入受限服务；认证成功后只更新 `last_used_at`、`last_used_ip`、`client_info`，不刷新到期时间。

- [ ] **步骤 4：实现个人 API**

成员只能列出、创建、撤销本人令牌；系统管理员可通过用户详情查看元数据并撤销，但任何角色都不能再次读取明文。创建响应字段 `token` 只出现一次。

- [ ] **步骤 5：运行绿灯**

运行：`go test ./internal/service/devicetoken ./internal/api -run 'TestDeviceToken|TestV1DeviceToken'`

- [ ] **步骤 6：提交任务 6**

提交意图：`feat(设备令牌): 允许成员自助创建可收窄的个人凭证`

---

### 任务 7：让 MCP 网关使用个人权限执行鉴权

**文件：**
- 修改：`internal/api/middleware.go`
- 修改：`internal/api/middleware_test.go`
- 重写：`internal/service/mcp/proxy_filter.go`
- 重写：`internal/service/mcp/proxy_filter_test.go`
- 修改：`internal/service/mcp/proxy.go`
- 修改：`internal/service/mcp/proxy_test.go`

- [ ] **步骤 1：写 MCP 鉴权红灯测试**

覆盖：缺少 Bearer 返回 401；Web Session Cookie 不能调用 `/mcp`；设备令牌成功注入真实用户、令牌和有效服务 ID；禁用用户/过期或撤销令牌返回 401；无权限访问服务时工具列表不可见且直接调用返回权限错误；权限组取消服务后已签发 inherit_all 和 restricted 令牌都立即失权。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/api ./internal/service/mcp -run 'TestDeviceTokenMCP|TestProxyToolFilter|TestAuthorizeProxy'`

- [ ] **步骤 3：实现 MCP 请求上下文**

使用私有 Context Key 类型，写入 `user_id`、`device_token_id` 和 `effective_service_ids`。过滤和直接调用都只消费同一份有效服务 ID 集合，不再读取服务器名称字符串白名单。

- [ ] **步骤 4：验证真实 MCP 协议路径**

通过 `mcp-go` 客户端或 JSON-RPC 请求依次执行 initialize、tools/list、tools/call，确认有权限令牌能看到并调用工具，无权限令牌在列表和调用两条路径都被拒绝。

- [ ] **步骤 5：运行绿灯**

运行：`go test ./internal/api ./internal/service/mcp`

- [ ] **步骤 6：提交任务 7**

提交意图：`feat(MCP 鉴权)!: 按个人设备令牌动态计算服务权限`

---

### 任务 8：删除旧凭证模型与旧 API 表面

**文件：**
- 删除：`internal/model/mcp_client.go`
- 删除：`internal/model/mcp_client_test.go`
- 删除：`internal/service/mcpclient/mcp_client.go`
- 删除：`internal/service/mcpclient/mcp_client_test.go`
- 删除：`internal/api/mcp_clients.go`
- 删除：`internal/api/mcp_clients_test.go`
- 删除：`pkg/types/mcp_client.go`
- 删除：`pkg/types/mcp_client_test.go`
- 删除：`client/mcp_clients.go`
- 删除：`client/mcp_clients_test.go`
- 修改：`cmd/create.go`、`cmd/list.go`、`cmd/update.go`、`cmd/delete.go` 及对应测试
- 修改：`internal/api/server.go`、`cmd/start.go`

- [ ] **步骤 1：写旧表面不存在的契约测试**

断言 `/api/v0/clients` 和 `/api/v0/users` 返回 404；启动迁移后没有 `mcp_clients` 表；CLI 根命令不再注册 `client` 和用户 AccessToken 子命令。

- [ ] **步骤 2：运行红灯**

运行：`go test ./internal/api ./cmd -run 'TestLegacyCredentialRoutesRemoved|TestLegacyCredentialCommandsRemoved'`

- [ ] **步骤 3：删除旧实现并修复依赖注入**

删除 `MCPClientService` 字段、构造参数和路由；删除所有 `GetUserByAccessToken`、`CheckAllowedServer`、`AllowList`、`AllowedServers` 调用。CLI 管理请求后续只接受登录 Session；本阶段先删除不再成立的凭证管理命令，不伪造兼容层。

- [ ] **步骤 4：做遗留关键词扫描**

运行：

```powershell
rg -n "McpClient|mcp_client|AccessToken|AllowedServers|AllowList|MCPJUNGLE_JWT_SECRET|golang-jwt" --glob '*.go' --glob 'go.mod'
```

预期：不存在身份/鉴权相关旧模型引用；MCP 上游配置中合法的 OAuth access token 字段不属于本次删除范围。

- [ ] **步骤 5：运行全仓编译与测试**

运行：`go test -run '^$' ./... && go test ./...`

- [ ] **步骤 6：提交任务 8**

提交意图：`refactor(鉴权)!: 删除旧客户端令牌与 v0 身份接口`

---

### 任务 9：更新 MySQL 部署入口并完成阶段验收

**文件：**
- 修改：`docker-compose.yaml`
- 修改：`docker-compose.prod.yaml`
- 修改：`.env.example`
- 修改：`README.md`
- 创建：`docs/internal-hub-access-control.md`

- [ ] **步骤 1：把 Compose 改成 MySQL 8.0**

两个 Compose 都使用 `mysql:8.0`、`utf8mb4`、UTC、健康检查和命名卷；应用只配置 `DATABASE_URL=mysql://...`。删除 pgAdmin 和所有 PostgreSQL 环境变量。

- [ ] **步骤 2：更新最小运行说明**

README 和 `.env.example` 只保留 MySQL 配置。内部指南说明：管理员创建用户、用户首次改密、管理员分配权限组、用户自建设备令牌、客户端配置 Bearer Token、撤销与禁用的即时效果。

- [ ] **步骤 3：运行静态验证**

运行：

```powershell
gofmt -w (rg --files -g '*.go')
go vet ./...
go test -run '^$' ./...
go test ./...
rg -n "sqlite|postgres|SQLITE_|POSTGRES_|mcp_clients|AllowedServers" --glob '!docs/superpowers/**' --glob '!CHANGELOG.md'
```

- [ ] **步骤 4：运行 MySQL 集成验证**

有 Docker 的环境执行：

```powershell
docker compose -f docker-compose.test.yaml up -d --wait
$env:MCPJUNGLE_TEST_DATABASE_URL='mysql://mcpjungle:mcpjungle@127.0.0.1:3307/mcpjungle_test'
go test ./...
docker compose -f docker-compose.test.yaml down -v
```

然后启动服务，使用真实 Cookie 登录、创建权限组与设备令牌，并通过 MCP initialize、tools/list、tools/call 验证授权链路。

- [ ] **步骤 5：构建前端与二进制**

运行：

```powershell
Push-Location frontend
npm ci
npm run build
Pop-Location
go build ./...
```

- [ ] **步骤 6：对照设计规范验收**

逐项确认：MySQL 单数据库；账户状态即时生效；四角色存在且管理权不等于使用权；无权限组默认拒绝；设备令牌只收窄权限；明文令牌只显示一次；旧凭证/API/数据库表已删除；MCP 列表与直接调用都执行同一授权判断。

- [ ] **步骤 7：提交任务 9**

提交意图：`docs(部署)!: 提供 MySQL 内部 Hub 的运行与验收说明`

---

## 后续独立计划

本计划完成后依次创建并执行：

1. `2026-07-20-mcp-service-lifecycle.md`：服务草稿、异步验证、发布、健康检查、责任人和密钥加密。
2. `2026-07-20-call-observability-audit.md`：个人级调用事件、聚合统计、保留策略与不可变审计事件。
3. `2026-07-20-internal-hub-frontend.md`：按角色重构导航、账户、权限组、设备令牌、服务目录与统计页面。

每个后续计划都必须独立产出可运行、可测试的软件，并沿用本计划建立的 MySQL、身份和权限契约。
