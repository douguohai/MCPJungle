import { useState } from "react";
import { Button, message, Tooltip } from "antd";
import { CopyOutlined } from "@ant-design/icons";

export default function CopyButton({ text, label }: { text: string; label?: string }) {
  const [copied, setCopied] = useState(false);

  const onCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      message.success("已复制");
      setTimeout(() => setCopied(false), 1500);
    } catch {
      message.error("复制失败");
    }
  };

  return (
    <Tooltip title={copied ? "已复制" : text}>
      <Button size="small" icon={<CopyOutlined />} onClick={onCopy}>
        {label ?? "复制"}
      </Button>
    </Tooltip>
  );
}
