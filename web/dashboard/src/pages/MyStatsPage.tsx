import { useEffect, useState } from "react";
import {
  Card,
  Alert,
  Spin,
  Typography,
  Statistic,
  Row,
  Col,
  Table,
  Segmented,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { statsApi } from "../api/stats";
import { getUser } from "../store/auth";
import { extractError } from "../api/client";
import type { CallStat } from "../types";

type Range = "today" | "7d" | "30d";

export default function MyStatsPage() {
  const [rows, setRows] = useState<CallStat[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [range, setRange] = useState<Range>("7d");
  const user = getUser();

  useEffect(() => {
    statsApi
      .list()
      .then((d) => {
        const myStats = (d.stats ?? []).filter(
          (s) => s.username === user?.username,
        );
        setRows(myStats);
        setError("");
      })
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  // Filter by time range
  const now = new Date();
  const todayStr = now.toISOString().slice(0, 10);
  const cutoffDate = new Date(now);
  if (range === "7d") cutoffDate.setDate(cutoffDate.getDate() - 7);
  if (range === "30d") cutoffDate.setDate(cutoffDate.getDate() - 30);
  const cutoffStr = cutoffDate.toISOString().slice(0, 10);

  const filtered = rows.filter((r) => {
    if (range === "today") return r.date === todayStr;
    return r.date >= cutoffStr;
  });

  // Aggregate by server
  const byServer = new Map<string, number>();
  const byDate = new Map<string, number>();
  filtered.forEach((r) => {
    byServer.set(r.server_name, (byServer.get(r.server_name) ?? 0) + r.count);
    byDate.set(r.date, (byDate.get(r.date) ?? 0) + r.count);
  });
  const totalCalls = filtered.reduce((s, r) => s + r.count, 0);

  const serverRows = [...byServer.entries()]
    .map(([server_name, count]) => ({ server_name, count }))
    .sort((a, b) => b.count - a.count);

  const dateRows = [...byDate.entries()]
    .map(([date, count]) => ({ date, count }))
    .sort((a, b) => a.date.localeCompare(b.date));

  const rangeLabel =
    range === "today" ? "今天" : range === "7d" ? "近7天" : "近30天";

  const serverColumns: ColumnsType<{ server_name: string; count: number }> = [
    { title: "MCP 服务", dataIndex: "server_name" },
    { title: "调用次数", dataIndex: "count", width: 100 },
  ];

  const dateColumns: ColumnsType<{ date: string; count: number }> = [
    { title: "日期", dataIndex: "date", width: 120 },
    { title: "调用次数", dataIndex: "count", width: 100 },
  ];

  return (
    <>
      <Typography.Title level={4} style={{ marginBottom: 4 }}>
        我的调用量
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        查看你在各 MCP 服务上的工具调用统计。
      </Typography.Paragraph>

      <div style={{ marginBottom: 16 }}>
        <Segmented
          value={range}
          onChange={(v) => setRange(v as Range)}
          options={[
            { label: "今天", value: "today" },
            { label: "近7天", value: "7d" },
            { label: "近30天", value: "30d" },
          ]}
        />
      </div>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic title={`${rangeLabel}总调用次数`} value={totalCalls} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="涉及服务数" value={byServer.size} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="活跃天数" value={byDate.size} />
          </Card>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={12}>
          <Card title="按服务汇总" size="small">
            <Table
              rowKey="server_name"
              dataSource={serverRows}
              columns={serverColumns}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
        <Col span={12}>
          <Card title="按日汇总" size="small">
            <Table
              rowKey="date"
              dataSource={dateRows}
              columns={dateColumns}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
      </Row>
    </>
  );
}
