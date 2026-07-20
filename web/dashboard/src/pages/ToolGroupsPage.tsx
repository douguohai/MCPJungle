import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Popconfirm, Drawer, List, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined } from "@ant-design/icons";
import { useSearchParams } from "react-router-dom";
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
import { useAuth } from "../store/auth";

export default function ToolGroupsPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { user } = useAuth();
  const canManage = user?.role === "system_admin";
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

  const initialTools = toolsData
    .filter((tool) => {
      const selectedTools = searchParams.get("tools")?.split(",").filter(Boolean) ?? [];
      const selectedServer = searchParams.get("server");
      return selectedTools.includes(tool.canonical_name) || (!!selectedServer && tool.server === selectedServer);
    })
    .map((tool) => tool.canonical_name);

  useEffect(() => {
    if (canManage && toolsData.length > 0 && initialTools.length > 0) {
      setFormOpen(true);
    }
  }, [canManage, toolsData.length, initialTools.length]);

  const closeForm = () => {
    setFormOpen(false);
    if (searchParams.has("tools") || searchParams.has("server")) {
      setSearchParams({});
    }
  };

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
          {canManage && (
            <Popconfirm title={`确认删除 ${row.name}？`} onConfirm={() => remove(row.name)}>
              <Button danger size="small">
                删除
              </Button>
            </Popconfirm>
          )}
        </>
      ),
    },
  ];

  const addButton = canManage ? (
    <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
      添加
    </Button>
  ) : null;

  if (data?.empty_state && (!data.tool_groups || data.tool_groups.length === 0)) {
    return (
      <>
        <EmptyStateCard
          state={data.empty_state}
          action={canManage ? (
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
              创建第一个能力组合
            </Button>
          ) : undefined}
        />
        <ToolGroupForm
          open={formOpen}
          onClose={closeForm}
          onCreated={load}
          tools={toolsData}
          initialTools={initialTools}
        />
      </>
    );
  }

  return (
    <Card title="能力组合" extra={addButton}>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        能力组合把多个 MCP 工具编排为一个独立调用端点，适合按业务场景交付给 AI 客户端。人员访问范围仍由权限组控制。
      </Typography.Paragraph>
      <Table
        rowKey="name"
        dataSource={data?.tool_groups ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
      <ToolGroupForm
        open={formOpen}
        onClose={closeForm}
        onCreated={load}
        tools={toolsData}
        initialTools={initialTools}
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
