import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Typography, Statistic, Row, Col, Button, message, Empty, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { statsApi } from "../api/stats";
import { extractError } from "../api/client";
import type { CallStat } from "../types";
import { DownloadOutlined } from "@ant-design/icons";

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

  const latestDate = rows.reduce<string | undefined>((latest, row) => {
    if (!latest || row.date > latest) return row.date;
    return latest;
  }, undefined);

  const columns: ColumnsType<CallStat> = [
    { title: "日期", dataIndex: "date", width: 120 },
    {
      title: "用户",
      dataIndex: "username",
      render: (value: string) => <Typography.Text strong>{value}</Typography.Text>,
    },
    {
      title: "MCP 服务",
      dataIndex: "server_name",
      render: (value: string) => <Tag color="blue">{value}</Tag>,
    },
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
    <div className="page-stack page-container">
      <div className="page-heading">
        <div className="page-heading-copy">
          <Typography.Title level={3} style={{ marginTop: 0 }}>调用统计</Typography.Title>
          <Typography.Paragraph type="secondary">
            用于内部 MCP 调用观察和基础审计。当前统计粒度为“日期 + 用户 + MCP 服务 + 次数”，不保存工具参数或返回结果。
          </Typography.Paragraph>
        </div>
        <Button icon={<DownloadOutlined />} onClick={exportCSV} disabled={!rows.length}>
          导出 CSV
        </Button>
      </div>

      <Alert
        showIcon
        type={rows.length ? "info" : "warning"}
        message={rows.length ? `最近有调用记录：${latestDate}` : "暂无调用记录"}
        description="如果要用于正式团队审计，下一阶段建议补充工具名、设备令牌、成功/失败、耗时和错误原因。"
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card className="metric-card">
            <Statistic title="总调用次数" value={totalCalls} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card className="metric-card">
            <Statistic title="活跃用户" value={byUser.size} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card className="metric-card">
            <Statistic title="被调用的 MCP 服务" value={byServer.size} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
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
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无用户调用" /> }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="按 MCP 服务汇总" size="small">
            <Table
              rowKey="server_name"
              dataSource={serverRows}
              columns={[
                { title: "MCP 服务", dataIndex: "server_name" },
                { title: "调用次数", dataIndex: "count" },
              ]}
              pagination={false}
              size="small"
              locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无服务调用" /> }}
            />
          </Card>
        </Col>
      </Row>

      <Card title="调用明细（按日）" className="responsive-card">
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          这里用于回答“谁在用、用了哪个 MCP 服务、每天用了多少次”。如果需要定位单次失败，请在后续版本补充请求级审计日志。
        </Typography.Paragraph>
        <Table
          rowKey={(r) => `${r.date}-${r.username}-${r.server_name}`}
          dataSource={rows}
          columns={columns}
          pagination={{ pageSize: 50 }}
          size="small"
          scroll={{ x: 760 }}
          locale={{ emptyText: <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无调用明细" /> }}
        />
      </Card>
    </div>
  );
}
