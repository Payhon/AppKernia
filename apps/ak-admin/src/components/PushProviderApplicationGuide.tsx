import {
  ExportOutlined,
  QuestionCircleOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Descriptions,
  Grid,
  Modal,
  Space,
  Steps,
  Tabs,
  Tag,
  Tooltip,
  Typography,
} from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { PushWritableProvider } from "../generated/api/types.gen";
import { pushProviderApplicationGuides } from "../features/push-channels/provider-application-guides";

function GuidePanel({ provider }: { provider: PushWritableProvider }) {
  const { t } = useTranslation();
  const guide = pushProviderApplicationGuides.find((item) => item.provider === provider);
  if (!guide) return null;

  const prefix = `push_channels.guide.provider.${provider}`;
  const steps = Array.from({ length: guide.stepCount }, (_, index) => ({
    title: t(`${prefix}.step.${String(index + 1)}.title`),
    content: t(`${prefix}.step.${String(index + 1)}.description`),
  }));

  return (
    <article className="ak-push-guide-panel" aria-labelledby={`ak-push-guide-${provider}`}>
      <div className="ak-push-guide-provider-heading">
        <div>
          <Typography.Title id={`ak-push-guide-${provider}`} level={4}>
            {t(`push_channels.provider.${provider}`)}
          </Typography.Title>
          <Typography.Paragraph type="secondary">{t(`${prefix}.summary`)}</Typography.Paragraph>
        </div>
        <Tag>{t(`${prefix}.account_type`)}</Tag>
      </div>

      <Typography.Title level={5}>{t("push_channels.guide.sections.application")}</Typography.Title>
      <Steps className="ak-push-guide-steps" current={-1} items={steps} orientation="vertical" responsive={false} size="small" />

      <Typography.Title level={5}>{t("push_channels.guide.sections.fields")}</Typography.Title>
      <Descriptions
        bordered
        className="ak-push-guide-fields"
        column={1}
        items={guide.fieldKeys.map((field) => ({
          key: field,
          label: t(`push_channels.field.${field}`),
          children: t(`${prefix}.field.${field}`),
        }))}
        size="small"
      />

      <Alert
        className="ak-push-guide-note"
        description={t(`${prefix}.note`)}
        showIcon
        title={t("push_channels.guide.sections.before_save")}
        type="warning"
      />

      <Typography.Title level={5}>{t("push_channels.guide.sections.official_resources")}</Typography.Title>
      <Space wrap size={[16, 8]}>
        {guide.links.map((link) => (
          <Typography.Link href={link.url} key={link.key} rel="noopener noreferrer" target="_blank">
            {t(`${prefix}.link.${link.key}`)} <ExportOutlined aria-hidden="true" />
          </Typography.Link>
        ))}
      </Space>
    </article>
  );
}

export function PushProviderApplicationGuide() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const [open, setOpen] = useState(false);
  const [provider, setProvider] = useState<PushWritableProvider>("apns");
  const triggerLabel = t("push_channels.guide.trigger");

  return (
    <>
      <Tooltip title={triggerLabel}>
        <Button
          aria-label={triggerLabel}
          className="ak-push-guide-trigger"
          icon={<QuestionCircleOutlined />}
          onClick={() => { setOpen(true); }}
          shape="circle"
          type="text"
        />
      </Tooltip>
      <Modal
        cancelButtonProps={{ style: { display: "none" } }}
        className="ak-push-guide-modal"
        okText={t("push_channels.guide.actions.close")}
        onCancel={() => { setOpen(false); }}
        onOk={() => { setOpen(false); }}
        open={open}
        title={(
          <Space size="small">
            <SafetyCertificateOutlined aria-hidden="true" />
            <span>{t("push_channels.guide.title")}</span>
          </Space>
        )}
        width={1040}
      >
        <Alert
          description={t("push_channels.guide.intro")}
          showIcon
          title={t("push_channels.guide.verified", { date: "2026-08-29" })}
          type="info"
        />
        <Tabs
          activeKey={provider}
          className="ak-push-guide-tabs"
          destroyOnHidden={false}
          items={pushProviderApplicationGuides.map((guide) => ({
            key: guide.provider,
            label: t(`push_channels.provider.${guide.provider}`),
            children: <GuidePanel provider={guide.provider} />,
          }))}
          onChange={(value) => { setProvider(value as PushWritableProvider); }}
          tabPlacement={screens.md ? "start" : "top"}
        />
      </Modal>
    </>
  );
}
