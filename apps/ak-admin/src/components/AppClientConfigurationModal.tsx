import { Grid, Modal, Tabs } from "antd";
import { type ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import type { ManagedApplication } from "../features/apps/model";
import { clientConfigTabPermissions, type ClientConfigTabId } from "../features/apps/client-config-registry";
import { AppLoginProviderConfigurationPanel } from "./AppLoginProviderConfigurationPanel";
import { AppScannerConfigurationPanel } from "./AppScannerConfigurationPanel";
import { AppShareConfigurationPanel } from "./AppShareConfigurationPanel";

interface ClientConfigTabRenderProps {
  appId: string;
  canOpenConfigurationPage: boolean;
  canUpdate: boolean;
  canReadLoginSettings: boolean;
  canReadProviders: boolean;
  canUpdateProviders: boolean;
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
  { ...clientConfigTabPermissions[0], labelKey: "apps.client_config.tabs.share", render: ({ appId, canUpdate, onDirtyChange }) => <AppShareConfigurationPanel appId={appId} canUpdate={canUpdate} onDirtyChange={onDirtyChange} /> },
  { ...clientConfigTabPermissions[1], labelKey: "apps.client_config.tabs.scanner", render: ({ appId, canUpdate, onDirtyChange }) => <AppScannerConfigurationPanel appId={appId} canUpdate={canUpdate} onDirtyChange={onDirtyChange} /> },
  { ...clientConfigTabPermissions[2], labelKey: "apps.client_config.tabs.login_providers", render: (props) => <AppLoginProviderConfigurationPanel {...props} /> },
];

const cleanDirtyState = (): Record<ClientConfigTabId, boolean> => Object.fromEntries(clientConfigTabs.map((tab) => [tab.id, false])) as Record<ClientConfigTabId, boolean>;

interface Props {
  app: ManagedApplication | null;
  permissions: ReadonlySet<string>;
  onClose: () => void;
}

export function AppClientConfigurationModal({ app, permissions, onClose }: Props) {
  const { t } = useTranslation();
  const screens = Grid.useBreakpoint();
  const availableTabs = useMemo(() => clientConfigTabs.filter((tab) => permissions.has(tab.readPermission) || tab.id === "login-providers" && permissions.has("app.login_provider_binding.read")), [permissions]);
  const [activeTab, setActiveTab] = useState<ClientConfigTabId>("share");
  const [dirty, setDirty] = useState<Record<ClientConfigTabId, boolean>>(cleanDirtyState);

  useEffect(() => {
    const first = availableTabs[0]?.id;
    if (first && !availableTabs.some((tab) => tab.id === activeTab)) setActiveTab(first);
  }, [app?.id, availableTabs, activeTab]);

  useEffect(() => {
    if (app) setDirty(cleanDirtyState());
  }, [app?.id]);

  const markDirty = useCallback((id: ClientConfigTabId, value: boolean) => {
    setDirty((current) => current[id] === value ? current : { ...current, [id]: value });
  }, []);

  const close = () => {
    if (!Object.values(dirty).some(Boolean)) {
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
    children: tab.render({ appId: app.id, canOpenConfigurationPage: permissions.has("sys.login_provider_config.read"), canUpdate: permissions.has(tab.updatePermission), canReadLoginSettings: permissions.has("app.login_settings.read"), canReadProviders: permissions.has("app.login_provider_binding.read"), canUpdateProviders: permissions.has("app.login_provider_binding.update"), onDirtyChange: (value) => { markDirty(tab.id, value); } }),
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
