import { useState } from "react";
import { Alert, Button, Card, Form, Input, Typography, message } from "antd";
import { useNavigate } from "react-router-dom";
import { authApi } from "../api/auth";
import { extractError } from "../api/client";
import { useAuth } from "../store/auth";

export default function ChangePasswordPage() {
  const [loading, setLoading] = useState(false);
  const { user, setUser } = useAuth();
  const navigate = useNavigate();

  const logout = async () => {
    try {
      await authApi.logout();
    } finally {
      setUser(null);
      navigate("/login", { replace: true });
    }
  };

  const submit = async (values: {
    current_password: string;
    new_password: string;
    confirm_password: string;
  }) => {
    setLoading(true);
    try {
      const account = await authApi.changePassword(
        values.current_password,
        values.new_password,
      );
      setUser(account);
      message.success("密码已更新");
      navigate("/", { replace: true });
    } catch (error) {
      message.error(extractError(error));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card title="修改密码" style={{ width: 440 }}>
      {user?.must_change_password && (
        <Alert
          type="warning"
          showIcon
          message="首次登录必须修改一次性初始密码"
          style={{ marginBottom: 16 }}
        />
      )}
      <Typography.Paragraph type="secondary">
        新密码至少 12 个字符。修改成功后才能访问 MCP 服务和创建设备令牌。
      </Typography.Paragraph>
      <Form layout="vertical" onFinish={submit}>
        <Form.Item
          name="current_password"
          label="当前密码"
          rules={[{ required: true, message: "请输入当前密码" }]}
        >
          <Input.Password autoComplete="current-password" />
        </Form.Item>
        <Form.Item
          name="new_password"
          label="新密码"
          rules={[
            { required: true, message: "请输入新密码" },
            { min: 12, message: "新密码至少 12 个字符" },
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          name="confirm_password"
          label="确认新密码"
          dependencies={["new_password"]}
          rules={[
            { required: true, message: "请再次输入新密码" },
            ({ getFieldValue }) => ({
              validator(_, value) {
                return !value || getFieldValue("new_password") === value
                  ? Promise.resolve()
                  : Promise.reject(new Error("两次输入的新密码不一致"));
              },
            }),
          ]}
        >
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Button type="primary" htmlType="submit" block loading={loading}>
          保存并继续
        </Button>
        <Button type="link" block onClick={() => void logout()}>
          退出登录
        </Button>
      </Form>
    </Card>
  );
}
