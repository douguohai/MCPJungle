import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Switch, Drawer, Button, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { promptsApi } from "../api/prompts";
import { extractError } from "../api/client";
import type { DashboardPrompt, DashboardPromptsResponse } from "../types";
import EmptyStateCard from "../components/EmptyStateCard";
import JsonViewer from "../components/JsonViewer";

export default function PromptsPage() {
  const [data, setData] = useState<DashboardPromptsResponse | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [toggling, setToggling] = useState<string | null>(null);
  const [detail, setDetail] = useState<DashboardPrompt | null>(null);

  const load = () => {
    setLoading(true);
    promptsApi
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
  if (data?.empty_state && (!data.prompts || data.prompts.length === 0))
    return <EmptyStateCard state={data.empty_state} />;

  const toggle = async (row: DashboardPrompt, enabled: boolean) => {
    setToggling(row.canonical_name);
    try {
      await promptsApi.setEnabled(row.canonical_name, enabled);
      message.success(`已${enabled ? "启用" : "禁用"} ${row.canonical_name}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setToggling(null);
    }
  };

  const columns: ColumnsType<DashboardPrompt> = [
    { title: "名称", dataIndex: "name" },
    { title: "所属服务器", dataIndex: "server" },
    { title: "描述", dataIndex: "description", ellipsis: true },
    {
      title: "启用",
      dataIndex: "enabled",
      render: (enabled: boolean, row) => (
        <Switch
          checked={enabled}
          loading={toggling === row.canonical_name}
          onChange={(c) => toggle(row, c)}
        />
      ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <Button size="small" onClick={() => setDetail(row)}>
          查看参数
        </Button>
      ),
    },
  ];

  return (
    <Card title="提示词">
      <Table
        rowKey="canonical_name"
        dataSource={data?.prompts ?? []}
        columns={columns}
        loading={loading}
        pagination={{ pageSize: 20 }}
      />
      <Drawer title={detail?.canonical_name} open={!!detail} onClose={() => setDetail(null)} width={560}>
        {detail && <JsonViewer value={detail.arguments ?? detail.arguments_preview} />}
      </Drawer>
    </Card>
  );
}
