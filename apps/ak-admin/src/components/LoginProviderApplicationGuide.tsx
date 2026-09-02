import { ExportOutlined, QuestionCircleOutlined, SafetyCertificateOutlined } from "@ant-design/icons";
import { Alert, Button, Descriptions, Grid, Modal, Space, Steps, Tabs, Tag, Tooltip, Typography } from "antd";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { loginProviderGuides } from "../features/login-providers/provider-guides";
import type { LoginProviderCode } from "../features/login-providers/model";
import { LoginProviderIcon } from "./LoginProviderIcon";

function GuidePanel({ provider }: { provider: LoginProviderCode }) {
  const { t } = useTranslation();
  const guide = loginProviderGuides.find((item) => item.provider === provider);
  if (!guide) return null;
  const prefix = `login_providers.guide.provider.${provider}`;
  return <article className="ak-login-provider-guide-panel" aria-labelledby={`ak-login-provider-guide-${provider}`}>
    <div className="ak-login-provider-guide-heading">
      <div>
        <Typography.Title id={`ak-login-provider-guide-${provider}`} level={3}>
          <Space size="small"><LoginProviderIcon provider={provider} />{t(`login_providers.provider.${provider}`)}</Space>
        </Typography.Title>
        <Typography.Paragraph type="secondary">{t(`${prefix}.summary`)}</Typography.Paragraph>
      </div>
      <Tag>{t(`${prefix}.account_type`)}</Tag>
    </div>
    <Typography.Title level={4}>{t("login_providers.guide.sections.application")}</Typography.Title>
    <Steps
      className="ak-login-provider-guide-steps"
      current={-1}
      items={Array.from({ length: guide.stepCount }, (_, index) => ({
        title: t(`${prefix}.step.${String(index + 1)}.title`),
        content: t(`${prefix}.step.${String(index + 1)}.description`),
      }))}
      orientation="vertical"
      responsive={false}
      size="small"
    />
    <Typography.Title level={4}>{t("login_providers.guide.sections.fields")}</Typography.Title>
    <Descriptions bordered className="ak-login-provider-guide-fields" column={1} size="small" items={guide.fieldKeys.map((key) => ({
      key,
      label: t(`login_providers.field.${key}`),
      children: t(`${prefix}.field.${key}`),
    }))} />
    <Alert className="ak-login-provider-guide-note" description={t(`${prefix}.note`)} showIcon title={t("login_providers.guide.sections.before_save")} type="warning" />
    <Typography.Title level={4}>{t("login_providers.guide.sections.official_resources")}</Typography.Title>
    <Space wrap size={[16, 8]}>{guide.links.map((link) => <Typography.Link href={link.url} key={link.key} rel="noopener noreferrer" target="_blank">
      {t(`${prefix}.link.${link.key}`)} <ExportOutlined aria-hidden="true" />
    </Typography.Link>)}</Space>
  </article>;
}

export function LoginProviderApplicationGuide() {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const [open, setOpen] = useState(false);
  const [provider, setProvider] = useState<LoginProviderCode>("wechat");
  const triggerLabel = t("login_providers.guide.trigger");
  return <>
    <Tooltip title={triggerLabel}>
      <Button aria-label={triggerLabel} className="ak-login-provider-guide-trigger" icon={<QuestionCircleOutlined />} onClick={() => { setOpen(true); }} shape="circle" type="text" />
    </Tooltip>
    <Modal
      cancelButtonProps={{ style: { display: "none" } }}
      className="ak-login-provider-guide-modal"
      okText={t("login_providers.guide.actions.close")}
      onCancel={() => { setOpen(false); }}
      onOk={() => { setOpen(false); }}
      open={open}
      title={<Typography.Title className="ak-login-provider-guide-title" level={2}><Space size="small"><SafetyCertificateOutlined aria-hidden="true" />{t("login_providers.guide.title")}</Space></Typography.Title>}
      width={1040}
    >
      <Alert description={t("login_providers.guide.intro")} showIcon title={t("login_providers.guide.verified")} type="info" />
      {screens.md ? <Tabs
          activeKey={provider}
          className="ak-login-provider-guide-tabs"
          destroyOnHidden={false}
          items={loginProviderGuides.map((guide) => ({
            key: guide.provider,
            label: <Space size="small"><LoginProviderIcon provider={guide.provider} />{t(`login_providers.provider.${guide.provider}`)}</Space>,
            children: <GuidePanel provider={guide.provider} />,
          }))}
          onChange={(value) => { setProvider(value as LoginProviderCode); }}
          tabPlacement="start"
        /> : <>
          <label className="ak-login-provider-guide-mobile-picker" htmlFor="ak-login-provider-guide-provider">
            <span>{t("login_providers.columns.provider")}</span>
            <select id="ak-login-provider-guide-provider" onChange={(event) => { setProvider(event.target.value as LoginProviderCode); }} value={provider}>
              {loginProviderGuides.map((guide) => <option key={guide.provider} value={guide.provider}>{t(`login_providers.provider.${guide.provider}`)}</option>)}
            </select>
          </label>
          <GuidePanel provider={provider} />
        </>}
    </Modal>
  </>;
}
