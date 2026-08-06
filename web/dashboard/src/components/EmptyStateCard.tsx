import { Card, Empty, Typography, Collapse } from "antd";
import type { ReactNode } from "react";
import type { DashboardEmptyState } from "../types";

// EmptyStateCard renders the backend-provided empty state. The primary call to
// action should be a UI button (the `action` prop); the backend's CLI commands
// are demoted into a collapsible "advanced" section so new users aren't pushed
// to the command line when a UI path exists.
export default function EmptyStateCard({
  state,
  action,
}: {
  state?: DashboardEmptyState;
  action?: ReactNode;
}) {
  if (!state) return null;
  return (
    <Card>
      <Empty
        description={
          <>
            <Typography.Text strong>{state.title}</Typography.Text>
            <br />
            <Typography.Text type="secondary">{state.description}</Typography.Text>
          </>
        }
      >
        {action && <div style={{ marginTop: 16 }}>{action}</div>}
        {state.commands && state.commands.length > 0 && (
          <Collapse
            style={{ marginTop: 16, textAlign: "left" }}
            items={[
              {
                key: "cli",
                label: "高级：改用命令行（CLI）",
                children: state.commands.map((c) => (
                  <Typography.Paragraph key={c} code copyable style={{ marginBottom: 4 }}>
                    {c}
                  </Typography.Paragraph>
                )),
              },
            ]}
          />
        )}
      </Empty>
    </Card>
  );
}
