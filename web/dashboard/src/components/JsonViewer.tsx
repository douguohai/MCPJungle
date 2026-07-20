import { Typography } from "antd";

// Pretty-prints any JSON-able value (string, object, array) in a scrollable block.
export default function JsonViewer({ value }: { value: unknown }) {
  let text: string;
  try {
    text =
      typeof value === "string"
        ? JSON.stringify(JSON.parse(value), null, 2)
        : JSON.stringify(value, null, 2);
  } catch {
    text = String(value ?? "");
  }
  return (
    <Typography.Paragraph
      style={{
        background: "#f5f5f5",
        padding: 12,
        borderRadius: 6,
        margin: 0,
        maxHeight: 400,
        overflow: "auto",
      }}
    >
      <pre style={{ margin: 0 }}>{text}</pre>
    </Typography.Paragraph>
  );
}
