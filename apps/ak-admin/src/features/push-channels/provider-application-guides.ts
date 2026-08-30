import type { PushWritableProvider } from "../../generated/api/types.gen";

export interface PushProviderGuideLink {
  key: string;
  url: string;
}

export interface PushProviderApplicationGuide {
  provider: PushWritableProvider;
  fieldKeys: readonly string[];
  stepCount: number;
  links: readonly PushProviderGuideLink[];
}

export const pushProviderApplicationGuides: readonly PushProviderApplicationGuide[] = [
  {
    provider: "apns",
    fieldKeys: ["team_id", "key_id", "bundle_id", "apns_environment", "private_key_p8"],
    stepCount: 4,
    links: [
      { key: "account", url: "https://developer.apple.com/account/resources/authkeys/list" },
      { key: "create_key", url: "https://developer.apple.com/help/account/keys/create-a-private-key/" },
      { key: "token_auth", url: "https://developer.apple.com/documentation/usernotifications/establishing-a-token-based-connection-to-apns" },
    ],
  },
  {
    provider: "fcm",
    fieldKeys: ["project_id", "package_name", "service_account_json"],
    stepCount: 4,
    links: [
      { key: "console", url: "https://console.firebase.google.com/" },
      { key: "android_setup", url: "https://firebase.google.com/docs/android/setup" },
      { key: "server_auth", url: "https://firebase.google.com/docs/cloud-messaging/server-environment" },
    ],
  },
  {
    provider: "huawei_android",
    fieldKeys: ["client_id", "package_name", "notification_channel_id", "client_secret"],
    stepCount: 4,
    links: [
      { key: "console", url: "https://developer.huawei.com/consumer/en/service/josp/agc/index.html" },
      { key: "configure", url: "https://developer.huawei.com/consumer/en/doc/HMS-Plugin-Guides-V1/config-agc-0000001050178043-V1" },
      { key: "push_kit", url: "https://developer.huawei.com/consumer/cn/hms/huawei-pushkit" },
    ],
  },
  {
    provider: "honor",
    fieldKeys: ["app_id", "client_id", "package_name", "notification_channel_id", "client_secret"],
    stepCount: 4,
    links: [
      { key: "apply", url: "https://developer.honor.com/cn/docs/11002/guides/app-registration" },
      { key: "server_api", url: "https://developer.honor.com/cn/docs/11002/reference/downlink-message" },
      { key: "classification", url: "https://developer.honor.com/cn/docs/11002/guides/notification-class" },
    ],
  },
  {
    provider: "xiaomi",
    fieldKeys: ["app_id", "app_key", "package_name", "region", "notification_channel_id", "app_secret"],
    stepCount: 4,
    links: [
      { key: "enable", url: "https://dev.mi.com/xiaomihyperos/documentation/detail?pId=1542" },
      { key: "regions", url: "https://dev.mi.com/xiaomihyperos/documentation/detail?pId=1691" },
      { key: "server_api", url: "https://dev.mi.com/xiaomihyperos/documentation/detail?pId=1559" },
    ],
  },
  {
    provider: "oppo",
    fieldKeys: ["app_key", "package_name", "notification_channel_id", "master_secret"],
    stepCount: 4,
    links: [
      { key: "client", url: "https://open.oppomobile.com/bbs/forum.php?mod=viewthread&tid=1918" },
      { key: "console", url: "https://open.oppomobile.com/bbs/forum.php?mod=viewthread&tid=1907" },
      { key: "auth", url: "https://open.oppomobile.com/bbs/forum.php?mod=viewthread&tid=1925" },
    ],
  },
  {
    provider: "vivo",
    fieldKeys: ["app_id", "app_key", "package_name", "service_category", "operations_category", "app_secret"],
    stepCount: 4,
    links: [
      { key: "product", url: "https://developers.vivo.com/product/d/messagePush" },
      { key: "enable", url: "https://developers.vivo.com/doc/d/6b683b474cf64fdab1a0738035c8868e" },
      { key: "privacy", url: "https://developers.vivo.com/doc/d/23807c559e844cbeb06049ee69e71833" },
    ],
  },
  {
    provider: "meizu",
    fieldKeys: ["app_id", "app_key", "package_name", "app_secret"],
    stepCount: 4,
    links: [
      { key: "platform", url: "https://open.flyme.cn/" },
      { key: "push", url: "https://open.flyme.cn/service/3" },
      { key: "client", url: "https://open-res.flyme.cn/fileserver/upload/file/202505/8a03286b43f34f28b379d0cfc11d622c.pdf" },
    ],
  },
  {
    provider: "harmony",
    fieldKeys: ["project_id", "client_id", "bundle_name", "service_category", "operations_category", "service_account_json"],
    stepCount: 4,
    links: [
      { key: "jwt", url: "https://developer.huawei.com/consumer/cn/doc/doccenter-capabilities/push-jwt-token" },
      { key: "classification", url: "https://developer.huawei.com/consumer/en/doc/harmonyos-guides-V5/push-apply-right-V5" },
      { key: "client", url: "https://developer.huawei.com/consumer/en/doc/harmonyos-references-V13/push-pushservice-V13" },
    ],
  },
] as const;

export const pushProviderGuideByProvider = new Map(
  pushProviderApplicationGuides.map((guide) => [guide.provider, guide]),
);
