export const wechatShareGuideLinks = {
  openPlatform: "https://open.weixin.qq.com/",
  applicationCenter:
    "https://open.weixin.qq.com/cgi-bin/frame?t=home/app_tmpl&lang=zh_CN",
  createApplication:
    "https://developers.weixin.qq.com/doc/oplatform/Mobile_App/guideline/create.html",
  androidAccess:
    "https://developers.weixin.qq.com/doc/oplatform/Mobile_App/Access_Guide/Android.html",
  iosAccess:
    "https://developers.weixin.qq.com/doc/oplatform/Mobile_App/Access_Guide/iOS.html",
  harmonyAccess:
    "https://developers.weixin.qq.com/doc/oplatform/Mobile_App/Access_Guide/ohos.html",
  sharing:
    "https://developers.weixin.qq.com/doc/oplatform/Mobile_App/Share_and_Favorites/Share_and_Favorites.html",
  dcloudModule:
    "https://doc.dcloud.net.cn/uni-app-x/collocation/manifest-modules.html",
} as const;

export type WechatShareGuideLinkKey = keyof typeof wechatShareGuideLinks;

interface WechatShareGuideStep {
  key: "account" | "application" | "identity" | "review" | "appkernia";
  linkKeys: readonly WechatShareGuideLinkKey[];
  requirementKeys: readonly string[];
}

export const wechatShareGuideSteps: readonly WechatShareGuideStep[] = [
  {
    key: "account",
    linkKeys: ["openPlatform"],
    requirementKeys: [
      "share_configs.guide.steps.account.requirement.subject",
      "share_configs.guide.steps.account.requirement.materials",
    ],
  },
  {
    key: "application",
    linkKeys: ["applicationCenter", "createApplication"],
    requirementKeys: [
      "share_configs.guide.steps.application.requirement.profile",
      "share_configs.guide.steps.application.requirement.consistency",
    ],
  },
  {
    key: "identity",
    linkKeys: ["androidAccess", "iosAccess", "harmonyAccess"],
    requirementKeys: [
      "share_configs.guide.steps.identity.requirement.android",
      "share_configs.guide.steps.identity.requirement.ios",
      "share_configs.guide.steps.identity.requirement.harmony",
    ],
  },
  {
    key: "review",
    linkKeys: ["sharing"],
    requirementKeys: [
      "share_configs.guide.steps.review.requirement.submit",
      "share_configs.guide.steps.review.requirement.app_id",
    ],
  },
  {
    key: "appkernia",
    linkKeys: ["dcloudModule"],
    requirementKeys: [
      "share_configs.guide.steps.appkernia.requirement.configure",
      "share_configs.guide.steps.appkernia.requirement.build",
      "share_configs.guide.steps.appkernia.requirement.verify",
    ],
  },
] as const;

export const wechatShareGuideChecklistKeys = [
  "share_configs.guide.checklist.approved",
  "share_configs.guide.checklist.identity",
  "share_configs.guide.checklist.universal_link",
  "share_configs.guide.checklist.device",
] as const;
