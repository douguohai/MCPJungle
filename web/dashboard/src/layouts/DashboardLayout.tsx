import { useState } from "react";
import { Layout, Menu, Button, Dropdown, theme } from "antd";
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
  SafetyCertificateOutlined,
  KeyOutlined,
} from "@ant-design/icons";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../store/auth";
import { authApi } from "../api/auth";

const { Header, Sider, Content } = Layout;

const baseMenuItems = [
  { key: "/", icon: <DashboardOutlined />, label: "概览" },
  { key: "/servers", icon: <CloudServerOutlined />, label: "MCP 服务器" },
  { key: "/tools", icon: <ToolOutlined />, label: "工具" },
  { key: "/tool-groups", icon: <GroupOutlined />, label: "工具组" },
  { key: "/prompts", icon: <FileTextOutlined />, label: "提示词" },
  { key: "/resources", icon: <FileOutlined />, label: "资源" },
  { key: "/diagnostics", icon: <ApiOutlined />, label: "诊断" },
];

// Management pages (MCP clients / users / tokens) only make sense in enterprise
// mode — dev mode has no auth concepts and the backing /v0 endpoints return 403.
const enterpriseMenuItems = [
  { key: "/device-tokens", icon: <DesktopOutlined />, label: "我的设备令牌" },
];

const adminMenuItems = [
  { key: "/users", icon: <TeamOutlined />, label: "用户" },
  { key: "/permission-groups", icon: <SafetyCertificateOutlined />, label: "权限组" },
  { key: "/stats", icon: <BarChartOutlined />, label: "调用统计" },
];

export default function DashboardLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const nav = useNavigate();
  const loc = useLocation();
  const { mode, user, setUser } = useAuth();
  const { token: themeToken } = theme.useToken();

  const items =
    mode === "development"
      ? baseMenuItems
      : [
          ...baseMenuItems,
          ...enterpriseMenuItems,
          ...(user?.role === "system_admin" ? adminMenuItems : []),
        ];

  const onLogout = async () => {
    try {
      await authApi.logout();
    } finally {
      setUser(null);
    }
    nav("/login", { replace: true });
  };

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
                {
                  key: "password",
                  icon: <KeyOutlined />,
                  label: "修改密码",
                  onClick: () => nav("/change-password"),
                },
                { key: "logout", icon: <LogoutOutlined />, label: "退出登录", onClick: () => void onLogout() },
              ],
            }}
          >
            <Button type="text">
              <UserOutlined /> {user?.username ?? "未登录"}
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
