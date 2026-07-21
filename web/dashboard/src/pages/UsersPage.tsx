import { useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { PlusOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import { usersApi } from "../api/users";
import { extractError } from "../api/client";
import type { UserAccount, UserRole, UserStatus } from "../types";

const roleOptions: Array<{ value: UserRole; label: string }> = [
  { value: "system_admin", label: "系统管理员" },
  { value: "service_admin", label: "服务管理员" },
  { value: "member", label: "成员" },
  { value: "auditor", label: "审计员" },
];

const roleLabels = Object.fromEntries(
  roleOptions.map((option) => [option.value, option.label]),
) as Record<UserRole, string>;

const statusMeta: Record<UserStatus, { label: string; color: string }> = {
  pending: { label: "待首次登录", color: "gold" },
  active: { label: "已启用", color: "green" },
  disabled: { label: "已禁用", color: "default" },
};

export default function UsersPage() {
  const [accounts, setAccounts] = useState<UserAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<UserAccount | null>(null);
  const [initialCredential, setInitialCredential] = useState<{
    username: string;
    password: string;
  } | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const load = async () => {
    setLoading(true);
    try {
      setAccounts(await usersApi.list());
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

  const create = async (values: {
    username: string;
    display_name?: string;
    role: UserRole;
  }) => {
    setSubmitting(true);
    try {
      const result = await usersApi.create({
        username: values.username.trim(),
        display_name: values.display_name?.trim(),
        role: values.role,
      });
      setCreateOpen(false);
      createForm.resetFields();
      setInitialCredential({
        username: result.user.username,
        password: result.initial_password,
      });
      await load();
    } catch (createError) {
      message.error(extractError(createError));
    } finally {
      setSubmitting(false);
    }
  };

  const update = async (values: { display_name: string; role: UserRole }) => {
    if (!editing) return;
    setSubmitting(true);
    try {
      await usersApi.update(editing.ID, {
        display_name: values.display_name.trim(),
        role: values.role,
      });
      message.success("用户信息已更新");
      setEditing(null);
      await load();
    } catch (updateError) {
      message.error(extractError(updateError));
    } finally {
      setSubmitting(false);
    }
  };

  const toggle = async (account: UserAccount, enabled: boolean) => {
    try {
      await usersApi.setEnabled(account.ID, enabled);
      message.success(`${account.username} 已${enabled ? "启用" : "禁用"}`);
      await load();
    } catch (toggleError) {
      message.error(extractError(toggleError));
    }
  };

  const columns: ColumnsType<UserAccount> = [
    {
      title: "用户",
      key: "identity",
      render: (_, account) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{account.display_name || account.username}</Typography.Text>
          <Typography.Text type="secondary">{account.username}</Typography.Text>
        </Space>
      ),
    },
    {
      title: "角色",
      dataIndex: "role",
      render: (role: UserRole) => roleLabels[role] ?? role,
    },
    {
      title: "状态",
      dataIndex: "status",
      render: (status: UserStatus, account) => (
        <Space>
          <Tag color={statusMeta[status]?.color}>{statusMeta[status]?.label ?? status}</Tag>
          {account.must_change_password && <Tag color="orange">需改密码</Tag>}
        </Space>
      ),
    },
    {
      title: "最近登录",
      dataIndex: "last_login_at",
      render: (value?: string) =>
        value ? new Date(value).toLocaleString("zh-CN") : "尚未登录",
    },
    {
      title: "操作",
      key: "actions",
      render: (_, account) => (
        <Space>
          <Button
            size="small"
            onClick={() => {
              setEditing(account);
              editForm.setFieldsValue({
                display_name: account.display_name,
                role: account.role,
              });
            }}
          >
            编辑
          </Button>
          <Switch
            checked={account.status === "active"}
            checkedChildren="启用"
            unCheckedChildren="禁用"
            onChange={(enabled) => void toggle(account, enabled)}
          />
        </Space>
      ),
    },
  ];

  if (loading && accounts.length === 0) return <Spin />;

  return (
    <div className="page-stack page-container">
      <Card
        title="内部用户"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建用户
          </Button>
        }
        className="responsive-card"
      >
        <Typography.Paragraph type="secondary">
          新用户以待激活状态创建。系统只展示一次初始密码，用户首次登录并修改密码后才会激活。
        </Typography.Paragraph>
        {error && <Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} />}
        <Table
          rowKey="ID"
          dataSource={accounts}
          columns={columns}
          loading={loading}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 860 }}
        />
      </Card>

      <Modal
        title="创建内部用户"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        okText="创建"
        cancelText="取消"
        destroyOnClose
      >
        <Form
          form={createForm}
          layout="vertical"
          initialValues={{ role: "member" }}
          onFinish={create}
        >
          <Form.Item name="username" label="登录名" rules={[{ required: true, message: "请输入登录名" }]}>
            <Input placeholder="例如 alice" autoComplete="off" />
          </Form.Item>
          <Form.Item name="display_name" label="显示名称">
            <Input placeholder="例如 张三" />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true }]}>
            <Select options={roleOptions} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`编辑用户：${editing?.username ?? ""}`}
        open={!!editing}
        onCancel={() => setEditing(null)}
        onOk={() => editForm.submit()}
        confirmLoading={submitting}
        okText="保存"
        cancelText="取消"
      >
        <Form form={editForm} layout="vertical" onFinish={update}>
          <Form.Item name="display_name" label="显示名称" rules={[{ required: true, message: "请输入显示名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="role" label="角色" rules={[{ required: true }]}>
            <Select options={roleOptions} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="一次性初始密码"
        open={!!initialCredential}
        closable={false}
        maskClosable={false}
        footer={
          <Button type="primary" onClick={() => setInitialCredential(null)}>
            我已安全保存
          </Button>
        }
      >
        <Alert
          type="warning"
          showIcon
          message="关闭后无法再次查看，请通过安全渠道交给用户。"
          style={{ marginBottom: 16 }}
        />
        <Typography.Paragraph>
          用户名：<Typography.Text code copyable>{initialCredential?.username}</Typography.Text>
        </Typography.Paragraph>
        <Typography.Paragraph>
          初始密码：<Typography.Text code copyable>{initialCredential?.password}</Typography.Text>
        </Typography.Paragraph>
      </Modal>
    </div>
  );
}
