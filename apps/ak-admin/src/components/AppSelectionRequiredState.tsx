import { Typography } from "antd";
import { useTranslation } from "react-i18next";

export function AppSelectionRequiredState() {
  const { t } = useTranslation();

  return (
    <div aria-live="polite" className="ak-app-selection-required" data-testid="app-selection-required" role="status">
      <Typography.Text type="secondary">{t("apps.scope.required")}</Typography.Text>
    </div>
  );
}
