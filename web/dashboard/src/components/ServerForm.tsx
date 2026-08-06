import { useEffect, useState } from "react";
import { Modal, Form, Input, Select, Button, Space, message, Typography } from "antd";
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

const TRANSPORT_HINT: Record<McpTransport, string> = {
  streamable_http: "远程 HTTP 服务，最常用。适用于在线/托管的 MCP 服务。",
  stdio: "本地命令行程序，例如用 npx / node / python 启动的工具。",
  sse: "基于 SSE 的旧版 HTTP 协议，仅用于兼容老版本服务。",
};

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
        <Form.Item
          name="name"
          label="名称"
          tooltip="服务器的唯一标识（注册后不可更改）。AI 客户端通过它引用此服务。"
          rules={[{ required: true, message: "请输入服务器名称" }]}
        >
          <Input placeholder="例如 context7、filesystem" />
        </Form.Item>
        <Form.Item
          name="transport"
          label="传输协议"
          tooltip="决定 MCPJungle 如何连接该服务。不确定时选 Streamable HTTP。"
          rules={[{ required: true }]}
        >
          <Select
            onChange={(v) => setTransport(v)}
            options={[
              { value: "streamable_http", label: "Streamable HTTP（远程 HTTP 服务，推荐）" },
              { value: "stdio", label: "STDIO（本地命令行程序）" },
              { value: "sse", label: "SSE（旧版 HTTP，兼容用）" },
            ]}
          />
        </Form.Item>
        <Typography.Paragraph type="secondary" style={{ marginTop: -8, marginBottom: 16, fontSize: 12 }}>
          {TRANSPORT_HINT[transport]}
        </Typography.Paragraph>

        <Form.Item name="description" label="描述">
          <Input placeholder="这个服务提供什么能力（可选）" />
        </Form.Item>

        {(transport === "streamable_http" || transport === "sse") && (
          <>
            <Form.Item
              name="url"
              label="URL"
              tooltip="MCP 服务的完整 HTTP 端点地址。"
              rules={[{ required: true, message: "请输入 URL" }]}
            >
              <Input placeholder="https://example.com/mcp" />
            </Form.Item>
            <Form.Item
              name="bearer_token"
              label="Bearer Token（可选）"
              tooltip="若上游需要鉴权，填这里；会作为 Authorization: Bearer 请求头发送。"
            >
              <Input.Password placeholder="留空表示无需鉴权" />
            </Form.Item>
          </>
        )}

        {transport === "streamable_http" && (
          <>
            <Form.Item label="自定义 Headers（可选）" tooltip="需要额外 HTTP 头时填写，例如自定义鉴权头。">
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
            </Form.Item>
          </>
        )}

        {transport === "stdio" && (
          <>
            <Form.Item
              name="command"
              label="命令"
              tooltip="启动本地 MCP 程序的命令。"
              rules={[{ required: true, message: "请输入命令" }]}
            >
              <Input placeholder="例如 npx、node、python" />
            </Form.Item>
            <Form.Item
              name="args"
              label="参数（每行一个）"
              tooltip="传给命令的参数，每行一个。"
            >
              <Input.TextArea autoSize={{ minRows: 2 }} placeholder={"例如：\n-y\n@modelcontextprotocol/server-filesystem\n/tmp"} />
            </Form.Item>
            <Form.Item
              name="env"
              label="环境变量（JSON 对象）"
              tooltip="传给程序的环境变量，必须是合法的 JSON 对象。"
            >
              <Input.TextArea autoSize={{ minRows: 2 }} placeholder='{"API_KEY":"xxx"}' />
            </Form.Item>
          </>
        )}

        <Form.Item
          name="session_mode"
          label="会话模式"
          tooltip="无状态：每次工具调用新建连接（默认，推荐）。有状态：保持长连接，少数服务要求此模式。"
        >
          <Select
            options={[
              { value: "stateless", label: "无状态（默认，推荐）" },
              { value: "stateful", label: "有状态" },
            ]}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
