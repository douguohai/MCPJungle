import { useEffect, useState } from "react";
import { Table, Card, Alert, Spin, Button, Popconfirm, Modal, Form, Input, Select, Typography, Tag, message } from "antd";
import { PlusOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { usersApi } from "../api/users";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type { UserListItem, DashboardServer } from "../types";

export default function UsersPage() {
  const [data, setData] = useState<UserListItem[]>([]);
  const [servers, setServers] = useState<DashboardServer[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<UserListItem | null>(null);
  const [created, setCreated] = useState<{ username: string; token: string } | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const load = () => {
    setLoading(true);
    Promise.all([usersApi.list(), serversApi.list()])
      .then(([us, ss]) => {
        setData(us);
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

  const onCreate = async (values: { username: string }) => {
    setSubmitting(true);
    try {
      const resp = await usersApi.create({ username: values.username.trim() });
      message.success(`已创建用户 ${resp.username}`);
      setCreated({ username: resp.username, token: resp.access_token });
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const onEdit = async (values: { allowed_servers?: string[] }) => {
    if (!editTarget) return;
    setSubmitting(true);
    try {
      await usersApi.update(editTarget.username, { allowed_servers: values.allowed_servers ?? [] });
      message.success(`已更新 ${editTarget.username} 的可访问服务器`);
      setEditTarget(null);
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const remove = async (username: string) => {
    try {
      await usersApi.remove(username);
      message.success(`已删除用户 ${username}`);
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const allowOptions = [
    { value: "*", label: "* （全部服务器）" },
    ...servers.map((s) => ({ value: s.name, label: s.name })),
  ];

  const columns: ColumnsType<UserListItem> = [
    { title: "用户名", dataIndex: "username" },
    { title: "角色", dataIndex: "role" },
    {
      title: "可访问的服务器",
      dataIndex: "allowed_servers",
      render: (list?: string[] | null) =>
        !list || list.length === 0 ? (
          <Typography.Text type="secondary">全部</Typography.Text>
        ) : (
          list.map((s) => <Tag key={s}>{s}</Tag>)
        ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <>
          <Button
            size="small"
            onClick={() => {
              setEditTarget(row);
              editForm.setFieldsValue({ allowed_servers: row.allowed_servers ?? [] });
            }}
          >
            配置授权
          </Button>{" "}
          {row.role !== "admin" && (
            <Popconfirm title={`确认删除用户 ${row.username}？`} onConfirm={() => remove(row.username)}>
              <Button danger size="small">
                删除
              </Button>
            </Popconfirm>
          )}
        </>
      ),
    },
  ];

  return (
    <>
      <Card
        title="用户"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            新建用户
          </Button>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          用户是可以登录 dashboard 和使用 CLI 的账号。管理员（admin）可管理一切；普通用户（user）只能查看和调用工具。「可访问的服务器」限制该用户（及其所有客户端）最多能用哪些 MCP 服务器——留空表示全部。
        </Typography.Paragraph>
        <Table rowKey="username" dataSource={data} columns={columns} loading={loading} pagination={{ pageSize: 20 }} />
      </Card>

      <Modal
        title="新建用户"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        okText="创建"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={onCreate}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: "请输入用户名" }]}>
            <Input placeholder="例如 alice" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`配置授权：${editTarget?.username}`}
        open={!!editTarget}
        onCancel={() => setEditTarget(null)}
        onOk={() => editForm.submit()}
        confirmLoading={submitting}
        okText="保存"
        cancelText="取消"
      >
        <Form form={editForm} layout="vertical" onFinish={onEdit}>
          <Form.Item
            name="allowed_servers"
            label="可访问的服务器"
            tooltip="选择该用户能用哪些 MCP 服务器。* 或留空 = 全部。列表来自已注册服务器，可在「MCP 服务器」页注册新的。"
          >
            <Select mode="multiple" placeholder="选择允许访问的服务器（留空=全部）" options={allowOptions} allowClear />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="用户访问令牌（仅显示这一次）"
        open={!!created}
        onCancel={() => setCreated(null)}
        footer={
          <Button type="primary" onClick={() => setCreated(null)}>
            我已保存
          </Button>
        }
      >
        <Alert type="warning" showIcon message="此令牌只显示一次，请立即复制并交给该用户。" style={{ marginBottom: 12 }} />
        <Typography.Paragraph code copyable>
          {created?.token}
        </Typography.Paragraph>
      </Modal>
    </>
  );
}
