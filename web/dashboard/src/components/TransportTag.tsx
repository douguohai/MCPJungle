import { Tag } from "antd";

const colors: Record<string, string> = {
  streamable_http: "blue",
  stdio: "purple",
  sse: "cyan",
};

const labels: Record<string, string> = {
  streamable_http: "Streamable HTTP",
  stdio: "STDIO",
  sse: "SSE",
};

export default function TransportTag({ transport }: { transport: string }) {
  return <Tag color={colors[transport] ?? "default"}>{labels[transport] ?? transport}</Tag>;
}
