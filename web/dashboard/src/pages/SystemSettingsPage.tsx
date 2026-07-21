import { ApiOutlined, ApartmentOutlined, RightOutlined } from "@ant-design/icons";
import { Button, Card, Col, Form, Input, message, Row, Space, Typography } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { extractError } from "../api/client";
import { defaultDashboardSettings, settingsApi, type DashboardSettings } from "../api/settings";

const settings = [
  {
    path: "/settings/diagnostics",
    icon: <ApiOutlined style={{ fontSize: 24, color: "#1677ff" }} />,
    title: "运行诊断",
    description: "查看版本、MySQL、服务端点、传输协议和系统排查建议。",
  },
  {
    path: "/settings/ability-combinations",
    icon: <ApartmentOutlined style={{ fontSize: 24, color: "#722ed1" }} />,
    title: "能力组合",
    description: "把多个 MCP 工具编排为独立调用端点；它不用于分配人员权限。",
  },
];

export default function SystemSettingsPage() {
  const [form] = Form.useForm<DashboardSettings>();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setLoading(true);
    settingsApi
      .get()
      .then((value) => form.setFieldsValue(value))
      .catch((e) => message.error(extractError(e)))
      .finally(() => setLoading(false));
  }, [form]);

  const onSave = async (values: DashboardSettings) => {
    setSaving(true);
    try {
      const saved = await settingsApi.update(values);
      form.setFieldsValue(saved);
      message.success("系统名称已更新");
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="page-stack">
      <div>
        <Typography.Title level={3} style={{ marginBottom: 4 }}>
          系统设置
        </Typography.Title>
        <Typography.Text type="secondary">
          面向系统管理员的运行维护与高级能力配置。
        </Typography.Text>
      </div>
      <Card title="基础信息" loading={loading}>
        <Form
          form={form}
          layout="vertical"
          onFinish={onSave}
          initialValues={defaultDashboardSettings}
        >
          <Form.Item
            label="系统名称"
            name="system_display_name"
            extra="展示在登录页和后台左侧导航，不改变接口路径、数据库名称或 MCP 服务名称。"
            rules={[
              { required: true, message: "请输入系统名称" },
              { max: 64, message: "系统名称最多 64 个字符" },
            ]}
          >
            <Input placeholder={defaultDashboardSettings.system_display_name} maxLength={64} showCount />
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={saving}>
            保存设置
          </Button>
        </Form>
      </Card>
      <Row gutter={[16, 16]}>
        {settings.map((item) => (
          <Col xs={24} lg={12} key={item.path}>
            <Link to={item.path} className="settings-link">
              <Card hoverable className="settings-card">
                <Space align="start" size={16}>
                  {item.icon}
                  <div>
                    <Typography.Title level={5} style={{ margin: 0 }}>
                      {item.title}
                    </Typography.Title>
                    <Typography.Paragraph type="secondary" style={{ margin: "8px 0 0" }}>
                      {item.description}
                    </Typography.Paragraph>
                  </div>
                  <RightOutlined className="settings-arrow" />
                </Space>
              </Card>
            </Link>
          </Col>
        ))}
      </Row>
    </div>
  );
}
