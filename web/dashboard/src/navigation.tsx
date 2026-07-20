import type { MenuProps } from "antd";
import {
  BarChartOutlined,
  CloudServerOutlined,
  DashboardOutlined,
  DesktopOutlined,
  ReadOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import type { UserRole } from "./types";

type MenuItem = Required<MenuProps>["items"][number];

function group(key: string, label: string, children: MenuItem[]): MenuItem {
  return { key, type: "group", label, children };
}

export function buildMenuItems(role?: UserRole): MenuItem[] {
  const items: MenuItem[] = [
    group("workspace", "工作台", [
      { key: "/", icon: <DashboardOutlined />, label: "概览" },
    ]),
    group("services", "服务中心", [
      { key: "/servers", icon: <CloudServerOutlined />, label: "MCP 服务" },
    ]),
    group("personal", "个人接入", [
      { key: "/device-tokens", icon: <DesktopOutlined />, label: "我的设备令牌" },
      { key: "/tutorial", icon: <ReadOutlined />, label: "使用教程" },
    ]),
  ];

  if (role === "system_admin") {
    items.push(
      group("access", "访问控制", [
        { key: "/users", icon: <TeamOutlined />, label: "用户" },
        {
          key: "/permission-groups",
          icon: <SafetyCertificateOutlined />,
          label: "权限组",
        },
      ]),
    );
  }

  if (role === "system_admin" || role === "auditor") {
    items.push(
      group("observability", "运行观测", [
        { key: "/stats", icon: <BarChartOutlined />, label: "调用统计" },
      ]),
    );
  }

  if (role === "system_admin") {
    items.push(
      group("system", "系统管理", [
        { key: "/settings", icon: <SettingOutlined />, label: "系统设置" },
      ]),
    );
  }

  return items;
}

export function selectedMenuKey(pathname: string): string {
  if (pathname.startsWith("/servers/")) return "/servers";
  if (pathname.startsWith("/settings/")) return "/settings";
  return pathname;
}
