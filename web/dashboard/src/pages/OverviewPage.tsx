import { useEffect, useState } from "react";
import { Row, Col, Card, Statistic, Alert, Descriptions, Typography, Spin, Button } from "antd";
import { Link } from "react-router-dom";
import { overviewApi } from "../api/overview";
import { extractError } from "../api/client";
import { useDashboardSettings } from "../hooks/useDashboardSettings";
import type { DashboardOverviewResponse } from "../types";
import EmptyStateCard from "../components/EmptyStateCard";

export default function OverviewPage() {
  const [data, setData] = useState<DashboardOverviewResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const settings = useDashboardSettings();

  useEffect(() => {
    overviewApi
      .get()
      .then(setData)
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spin />;
  if (error) return <Alert type="error" message={error} />;
  if (!data) return null;
  if (data.empty_state) {
    return (
      <EmptyStateCard
        state={data.empty_state}
        action={
          <Link to="/servers">
            <Button type="primary">添加第一个 MCP 服务</Button>
          </Link>
        }
      />
    );
  }

  const statusType = data.status === "running" ? "success" : data.status === "degraded" ? "warning" : "info";
  const statusText = data.status === "running" ? "运行正常" : data.status === "degraded" ? "运行降级" : "状态未知";

  return (
    <>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        {settings.system_display_name} 统一接入团队内部的 MCP 服务，并按用户和权限组分配访问范围；每次调用都可追溯到具体身份和设备令牌。
      </Typography.Paragraph>
      <Alert showIcon type={statusType} message={statusText} style={{ marginBottom: 16 }} />
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} xl={6}>
          <Card className="metric-card" extra={<Link to="/servers">查看</Link>}>
            <Statistic title="MCP 服务" value={data.server_count} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card className="metric-card" extra={<Link to="/servers">查看</Link>}>
            <Statistic title="工具" value={data.tool_count} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card className="metric-card" extra={<Link to="/servers">查看</Link>}>
            <Statistic title="提示模板" value={data.prompt_count} />
          </Card>
        </Col>
        <Col xs={24} sm={12} xl={6}>
          <Card className="metric-card" extra={<Link to="/servers">查看</Link>}>
            <Statistic title="数据资源" value={data.resource_count} />
          </Card>
        </Col>
      </Row>
      <Card style={{ marginTop: 16 }}>
        <Descriptions title="服务信息" column={1}>
          <Descriptions.Item label="版本">{data.version}</Descriptions.Item>
          <Descriptions.Item label="模式">{data.mode}</Descriptions.Item>
          {data.endpoints.map((e) => (
            <Descriptions.Item key={e.url} label={e.label}>
              <Typography.Link href={e.url} target="_blank" rel="noreferrer">
                {e.url}
              </Typography.Link>
            </Descriptions.Item>
          ))}
        </Descriptions>
        {data.troubleshooting && data.troubleshooting.length > 0 && (
          <Alert
            type="warning"
            style={{ marginTop: 8 }}
            message="排查提示"
            description={
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                {data.troubleshooting.map((t) => (
                  <li key={t}>{t}</li>
                ))}
              </ul>
            }
          />
        )}
      </Card>
    </>
  );
}
