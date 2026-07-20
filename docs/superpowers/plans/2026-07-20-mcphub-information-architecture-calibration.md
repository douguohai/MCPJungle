# MCPJungle 企业版信息架构校准实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 把管理后台调整为以 MCP 服务为核心、按内部团队任务组织导航的企业 MCP Hub。

**架构：** 保留现有 React、Ant Design 和后端聚合 API，通过服务详情页在客户端按 `server` 字段过滤能力数据。路由守卫与菜单模型共同执行角色可见性，系统诊断和能力组合收敛到管理员系统设置页。

**技术栈：** React 18、TypeScript、React Router 6、Ant Design 5、Vite、Go。

---

## 文件结构

- 创建 `web/dashboard/src/navigation.tsx`：集中定义分组菜单与角色可见性。
- 创建 `web/dashboard/src/pages/ServerDetailPage.tsx`：服务详情及五个能力页签。
- 创建 `web/dashboard/src/pages/SystemSettingsPage.tsx`：管理员系统入口。
- 修改 `web/dashboard/src/layouts/DashboardLayout.tsx`：使用分组导航和服务详情选中态。
- 修改 `web/dashboard/src/App.tsx`：注册详情、设置及角色守卫路由，移除能力类型一级页面。
- 修改 `web/dashboard/src/pages/ServersPage.tsx`：统一产品文案并增加详情入口。
- 修改 `web/dashboard/src/pages/ToolGroupsPage.tsx`：改名为能力组合并校准说明。
- 修改 `web/dashboard/src/components/EmptyStateCard.tsx`：删除 CLI 展示，只保留浏览器动作。
- 修改 `web/dashboard/src/index.css`：补充页面、导航和详情布局样式。
- 修改 `internal/service/dashboard/service.go` 与相关测试：统一 API 空状态中文文案。

### 任务 1：锁定导航与路由规则

- [ ] **步骤 1：编写失败的源级回归测试**

在现有 Go 测试中增加断言：构建后的前端入口必须包含 `/servers/:serverName` 和 `/settings`，且旧能力页不再出现在 `DashboardLayout` 菜单定义中。

- [ ] **步骤 2：运行测试验证失败**

运行：`go test ./internal/api/... ./internal/service/dashboard/...`

预期：新增断言因详情路由和分组导航尚不存在而失败。

- [ ] **步骤 3：实现集中导航和受保护路由**

`navigation.tsx` 导出 `buildMenuItems(role)`；成员获得概览、服务和设备令牌，审计员额外获得调用统计，系统管理员获得访问控制、调用统计和系统管理。`App.tsx` 对系统设置、诊断和能力组合使用 `RequireAuth requiredRole="system_admin"`。

- [ ] **步骤 4：运行前端类型检查**

运行：`npm run typecheck --prefix web/dashboard`

预期：PASS。

### 任务 2：实现服务详情主流程

- [ ] **步骤 1：建立服务详情页的失败构建基线**

先在 `App.tsx` 引用尚未创建的 `ServerDetailPage` 并运行类型检查。

预期：FAIL，提示找不到模块。

- [ ] **步骤 2：实现最小详情页面**

详情页并行读取服务、工具、提示模板和数据资源 API，按 URL 解码后的服务名筛选。使用 Ant Design `Tabs` 提供概览、工具、提示模板、数据资源、连接诊断，空数组使用中文 `Empty`。

- [ ] **步骤 3：补齐管理操作与错误态**

工具和提示模板启停只对系统管理员开放；请求失败显示 `Alert`；服务不存在显示“未找到 MCP 服务”并提供返回按钮。

- [ ] **步骤 4：运行类型检查**

运行：`npm run typecheck --prefix web/dashboard`

预期：PASS。

### 任务 3：校准服务列表与系统管理

- [ ] **步骤 1：修改服务列表**

将标题和说明改为“MCP 服务”，服务名链接到 `/servers/{encodeURIComponent(name)}`，操作区始终提供查看详情，管理员操作保持可用。

- [ ] **步骤 2：创建系统设置页**

使用两个 `Card` 展示“运行诊断”和“能力组合”，明确二者分别属于系统运维和高级能力编排。

- [ ] **步骤 3：校准能力组合文案**

把 `ToolGroupsPage` 的“工具组”用户文案改为“能力组合”，不修改后端 API 名称。

- [ ] **步骤 4：删除浏览器 CLI 空状态**

`EmptyStateCard` 只渲染标题、描述和 UI 动作，不读取 `commands`。

### 任务 4：统一空状态并完成验证

- [ ] **步骤 1：编写中文空状态测试**

在 dashboard service 测试中断言服务器、工具、提示模板、数据资源和能力组合的空状态标题不包含英文产品引导。

- [ ] **步骤 2：修改后端空状态文案**

保持 JSON 字段不变，把标题、描述和排查建议改为简洁中文；CLI 命令可继续供 API/文档使用，但前端不展示。

- [ ] **步骤 3：运行后端测试**

运行：`go test ./internal/api/... ./internal/service/dashboard/...`

预期：PASS。

- [ ] **步骤 4：运行完整前端验证**

运行：`npm run typecheck --prefix web/dashboard` 和 `npm run build --prefix web/dashboard`。

预期：两项 PASS。

- [ ] **步骤 5：本地企业模式人工验收**

启动现有 MySQL 企业模式环境，登录后验证 MCP 服务列表、服务详情五页签、我的设备令牌、调用统计和系统设置；分别检查系统管理员与非管理员菜单。

