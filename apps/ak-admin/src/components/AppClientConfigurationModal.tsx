import { Grid, Modal, Tabs } from "antd";
import { type ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import type { ManagedApplication } from "../features/apps/model";
import { AppScannerConfigurationPanel } from "./AppScannerConfigurationPanel";
import { AppShareConfigurationPanel } from "./AppShareConfigurationPanel";

export type ClientConfigTabId = "share" | "scanner";

interface ClientConfigTabRenderProps {
  appId: string;
  canUpdate: boolean;
  onDirtyChange: (dirty: boolean) => void;
}

export interface ClientConfigTabDefinition {
  id: ClientConfigTabId;
  labelKey: string;
  readPermission: string;
  updatePermission: string;
  render: (props: ClientConfigTabRenderProps) => ReactNode;
}

export const clientConfigTabs: ClientConfigTabDefinition[] = [
  { id: "share", labelKey: "apps.client_config.tabs.share", readPermission: "app.share_binding.read", updatePermission: "app.share_binding.update", render: (props) => <AppShareConfigurationPanel {...props} /> },
  { id: "scanner", labelKey: "apps.client_config.tabs.scanner", readPermission: "app.scanner_config.read", updatePermission: "app.scanner_config.update", render: (props) => <AppScannerConfigurationPanel {...props} /> },
];

interface Props {
  app: ManagedApplication | null;
  permissions: ReadonlySet<string>;
  onClose: () => void;
}

export function AppClientConfigurationModal({ app, permissions, onClose }: Props) {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const availableTabs = useMemo(() => clientConfigTabs.filter((tab) => permissions.has(tab.readPermission)), [permissions]);
  const [activeTab, setActiveTab] = useState<ClientConfigTabId>("share");
  const [dirty, setDirty] = useState<Record<ClientConfigTabId, boolean>>({ share: false, scanner: false });

  useEffect(() => {
    const first = availableTabs[0]?.id;
    if (first && !availableTabs.some((tab) => tab.id === activeTab)) setActiveTab(first);
  }, [app?.id, availableTabs, activeTab]);

  useEffect(() => {
    if (app) setDirty({ share: false, scanner: false });
  }, [app?.id]);

  const markDirty = useCallback((id: ClientConfigTabId, value: boolean) => {
    setDirty((current) => current[id] === value ? current : { ...current, [id]: value });
  }, []);

  const close = () => {
    if (!dirty.share && !dirty.scanner) {
      onClose();
      return;
    }
    Modal.confirm({
      title: t("apps.client_config.unsaved.title"),
      content: t("apps.client_config.unsaved.description"),
      okText: t("apps.client_config.unsaved.discard"),
      okButtonProps: { danger: true },
      cancelText: t("common.actions.cancel"),
      onOk: onClose,
    });
  };

  const items = app ? availableTabs.map((tab) => ({
    key: tab.id,
    label: t(tab.labelKey),
    children: tab.render({ appId: app.id, canUpdate: permissions.has(tab.updatePermission), onDirtyChange: (value) => { markDirty(tab.id, value); } }),
  })) : [];

  return <Modal
    centered={Boolean(screens.md)}
    className="ak-client-config-modal"
    destroyOnHidden
    footer={null}
    open={app !== null && items.length > 0}
    title={app ? `${t("apps.client_config.title")} · ${app.name}` : t("apps.client_config.title")}
    width={screens.md ? 760 : "calc(100vw - 16px)"}
    onCancel={close}
  >
    <Tabs activeKey={activeTab} destroyOnHidden={false} items={items} onChange={(key) => { setActiveTab(key as ClientConfigTabId); }} />
  </Modal>;
}
