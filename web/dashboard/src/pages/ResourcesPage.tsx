import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Space, Typography, Button } from "antd";
import { Link } from "react-router-dom";
import type { ColumnsType } from "antd/es/table";
import { resourcesApi } from "../api/resources";
import { extractError } from "../api/client";
import type { DashboardResource, DashboardResourcesResponse } from "../types";
import CopyButton from "../components/CopyButton";
import EmptyStateCard from "../components/EmptyStateCard";

export default function ResourcesPage() {
  const [data, setData] = useState<DashboardResourcesResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    resourcesApi
      .list(controller.signal)
      .then(setData)
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, []);

  if (loading) return <Spin />;
  if (error) return <Alert type="error" message={error} />;
  if (data?.empty_state && (!data.resources || data.resources.length === 0))
    return (
      <EmptyStateCard
        state={data.empty_state}
        action={
          <Link to="/servers">
            <Button type="primary">先去注册 MCP 服务器</Button>
          </Link>
        }
      />
    );

  const columns: ColumnsType<DashboardResource> = [
    {
      title: "URI",
      dataIndex: "uri",
      render: (uri: string) => (
        <Space>
          <Typography.Text code>{uri}</Typography.Text>
          <CopyButton text={uri} />
        </Space>
      ),
    },
    { title: "名称", dataIndex: "name" },
    { title: "所属服务器", dataIndex: "server" },
    { title: "MIME 类型", dataIndex: "mime_type" },
    { title: "描述", dataIndex: "description", ellipsis: true },
  ];

  return (
    <Card title="资源">
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        资源是 MCP 服务器暴露的可读数据（如文件、配置、上下文），AI 客户端可按 URI 读取。
      </Typography.Paragraph>
      <Table
        rowKey="uri"
        dataSource={data?.resources ?? []}
        columns={columns}
        pagination={{ pageSize: 20 }}
      />
    </Card>
  );
}
