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
import { permissionGroupsApi } from "../api/permissionGroups";
import { usersApi } from "../api/users";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type {
  DashboardServer,
  PermissionGroup,
  PermissionGroupDetail,
  UserAccount,
} from "../types";

interface GroupFormValues {
  name: string;
  display_name: string;
  description?: string;
}

interface ConfigureFormValues {
  display_name: string;
  description?: string;
  user_ids: number[];
  service_ids: number[];
}

export default function PermissionGroupsPage() {
  const [groups, setGroups] = useState<PermissionGroup[]>([]);
  const [users, setUsers] = useState<UserAccount[]>([]);
  const [servers, setServers] = useState<DashboardServer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [detail, setDetail] = useState<PermissionGroupDetail | null>(null);
  const [createForm] = Form.useForm<GroupFormValues>();
  const [configureForm] = Form.useForm<ConfigureFormValues>();

  const load = async () => {
    setLoading(true);
    try {
      const [groupRows, userRows, serverResponse] = await Promise.all([
        permissionGroupsApi.list(),
        usersApi.list(),
        serversApi.list(),
      ]);
      setGroups(groupRows);
      setUsers(userRows);
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

  const create = async (values: GroupFormValues) => {
    setSubmitting(true);
    try {
      await permissionGroupsApi.create({
        name: values.name.trim(),
        display_name: values.display_name.trim(),
        description: values.description?.trim(),
      });
      message.success("权限组已创建");
      setCreateOpen(false);
      createForm.resetFields();
      await load();
    } catch (createError) {
      message.error(extractError(createError));
    } finally {
      setSubmitting(false);
    }
  };

  const openConfigure = async (group: PermissionGroup) => {
    try {
      const next = await permissionGroupsApi.get(group.ID);
      setDetail(next);
      configureForm.setFieldsValue({
        display_name: next.group.display_name,
        description: next.group.description,
        user_ids: next.user_ids,
        service_ids: next.service_ids,
      });
    } catch (detailError) {
      message.error(extractError(detailError));
    }
  };

  const save = async (values: ConfigureFormValues) => {
    if (!detail) return;
    setSubmitting(true);
    try {
      const id = detail.group.ID;
      await permissionGroupsApi.update(id, {
        display_name: values.display_name.trim(),
        description: values.description?.trim() ?? "",
      });
      await permissionGroupsApi.replaceUsers(id, values.user_ids ?? []);
      await permissionGroupsApi.replaceServices(id, values.service_ids ?? []);
      message.success("权限组配置已保存");
      setDetail(null);
      await load();
    } catch (saveError) {
      message.error(extractError(saveError));
    } finally {
      setSubmitting(false);
    }
  };

  const toggle = async (group: PermissionGroup, enabled: boolean) => {
    try {
      await permissionGroupsApi.setEnabled(group.ID, enabled);
      message.success(`${group.display_name} 已${enabled ? "启用" : "停用"}`);
      await load();
    } catch (toggleError) {
      message.error(extractError(toggleError));
    }
  };

  const columns: ColumnsType<PermissionGroup> = [
    {
      title: "权限组",
      key: "identity",
      render: (_, group) => (
        <Space direction="vertical" size={0}>
          <Typography.Text strong>{group.display_name}</Typography.Text>
          <Typography.Text type="secondary">{group.name}</Typography.Text>
        </Space>
      ),
    },
    { title: "说明", dataIndex: "description", ellipsis: true },
    {
      title: "状态",
      dataIndex: "enabled",
      render: (enabled: boolean) => (
        <Tag color={enabled ? "green" : "default"}>{enabled ? "已启用" : "已停用"}</Tag>
      ),
    },
    {
      title: "操作",
      key: "actions",
      render: (_, group) => (
        <Space>
          <Button size="small" onClick={() => void openConfigure(group)}>
            配置成员与服务
          </Button>
          <Switch
            checked={group.enabled}
            checkedChildren="启用"
            unCheckedChildren="停用"
            onChange={(enabled) => void toggle(group, enabled)}
          />
        </Space>
      ),
    },
  ];

  if (loading && groups.length === 0) return <Spin />;

  const userOptions = users.map((user) => ({
    value: user.ID,
    label: `${user.display_name || user.username}（${user.username}）`,
    disabled: user.status === "disabled",
  }));
  const serviceOptions = servers.map((server) => ({
    value: server.id,
    label: server.name,
    disabled: !server.enabled,
  }));

  return (
    <div className="page-stack page-container">
      <Card
        title="权限组"
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
            创建权限组
          </Button>
        }
        className="responsive-card"
      >
        <Typography.Paragraph type="secondary">
          用户最终可访问的 MCP 服务，是其所有已启用权限组所绑定服务的并集。停用权限组会立即撤销该组授权。
        </Typography.Paragraph>
        {error && <Alert type="error" showIcon message={error} style={{ marginBottom: 16 }} />}
        <Table
          rowKey="ID"
          dataSource={groups}
          columns={columns}
          loading={loading}
          pagination={{ pageSize: 20 }}
          scroll={{ x: 820 }}
        />
      </Card>

      <Modal
        title="创建权限组"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        okText="创建"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={create}>
          <Form.Item
            name="name"
            label="标识"
            rules={[
              { required: true, message: "请输入权限组标识" },
              { pattern: /^[a-z0-9][a-z0-9_-]*$/, message: "仅支持小写字母、数字、下划线和连字符" },
            ]}
          >
            <Input placeholder="例如 data_team" />
          </Form.Item>
          <Form.Item name="display_name" label="显示名称" rules={[{ required: true, message: "请输入显示名称" }]}>
            <Input placeholder="例如 数据团队" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`配置权限组：${detail?.group.display_name ?? ""}`}
        open={!!detail}
        onCancel={() => setDetail(null)}
        onOk={() => configureForm.submit()}
        confirmLoading={submitting}
        okText="保存"
        cancelText="取消"
        width={640}
      >
        <Form form={configureForm} layout="vertical" onFinish={save}>
          <Form.Item name="display_name" label="显示名称" rules={[{ required: true, message: "请输入显示名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="user_ids" label="成员">
            <Select mode="multiple" options={userOptions} optionFilterProp="label" placeholder="选择内部用户" />
          </Form.Item>
          <Form.Item
            name="service_ids"
            label="可访问的 MCP 服务"
            extra="只显示当前已注册的服务；停用的服务不可新增选择。"
          >
            <Select mode="multiple" options={serviceOptions} optionFilterProp="label" placeholder="选择 MCP 服务" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
