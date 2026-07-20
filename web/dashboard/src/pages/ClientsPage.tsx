import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Popconfirm, Modal, Form, Input, Select, Typography, Tag, message } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { clientsApi } from "../api/clients";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type { McpClient, DashboardServer } from "../types";
import { getUser } from "../store/auth";

export default function ClientsPage() {
  const user = getUser();
  const [data, setData] = useState<McpClient[]>([]);
  const [servers, setServers] = useState<DashboardServer[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<McpClient | null>(null);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const load = () => {
    setLoading(true);
    Promise.all([clientsApi.list(), serversApi.list()])
      .then(([cs, ss]) => {
        setData(cs);
        setServers(ss.servers ?? []);
        setError("");
      })
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  };
  useEffect(() => {
    load();
  }, []);

  if (loading && !data.length) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  const onCreate = async (values: { name: string; description?: string; allow_list?: string[] }) => {
    setSubmitting(true);
    try {
      const client = await clientsApi.create({
        name: values.name.trim(),
        description: values.description,
        allow_list: values.allow_list ?? [],
      });
      message.success(`已创建客户端 ${client.name}`);
      if (client.access_token) setCreatedToken(client.access_token);
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const onEdit = async (values: { allow_list?: string[] }) => {
    if (!editTarget) return;
    setSubmitting(true);
    try {
      await clientsApi.update(editTarget.name, { allow_list: values.allow_list ?? [] });
      message.success(`已更新 ${editTarget.name} 的授权`);
      setEditTarget(null);
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const remove = async (name: string) => {
    try {
      await clientsApi.remove(name);
      message.success(`已删除客户端 ${name}，其 Token 立即失效`);
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  // AllowList options: a wildcard plus every registered server.
  const allowListOptions = [
    { value: "*", label: "* （全部服务器，不推荐）" },
    ...servers.map((s) => ({ value: s.name, label: s.name })),
  ];

  const columns: ColumnsType<McpClient> = [
    { title: "名称", dataIndex: "name" },
    { title: "描述", dataIndex: "description", ellipsis: true },
    {
      title: "可访问的服务器（AllowList）",
      dataIndex: "allow_list",
      render: (list: string[]) =>
        list.length ? (
          list.map((s) => <Tag key={s}>{s}</Tag>)
        ) : (
          <Typography.Text type="secondary">无（不能访问任何服务器）</Typography.Text>
        ),
    },
    // admins see which user each client belongs to
    ...(user?.role === "admin"
      ? [{ title: "归属用户", key: "owner", render: (_: unknown, row: McpClient) => `#${row.user_id ?? "-"}` }]
      : []),
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <>
          <Button
            size="small"
            onClick={() => {
              setEditTarget(row);
              editForm.setFieldsValue({ allow_list: row.allow_list });
            }}
          >
            编辑授权
          </Button>{" "}
          <Popconfirm title={`确认删除客户端 ${row.name}？其 Token 立即失效。`} onConfirm={() => remove(row.name)}>
            <Button danger size="small">
              删除
            </Button>
          </Popconfirm>
        </>
      ),
    },
  ];

  return (
    <>
      <Card
        title="MCP 客户端"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建客户端
          </Button>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          MCP 客户端代表连接 MCPJungle 使用工具的 AI 应用（如 Claude Desktop、Cursor）。为每个应用创建一个客户端，通过 AllowList 控制它能访问哪些 MCP 服务器；创建时生成的 Token 需配置到该应用的 MCP 连接（请求头 <Typography.Text code>Authorization: Bearer {"<token>"}</Typography.Text>）。
        </Typography.Paragraph>
        <Table rowKey="name" dataSource={data} columns={columns} loading={loading} pagination={{ pageSize: 20 }} />
      </Card>

      <Modal
        title="新建 MCP 客户端"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        okText="创建"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={onCreate}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入客户端名称" }]} tooltip="客户端唯一标识，如 claude-desktop">
            <Input placeholder="例如 claude-desktop" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input placeholder="这个客户端给谁/什么应用用" />
          </Form.Item>
          <Form.Item
            name="allow_list"
            label="可访问的服务器（AllowList）"
            tooltip="选择该客户端能访问的 MCP 服务器。* 表示全部（不推荐）。不选 = 不能访问任何服务器。列表来自已注册的服务器，可在「MCP 服务器」页注册新的。"
          >
            <Select
              mode="multiple"
              placeholder={servers.length ? "选择允许访问的服务器" : "暂无已注册服务器（可选 * 或先去注册服务器）"}
              options={allowListOptions}
              allowClear
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`编辑授权：${editTarget?.name}`}
        open={!!editTarget}
        onCancel={() => setEditTarget(null)}
        onOk={() => editForm.submit()}
        confirmLoading={submitting}
        okText="保存"
        cancelText="取消"
      >
        <Form form={editForm} layout="vertical" onFinish={onEdit}>
          <Form.Item
            name="allow_list"
            label="可访问的服务器（AllowList）"
            tooltip="选择允许访问的服务器。* 表示全部。留空 = 不能访问任何服务器。"
          >
            <Select mode="multiple" placeholder="选择允许访问的服务器" options={allowListOptions} allowClear />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="客户端 Token（仅显示这一次）"
        open={!!createdToken}
        onCancel={() => setCreatedToken(null)}
        footer={
          <Button type="primary" onClick={() => setCreatedToken(null)}>
            我已保存
          </Button>
        }
      >
        <Alert type="warning" showIcon message="此 Token 只显示一次，关闭后无法再查看。请立即复制保存。" style={{ marginBottom: 12 }} />
        <Typography.Paragraph code copyable>
          {createdToken}
        </Typography.Paragraph>
      </Modal>
    </>
  );
}
