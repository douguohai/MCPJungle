import { useEffect, useState } from "react";
import {
  Card,
  Alert,
  Spin,
  Typography,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Tag,
  Popconfirm,
  Select,
  message,
  Drawer,
  List,
  Space,
  Descriptions,
} from "antd";
import {
  PlusOutlined,
  TeamOutlined,
  CloudServerOutlined,
  DeleteOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import {
  permissionGroupsApi,
  type PermissionGroup,
  type PermissionGroupDetail,
  type PermissionGroupMember,
  type PermissionGroupService,
} from "../api/permissionGroups";
import { usersApi } from "../api/users";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type { UserListItem, DashboardServer } from "../types";

export default function PermissionGroupsPage() {
  const [groups, setGroups] = useState<PermissionGroup[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<PermissionGroup | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // Detail drawer state
  const [detailGroup, setDetailGroup] = useState<PermissionGroupDetail | null>(
    null,
  );
  const [detailLoading, setDetailLoading] = useState(false);
  const [users, setUsers] = useState<UserListItem[]>([]);
  const [servers, setServers] = useState<DashboardServer[]>([]);
  const [addMemberId, setAddMemberId] = useState<number | undefined>();
  const [addServerId, setAddServerId] = useState<number | undefined>();

  const load = (signal?: AbortSignal) => {
    setLoading(true);
    permissionGroupsApi
      .list(signal)
      .then(setGroups)
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  };

  const loadUsersAndServers = (signal?: AbortSignal) => {
    usersApi.list(signal).then(setUsers).catch(() => {});
    serversApi.list(signal).then((r) => setServers(r.servers ?? [])).catch(() => {});
  };

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    loadUsersAndServers(controller.signal);
    return () => controller.abort();
  }, []);

  if (loading && !groups.length) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  const onCreate = async (values: {
    name: string;
    display_name?: string;
    description?: string;
  }) => {
    setSubmitting(true);
    try {
      await permissionGroupsApi.create({
        name: values.name.trim(),
        display_name: values.display_name?.trim(),
        description: values.description?.trim(),
      });
      message.success("已创建权限组");
      setCreateOpen(false);
      createForm.resetFields();
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const onEdit = async (values: {
    display_name: string;
    description: string;
  }) => {
    if (!editTarget) return;
    setSubmitting(true);
    try {
      await permissionGroupsApi.update(editTarget.id, {
        display_name: values.display_name,
        description: values.description,
      });
      message.success("已更新权限组");
      setEditTarget(null);
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const onDisable = async (id: number) => {
    try {
      await permissionGroupsApi.disable(id);
      message.success("已停用权限组");
      load();
      if (detailGroup?.group.id === id) {
        openDetail(id);
      }
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const openDetail = async (id: number) => {
    setDetailLoading(true);
    try {
      const d = await permissionGroupsApi.get(id);
      setDetailGroup(d);
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setDetailLoading(false);
    }
  };

  const handleAddMember = async () => {
    if (!detailGroup || !addMemberId) return;
    try {
      await permissionGroupsApi.addMember(detailGroup.group.id, addMemberId);
      message.success("已添加成员");
      setAddMemberId(undefined);
      openDetail(detailGroup.group.id);
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const handleRemoveMember = async (userId: number) => {
    if (!detailGroup) return;
    try {
      await permissionGroupsApi.removeMember(detailGroup.group.id, userId);
      message.success("已移除成员");
      openDetail(detailGroup.group.id);
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const handleAddService = async () => {
    if (!detailGroup || !addServerId) return;
    try {
      await permissionGroupsApi.addService(detailGroup.group.id, addServerId);
      message.success("已添加服务授权");
      setAddServerId(undefined);
      openDetail(detailGroup.group.id);
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const handleRemoveService = async (serverId: number) => {
    if (!detailGroup) return;
    try {
      await permissionGroupsApi.removeService(detailGroup.group.id, serverId);
      message.success("已移除服务授权");
      openDetail(detailGroup.group.id);
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const getUserLabel = (userId: number) => {
    const u = users.find((u) => u.id === userId);
    return u ? u.username : `用户 #${userId}`;
  };

  const getServerLabel = (serverId: number) => {
    const s = servers.find((s) => s.id === serverId);
    return s ? s.name : `服务 #${serverId}`;
  };

  const columns: ColumnsType<PermissionGroup> = [
    {
      title: "名称",
      dataIndex: "name",
      render: (name: string, row) => (
        <a
          onClick={() => openDetail(row.id)}
          style={{ cursor: "pointer" }}
        >
          {row.display_name || name}
        </a>
      ),
    },
    { title: "标识", dataIndex: "name", render: (n: string) => <Tag>{n}</Tag> },
    {
      title: "状态",
      dataIndex: "status",
      render: (s: string) => (
        <Tag color={s === "active" ? "green" : "red"}>{s === "active" ? "启用" : "停用"}</Tag>
      ),
    },
    { title: "说明", dataIndex: "description", ellipsis: true },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <Space>
          <Button
            size="small"
            onClick={() => {
              setEditTarget(row);
              editForm.setFieldsValue({
                display_name: row.display_name,
                description: row.description,
              });
            }}
          >
            编辑
          </Button>
          <Button size="small" onClick={() => openDetail(row.id)}>
            详情
          </Button>
          {row.status === "active" && (
            <Popconfirm
              title="确认停用该权限组？停用后其服务授权将不再生效。"
              onConfirm={() => onDisable(row.id)}
            >
              <Button size="small" danger>
                停用
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  // Members that are not yet in the group
  const memberIds = new Set(
    detailGroup?.members?.map((m) => m.user_id) ?? [],
  );
  const availableUsers = users.filter((u) => u.id && !memberIds.has(u.id));

  // Servers not yet authorized for this group
  const serviceIds = new Set(
    detailGroup?.services?.map((s) => s.mcp_server_id) ?? [],
  );
  const availableServers = servers.filter(
    (s) => s.id && !serviceIds.has(s.id),
  );

  return (
    <>
      <Card
        title="权限组"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            新建权限组
          </Button>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          权限组用于将用户分组并统一授权 MCP 服务。用户通过所属权限组获得对指定服务的访问权限。
        </Typography.Paragraph>
        <Table
          rowKey="id"
          dataSource={groups}
          columns={columns}
          loading={loading}
          pagination={{ pageSize: 20 }}
        />
      </Card>

      {/* Create modal */}
      <Modal
        title="新建权限组"
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => createForm.submit()}
        confirmLoading={submitting}
        okText="创建"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={createForm} layout="vertical" onFinish={onCreate}>
          <Form.Item
            name="name"
            label="标识（英文）"
            rules={[{ required: true, message: "请输入权限组标识" }]}
            tooltip="用于系统内部引用，如 dev-team"
          >
            <Input placeholder="例如 dev-team" />
          </Form.Item>
          <Form.Item name="display_name" label="显示名称">
            <Input placeholder="例如 开发团队" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} placeholder="权限组用途描述" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Edit modal */}
      <Modal
        title={`编辑权限组：${editTarget?.name}`}
        open={!!editTarget}
        onCancel={() => setEditTarget(null)}
        onOk={() => editForm.submit()}
        confirmLoading={submitting}
        okText="保存"
        cancelText="取消"
        destroyOnClose
      >
        <Form form={editForm} layout="vertical" onFinish={onEdit}>
          <Form.Item name="display_name" label="显示名称">
            <Input />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Detail drawer */}
      <Drawer
        title={`权限组详情：${detailGroup?.group?.display_name || detailGroup?.group?.name || ""}`}
        open={!!detailGroup}
        onClose={() => setDetailGroup(null)}
        width={640}
        loading={detailLoading}
      >
        {detailGroup && (
          <>
            <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="标识">
                <Tag>{detailGroup.group.name}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="显示名称">
                {detailGroup.group.display_name || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="说明">
                {detailGroup.group.description || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag
                  color={detailGroup.group.status === "active" ? "green" : "red"}
                >
                  {detailGroup.group.status === "active" ? "启用" : "停用"}
                </Tag>
              </Descriptions.Item>
            </Descriptions>

            {/* Members section */}
            <Card
              size="small"
              title={
                <>
                  <TeamOutlined /> 成员
                </>
              }
              style={{ marginBottom: 16 }}
              extra={
                <Space>
                  <Select
                    style={{ width: 200 }}
                    placeholder="选择用户"
                    allowClear
                    value={addMemberId}
                    onChange={setAddMemberId}
                    options={availableUsers.map((u) => ({
                      value: u.id,
                      label: u.username,
                    }))}
                    showSearch
                    optionFilterProp="label"
                  />
                  <Button
                    size="small"
                    type="primary"
                    disabled={!addMemberId}
                    onClick={handleAddMember}
                  >
                    添加
                  </Button>
                </Space>
              }
            >
              <List
                size="small"
                dataSource={detailGroup.members ?? []}
                locale={{ emptyText: "暂无成员" }}
                renderItem={(m: PermissionGroupMember) => (
                  <List.Item
                    actions={[
                      <Popconfirm
                        key="rm"
                        title="确认移除该成员？"
                        onConfirm={() => handleRemoveMember(m.user_id)}
                      >
                        <Button size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>,
                    ]}
                  >
                    <List.Item.Meta
                      title={getUserLabel(m.user_id)}
                      description={`用户ID: ${m.user_id}`}
                    />
                  </List.Item>
                )}
              />
            </Card>

            {/* Services section */}
            <Card
              size="small"
              title={
                <>
                  <CloudServerOutlined /> 授权服务
                </>
              }
              extra={
                <Space>
                  <Select
                    style={{ width: 200 }}
                    placeholder="选择服务"
                    allowClear
                    value={addServerId}
                    onChange={setAddServerId}
                    options={availableServers.map((s) => ({
                      value: s.id,
                      label: s.name,
                    }))}
                    showSearch
                    optionFilterProp="label"
                  />
                  <Button
                    size="small"
                    type="primary"
                    disabled={!addServerId}
                    onClick={handleAddService}
                  >
                    添加
                  </Button>
                </Space>
              }
            >
              <List
                size="small"
                dataSource={detailGroup.services ?? []}
                locale={{ emptyText: "暂无授权服务" }}
                renderItem={(sv: PermissionGroupService) => (
                  <List.Item
                    actions={[
                      <Popconfirm
                        key="rm"
                        title="确认移除该服务授权？"
                        onConfirm={() => handleRemoveService(sv.mcp_server_id)}
                      >
                        <Button size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>,
                    ]}
                  >
                    <List.Item.Meta
                      title={getServerLabel(sv.mcp_server_id)}
                      description={`服务ID: ${sv.mcp_server_id}`}
                    />
                  </List.Item>
                )}
              />
            </Card>
          </>
        )}
      </Drawer>
    </>
  );
}
