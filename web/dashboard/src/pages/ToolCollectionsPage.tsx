import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Popconfirm, Drawer, List, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined } from "@ant-design/icons";
import { toolCollectionsApi } from "../api/toolCollections";
import { toolsApi } from "../api/tools";
import { extractError } from "../api/client";
import type {
  DashboardToolCollection,
  DashboardToolCollectionsResponse,
  DashboardTool,
} from "../types";
import CopyButton from "../components/CopyButton";
import EmptyStateCard from "../components/EmptyStateCard";
import ToolCollectionForm from "../components/ToolCollectionForm";

export default function ToolCollectionsPage() {
  const [data, setData] = useState<DashboardToolCollectionsResponse | null>(null);
  const [toolsData, setToolsData] = useState<DashboardTool[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [detail, setDetail] = useState<DashboardToolCollection | null>(null);

  const load = () => {
    setLoading(true);
    Promise.all([toolCollectionsApi.list(), toolsApi.list()])
      .then(([g, t]) => {
        setData(g);
        setToolsData(t.tools ?? []);
      })
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  };
  useEffect(() => {
    load();
  }, []);

  if (loading && !data) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  const remove = async (name: string) => {
    try {
      await toolCollectionsApi.remove(name);
      message.success(`已删除 ${name}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const columns: ColumnsType<DashboardToolCollection> = [
    { title: "名称", dataIndex: "name" },
    { title: "描述", dataIndex: "description", ellipsis: true },
    { title: "工具数", dataIndex: "tool_count", width: 80 },
    {
      title: "端点",
      key: "endpoints",
      render: (_, row) => (
        <>
          <CopyButton text={row.streamable_http_endpoint} label="HTTP" />{" "}
          <CopyButton text={row.sse_endpoint} label="SSE" />
        </>
      ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <>
          <Button size="small" onClick={() => setDetail(row)}>
            查看工具
          </Button>{" "}
          <Popconfirm title={`确认删除 ${row.name}？`} onConfirm={() => remove(row.name)}>
            <Button danger size="small">
              删除
            </Button>
          </Popconfirm>
        </>
      ),
    },
  ];

  const addButton = (
    <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
      添加
    </Button>
  );

  if (data?.empty_state && (!data.tool_collections || data.tool_collections.length === 0)) {
    return (
      <>
        <EmptyStateCard
          state={data.empty_state}
          action={
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
              创建第一个工具集合
            </Button>
          }
        />
        <ToolCollectionForm
          open={formOpen}
          onClose={() => setFormOpen(false)}
          onCreated={load}
          tools={toolsData}
        />
      </>
    );
  }

  return (
    <Card title="工具集合" extra={addButton}>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        工具集合把来自不同 MCP 服务器的多个工具打包成一个虚拟服务，提供独立的调用端点，方便按场景分发给 AI 客户端。
      </Typography.Paragraph>
      <Table
        rowKey="name"
        dataSource={data?.tool_collections ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
      <ToolCollectionForm
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onCreated={load}
        tools={toolsData}
      />
      <Drawer title={detail?.name} open={!!detail} onClose={() => setDetail(null)} width={480}>
        <List
          size="small"
          dataSource={detail?.tools ?? []}
          renderItem={(t) => <List.Item>{t.canonical_name}</List.Item>}
        />
      </Drawer>
    </Card>
  );
}
