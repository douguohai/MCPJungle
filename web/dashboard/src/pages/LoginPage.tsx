import { useState } from "react";
import { Card, Input, Button, message, Typography, Form, Space, Tag } from "antd";
import {
  ApartmentOutlined,
  ApiOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import { authApi } from "../api/auth";
import { extractError } from "../api/client";
import { useDashboardSettings } from "../hooks/useDashboardSettings";
import { useAuth } from "../store/auth";

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const settings = useDashboardSettings();
  const nav = useNavigate();
  const { setUser } = useAuth();

  const onSubmit = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await authApi.login(values.username.trim(), values.password);
      setUser(res.user);
      message.success("登录成功");
      nav(res.must_change_password ? "/change-password" : "/", { replace: true });
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-shell">
      <section className="login-hero">
        <Tag className="login-mode-tag">企业内部 MCP Hub</Tag>
        <Typography.Title className="login-title">
          {settings.system_display_name}
        </Typography.Title>
        <Typography.Paragraph className="login-subtitle">
          面向内部团队的 MCP 服务入口。统一登记服务能力，按账号授权访问，并把调用量归属到具体成员。
        </Typography.Paragraph>
        <div className="login-feature-grid">
          <div className="login-feature">
            <ApiOutlined />
            <span>服务能力统一登记</span>
          </div>
          <div className="login-feature">
            <SafetyCertificateOutlined />
            <span>账号权限集中分配</span>
          </div>
          <div className="login-feature">
            <ApartmentOutlined />
            <span>个人令牌追踪调用</span>
          </div>
        </div>
      </section>

      <Card className="login-card" bordered={false}>
        <Space direction="vertical" size={6} style={{ width: "100%" }}>
          <Typography.Title level={3} style={{ margin: 0 }}>
            登录控制台
          </Typography.Title>
          <Typography.Text type="secondary">
            使用内部账号访问 MCP 服务、权限和调用观察能力。
          </Typography.Text>
        </Space>
        <Form layout="vertical" onFinish={onSubmit} className="login-form">
          <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input size="large" placeholder="输入内部账号" autoComplete="username" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
            <Input.Password size="large" placeholder="输入登录密码" autoComplete="current-password" />
          </Form.Item>
          <Button type="primary" size="large" htmlType="submit" block loading={loading}>
            登录
          </Button>
        </Form>
      </Card>
    </div>
  );
}
