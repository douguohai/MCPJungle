import { useState } from "react";
import { Layout, Menu, Button, Dropdown, theme } from "antd";
import {
  LogoutOutlined,
  UserOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  KeyOutlined,
} from "@ant-design/icons";
import { Outlet, useNavigate, useLocation } from "react-router-dom";
import { useAuth } from "../store/auth";
import { authApi } from "../api/auth";
import { buildMenuItems, selectedMenuKey } from "../navigation";

const { Header, Sider, Content } = Layout;

export default function DashboardLayout() {
  const [collapsed, setCollapsed] = useState(false);
  const nav = useNavigate();
  const loc = useLocation();
  const { user, setUser } = useAuth();
  const { token: themeToken } = theme.useToken();

  const items = buildMenuItems(user?.role);

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
      <Sider width={224} trigger={null} collapsible collapsed={collapsed}>
        <div className={collapsed ? "dashboard-brand collapsed" : "dashboard-brand"}>
          <div>{collapsed ? "M" : "MCPJungle"}</div>
          {!collapsed && <small>企业内部 MCP Hub</small>}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedMenuKey(loc.pathname)]}
          items={items}
          onClick={({ key }) => nav(key)}
        />
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
            margin: 24,
            minHeight: 280,
          }}
        >
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
