import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Popconfirm, Drawer, List, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined } from "@ant-design/icons";
import { toolGroupsApi } from "../api/toolGroups";
import { toolsApi } from "../api/tools";
import { extractError } from "../api/client";
import type {
  DashboardToolGroup,
  DashboardToolGroupsResponse,
  DashboardTool,
} from "../types";
import CopyButton from "../components/CopyButton";
import EmptyStateCard from "../components/EmptyStateCard";
import ToolGroupForm from "../components/ToolGroupForm";

export default function ToolGroupsPage() {
  const [data, setData] = useState<DashboardToolGroupsResponse | null>(null);
  const [toolsData, setToolsData] = useState<DashboardTool[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [formOpen, setFormOpen] = useState(false);
  const [detail, setDetail] = useState<DashboardToolGroup | null>(null);

  const load = () => {
    setLoading(true);
    Promise.all([toolGroupsApi.list(), toolsApi.list()])
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
      await toolGroupsApi.remove(name);
      message.success(`已删除 ${name}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const columns: ColumnsType<DashboardToolGroup> = [
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

  if (data?.empty_state && (!data.tool_groups || data.tool_groups.length === 0)) {
    return (
      <>
        <div style={{ marginBottom: 16 }}>{addButton}</div>
        <EmptyStateCard state={data.empty_state} />
        <ToolGroupForm
          open={formOpen}
          onClose={() => setFormOpen(false)}
          onCreated={load}
          tools={toolsData}
        />
      </>
    );
  }

  return (
    <Card title="工具组" extra={addButton}>
      <Table
        rowKey="name"
        dataSource={data?.tool_groups ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
      <ToolGroupForm
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
