import { useEffect, useState } from "react";
import { Modal, Form, Input, Select, message } from "antd";
import { toolGroupsApi } from "../api/toolGroups";
import { extractError } from "../api/client";
import type { DashboardTool } from "../types";

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
  // available tools, used to populate the multi-select options
  tools: DashboardTool[];
}

export default function ToolGroupForm({ open, onClose, onCreated, tools }: Props) {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open) form.resetFields();
  }, [open, form]);

  const options = tools.map((t) => ({ label: t.canonical_name, value: t.canonical_name }));

  const onSubmit = async () => {
    let values: Record<string, unknown>;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    setLoading(true);
    try {
      await toolGroupsApi.create({
        name: values.name as string,
        description: (values.description as string) || undefined,
        tools: (values.tools as string[]) ?? [],
      });
      message.success(`已创建工具组 ${values.name as string}`);
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
      title="添加工具组"
      open={open}
      onCancel={onClose}
      onOk={onSubmit}
      confirmLoading={loading}
      okText="创建"
      cancelText="取消"
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item name="name" label="组名" rules={[{ required: true, message: "请输入组名" }]}>
          <Input />
        </Form.Item>
        <Form.Item name="description" label="描述">
          <Input />
        </Form.Item>
        <Form.Item name="tools" label="包含工具">
          <Select
            mode="multiple"
            showSearch
            options={options}
            placeholder="选择要包含的工具"
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
