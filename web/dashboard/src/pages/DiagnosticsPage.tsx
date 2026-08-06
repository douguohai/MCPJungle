import { useEffect, useState } from "react";
import { Descriptions, Statistic, Card, Row, Col, Tag, List, Alert, Spin } from "antd";
import { diagnosticsApi } from "../api/diagnostics";
import { extractError } from "../api/client";
import type { DashboardDiagnosticsResponse } from "../types";

export default function DiagnosticsPage() {
  const [data, setData] = useState<DashboardDiagnosticsResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    diagnosticsApi
      .get(controller.signal)
      .then(setData)
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, []);

  if (loading) return <Spin />;
  if (error) return <Alert type="error" message={error} />;
  if (!data) return null;

  return (
    <>
      <Card>
        <Descriptions title="诊断信息" column={1}>
          <Descriptions.Item label="版本">{data.version}</Descriptions.Item>
          <Descriptions.Item label="模式">{data.mode}</Descriptions.Item>
          <Descriptions.Item label="数据库">{data.database}</Descriptions.Item>
          <Descriptions.Item label="配置来源">{data.config_source ?? "-"}</Descriptions.Item>
          <Descriptions.Item label="主端点">{data.primary_endpoint}</Descriptions.Item>
          {data.metrics_endpoint && (
            <Descriptions.Item label="Metrics 端点">{data.metrics_endpoint}</Descriptions.Item>
          )}
          <Descriptions.Item label="启用的传输协议">
            {data.enabled_transports.map((t) => (
              <Tag key={t}>{t}</Tag>
            ))}
          </Descriptions.Item>
        </Descriptions>
      </Card>
      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic title="服务器" value={data.server_count} />
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
      {data.troubleshooting_hints && data.troubleshooting_hints.length > 0 && (
        <Card title="排查提示" style={{ marginTop: 16 }}>
          <List
            size="small"
            dataSource={data.troubleshooting_hints}
            renderItem={(h) => <List.Item>{h}</List.Item>}
          />
        </Card>
      )}
    </>
  );
}
