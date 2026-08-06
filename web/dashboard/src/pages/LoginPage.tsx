import { useState } from "react";
import { Card, Input, Button, message, Typography, Form } from "antd";
import { useNavigate } from "react-router-dom";
import { authApi } from "../api/auth";
import { extractError } from "../api/client";
import { setUser } from "../store/auth";

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
  const nav = useNavigate();

  const onSubmit = async (values: { username: string; password: string }) => {
    setLoading(true);
    try {
      const res = await authApi.login(values.username.trim(), values.password);
      setUser({ username: res.user.username, role: res.user.role });
      message.success("登录成功");
      nav("/", { replace: true });
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card title="MCPJungle 管理端登录" style={{ width: 420 }}>
      <Typography.Paragraph type="secondary">
        使用管理员账号登录。首次使用请通过{" "}
        <Typography.Text code>mcpjungle init-server</Typography.Text> 初始化并设置管理员密码。
      </Typography.Paragraph>
      <Form layout="vertical" onFinish={onSubmit}>
        <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
          <Input placeholder="admin" autoComplete="username" />
        </Form.Item>
        <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
          <Input.Password placeholder="密码" autoComplete="current-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={loading}>
          登录
        </Button>
      </Form>
    </Card>
  );
}
