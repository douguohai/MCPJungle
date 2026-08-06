import { useState } from "react";
import { Card, Typography, Table, Input, Select, Space, Tag, DatePicker } from "antd";
import type { ColumnsType } from "antd/es/table";
import dayjs from "dayjs";

const { RangePicker } = DatePicker;

// AuditEvent placeholder type — will be replaced when AuditEvent API (design doc step 6) is implemented.
interface AuditEvent {
  id: number;
  actor: string;
  action: string;
  target: string;
  result: string;
  created_at: string;
}

export default function AuditPage() {
  // TODO: Replace with real API call once AuditEvent API is available (design doc step 6).
  const [data] = useState<AuditEvent[]>([]);
  const [loading] = useState(false);

  // Filter state
  const [actionFilter, setActionFilter] = useState<string | undefined>();
  const [actorFilter, setActorFilter] = useState("");

  // Filtered data
  const filtered = data.filter((e) => {
    if (actionFilter && e.action !== actionFilter) return false;
    if (actorFilter && !e.actor.includes(actorFilter)) return false;
    return true;
  });

  // Collect unique action types for filter dropdown
  const actionTypes = [...new Set(data.map((e) => e.action))];

  const columns: ColumnsType<AuditEvent> = [
    { title: "时间", dataIndex: "created_at", width: 180 },
    {
      title: "操作人",
      dataIndex: "actor",
      render: (v: string) => <Tag>{v}</Tag>,
    },
    {
      title: "操作类型",
      dataIndex: "action",
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
    { title: "操作目标", dataIndex: "target", ellipsis: true },
    {
      title: "结果",
      dataIndex: "result",
      width: 80,
      render: (v: string) => (
        <Tag color={v === "success" ? "green" : "red"}>
          {v === "success" ? "成功" : "失败"}
        </Tag>
      ),
    },
  ];

  return (
    <>
      <Typography.Title level={4} style={{ marginBottom: 4 }}>
        审计日志
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        记录系统中的关键操作，包括用户管理、权限变更、服务配置等。审计日志 API 将在后续版本中提供。
      </Typography.Paragraph>

      <Card>
        <Space style={{ marginBottom: 16 }} wrap>
          <Input
            placeholder="按操作人筛选"
            value={actorFilter}
            onChange={(e) => setActorFilter(e.target.value)}
            allowClear
            style={{ width: 180 }}
          />
          <Select
            placeholder="按操作类型筛选"
            value={actionFilter}
            onChange={setActionFilter}
            allowClear
            style={{ width: 180 }}
            options={actionTypes.map((a) => ({ value: a, label: a }))}
          />
        </Space>

        {data.length === 0 && !loading ? (
          <div style={{ textAlign: "center", padding: "48px 0" }}>
            <Typography.Title
              level={5}
              type="secondary"
              style={{ marginBottom: 8 }}
            >
              暂无审计记录
            </Typography.Title>
            <Typography.Text type="secondary">
              审计日志功能将在后续版本中启用（设计文档第 6
              步）。届时，系统中的关键操作将自动记录并在此展示。
            </Typography.Text>
          </div>
        ) : (
          <Table
            rowKey="id"
            dataSource={filtered}
            columns={columns}
            loading={loading}
            pagination={{ pageSize: 50 }}
            size="small"
          />
        )}
      </Card>
    </>
  );
}
