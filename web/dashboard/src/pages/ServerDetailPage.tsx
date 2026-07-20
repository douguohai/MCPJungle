import { useEffect, useMemo, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Row,
  Space,
  Spin,
  Statistic,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { ArrowLeftOutlined } from "@ant-design/icons";
import { Link, useParams } from "react-router-dom";
import { serversApi } from "../api/servers";
import { toolsApi } from "../api/tools";
import { promptsApi } from "../api/prompts";
import { resourcesApi } from "../api/resources";
import { extractError } from "../api/client";
import type {
  DashboardPrompt,
  DashboardResource,
  DashboardServer,
  DashboardTool,
} from "../types";
import { useAuth } from "../store/auth";
import CopyButton from "../components/CopyButton";
import JsonViewer from "../components/JsonViewer";
import StatusBadge from "../components/StatusBadge";
import TransportTag from "../components/TransportTag";

interface DetailData {
  server: DashboardServer;
  tools: DashboardTool[];
  prompts: DashboardPrompt[];
  resources: DashboardResource[];
}

function CapabilityEmpty({ label }: { label: string }) {
  return (
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={`该服务暂未发现${label}。请确认上游服务已启用并支持此类能力。`}
    />
  );
}

export default function ServerDetailPage() {
  const { serverName = "" } = useParams();
  const { user } = useAuth();
  const canManage = user?.role === "system_admin";
  const [data, setData] = useState<DetailData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [toolDetail, setToolDetail] = useState<DashboardTool | null>(null);
  const [promptDetail, setPromptDetail] = useState<DashboardPrompt | null>(null);
  const [toggling, setToggling] = useState<string | null>(null);

  const load = () => {
    setLoading(true);
    setError("");
    Promise.all([serversApi.list(), toolsApi.list(), promptsApi.list(), resourcesApi.list()])
      .then(([servers, tools, prompts, resources]) => {
        const server = servers.servers.find((item) => item.name === serverName);
        if (!server) {
          setData(null);
          return;
        }
        setData({
          server,
          tools: tools.tools.filter((item) => item.server === server.name),
          prompts: prompts.prompts.filter((item) => item.server === server.name),
          resources: resources.resources.filter((item) => item.server === server.name),
        });
      })
      .catch((reason) => setError(extractError(reason)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
  }, [serverName]);

  const toggleTool = async (row: DashboardTool, enabled: boolean) => {
    setToggling(row.canonical_name);
    try {
      await toolsApi.setEnabled(row.canonical_name, enabled);
      message.success(`已${enabled ? "启用" : "禁用"}工具 ${row.name}`);
      load();
    } catch (reason) {
      message.error(extractError(reason));
    } finally {
      setToggling(null);
    }
  };

  const togglePrompt = async (row: DashboardPrompt, enabled: boolean) => {
    setToggling(row.canonical_name);
    try {
      await promptsApi.setEnabled(row.canonical_name, enabled);
      message.success(`已${enabled ? "启用" : "禁用"}提示模板 ${row.name}`);
      load();
    } catch (reason) {
      message.error(extractError(reason));
    } finally {
      setToggling(null);
    }
  };

  const toolColumns = useMemo<ColumnsType<DashboardTool>>(
    () => [
      {
        title: "名称",
        dataIndex: "name",
        render: (name: string, row) => (
          <Space size={8}>
            <Typography.Text>{name}</Typography.Text>
            <CopyButton text={row.canonical_name} />
          </Space>
        ),
      },
      { title: "描述", dataIndex: "description", ellipsis: true },
      {
        title: "注解",
        dataIndex: "annotation_keys",
        render: (keys: string[] = []) => keys.map((key) => <Tag key={key}>{key}</Tag>),
      },
      {
        title: "启用",
        dataIndex: "enabled",
        width: 90,
        render: (enabled: boolean, row) => (
          <Tooltip title={canManage ? undefined : "当前账号只有查看权限"}>
            <Switch
              checked={row.server_enabled && enabled}
              disabled={!canManage || !row.server_enabled}
              loading={toggling === row.canonical_name}
              onChange={(checked) => void toggleTool(row, checked)}
            />
          </Tooltip>
        ),
      },
      {
        title: "操作",
        width: 200,
        render: (_, row) => (
          <Space size={8}>
            <Button onClick={() => setToolDetail(row)}>查看输入</Button>
            {canManage && (
              <Link to={`/settings/ability-combinations?tools=${encodeURIComponent(row.canonical_name)}`}>
                <Button>加入组合</Button>
              </Link>
            )}
          </Space>
        ),
      },
    ],
    [canManage, toggling],
  );

  const promptColumns = useMemo<ColumnsType<DashboardPrompt>>(
    () => [
      {
        title: "名称",
        dataIndex: "name",
        render: (name: string, row) => (
          <Space size={8}>
            <Typography.Text>{name}</Typography.Text>
            <CopyButton text={row.canonical_name} />
          </Space>
        ),
      },
      { title: "描述", dataIndex: "description", ellipsis: true },
      {
        title: "启用",
        dataIndex: "enabled",
        width: 90,
        render: (enabled: boolean, row) => (
          <Tooltip title={canManage ? undefined : "当前账号只有查看权限"}>
            <Switch
              checked={row.server_enabled && enabled}
              disabled={!canManage}
              loading={toggling === row.canonical_name}
              onChange={(checked) => void togglePrompt(row, checked)}
            />
          </Tooltip>
        ),
      },
      {
        title: "操作",
        width: 100,
        render: (_, row) => <Button onClick={() => setPromptDetail(row)}>查看参数</Button>,
      },
    ],
    [canManage, toggling],
  );

  const resourceColumns: ColumnsType<DashboardResource> = [
    {
      title: "URI",
      dataIndex: "uri",
      render: (uri: string) => (
        <Space>
          <Typography.Text code>{uri}</Typography.Text>
          <CopyButton text={uri} />
        </Space>
      ),
    },
    { title: "名称", dataIndex: "name" },
    { title: "MIME 类型", dataIndex: "mime_type" },
    { title: "描述", dataIndex: "description", ellipsis: true },
  ];

  if (loading && !data) return <Spin size="large" />;
  if (error) return <Alert type="error" showIcon message="服务详情加载失败" description={error} />;
  if (!data) {
    return (
      <Card>
        <Empty description={`未找到 MCP 服务“${serverName}”`}>
          <Link to="/servers"><Button type="primary">返回 MCP 服务</Button></Link>
        </Empty>
      </Card>
    );
  }

  const { server, tools, prompts, resources } = data;
  const overview = (
    <div className="page-stack">
      <Row gutter={[16, 16]}>
        <Col xs={12} lg={6}><Card><Statistic title="工具" value={tools.length} /></Card></Col>
        <Col xs={12} lg={6}><Card><Statistic title="提示模板" value={prompts.length} /></Card></Col>
        <Col xs={12} lg={6}><Card><Statistic title="数据资源" value={resources.length} /></Card></Col>
        <Col xs={12} lg={6}><Card><Statistic title="已启用能力" value={tools.filter((item) => item.enabled).length + prompts.filter((item) => item.enabled).length + resources.filter((item) => item.enabled).length} /></Card></Col>
      </Row>
      <Card title="服务信息">
        <Descriptions column={{ xs: 1, md: 2 }}>
          <Descriptions.Item label="传输协议"><TransportTag transport={server.transport} /></Descriptions.Item>
          <Descriptions.Item label="运行状态"><StatusBadge status={server.status} /></Descriptions.Item>
          <Descriptions.Item label="配置摘要" span={2}>{server.config_summary?.sanitized_summary || server.connection_summary}</Descriptions.Item>
          <Descriptions.Item label="最近发现">{server.last_discovered_at || "尚未完成发现"}</Descriptions.Item>
          <Descriptions.Item label="最近更新">{server.updated_at || "-"}</Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  );

  const tabs = [
    { key: "overview", label: "概览", children: overview },
    {
      key: "tools",
      label: `工具 ${tools.length}`,
      children: tools.length ? (
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          {canManage && (
            <Link to={`/settings/ability-combinations?server=${encodeURIComponent(server.name)}`}>
              <Button>用该服务工具创建能力组合</Button>
            </Link>
          )}
          <Table rowKey="canonical_name" columns={toolColumns} dataSource={tools} pagination={{ pageSize: 20 }} />
        </Space>
      ) : <CapabilityEmpty label="工具" />,
    },
    {
      key: "prompts",
      label: `提示模板 ${prompts.length}`,
      children: prompts.length ? <Table rowKey="canonical_name" columns={promptColumns} dataSource={prompts} pagination={{ pageSize: 20 }} /> : <CapabilityEmpty label="提示模板" />,
    },
    {
      key: "resources",
      label: `数据资源 ${resources.length}`,
      children: resources.length ? <Table rowKey="uri" columns={resourceColumns} dataSource={resources} pagination={{ pageSize: 20 }} /> : <CapabilityEmpty label="数据资源" />,
    },
    {
      key: "diagnostics",
      label: "连接诊断",
      children: (
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <Alert
            showIcon
            type={server.enabled && server.status !== "failed" ? "success" : "warning"}
            message={server.enabled ? "该服务已启用" : "该服务当前已停用"}
            description={server.status === "failed" ? "最近一次能力发现失败，请检查上游地址、认证信息和网络连接。" : "连接状态来自最近一次能力发现结果。"}
          />
          <Descriptions bordered column={1}>
            <Descriptions.Item label="发现状态"><StatusBadge status={server.status} /></Descriptions.Item>
            <Descriptions.Item label="连接信息">{server.config_summary?.sanitized_summary || server.connection_summary}</Descriptions.Item>
            <Descriptions.Item label="最近发现">{server.last_discovered_at || "尚未完成发现"}</Descriptions.Item>
            <Descriptions.Item label="能力总数">{tools.length + prompts.length + resources.length}</Descriptions.Item>
          </Descriptions>
        </Space>
      ),
    },
  ];

  return (
    <div className="page-stack">
      <Link to="/servers"><Button type="link" icon={<ArrowLeftOutlined />} style={{ padding: 0 }}>返回 MCP 服务</Button></Link>
      <div className="service-detail-title">
        <div>
          <Space align="center">
            <Typography.Title level={3} style={{ margin: 0 }}>{server.name}</Typography.Title>
            <StatusBadge status={server.status} />
          </Space>
          <Typography.Text type="secondary">{server.connection_summary}</Typography.Text>
        </div>
        <TransportTag transport={server.transport} />
      </div>
      <Card className="service-tabs-card"><Tabs items={tabs} /></Card>
      <Drawer title={toolDetail?.canonical_name} open={!!toolDetail} onClose={() => setToolDetail(null)} width={560}>
        {toolDetail && <JsonViewer value={toolDetail.input_schema ?? toolDetail.input_preview} />}
      </Drawer>
      <Drawer title={promptDetail?.canonical_name} open={!!promptDetail} onClose={() => setPromptDetail(null)} width={560}>
        {promptDetail && <JsonViewer value={promptDetail.arguments ?? promptDetail.arguments_preview} />}
      </Drawer>
    </div>
  );
}
