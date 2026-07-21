import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { useDashboardSettings } from "./useDashboardSettings";

const pageTitles: Array<[RegExp, string]> = [
  [/^\/login$/, "登录"],
  [/^\/change-password$/, "修改密码"],
  [/^\/$/, "概览"],
  [/^\/servers\/[^/]+$/, "服务详情"],
  [/^\/servers$/, "MCP 服务"],
  [/^\/device-tokens$/, "我的设备令牌"],
  [/^\/tutorial$/, "使用教程"],
  [/^\/users$/, "内部账户"],
  [/^\/permission-groups$/, "权限组"],
  [/^\/stats$/, "调用观察"],
  [/^\/settings\/diagnostics$/, "运行诊断"],
  [/^\/settings\/ability-combinations$/, "能力组合"],
  [/^\/settings$/, "系统设置"],
];

export function useDocumentTitle() {
  const location = useLocation();
  const settings = useDashboardSettings();

  useEffect(() => {
    const pageTitle = pageTitles.find(([pattern]) => pattern.test(location.pathname))?.[1];
    document.title = pageTitle
      ? `${pageTitle} - ${settings.system_display_name}`
      : settings.system_display_name;
  }, [location.pathname, settings.system_display_name]);
}
