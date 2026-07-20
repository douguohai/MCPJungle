import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Switch, Popconfirm, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { PlusOutlined } from "@ant-design/icons";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type { DashboardServer, DashboardServersResponse } from "../types";
import TransportTag from "../components/TransportTag";
import StatusBadge from "../components/StatusBadge";
import EmptyStateCard from "../components/EmptyStateCard";
import ServerForm from "../components/ServerForm";
import { useAuth } from "../store/auth";

export default function ServersPage() {
  const { user } = useAuth();
  const canManage = user?.role === "system_admin";
  const [data, setData] = useState<DashboardServersResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [formOpen, setFormOpen] = useState(false);

  const load = () => {
    setLoading(true);
    serversApi
      .list()
      .then(setData)
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  };
  useEffect(() => {
    load();
  }, []);

  if (loading && !data) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  const toggle = async (row: DashboardServer, enabled: boolean) => {
    setToggling(row.name);
    try {
      await serversApi.setEnabled(row.name, enabled);
      message.success(`已${enabled ? "启用" : "禁用"} ${row.name}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setToggling(null);
    }
  };

  const remove = async (name: string) => {
    try {
      await serversApi.remove(name);
      message.success(`已删除 ${name}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const columns: ColumnsType<DashboardServer> = [
    { title: "名称", dataIndex: "name" },
    { title: "传输", dataIndex: "transport", render: (t: string) => <TransportTag transport={t} /> },
    { title: "状态", dataIndex: "status", render: (s: string) => <StatusBadge status={s} /> },
    {
      title: "启用",
      dataIndex: "enabled",
      render: (enabled: boolean, row) => (
        <Switch
          checked={enabled}
          disabled={!canManage}
          loading={toggling === row.name}
          onChange={(c) => toggle(row, c)}
        />
      ),
    },
    { title: "工具", dataIndex: "tool_count", width: 70 },
    { title: "提示词", dataIndex: "prompt_count", width: 80 },
    { title: "资源", dataIndex: "resource_count", width: 70 },
    {
      title: "配置摘要",
      dataIndex: "connection_summary",
      ellipsis: true,
      render: (s: string, row) => (
        <Tooltip title={row.config_summary?.sanitized_summary ?? s}>
          <span>{s}</span>
        </Tooltip>
      ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => canManage ? (
        <Popconfirm title={`确认删除 ${row.name}？`} onConfirm={() => remove(row.name)}>
          <Button danger size="small">
            删除
          </Button>
        </Popconfirm>
      ) : null,
    },
  ];

  const addButton = canManage ? (
    <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
      添加
    </Button>
  ) : null;

  if (data?.empty_state && (!data.servers || data.servers.length === 0)) {
    return (
      <>
        <EmptyStateCard
          state={data.empty_state}
          action={canManage ? (
            <Button type="primary" icon={<PlusOutlined />} onClick={() => setFormOpen(true)}>
              注册第一个 MCP 服务器
            </Button>
          ) : undefined}
        />
        <ServerForm open={formOpen} onClose={() => setFormOpen(false)} onCreated={load} />
      </>
    );
  }

  return (
    <Card title="MCP 服务器" extra={addButton}>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        MCP 服务器是你接入的上游 MCP 服务（在线 HTTP 服务或本地命令行程序）。注册后，MCPJungle 会自动发现并代理它提供的工具、提示词和资源。
      </Typography.Paragraph>
      <Table
        rowKey="name"
        dataSource={data?.servers ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
      <ServerForm open={formOpen} onClose={() => setFormOpen(false)} onCreated={load} />
    </Card>
  );
}
