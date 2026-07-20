import { Card, Empty, Typography } from "antd";
import type { DashboardEmptyState } from "../types";

export default function EmptyStateCard({ state }: { state?: DashboardEmptyState }) {
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
        {state.commands && state.commands.length > 0 && (
          <div style={{ marginTop: 16, textAlign: "left" }}>
            {state.commands.map((c) => (
              <Typography.Paragraph key={c} code copyable style={{ marginBottom: 4 }}>
                {c}
              </Typography.Paragraph>
            ))}
          </div>
        )}
      </Empty>
    </Card>
  );
}
