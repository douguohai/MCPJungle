import { ApiOutlined, ApartmentOutlined, RightOutlined } from "@ant-design/icons";
import { Card, Col, Row, Space, Typography } from "antd";
import { Link } from "react-router-dom";

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

