import { useEffect, useState } from "react";
import { Row, Col, Card, Statistic, Alert, Descriptions, Typography, Spin } from "antd";
import { overviewApi } from "../api/overview";
import { extractError } from "../api/client";
import type { DashboardOverviewResponse } from "../types";
import EmptyStateCard from "../components/EmptyStateCard";

export default function OverviewPage() {
  const [data, setData] = useState<DashboardOverviewResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

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
  if (data.empty_state) return <EmptyStateCard state={data.empty_state} />;

  const statusType = data.status === "running" ? "success" : data.status === "degraded" ? "warning" : "info";
  const statusText = data.status === "running" ? "运行正常" : data.status === "degraded" ? "运行降级" : "状态未知";

  return (
    <>
      <Alert showIcon type={statusType} message={statusText} style={{ marginBottom: 16 }} />
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic title="MCP 服务器" value={data.server_count} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="工具" value={data.tool_count} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="提示词" value={data.prompt_count} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic title="资源" value={data.resource_count} />
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
