// This file is generated from blueprint/admin-frontend/spec/admin-route-registry.json.
// Do not edit manually.

export const generatedRouteRegistry = [
  {
    "componentKey": "dashboard",
    "path": "/dashboard",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.dashboard.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": "dashboard"
  },
  {
    "componentKey": "system.settings.configs",
    "path": "/system/settings/configs",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.settings.configs.title",
    "permissions": [
      "sys.config.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.settings.configs"
  },
  {
    "componentKey": "system.settings.dictionaries",
    "path": "/system/settings/dictionaries",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.settings.dictionaries.title",
    "permissions": [
      "sys.dictionary.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.settings.dictionaries"
  },
  {
    "componentKey": "system.settings.regions",
    "path": "/system/settings/regions",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.settings.regions.title",
    "permissions": [
      "sys.region.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.settings.regions"
  },
  {
    "componentKey": "system.settings.modules",
    "path": "/system/settings/modules",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.settings.modules.title",
    "permissions": [
      "sys.module.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.settings.modules"
  },
  {
    "componentKey": "system.users.departments",
    "path": "/system/users/departments",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.users.departments.title",
    "permissions": [
      "org.unit.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.users.departments"
  },
  {
    "componentKey": "system.users.accounts",
    "path": "/system/users/accounts",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.users.accounts.title",
    "permissions": [
      "iam.user.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.users.accounts"
  },
  {
    "componentKey": "system.users.positions",
    "path": "/system/users/positions",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.users.positions.title",
    "permissions": [
      "org.position.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.users.positions"
  },
  {
    "componentKey": "system.users.tenants",
    "path": "/system/users/tenants",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.users.tenants.title",
    "permissions": [
      "iam.tenant.read"
    ],
    "featureFlag": "multi_tenant",
    "activeMenuCode": "system.users.tenants"
  },
  {
    "componentKey": "system.access.roles",
    "path": "/system/access/roles",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.access.roles.title",
    "permissions": [
      "iam.role.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.access.roles"
  },
  {
    "componentKey": "system.access.permissions",
    "path": "/system/access/permissions",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.access.permissions.title",
    "permissions": [
      "iam.permission.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.access.permissions"
  },
  {
    "componentKey": "system.access.menus",
    "path": "/system/access/menus",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.access.menus.title",
    "permissions": [
      "sys.menu.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.access.menus"
  },
  {
    "componentKey": "system.storage.files",
    "path": "/system/storage/files",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.storage.files.title",
    "permissions": [
      "storage.file.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.storage.files"
  },
  {
    "componentKey": "system.notifications.notices",
    "path": "/system/notifications/notices",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.notifications.notices.title",
    "permissions": [
      "notify.notice.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.notifications.notices"
  },
  {
    "componentKey": "system.notifications.messages",
    "path": "/system/notifications/messages",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.notifications.messages.title",
    "permissions": [
      "notify.message.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.notifications.messages"
  },
  {
    "componentKey": "system.notifications.templates",
    "path": "/system/notifications/templates",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.notifications.templates.title",
    "permissions": [
      "notify.template.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.notifications.templates"
  },
  {
    "componentKey": "system.notifications.deliveries",
    "path": "/system/notifications/deliveries",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.notifications.deliveries.title",
    "permissions": [
      "notify.delivery.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.notifications.deliveries"
  },
  {
    "componentKey": "system.integrations.schedules",
    "path": "/system/integrations/schedules",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.integrations.schedules.title",
    "permissions": [
      "jobs.schedule.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.integrations.schedules"
  },
  {
    "componentKey": "system.integrations.api-clients",
    "path": "/system/integrations/api-clients",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.integrations.api-clients.title",
    "permissions": [
      "sys.api_client.read"
    ],
    "featureFlag": "api_clients",
    "activeMenuCode": "system.integrations.api-clients"
  },
  {
    "componentKey": "system.integrations.webhooks",
    "path": "/system/integrations/webhooks",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.integrations.webhooks.title",
    "permissions": [
      "sys.webhook.read"
    ],
    "featureFlag": "webhooks",
    "activeMenuCode": "system.integrations.webhooks"
  },
  {
    "componentKey": "system.security.operation-logs",
    "path": "/system/security/operation-logs",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.security.operation-logs.title",
    "permissions": [
      "audit.operation.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.security.operation-logs"
  },
  {
    "componentKey": "system.security.login-logs",
    "path": "/system/security/login-logs",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.security.login-logs.title",
    "permissions": [
      "audit.login.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.security.login-logs"
  },
  {
    "componentKey": "system.security.events",
    "path": "/system/security/events",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.security.events.title",
    "permissions": [
      "audit.security.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.security.events"
  },
  {
    "componentKey": "system.security.block-rules",
    "path": "/system/security/block-rules",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.security.block-rules.title",
    "permissions": [
      "iam.block_rule.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.security.block-rules"
  },
  {
    "componentKey": "system.monitoring.sessions",
    "path": "/system/monitoring/sessions",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.monitoring.sessions.title",
    "permissions": [
      "iam.session.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.monitoring.sessions"
  },
  {
    "componentKey": "system.monitoring.health",
    "path": "/system/monitoring/health",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.monitoring.health.title",
    "permissions": [
      "ops.health.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.monitoring.health"
  },
  {
    "componentKey": "profile.basic",
    "path": "/profile/basic",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.profile.basic.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": null
  },
  {
    "componentKey": "profile.security",
    "path": "/profile/security",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.profile.security.title",
    "permissions": [
      "iam.session.read_self",
      "iam.device.read_self"
    ],
    "featureFlag": null,
    "activeMenuCode": null
  },
  {
    "componentKey": "profile.connections",
    "path": "/profile/connections",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.profile.connections.title",
    "permissions": [
      "iam.oauth.manage_self"
    ],
    "featureFlag": "oauth",
    "activeMenuCode": null
  },
  {
    "componentKey": "system.users.account-detail",
    "path": "/system/users/accounts/$userId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.users.account-detail.title",
    "permissions": [
      "iam.user.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.users.accounts"
  },
  {
    "componentKey": "system.users.tenant-detail",
    "path": "/system/users/tenants/$tenantId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.users.tenant-detail.title",
    "permissions": [
      "iam.tenant.read"
    ],
    "featureFlag": "multi_tenant",
    "activeMenuCode": "system.users.tenants"
  },
  {
    "componentKey": "system.access.role-detail",
    "path": "/system/access/roles/$roleId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.access.role-detail.title",
    "permissions": [
      "iam.role.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.access.roles"
  },
  {
    "componentKey": "system.storage.file-detail",
    "path": "/system/storage/files/$fileId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.storage.file-detail.title",
    "permissions": [
      "storage.file.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.storage.files"
  },
  {
    "componentKey": "system.notifications.notice-detail",
    "path": "/system/notifications/notices/$noticeId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.notifications.notice-detail.title",
    "permissions": [
      "notify.notice.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.notifications.notices"
  },
  {
    "componentKey": "system.notifications.message-detail",
    "path": "/system/notifications/messages/$messageId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.notifications.message-detail.title",
    "permissions": [
      "notify.message.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.notifications.messages"
  },
  {
    "componentKey": "system.integrations.schedule-runs",
    "path": "/system/integrations/schedules/$scheduleId/runs",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.integrations.schedule-runs.title",
    "permissions": [
      "jobs.run.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.integrations.schedules"
  },
  {
    "componentKey": "system.integrations.api-client-detail",
    "path": "/system/integrations/api-clients/$clientId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.integrations.api-client-detail.title",
    "permissions": [
      "sys.api_client.read"
    ],
    "featureFlag": "api_clients",
    "activeMenuCode": "system.integrations.api-clients"
  },
  {
    "componentKey": "system.integrations.webhook-deliveries",
    "path": "/system/integrations/webhooks/$webhookId/deliveries",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.integrations.webhook-deliveries.title",
    "permissions": [
      "sys.webhook.delivery.read"
    ],
    "featureFlag": "webhooks",
    "activeMenuCode": "system.integrations.webhooks"
  },
  {
    "componentKey": "system.security.event-detail",
    "path": "/system/security/events/$eventId",
    "auth": "required",
    "layout": "app",
    "titleKey": "routes.system.security.event-detail.title",
    "permissions": [
      "audit.security.read"
    ],
    "featureFlag": null,
    "activeMenuCode": "system.security.events"
  },
  {
    "componentKey": "auth.login",
    "path": "/login",
    "auth": "anonymous_or_callback",
    "layout": "auth",
    "titleKey": "routes.auth.login.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": null
  },
  {
    "componentKey": "auth.register",
    "path": "/register",
    "auth": "anonymous_or_callback",
    "layout": "auth",
    "titleKey": "routes.auth.register.title",
    "permissions": [],
    "featureFlag": "admin_registration",
    "activeMenuCode": null
  },
  {
    "componentKey": "auth.forgot-password",
    "path": "/forgot-password",
    "auth": "anonymous_or_callback",
    "layout": "auth",
    "titleKey": "routes.auth.forgot-password.title",
    "permissions": [],
    "featureFlag": "password_recovery",
    "activeMenuCode": null
  },
  {
    "componentKey": "auth.reset-password",
    "path": "/reset-password",
    "auth": "anonymous_or_callback",
    "layout": "auth",
    "titleKey": "routes.auth.reset-password.title",
    "permissions": [],
    "featureFlag": "password_recovery",
    "activeMenuCode": null
  },
  {
    "componentKey": "auth.oauth-callback",
    "path": "/auth/callback/$provider",
    "auth": "anonymous_or_callback",
    "layout": "auth",
    "titleKey": "routes.auth.oauth-callback.title",
    "permissions": [],
    "featureFlag": "oauth",
    "activeMenuCode": null
  },
  {
    "componentKey": "errors.forbidden",
    "path": "/403",
    "auth": "public",
    "layout": "plain",
    "titleKey": "routes.errors.forbidden.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": null
  },
  {
    "componentKey": "errors.not-found",
    "path": "/404",
    "auth": "public",
    "layout": "plain",
    "titleKey": "routes.errors.not-found.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": null
  },
  {
    "componentKey": "errors.server-error",
    "path": "/500",
    "auth": "public",
    "layout": "plain",
    "titleKey": "routes.errors.server-error.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": null
  },
  {
    "componentKey": "errors.offline",
    "path": "/offline",
    "auth": "public",
    "layout": "plain",
    "titleKey": "routes.errors.offline.title",
    "permissions": [],
    "featureFlag": null,
    "activeMenuCode": null
  }
] as const
