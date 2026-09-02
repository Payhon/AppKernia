import {
  CheckCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  FileTextOutlined,
  MobileOutlined,
  RocketOutlined,
  StopOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";

import type { ManagedApplication } from "./model";
import { canReadAnyClientConfig } from "./client-config-registry";

export interface ApplicationActionLabels {
  edit: string;
  upgradeCenter: string;
  content: string;
  clientConfig: string;
  enable: string;
  disable: string;
  delete: string;
}

export interface ApplicationActionHandlers {
  edit: () => void;
  upgradeCenter: () => void;
  content: () => void;
  clientConfig: () => void;
  changeStatus: () => void;
  delete: () => void;
}

export function createApplicationActionItems(
  item: ManagedApplication,
  permissions: ReadonlySet<string>,
  labels: ApplicationActionLabels,
  handlers: ApplicationActionHandlers,
  statusPending: boolean,
): NonNullable<MenuProps["items"]> {
  const items: NonNullable<MenuProps["items"]> = [];

  if (permissions.has("app.application.update")) {
    items.push({ key: "edit", icon: <EditOutlined />, label: labels.edit, onClick: handlers.edit });
  }
  items.push(
    { key: "upgrade-center", icon: <RocketOutlined />, label: labels.upgradeCenter, onClick: handlers.upgradeCenter },
    { key: "content", icon: <FileTextOutlined />, label: labels.content, onClick: handlers.content },
  );
  if (canReadAnyClientConfig(permissions)) {
    items.push({ key: "client-config", icon: <MobileOutlined />, label: labels.clientConfig, onClick: handlers.clientConfig });
  }
  if (permissions.has("app.application.disable")) {
    const disabling = item.status === "active";
    items.push({
      key: "status",
      danger: disabling,
      disabled: statusPending,
      icon: disabling ? <StopOutlined /> : <CheckCircleOutlined />,
      label: disabling ? labels.disable : labels.enable,
      onClick: handlers.changeStatus,
    });
  }
  if (permissions.has("app.application.delete") && item.status === "disabled" && !item.is_default) {
    items.push({ key: "delete", danger: true, icon: <DeleteOutlined />, label: labels.delete, onClick: handlers.delete });
  }
  return items;
}
