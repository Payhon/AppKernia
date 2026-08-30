import type { AdminShareConfigInput } from "../../generated/api/types.gen";

export interface ShareConfigFilters {
  page: number;
  page_size: number;
  provider_code?: "wechat";
  q?: string;
  status?: "draft" | "active" | "disabled";
}

export const emptyWechatShareConfig = (): AdminShareConfigInput => ({
  name: "",
  description: "",
  provider_code: "wechat",
  external_app_id: "",
  config_schema_version: 1,
  public_config: {
    android: { enabled: false },
    ios: { enabled: false },
    harmony: { enabled: false },
  },
});
