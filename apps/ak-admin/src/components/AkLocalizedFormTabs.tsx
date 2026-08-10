import { ExclamationCircleFilled } from "@ant-design/icons";
import { Alert, Button, Spin, Tabs, Tooltip, Typography } from "antd";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

import type {
  SystemLanguagesState,
} from "../features/settings/system-languages";
import type { AdminLocale } from "../shared/i18n";

interface AkLocalizedFormTabsProps {
  activeLocale: AdminLocale;
  errorLocales?: Partial<Record<AdminLocale, boolean>>;
  languages: SystemLanguagesState;
  onActiveLocaleChange: (locale: AdminLocale) => void;
  renderFields: (locale: AdminLocale, label: string) => ReactNode;
}

export function AkLocalizedFormTabs({
  activeLocale,
  errorLocales = {},
  languages,
  onActiveLocaleChange,
  renderFields,
}: AkLocalizedFormTabsProps) {
  const { t } = useTranslation();

  if (languages.isPending) {
    return (
      <div
        aria-live="polite"
        className="ak-localized-form-status"
        role="status"
      >
        <Spin size="small" />
        <Typography.Text type="secondary">
          {t("common.localized_form.loading")}
        </Typography.Text>
      </div>
    );
  }

  if (!languages.isReady) {
    return (
      <Alert
        action={
          <Button onClick={() => void languages.refetch()} size="small">
            {t("common.actions.retry")}
          </Button>
        }
        role="alert"
        showIcon
        title={t("common.localized_form.load_error")}
        type="error"
      />
    );
  }

  return (
    <Tabs
      activeKey={activeLocale}
      aria-label={t("common.localized_form.languages")}
      className="ak-localized-form-tabs"
      destroyOnHidden={false}
      items={languages.languages.map((language) => {
        const hasError = Boolean(errorLocales[language.value]);
        return {
          children: renderFields(language.value, language.label),
          key: language.value,
          label: (
            <span className="ak-localized-form-tab-label">
              <span>{language.label}</span>
              {hasError ? (
                <Tooltip title={t("common.localized_form.language_has_errors")}>
                  <Typography.Text type="danger">
                    <ExclamationCircleFilled
                      aria-label={t("common.localized_form.language_has_errors")}
                      className="ak-localized-form-tab-error"
                    />
                  </Typography.Text>
                </Tooltip>
              ) : null}
            </span>
          ),
        };
      })}
      onChange={(locale) => {
        onActiveLocaleChange(locale as AdminLocale);
      }}
    />
  );
}
