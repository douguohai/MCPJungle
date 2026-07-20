import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Switch, Popconfirm, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { EyeOutlined, PlusOutlined } from "@ant-design/icons";
import { Link } from "react-router-dom";
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
    {
      title: "名称",
      dataIndex: "name",
      width: 160,
      ellipsis: true,
      render: (name: string) => (
        <Link to={`/servers/${encodeURIComponent(name)}`} title={name}>
          {name}
        </Link>
      ),
    },
    { title: "传输", dataIndex: "transport", width: 100, render: (t: string) => <TransportTag transport={t} /> },
    { title: "状态", dataIndex: "status", width: 90, render: (s: string) => <StatusBadge status={s} /> },
    {
      title: "启用",
      dataIndex: "enabled",
      width: 80,
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
    { title: "提示模板", dataIndex: "prompt_count", width: 96 },
    { title: "数据资源", dataIndex: "resource_count", width: 96 },
    {
      title: "配置摘要",
      dataIndex: "connection_summary",
      width: 220,
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
      width: 190,
      fixed: "right",
      render: (_, row) => (
        <span style={{ display: "flex", gap: 8 }}>
          <Link to={`/servers/${encodeURIComponent(row.name)}`}>
            <Button size="small" icon={<EyeOutlined />}>查看详情</Button>
          </Link>
          {canManage && (
            <Popconfirm title={`确认删除 ${row.name}？`} onConfirm={() => remove(row.name)}>
              <Button danger size="small">删除</Button>
            </Popconfirm>
          )}
        </span>
      ),
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
              添加第一个 MCP 服务
            </Button>
          ) : undefined}
        />
        <ServerForm open={formOpen} onClose={() => setFormOpen(false)} onCreated={load} />
      </>
    );
  }

  return (
    <Card title="MCP 服务" extra={addButton}>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        集中管理内部团队可访问的 MCP 服务。进入服务详情可查看工具、提示模板、数据资源和连接状态。
      </Typography.Paragraph>
      <Table
        rowKey="name"
        dataSource={data?.servers ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
        scroll={{ x: 1100 }}
      />
      <ServerForm open={formOpen} onClose={() => setFormOpen(false)} onCreated={load} />
    </Card>
  );
}
