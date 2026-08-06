import { useEffect, useState } from "react";
import {
  Card,
  Alert,
  Spin,
  Typography,
  Tag,
  Row,
  Col,
  Select,
  Button,
  Tooltip,
  message,
} from "antd";
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  DesktopOutlined,
} from "@ant-design/icons";
import { serversApi } from "../api/servers";
import { deviceTokensApi, type DeviceToken } from "../api/device_tokens";
import { extractError } from "../api/client";
import type { DashboardServer } from "../types";

export default function MyServicesPage() {
  const [servers, setServers] = useState<DashboardServer[]>([]);
  const [tokens, setTokens] = useState<DeviceToken[]>([]);
  const [selectedToken, setSelectedToken] = useState<string | null>(null);
  const [selectedRawToken, setSelectedRawToken] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([serversApi.list(controller.signal), deviceTokensApi.list(controller.signal)])
      .then(([s, t]) => {
        setServers(
          (s.servers ?? []).filter(
            (srv) =>
              srv.enabled &&
              (srv.status === "connected" || srv.status === "reachable"),
          ),
        );
        setTokens(t.filter((tk) => tk.status === "active"));
        setError("");
      })
      .catch((e) => setError(extractError(e)))
      .finally(() => setLoading(false));
    return () => controller.abort();
  }, []);

  if (loading) return <Spin />;
  if (error) return <Alert type="error" message={error} />;

  const copyConfig = (config: string) => {
    navigator.clipboard.writeText(config).then(() => {
      message.success("已复制客户端配置");
    });
  };

  const generateConfig = (serverName: string) => {
    if (!selectedToken) return "";
    const tokenObj = tokens.find((t) => t.token_prefix?.startsWith(selectedToken) || String(t.id) === selectedToken);
    // Retrieve raw token from sessionStorage (stored at creation time in DeviceTokensPage).
    const tokenId = tokenObj ? String(tokenObj.id) : selectedToken;
    const rawToken = selectedRawToken || sessionStorage.getItem(`device_token_raw_${tokenId}`) || "<your-device-token>";
    // Generate a Cursor / Claude Desktop compatible MCP configuration
    return JSON.stringify(
      {
        mcpServers: {
          mcpjungle: {
            url: `${window.location.origin}/mcp`,
            headers: {
              Authorization: `Bearer ${rawToken}`,
            },
          },
        },
      },
      null,
      2,
    );
  };

  return (
    <>
      <Typography.Title level={4} style={{ marginBottom: 4 }}>
        我的服务
      </Typography.Title>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
        这些是你有权访问且当前在线的 MCP 服务。选择一个设备令牌后可生成客户端配置片段，用于
        Cursor、Claude Desktop 等 AI 客户端。
      </Typography.Paragraph>

      {tokens.length > 0 && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Row align="middle" gutter={16}>
            <Col>
              <Typography.Text strong>
                <DesktopOutlined /> 选择设备令牌以生成配置：
              </Typography.Text>
            </Col>
            <Col flex="auto">
              <Select
                style={{ width: "100%" }}
                placeholder="选择一个活跃的设备令牌"
                allowClear
                value={selectedToken}
                onChange={(val) => {
                  setSelectedToken(val);
                  if (val) {
                    const tk = tokens.find((t) => String(t.id) === val);
                    const id = tk ? String(tk.id) : val;
                    setSelectedRawToken(sessionStorage.getItem(`device_token_raw_${id}`));
                  } else {
                    setSelectedRawToken(null);
                  }
                }}
                options={tokens.map((t) => ({
                  value: String(t.id),
                  label: `${t.name} (${t.token_prefix}...)`,
                }))}
              />
            </Col>
          </Row>
        </Card>
      )}

      {servers.length === 0 ? (
        <Alert
          type="info"
          showIcon
          message="暂无可用服务"
          description="当前没有在线且启用的 MCP 服务。请联系管理员配置权限组或注册新的服务。"
        />
      ) : (
        <Row gutter={[16, 16]}>
          {servers.map((srv) => (
            <Col key={srv.name} xs={24} sm={12} lg={8}>
              <Card
                title={srv.name}
                extra={
                  srv.status === "connected" ? (
                    <Tag icon={<CheckCircleOutlined />} color="success">
                      在线
                    </Tag>
                  ) : (
                    <Tag icon={<CheckCircleOutlined />} color="processing">
                      可达
                    </Tag>
                  )
                }
                actions={
                  selectedToken
                    ? [
                        <Tooltip title="复制客户端配置" key="copy">
                          <Button
                            type="link"
                            icon={<CopyOutlined />}
                            onClick={() =>
                              copyConfig(generateConfig(srv.name))
                            }
                          >
                            复制配置
                          </Button>
                        </Tooltip>,
                      ]
                    : undefined
                }
              >
                {srv.config_summary?.description && (
                  <Typography.Paragraph
                    ellipsis={{ rows: 2 }}
                    type="secondary"
                  >
                    {srv.config_summary.description}
                  </Typography.Paragraph>
                )}
                <Typography.Text type="secondary">
                  工具数：{srv.tool_count}
                </Typography.Text>
                <br />
                <Typography.Text type="secondary">
                  传输方式：{srv.transport}
                </Typography.Text>
                {srv.connection_summary && (
                  <>
                    <br />
                    <Tooltip title={srv.config_summary?.sanitized_summary}>
                      <Typography.Text
                        type="secondary"
                        ellipsis
                        style={{ maxWidth: 240 }}
                      >
                        {srv.connection_summary}
                      </Typography.Text>
                    </Tooltip>
                  </>
                )}
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {selectedToken && servers.length > 0 && (
        <Card
          title="客户端配置预览"
          style={{ marginTop: 16 }}
          extra={
            <Button
              icon={<CopyOutlined />}
              onClick={() => copyConfig(generateConfig(""))}
            >
              复制
            </Button>
          }
        >
          <Typography.Paragraph type="secondary">
            将以下配置粘贴到你的 AI 客户端（如 Cursor Settings、Claude Desktop config）的
            MCP servers 配置中。
            {selectedRawToken
              ? " 令牌已自动填入。"
              : (<>请将{" "}
                <Typography.Text code>{`<your-device-token>`}</Typography.Text>{" "}
                替换为实际的设备令牌（仅创建时可见）。</>)}
          </Typography.Paragraph>
          <Typography.Paragraph
            code
            copyable
            style={{ whiteSpace: "pre-wrap" }}
          >
            {generateConfig("")}
          </Typography.Paragraph>
        </Card>
      )}
    </>
  );
}
