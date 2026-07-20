import { useState } from "react";
import { Card, Input, Button, message, Typography, Form } from "antd";
import { useNavigate } from "react-router-dom";
import { authApi } from "../api/auth";
import { extractError } from "../api/client";
import { useAuth } from "../store/auth";

export default function LoginPage() {
  const [loading, setLoading] = useState(false);
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
    <Card title="MCPJungle 管理端登录" style={{ width: 420 }}>
      <Typography.Paragraph type="secondary">
        使用内部账号登录。若管理员刚为你创建账号，登录后需要先修改一次性初始密码。
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
