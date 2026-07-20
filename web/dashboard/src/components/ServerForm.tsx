import { useEffect, useState } from "react";
import { Modal, Form, Input, Select, Button, Space, message } from "antd";
import { serversApi } from "../api/servers";
import { extractError } from "../api/client";
import type { DashboardRegisterServerInput, McpTransport } from "../types";

interface HeaderPair {
  key?: string;
  value?: string;
}

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}

// Modal-based form to register an MCP server. Fields change with the selected
// transport. Form.List entries and textarea inputs are converted into the
// structured RegisterServerInput expected by the backend before submission.
export default function ServerForm({ open, onClose, onCreated }: Props) {
  const [form] = Form.useForm();
  const [transport, setTransport] = useState<McpTransport>("streamable_http");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) {
      form.resetFields();
      setTransport("streamable_http");
    }
  }, [open, form]);

  const onSubmit = async () => {
    let values: Record<string, unknown>;
    try {
      values = await form.validateFields();
    } catch {
      return; // validation errors are shown inline by the Form
    }

    const body: DashboardRegisterServerInput = {
      name: values.name as string,
      transport: values.transport as McpTransport,
      description: (values.description as string) || undefined,
      session_mode: (values.session_mode as "stateless" | "stateful") || undefined,
    };

    if (body.transport === "streamable_http" || body.transport === "sse") {
      body.url = values.url as string;
      if (values.bearer_token) body.bearer_token = values.bearer_token as string;
    }
    if (body.transport === "streamable_http") {
      const headers: Record<string, string> = {};
      (values.headersList as HeaderPair[] | undefined)?.forEach((h) => {
        if (h?.key) headers[h.key] = h.value ?? "";
      });
      if (Object.keys(headers).length) body.headers = headers;
    }
    if (body.transport === "stdio") {
      body.command = values.command as string;
      const argsText = (values.args as string) || "";
      body.args = argsText
        .split(/\r?\n/)
        .map((s) => s.trim())
        .filter(Boolean);
      const envText = (values.env as string) || "";
      if (envText.trim()) {
        try {
          body.env = JSON.parse(envText);
        } catch {
          message.error("环境变量不是合法的 JSON 对象");
          return;
        }
      }
    }

    setLoading(true);
    try {
      const res = await serversApi.create(body);
      if (res.authorization_required) {
        message.warning(
          "该上游需要 OAuth 授权，请通过 CLI 完成注册流程。",
        );
      } else {
        message.success(`已添加服务器 ${res.name ?? body.name}`);
      }
      onCreated();
      onClose();
    } catch (e) {
      message.error(extractError(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="添加 MCP 服务器"
      open={open}
      onCancel={onClose}
      onOk={onSubmit}
      confirmLoading={loading}
      okText="添加"
      cancelText="取消"
      destroyOnClose
      width={560}
    >
      <Form
        form={form}
        layout="vertical"
        initialValues={{ transport: "streamable_http", session_mode: "stateless" }}
      >
        <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入服务器名称" }]}>
          <Input placeholder="例如 context7" />
        </Form.Item>
        <Form.Item name="transport" label="传输协议" rules={[{ required: true }]}>
          <Select
            onChange={(v) => setTransport(v)}
            options={[
              { value: "streamable_http", label: "Streamable HTTP" },
              { value: "stdio", label: "STDIO" },
              { value: "sse", label: "SSE" },
            ]}
          />
        </Form.Item>
        <Form.Item name="description" label="描述">
          <Input />
        </Form.Item>

        {(transport === "streamable_http" || transport === "sse") && (
          <>
            <Form.Item name="url" label="URL" rules={[{ required: true, message: "请输入 URL" }]}>
              <Input placeholder="https://example.com/mcp" />
            </Form.Item>
            <Form.Item name="bearer_token" label="Bearer Token（可选）">
              <Input.Password />
            </Form.Item>
          </>
        )}

        {transport === "streamable_http" && (
          <>
            <div style={{ marginBottom: 8 }}>自定义 Headers</div>
            <Form.List name="headersList">
              {(fields, { add, remove }) => (
                <>
                  {fields.map((f) => (
                    <Space key={f.key} align="baseline" style={{ display: "flex", marginBottom: 8 }}>
                      <Form.Item name={[f.name, "key"]} noStyle>
                        <Input placeholder="Header 名" />
                      </Form.Item>
                      <Form.Item name={[f.name, "value"]} noStyle>
                        <Input placeholder="Header 值" />
                      </Form.Item>
                      <Button onClick={() => remove(f.name)}>删除</Button>
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()}>
                    添加 Header
                  </Button>
                </>
              )}
            </Form.List>
          </>
        )}

        {transport === "stdio" && (
          <>
            <Form.Item name="command" label="命令" rules={[{ required: true, message: "请输入命令" }]}>
              <Input placeholder="npx" />
            </Form.Item>
            <Form.Item name="args" label="参数（每行一个）">
              <Input.TextArea autoSize={{ minRows: 2 }} />
            </Form.Item>
            <Form.Item name="env" label="环境变量（JSON 对象）">
              <Input.TextArea autoSize={{ minRows: 2 }} placeholder='{"KEY":"value"}' />
            </Form.Item>
          </>
        )}

        <Form.Item name="session_mode" label="会话模式">
          <Select
            options={[
              { value: "stateless", label: "无状态（默认）" },
              { value: "stateful", label: "有状态" },
            ]}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
