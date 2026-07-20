import { Card, Empty, Typography } from "antd";
import type { ReactNode } from "react";
import type { DashboardEmptyState } from "../types";

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
      </Empty>
    </Card>
  );
}
