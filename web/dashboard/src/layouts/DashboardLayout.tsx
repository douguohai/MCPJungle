import { useEffect, useState } from "react";
import { Layout, Menu, Button, Dropdown, theme, Tag } from "antd";
import {
  DashboardOutlined,
  CloudServerOutlined,
  ToolOutlined,
  GroupOutlined,
  FileTextOutlined,
  FileOutlined,
  ApiOutlined,
  LogoutOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  DesktopOutlined,
  TeamOutlined,
  BarChartOutlined,
  AuditOutlined,
} from "@ant-design/icons";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { clearUser, getUser } from "../store/auth";
import { http } from "../api/client";
import { overviewApi } from "../api/overview";

const { Header, Sider, Content } = Layout;

// Development mode: show all base pages (no auth concept).
const devMenuItems = [
  { key: "/", icon: <DashboardOutlined />, label: "概览" },
  { key: "/servers", icon: <CloudServerOutlined />, label: "MCP 服务器" },
  { key: "/tools", icon: <ToolOutlined />, label: "工具" },
  { key: "/tool-collections", icon: <GroupOutlined />, label: "工具集合" },
  { key: "/prompts", icon: <FileTextOutlined />, label: "提示词" },
  { key: "/resources", icon: <FileOutlined />, label: "资源" },
  { key: "/diagnostics", icon: <ApiOutlined />, label: "诊断" },
];

// system_admin: full management menu
const systemAdminMenuItems = [
  { key: "/", icon: <DashboardOutlined />, label: "工作台" },
  { key: "/servers", icon: <CloudServerOutlined />, label: "MCP 服务" },
  { key: "/permission-groups", icon: <GroupOutlined />, label: "权限组" },
  { key: "/users", icon: <TeamOutlined />, label: "人员" },
  { key: "/tool-collections", icon: <GroupOutlined />, label: "工具集合" },
  { key: "/stats", icon: <BarChartOutlined />, label: "调用分析" },
  { key: "/audit", icon: <AuditOutlined />, label: "审计日志" },
];

// service_admin: member pages + management of their services
const serviceAdminMenuItems = [
  { key: "/my-services", icon: <CloudServerOutlined />, label: "我的服务" },
  { key: "/device-tokens", icon: <DesktopOutlined />, label: "我的设备令牌" },
  { key: "/my-stats", icon: <BarChartOutlined />, label: "我的调用量" },
  { key: "/servers", icon: <CloudServerOutlined />, label: "MCP 服务" },
  { key: "/tool-collections", icon: <GroupOutlined />, label: "工具集合" },
];

// member: personal workspace
const memberMenuItems = [
  { key: "/my-services", icon: <CloudServerOutlined />, label: "我的服务" },
  { key: "/device-tokens", icon: <DesktopOutlined />, label: "我的设备令牌" },
  { key: "/my-stats", icon: <BarChartOutlined />, label: "我的调用量" },
];

// auditor: read-only analytics and audit
const auditorMenuItems = [
  { key: "/stats", icon: <BarChartOutlined />, label: "调用分析" },
  { key: "/audit", icon: <AuditOutlined />, label: "审计日志" },
];

const roleLabels: Record<string, { text: string; color: string }> = {
  system_admin: { text: "系统管理员", color: "red" },
  service_admin: { text: "服务管理员", color: "orange" },
  member: { text: "成员", color: "blue" },
  auditor: { text: "审计员", color: "purple" },
};

function getMenuItems(mode: string, role?: string) {
  if (mode === "development") return devMenuItems;
  switch (role) {
    case "system_admin":
      return systemAdminMenuItems;
    case "service_admin":
      return serviceAdminMenuItems;
    case "auditor":
      return auditorMenuItems;
    case "member":
    default:
      return memberMenuItems;
  }
}

export default function DashboardLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const [mode, setMode] = useState("");
  const nav = useNavigate();
  const loc = useLocation();
  const user = getUser();
  const { token: themeToken } = theme.useToken();

  useEffect(() => {
    overviewApi
      .get()
      .then((d) => setMode(d.mode))
      .catch(() => {});
  }, []);

  const items = getMenuItems(mode, user?.role);

  const onLogout = async () => {
    try { await http.post("/auth/logout"); } catch {}
    clearUser();
    nav("/login", { replace: true });
  };

  const roleInfo = user?.role ? roleLabels[user.role] : null;

  return (
    <Layout style={{ minHeight: "100vh" }}>
      <Sider trigger={null} collapsible collapsed={collapsed}>
        <div
          style={{
            height: 48,
            margin: 12,
            color: "#fff",
            textAlign: "center",
            fontWeight: 600,
            lineHeight: "48px",
          }}
        >
          {collapsed ? "M" : "MCPJungle"}
        </div>
        <Menu theme="dark" mode="inline" selectedKeys={[loc.pathname]} items={items} onClick={({ key }) => nav(key)} />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: "0 16px",
            background: themeToken.colorBgContainer,
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <Button
            type="text"
            onClick={() => setCollapsed(!collapsed)}
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          />
          <Dropdown
            menu={{
              items: [
                { key: "logout", icon: <LogoutOutlined />, label: "退出登录", onClick: onLogout },
              ],
            }}
          >
            <Button type="text">
              <UserOutlined /> {user?.username ?? "未登录"}
              {roleInfo && (
                <Tag color={roleInfo.color} style={{ marginLeft: 8 }}>
                  {roleInfo.text}
                </Tag>
              )}
            </Button>
          </Dropdown>
        </Header>
        <Content
          style={{
            margin: 16,
            padding: 24,
            background: themeToken.colorBgContainer,
            borderRadius: 8,
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
