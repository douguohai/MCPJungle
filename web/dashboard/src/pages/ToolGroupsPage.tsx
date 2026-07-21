import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Popconfirm, Drawer, List, Typography, message, Row, Col, Statistic, Space, Tag } from "antd";
import type { ColumnsType } from "antd/es/table";
import { ApiOutlined, PlusOutlined, ToolOutlined } from "@ant-design/icons";
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
      创建能力组合
    </Button>
  ) : null;

  const groupRows = data?.tool_groups ?? [];
  const totalGroupedTools = groupRows.reduce((sum, groupItem) => sum + groupItem.tool_count, 0);

  const content = (
    <>
      <div className="page-heading">
        <div className="page-heading-copy">
          <Typography.Title level={3} style={{ marginTop: 0 }}>能力中心</Typography.Title>
          <Typography.Paragraph type="secondary">
            把多个 MCP 工具组合成面向场景的独立端点，例如“研发检索”“运维诊断”“数据查询”。外部客户端可以只接入一个组合端点，避免暴露全部工具。
          </Typography.Paragraph>
        </div>
        {addButton}
      </div>
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card className="metric-card">
            <Statistic prefix={<ApiOutlined />} title="能力组合" value={groupRows.length} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card className="metric-card">
            <Statistic prefix={<ToolOutlined />} title="可选工具" value={toolsData.length} />
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card className="metric-card">
            <Statistic title="已编排工具引用" value={totalGroupedTools} />
          </Card>
        </Col>
      </Row>
      <Alert
        showIcon
        type="info"
        message="什么时候使用能力组合"
        description="如果某个客户端或团队只需要固定几类工具，就为它创建能力组合；如果用户需要按权限组访问多个完整 MCP 服务，则直接使用全局 /mcp 入口即可。"
      />
    </>
  );

  if (data?.empty_state && (!data.tool_groups || data.tool_groups.length === 0)) {
    return (
      <div className="page-stack page-container">
        {content}
        <EmptyStateCard
          state={{
            title: "还没有能力组合",
            description: "可以从 MCP 服务详情页选择工具，也可以在这里手动创建面向场景的组合端点。",
          }}
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
      </div>
    );
  }

  return (
    <div className="page-stack page-container">
      {content}
      <Card title="组合列表" className="responsive-card">
        <Table
          rowKey="name"
          dataSource={groupRows}
          columns={columns}
          loading={loading}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 860 }}
        />
      </Card>
      <ToolGroupForm
        open={formOpen}
        onClose={closeForm}
        onCreated={load}
        tools={toolsData}
        initialTools={initialTools}
      />
      <Drawer title={detail?.name} open={!!detail} onClose={() => setDetail(null)} width={480}>
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Typography.Paragraph type="secondary">
            该组合会作为独立 MCP 端点暴露，客户端只会发现组合内的工具。
          </Typography.Paragraph>
          {detail && (
            <Space wrap>
              <Tag color="blue">工具 {detail.tool_count}</Tag>
              <CopyButton text={detail.streamable_http_endpoint} label="复制 HTTP 端点" />
              <CopyButton text={detail.sse_endpoint} label="复制 SSE 端点" />
            </Space>
          )}
        </Space>
        <List
          style={{ marginTop: 16 }}
          size="small"
          dataSource={detail?.tools ?? []}
          renderItem={(t) => (
            <List.Item>
              <List.Item.Meta
                title={<Typography.Text code>{t.canonical_name}</Typography.Text>}
                description={t.description || t.server}
              />
            </List.Item>
          )}
        />
      </Drawer>
    </div>
  );
}
