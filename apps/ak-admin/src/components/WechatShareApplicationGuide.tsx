import {
  Alert,
  Button,
  Divider,
  Modal,
  Space,
  Steps,
  Tooltip,
  Typography,
} from "antd";
import {
  ExportOutlined,
  QuestionCircleOutlined,
  WechatOutlined,
} from "@ant-design/icons";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import {
  wechatShareGuideChecklistKeys,
  wechatShareGuideLinks,
  wechatShareGuideSteps,
} from "../features/share-configs/wechat-application-guide";

export function WechatShareApplicationGuide() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const triggerLabel = t("share_configs.guide.trigger");

  const stepItems = wechatShareGuideSteps.map((step) => ({
    title: t(`share_configs.guide.steps.${step.key}.title`),
    description: (
      <div className="ak-share-guide-step">
        <Typography.Paragraph>
          {t(`share_configs.guide.steps.${step.key}.description`)}
        </Typography.Paragraph>
        <ul className="ak-share-guide-list">
          {step.requirementKeys.map((key) => <li key={key}>{t(key)}</li>)}
        </ul>
        <Space wrap size={[12, 8]}>
          {step.linkKeys.map((linkKey) => (
            <Typography.Link
              key={linkKey}
              href={wechatShareGuideLinks[linkKey]}
              rel="noopener noreferrer"
              target="_blank"
            >
              {t(`share_configs.guide.links.${linkKey}`)} <ExportOutlined aria-hidden="true" />
            </Typography.Link>
          ))}
        </Space>
      </div>
    ),
  }));

  return (
    <>
      <Tooltip title={triggerLabel}>
        <Button
          aria-label={triggerLabel}
          className="ak-share-guide-trigger"
          icon={<QuestionCircleOutlined />}
          onClick={() => { setOpen(true); }}
          shape="circle"
          size="small"
          type="text"
        />
      </Tooltip>
      <Modal
        cancelButtonProps={{ style: { display: "none" } }}
        className="ak-share-guide-modal"
        okText={t("share_configs.guide.actions.got_it")}
        onCancel={() => { setOpen(false); }}
        onOk={() => { setOpen(false); }}
        open={open}
        title={(
          <Space size="small">
            <WechatOutlined aria-hidden="true" />
            <span>{t("share_configs.guide.title")}</span>
          </Space>
        )}
        width={760}
        zIndex={1100}
      >
        <Alert
          description={t("share_configs.guide.intro")}
          showIcon
          title={t("share_configs.guide.welcome")}
          type="info"
        />
        <Steps
          className="ak-share-guide-steps"
          current={-1}
          items={stepItems}
          orientation="vertical"
          responsive={false}
          size="small"
        />
        <Divider />
        <Typography.Title level={5}>{t("share_configs.guide.checklist.title")}</Typography.Title>
        <ul className="ak-share-guide-list ak-share-guide-checklist">
          {wechatShareGuideChecklistKeys.map((key) => <li key={key}>{t(key)}</li>)}
        </ul>
        <Alert
          description={t("share_configs.guide.secret.description")}
          showIcon
          title={t("share_configs.guide.secret.title")}
          type="warning"
        />
      </Modal>
    </>
  );
}
