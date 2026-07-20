import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Switch, Drawer, Tag, Tooltip, Button, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { toolsApi } from "../api/tools";
import { extractError } from "../api/client";
import type { DashboardTool, DashboardToolsResponse } from "../types";
import EmptyStateCard from "../components/EmptyStateCard";
import JsonViewer from "../components/JsonViewer";

export default function ToolsPage() {
  const [data, setData] = useState<DashboardToolsResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [detail, setDetail] = useState<DashboardTool | null>(null);

  const load = () => {
    setLoading(true);
    toolsApi
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
  if (data?.empty_state && (!data.tools || data.tools.length === 0))
    return <EmptyStateCard state={data.empty_state} />;

  const toggle = async (row: DashboardTool, enabled: boolean) => {
    setToggling(row.canonical_name);
    try {
      await toolsApi.setEnabled(row.canonical_name, enabled);
      message.success(`已${enabled ? "启用" : "禁用"} ${row.canonical_name}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setToggling(null);
    }
  };

  const columns: ColumnsType<DashboardTool> = [
    { title: "名称", dataIndex: "name" },
    { title: "所属服务器", dataIndex: "server" },
    { title: "描述", dataIndex: "description", ellipsis: true },
    {
      title: "注解",
      dataIndex: "annotation_keys",
      render: (ks: string[] = []) => ks.map((k) => <Tag key={k}>{k}</Tag>),
    },
    {
      title: "启用",
      dataIndex: "enabled",
      render: (enabled: boolean, row) =>
        row.server_enabled ? (
          <Switch
            checked={enabled}
            loading={toggling === row.canonical_name}
            onChange={(c) => toggle(row, c)}
          />
        ) : (
          <Tooltip title="所属服务器已禁用">
            <Switch checked={false} disabled />
          </Tooltip>
        ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <Button size="small" onClick={() => setDetail(row)}>
          查看输入
        </Button>
      ),
    },
  ];

  return (
    <Card title="工具">
      <Table
        rowKey="canonical_name"
        dataSource={data?.tools ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
      <Drawer
        title={detail?.canonical_name}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={560}
      >
        {detail && <JsonViewer value={detail.input_schema ?? detail.input_preview} />}
      </Drawer>
    </Card>
  );
}
