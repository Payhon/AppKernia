import {
  ApiOutlined,
  ApartmentOutlined,
  AppstoreOutlined,
  AuditOutlined,
  BankOutlined,
  BookOutlined,
  CodeOutlined,
  ControlOutlined,
  DashboardOutlined,
  DesktopOutlined,
  FileOutlined,
  FileSearchOutlined,
  FolderOpenOutlined,
  FundProjectionScreenOutlined,
  GlobalOutlined,
  HeartOutlined,
  IdcardOutlined,
  KeyOutlined,
  LinkOutlined,
  LoginOutlined,
  MailOutlined,
  MenuOutlined,
  MessageOutlined,
  NotificationOutlined,
  SafetyCertificateOutlined,
  ScheduleOutlined,
  SendOutlined,
  SettingOutlined,
  SlidersOutlined,
  SoundOutlined,
  StopOutlined,
  TeamOutlined,
  UserOutlined,
  UserSwitchOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import type { ComponentType, HTMLAttributes } from 'react'

type MenuIconComponent = ComponentType<HTMLAttributes<HTMLSpanElement>>

const menuIconRegistry: Readonly<Record<string, MenuIconComponent>> = {
  ApiOutlined,
  ApartmentOutlined,
  AppstoreOutlined,
  AuditOutlined,
  BankOutlined,
  BookOutlined,
  CodeOutlined,
  ControlOutlined,
  DashboardOutlined,
  DesktopOutlined,
  FileOutlined,
  FileSearchOutlined,
  FolderOpenOutlined,
  FundProjectionScreenOutlined,
  GlobalOutlined,
  HeartOutlined,
  IdcardOutlined,
  KeyOutlined,
  LinkOutlined,
  LoginOutlined,
  MailOutlined,
  MenuOutlined,
  MessageOutlined,
  NotificationOutlined,
  SafetyCertificateOutlined,
  ScheduleOutlined,
  SendOutlined,
  SettingOutlined,
  SlidersOutlined,
  SoundOutlined,
  StopOutlined,
  TeamOutlined,
  UserOutlined,
  UserSwitchOutlined,
  WarningOutlined,
}

export function ConfiguredMenuIcon({ name }: { name: string | null }) {
  const Icon = name ? menuIconRegistry[name] ?? AppstoreOutlined : AppstoreOutlined
  return <Icon aria-hidden="true" className="ak-menu-item-icon" />
}

export function isConfiguredMenuIcon(name: string): boolean {
  return menuIconRegistry[name] !== undefined
}
