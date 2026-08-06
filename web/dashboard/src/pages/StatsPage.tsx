import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Typography, Statistic, Row, Col, Button, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { statsApi } from "../api/stats";
import { extractError } from "../api/client";
import type { CallStat } from "../types";

export default function StatsPage() {
  const [rows, setRows] = useState<CallStat[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    statsApi
      .list()
      .then((d) => {
        setRows(d.stats ?? []);
        setError("");
      })
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  // aggregate totals
  const byUser = new Map<string, number>();
  const byServer = new Map<string, number>();
  rows.forEach((r) => {
    byUser.set(r.username, (byUser.get(r.username) ?? 0) + r.count);
    byServer.set(r.server_name, (byServer.get(r.server_name) ?? 0) + r.count);
  });
  const totalCalls = rows.reduce((s, r) => s + r.count, 0);

  const columns: ColumnsType<CallStat> = [
    { title: "日期", dataIndex: "date", width: 120 },
    { title: "用户", dataIndex: "username" },
    { title: "服务器", dataIndex: "server_name" },
    { title: "调用次数", dataIndex: "count", width: 100 },
  ];

  const exportCSV = () => {
    const header = "date,username,server_name,count\n";
    const body = rows.map((r) => `${r.date},${r.username},${r.server_name},${r.count}`).join("\n");
    const blob = new Blob(["﻿" + header + body], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "mcp-call-stats.csv";
    a.click();
    URL.revokeObjectURL(url);
    message.success("已导出 CSV");
  };

  const userRows = [...byUser.entries()].map(([username, count]) => ({ username, count }));
  const serverRows = [...byServer.entries()].map(([server_name, count]) => ({ server_name, count }));

  return (
    <>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic title="总调用次数" value={totalCalls} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="活跃用户" value={byUser.size} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic title="被调用的服务器" value={byServer.size} />
          </Card>
        </Col>
      </Row>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={12}>
          <Card title="按用户汇总" size="small">
            <Table
              rowKey="username"
              dataSource={userRows}
              columns={[
                { title: "用户", dataIndex: "username" },
                { title: "调用次数", dataIndex: "count" },
              ]}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
        <Col span={12}>
          <Card title="按服务器汇总" size="small">
            <Table
              rowKey="server_name"
              dataSource={serverRows}
              columns={[
                { title: "服务器", dataIndex: "server_name" },
                { title: "调用次数", dataIndex: "count" },
              ]}
              pagination={false}
              size="small"
            />
          </Card>
        </Col>
      </Row>

      <Card
        title="调用明细（按日）"
        extra={
          <Button onClick={exportCSV} disabled={!rows.length}>
            导出 CSV
          </Button>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          按日记录每位用户对每个 MCP 服务器的工具调用次数（仅次数，不含工具名、参数或结果）。可用于用量观察、报表与后续统计。
        </Typography.Paragraph>
        <Table
          rowKey={(r) => `${r.date}-${r.username}-${r.server_name}`}
          dataSource={rows}
          columns={columns}
          pagination={{ pageSize: 50 }}
          size="small"
        />
      </Card>
    </>
  );
}
