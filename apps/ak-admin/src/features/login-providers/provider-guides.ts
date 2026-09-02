import type { LoginProviderCode } from "./model";

export interface LoginProviderGuideLink {
  key: string;
  url: string;
}

export interface LoginProviderGuide {
  provider: LoginProviderCode;
  fieldKeys: readonly string[];
  stepCount: number;
  links: readonly LoginProviderGuideLink[];
  verifiedAt: string;
}

export const loginProviderGuides: readonly LoginProviderGuide[] = [
  {
    provider: "wechat",
    fieldKeys: ["wechat_app_id", "app_secret", "android_package_name", "android_app_signature", "ios_bundle_id", "ios_universal_link", "harmony_bundle_name"],
    stepCount: 4,
    links: [
      { key: "platform", url: "https://open.weixin.qq.com/" },
      { key: "dcloud", url: "https://doc.dcloud.net.cn/uni-app-x/api/sign-in.html" },
    ],
    verifiedAt: "2026-09-01",
  },
  {
    provider: "github",
    fieldKeys: ["client_id", "client_secret", "app_return_uri", "callback_uri"],
    stepCount: 4,
    links: [
      { key: "apps", url: "https://github.com/settings/developers" },
      { key: "oauth", url: "https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps" },
    ],
    verifiedAt: "2026-09-01",
  },
  {
    provider: "apple",
    fieldKeys: ["apple_client_id", "team_id", "key_id", "private_key_p8"],
    stepCount: 4,
    links: [
      { key: "account", url: "https://developer.apple.com/account/resources/authkeys/list" },
      { key: "signin", url: "https://developer.apple.com/sign-in-with-apple/get-started/" },
    ],
    verifiedAt: "2026-09-01",
  },
  {
    provider: "google",
    fieldKeys: ["server_client_id", "android_package_name", "android_certificate_sha256"],
    stepCount: 4,
    links: [
      { key: "console", url: "https://console.cloud.google.com/apis/credentials" },
      { key: "credential_manager", url: "https://developer.android.com/identity/sign-in/credential-manager-siwg" },
    ],
    verifiedAt: "2026-09-01",
  },
] as const;

export const loginProviderGuideByProvider = new Map(loginProviderGuides.map((guide) => [guide.provider, guide]));
