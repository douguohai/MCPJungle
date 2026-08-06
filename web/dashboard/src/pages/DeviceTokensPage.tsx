import { useEffect, useState } from "react";
import {
  Table,
  Card,
  Alert,
  Spin,
  Button,
  Popconfirm,
  Modal,
  Form,
  Input,
  Select,
  Typography,
  Tag,
  message,
} from "antd";
import { PlusOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import {
  deviceTokensApi,
  type DeviceToken,
} from "../api/device_tokens";
import { extractError } from "../api/client";

export default function DeviceTokensPage() {
  const [data, setData] = useState<DeviceToken[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [createdToken, setCreatedToken] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [createForm] = Form.useForm();

  const load = (signal?: AbortSignal) => {
    setLoading(true);
    deviceTokensApi
      .list(signal)
      .then((ts) => {
        setData(ts);
        setError("");
      })
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
  };
  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, []);

  if (loading && !data.length) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  const onCreate = async (values: {
    name: string;
    scope_mode: string;
    restricted_server_names?: string;
  }) => {
    setSubmitting(true);
    try {
      const restricted =
        values.scope_mode === "restricted" && values.restricted_server_names
          ? values.restricted_server_names
              .split(",")
              .map((s) => s.trim())
              .filter(Boolean)
          : [];
      const res = await deviceTokensApi.create({
        name: values.name.trim(),
        scope_mode: values.scope_mode || "inherit_all",
        restricted_server_names: restricted,
      });
      message.success(`已创建设备令牌 ${res.token.name}`);
      setCreatedToken(res.raw_token);
      setCreateOpen(false);
      createForm.resetFields();
      // Persist raw token so MyServicesPage can use it for config generation.
      try {
        sessionStorage.setItem(`device_token_raw_${res.token.id}`, res.raw_token);
      } catch { /* quota exceeded, ignore */ }
      load();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setSubmitting(false);
    }
  };

  const revoke = async (id: number) => {
    try {
      await deviceTokensApi.revoke(id);
      message.success("已撤销令牌");
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const remove = async (id: number) => {
    try {
      await deviceTokensApi.remove(id);
      message.success("已删除令牌");
      load();
    } catch (e) {
      message.error(extractError(e));
    }
  };

  const columns: ColumnsType<DeviceToken> = [
    { title: "名称", dataIndex: "name" },
    {
      title: "范围模式",
      dataIndex: "scope_mode",
      render: (mode: string) =>
        mode === "inherit_all" ? "继承全部权限" : "限定服务范围",
    },
    {
      title: "状态",
      dataIndex: "status",
      render: (status: string) => (
        <Tag
          color={
            status === "active"
              ? "green"
              : status === "revoked"
                ? "red"
                : "default"
          }
        >
          {status}
        </Tag>
      ),
    },
    { title: "令牌前缀", dataIndex: "token_prefix" },
    { title: "创建时间", dataIndex: "created_at" },
    {
      title: "操作",
      key: "actions",
      render: (_, row) => (
        <>
          {row.status === "active" && (
            <Popconfirm
              title="确认撤销该令牌？撤销后立即失效。"
              onConfirm={() => revoke(row.id)}
            >
              <Button size="small" danger>
                撤销
              </Button>
            </Popconfirm>
          )}{" "}
          <Popconfirm
            title="确认删除该令牌？"
            onConfirm={() => remove(row.id)}
          >
            <Button size="small" danger>
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
        title="设备令牌"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            新建令牌
          </Button>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          设备令牌用于 MCP 客户端（如 Cursor、Claude
          Desktop）认证。每个令牌只在创建时显示完整内容，请立即复制保存。继承全部权限的令牌自动获得你权限组授权的所有服务；限定范围的令牌只能访问你指定的服务。
        </Typography.Paragraph>
        <Table
          rowKey="id"
          dataSource={data}
          columns={columns}
          loading={loading}
          pagination={{ pageSize: 20 }}
        />
      </Card>

      <Modal
        title="新建设备令牌"
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
          onFinish={onCreate}
        >
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: "请输入令牌名称" }]}
            tooltip="如 Cursor-办公电脑"
          >
            <Input placeholder="例如 Cursor-办公电脑" />
          </Form.Item>
          <Form.Item
            name="scope_mode"
            label="范围模式"
            initialValue="inherit_all"
          >
            <Select
              options={[
                { value: "inherit_all", label: "继承全部权限（推荐）" },
                { value: "restricted", label: "限定服务范围" },
              ]}
            />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prev, cur) => prev.scope_mode !== cur.scope_mode}
          >
            {({ getFieldValue }) =>
              getFieldValue("scope_mode") === "restricted" && (
                <Form.Item
                  name="restricted_server_names"
                  label="限定的服务名称（逗号分隔）"
                  rules={[
                    {
                      required: true,
                      message: "请填写至少一个服务名称",
                    },
                  ]}
                  tooltip="输入 MCP 服务名称，多个用逗号分隔。必须是已存在的服务。"
                >
                  <Input placeholder="例如 notion,github" />
                </Form.Item>
              )
            }
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="设备令牌（仅显示这一次）"
        open={!!createdToken}
        onCancel={() => setCreatedToken(null)}
        footer={
          <Button type="primary" onClick={() => setCreatedToken(null)}>
            我已保存
          </Button>
        }
      >
        <Alert
          type="warning"
          showIcon
          message="此令牌只显示一次，关闭后无法再查看。请立即复制保存。"
          style={{ marginBottom: 12 }}
        />
        <Typography.Paragraph code copyable>
          {createdToken}
        </Typography.Paragraph>
      </Modal>
    </>
  );
}
