import { Button, Card, Space, Typography } from "antd";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";

export function OfflinePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  return (
    <main className="ak-offline-page">
      <Card className="ak-offline-card">
        <Typography.Title level={1}>
          {t("routes.errors.offline.title")}
        </Typography.Title>
        <Typography.Paragraph type="secondary">
          {t("routes.errors.offline.description")}
        </Typography.Paragraph>
        <Space wrap>
          <Button
            type="primary"
            onClick={() => {
              window.location.reload();
            }}
          >
            {t("routes.errors.offline.retry")}
          </Button>
          <Button
            onClick={() =>
              void navigate({ to: "/dashboard", search: { range: "30d" } })
            }
          >
            {t("routes.errors.offline.dashboard")}
          </Button>
        </Space>
      </Card>
    </main>
  );
}
