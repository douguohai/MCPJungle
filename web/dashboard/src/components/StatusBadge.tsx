import { Badge, Tag } from "antd";

const map: Record<string, { status: "success" | "processing" | "error" | "default"; text: string }> = {
  connected: { status: "success", text: "已连接" },
  reachable: { status: "processing", text: "可达" },
  failed: { status: "error", text: "失败" },
  unknown: { status: "default", text: "未知" },
};

export default function StatusBadge({ status }: { status?: string }) {
  if (!status) return <Tag>未知</Tag>;
  const m = map[status] ?? map.unknown;
  return <Badge status={m.status} text={m.text} />;
}
