import { AppleOutlined, GithubOutlined, GoogleOutlined, WechatOutlined } from "@ant-design/icons";

import type { LoginProviderCode } from "../features/login-providers/model";

export function LoginProviderIcon({ provider }: { provider: LoginProviderCode }) {
  const props = { "aria-hidden": true } as const;
  if (provider === "wechat") return <WechatOutlined {...props} />;
  if (provider === "github") return <GithubOutlined {...props} />;
  if (provider === "apple") return <AppleOutlined {...props} />;
  return <GoogleOutlined {...props} />;
}
