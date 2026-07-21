import { Alert, Card, Col, Divider, Row, Space, Steps, Typography } from "antd";
import { Link } from "react-router-dom";
import CopyButton from "../components/CopyButton";
import { useDashboardSettings } from "../hooks/useDashboardSettings";

const { Paragraph, Text, Title } = Typography;

const cursorConfig = `{
  "mcpServers": {
    "internal-mcphub": {
      "url": "http://127.0.0.1:8080/mcp",
      "headers": {
        "Authorization": "Bearer <DEVICE_TOKEN>"
      }
    }
  }
}`;

const claudeConfig = `{
  "mcpServers": {
    "internal-mcphub": {
      "command": "npx",
      "args": [
        "-y",
        "mcp-remote",
        "http://127.0.0.1:8080/mcp",
        "--header",
        "Authorization: Bearer <DEVICE_TOKEN>"
      ]
    }
  }
}`;

const groupEndpoint = "http://127.0.0.1:8080/v0/groups/<group-name>/mcp";

export default function TutorialPage() {
  const settings = useDashboardSettings();

  return (
    <Space direction="vertical" size={16} style={{ width: "100%" }}>
      <Card>
        <Title level={3} style={{ marginTop: 0 }}>
          外部如何使用内部 MCP
        </Title>
        <Paragraph type="secondary">
          外部客户端只需要 {settings.system_display_name} 地址和一个设备令牌。工具、提示模板、数据资源由客户端通过 MCP 协议自动发现；管理员在控制台内负责服务注册、权限组授权、能力组合和调用观察。
        </Paragraph>
        <Alert
          showIcon
          type="info"
          message="不要把 dashboard 登录密码或会话 Token 配到外部客户端。外部客户端应使用「我的设备令牌」里创建的设备令牌。"
        />
      </Card>

      <Card title="交付流程">
        <Steps
          direction="vertical"
          current={-1}
          items={[
            {
              title: "管理员注册 MCP 服务",
              description: (
                <>
                  在 <Link to="/servers">MCP 服务</Link> 添加 HTTP 或本地 STDIO 服务，确认状态为 connected/reachable。
                </>
              ),
            },
            {
              title: "管理员分配访问范围",
              description: (
                <>
                  在 <Link to="/permission-groups">权限组</Link> 把成员和可访问的 MCP 服务绑定起来。普通成员只能看到自己被授权的能力。
                </>
              ),
            },
            {
              title: "使用者创建设备令牌",
              description: (
                <>
                  在 <Link to="/device-tokens">我的设备令牌</Link> 为 Cursor、Claude Desktop、Codex 或自研客户端创建独立令牌。
                </>
              ),
            },
            {
              title: "按场景复用工具",
              description: (
                <>
                  如果某个客户端只需要一组固定工具，管理员在 <Link to="/settings/ability-combinations">能力组合</Link> 创建专用端点。
                </>
              ),
            },
          ]}
        />
      </Card>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title="Cursor / 支持 Streamable HTTP 的客户端"
            extra={<CopyButton text={cursorConfig} label="复制配置" />}
          >
            <Paragraph>
              使用全局入口 <Text code>http://127.0.0.1:8080/mcp</Text>，请求头带设备令牌。
            </Paragraph>
            <pre style={{ whiteSpace: "pre-wrap", margin: 0 }}>{cursorConfig}</pre>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Claude Desktop" extra={<CopyButton text={claudeConfig} label="复制配置" />}>
            <Paragraph>
              Claude Desktop 通常通过 <Text code>mcp-remote</Text> 连接 HTTP MCP 服务。
            </Paragraph>
            <pre style={{ whiteSpace: "pre-wrap", margin: 0 }}>{claudeConfig}</pre>
          </Card>
        </Col>
      </Row>

      <Card title="工具、提示模板、数据资源怎么用">
        <Row gutter={[16, 16]}>
          <Col xs={24} md={8}>
            <Title level={5}>工具 Tools</Title>
            <Paragraph type="secondary">
              工具是可执行函数。客户端连接后通过 <Text code>tools/list</Text> 发现工具，再用{" "}
              <Text code>tools/call</Text> 按输入 schema 调用。适合封装查询、写入、计算、第三方系统操作。
            </Paragraph>
            <Paragraph>
              在服务详情页可查看输入 schema、复制工具名，并把工具加入能力组合。
            </Paragraph>
          </Col>
          <Col xs={24} md={8}>
            <Title level={5}>提示模板 Prompts</Title>
            <Paragraph type="secondary">
              提示模板是可复用 Prompt。客户端通过 <Text code>prompts/list</Text> 发现模板，再用{" "}
              <Text code>prompts/get</Text> 带参数取回内容。适合团队沉淀固定分析流程、审查模板、报告模板。
            </Paragraph>
            <Paragraph>在服务详情页可查看参数定义并复制模板名。</Paragraph>
          </Col>
          <Col xs={24} md={8}>
            <Title level={5}>数据资源 Resources</Title>
            <Paragraph type="secondary">
              资源是可读取上下文。客户端通过 <Text code>resources/list</Text> 发现 URI，再用{" "}
              <Text code>resources/read</Text> 读取内容。适合暴露配置、知识库片段、文件或业务上下文。
            </Paragraph>
            <Paragraph>在服务详情页可复制资源 URI。</Paragraph>
          </Col>
        </Row>
        <Divider />
        <Alert
          showIcon
          type="warning"
          message="能力组合当前只复用工具"
          description={
            <>
              能力组合端点 <Text code>{groupEndpoint}</Text> 只暴露选中的工具。提示模板和数据资源仍从全局{" "}
              <Text code>/mcp</Text> 入口读取，并由用户、设备令牌和权限组控制可见范围。
            </>
          }
        />
      </Card>
    </Space>
  );
}
