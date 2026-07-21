import { useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  DatePicker,
  Form,
  Input,
  Modal,
  Popconfirm,
  Radio,
  Select,
  Space,
  Steps,
  Spin,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { PlusOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import dayjs, { type Dayjs } from "dayjs";
import { deviceTokensApi } from "../api/deviceTokens";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type { DashboardServer, DeviceToken, DeviceTokenScope } from "../types";
import CopyButton from "../components/CopyButton";

interface TokenFormValues {
  name: string;
  scope: DeviceTokenScope;
  expires_at?: Dayjs;
  service_ids?: number[];
}

export default function DeviceTokensPage() {
  const [tokens, setTokens] = useState<DeviceToken[]>([]);
  const [servers, setServers] = useState<DashboardServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [plainToken, setPlainToken] = useState<string | null>(null);
  const [form] = Form.useForm<TokenFormValues>();
  const scope = Form.useWatch("scope", form);
  const endpoint = `${window.location.origin}/mcp`;

  const load = async () => {
    setLoading(true);
    try {
      const [tokenRows, serverResponse] = await Promise.all([
        deviceTokensApi.list(),
        serversApi.list(),
      ]);
      setTokens(tokenRows);
      setServers(serverResponse.servers ?? []);
      setError("");
    } catch (loadError) {
      setError(extractError(loadError));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const create = async (values: TokenFormValues) => {
    setSubmitting(true);
    try {
      const result = await deviceTokensApi.create({
        name: values.name.trim(),
        scope: values.scope,
        expires_at: values.expires_at?.toISOString(),
        service_ids:
          values.scope === "restricted" ? values.service_ids ?? [] : undefined,
      });
      setCreateOpen(false);
      form.resetFields();
      setPlainToken(result.token);
      await load();
    } catch (createError) {
      message.error(extractError(createError));
    } finally {
      setSubmitting(false);
    }
  };

  const revoke = async (token: DeviceToken) => {
    try {
      await deviceTokensApi.revoke(token.ID);
      message.success(`已吊销 ${token.name}`);
      await load();
    } catch (revokeError) {
      message.error(extractError(revokeError));
    }
  };

  const columns: ColumnsType<DeviceToken> = [
    { title: "设备名称", dataIndex: "name", width: 200, ellipsis: true },
    {
      title: "令牌前缀",
      dataIndex: "token_prefix",
      width: 150,
      render: (prefix: string) => <Typography.Text code>{prefix}…</Typography.Text>,
    },
    {
      title: "权限范围",
      dataIndex: "scope",
      width: 160,
      render: (value: DeviceTokenScope) =>
        value === "inherit_all" ? "继承当前全部权限" : "仅指定服务",
    },
    {
      title: "有效期至",
      dataIndex: "expires_at",
      width: 180,
      render: (value: string) => new Date(value).toLocaleString("zh-CN"),
    },
    {
      title: "最近使用",
      key: "last_used",
      width: 220,
      render: (_, token) =>
        token.last_used_at ? (
          <Space direction="vertical" size={0}>
            <Typography.Text>{new Date(token.last_used_at).toLocaleString("zh-CN")}</Typography.Text>
            {token.last_used_ip && <Typography.Text type="secondary">{token.last_used_ip}</Typography.Text>}
          </Space>
        ) : "尚未使用",
    },
    {
      title: "状态",
      key: "status",
      width: 90,
      render: (_, token) =>
        token.revoked_at ? (
          <Tag>已吊销</Tag>
        ) : dayjs(token.expires_at).isBefore(dayjs()) ? (
          <Tag color="orange">已过期</Tag>
        ) : (
          <Tag color="green">有效</Tag>
        ),
    },
    {
      title: "操作",
      key: "actions",
      width: 90,
      fixed: "right",
      render: (_, token) =>
        token.revoked_at ? null : (
          <Popconfirm title={`确认吊销 ${token.name}？`} onConfirm={() => void revoke(token)}>
            <Button danger size="small">吊销</Button>
          </Popconfirm>
        ),
    },
  ];

  if (loading && tokens.length === 0) return <Spin />;

  const serviceOptions = servers
    .filter((server) => server.enabled)
    .map((server) => ({ value: server.id, label: server.name }));

  const cursorConfig = plainToken
    ? JSON.stringify(
        {
          mcpServers: {
            mcpjungle: {
              url: endpoint,
              headers: {
                Authorization: `Bearer ${plainToken}`,
              },
            },
          },
        },
        null,
        2,
      )
    : "";

  const mcpRemoteConfig = plainToken
    ? JSON.stringify(
        {
          mcpServers: {
            mcpjungle: {
              command: "npx",
              args: ["-y", "mcp-remote", endpoint, "--header", `Authorization: Bearer ${plainToken}`],
            },
          },
        },
        null,
        2,
      )
    : "";

  return (
    <div className="page-stack page-container">
      <Card
        title="我的设备令牌"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建设备令牌
          </Button>
        }
      >
        <Alert
          showIcon
          type="info"
          style={{ marginBottom: 16 }}
          message="设备令牌是 MCP 客户端专用凭据"
          description={
            <Steps
              size="small"
              responsive
              items={[
                { title: "创建令牌", description: "每台电脑或客户端一个令牌" },
                { title: "复制配置", description: "粘贴到 Cursor、Claude Desktop 或自研客户端" },
                { title: "按人追踪", description: "调用量归属到当前登录用户和设备令牌" },
              ]}
            />
          }
        />
        <Typography.Paragraph type="secondary">
          设备令牌用于 MCP 客户端连接，不用于登录管理后台。令牌明文只在创建成功时展示一次；如果丢失，请吊销后重新创建。
        </Typography.Paragraph>
        {error && <Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} />}
        <Table
          rowKey="ID"
          dataSource={tokens}
          columns={columns}
          loading={loading}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 1050 }}
        />
      </Card>

      <Modal
        title="创建设备令牌"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={submitting}
        okText="创建"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ scope: "inherit_all" }} onFinish={create}>
          <Form.Item name="name" label="设备名称" rules={[{ required: true, message: "请输入设备名称" }]}>
            <Input placeholder="例如 张三的 Claude Desktop" />
          </Form.Item>
          <Form.Item name="scope" label="权限范围" rules={[{ required: true }]}>
            <Radio.Group>
              <Space direction="vertical">
                <Radio value="inherit_all">继承我当前可访问的全部 MCP 服务</Radio>
                <Radio value="restricted">进一步限制到指定 MCP 服务</Radio>
              </Space>
            </Radio.Group>
          </Form.Item>
          {scope === "restricted" && (
            <Form.Item
              name="service_ids"
              label="允许的 MCP 服务"
              rules={[{ required: true, message: "请至少选择一个服务" }]}
              extra="最终权限始终不会超过你所在权限组授予的范围。"
            >
              <Select mode="multiple" options={serviceOptions} placeholder="选择服务" optionFilterProp="label" />
            </Form.Item>
          )}
          <Form.Item name="expires_at" label="到期时间" extra="不填写时由系统采用默认有效期。">
            <DatePicker showTime style={{ width: "100%" }} disabledDate={(date) => date.isBefore(dayjs(), "day")} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="设备令牌（仅显示这一次）"
        open={!!plainToken}
        closable={false}
        maskClosable={false}
        footer={
          <Button type="primary" onClick={() => setPlainToken(null)}>
            我已安全保存
          </Button>
        }
      >
        <Alert type="warning" showIcon message="关闭后无法再次查看，请立即复制到 MCP 客户端。" style={{ marginBottom: 16 }} />
        <Space direction="vertical" size={16} style={{ width: "100%" }}>
          <div>
            <Typography.Text strong>令牌</Typography.Text>
            <Typography.Paragraph code copyable style={{ wordBreak: "break-all", marginTop: 8 }}>
              {plainToken}
            </Typography.Paragraph>
          </div>
          <div>
            <Space align="center" style={{ marginBottom: 8 }}>
              <Typography.Text strong>Cursor / 支持 Streamable HTTP 的客户端</Typography.Text>
              <CopyButton text={cursorConfig} />
            </Space>
            <Typography.Paragraph code style={{ whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
              {cursorConfig}
            </Typography.Paragraph>
          </div>
          <div>
            <Space align="center" style={{ marginBottom: 8 }}>
              <Typography.Text strong>Claude Desktop / mcp-remote</Typography.Text>
              <CopyButton text={mcpRemoteConfig} />
            </Space>
            <Typography.Paragraph code style={{ whiteSpace: "pre-wrap", wordBreak: "break-all" }}>
              {mcpRemoteConfig}
            </Typography.Paragraph>
          </div>
        </Space>
      </Modal>
    </div>
  );
}
