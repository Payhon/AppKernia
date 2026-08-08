---
title: Authorization and multi-tenancy
description: AppKernia RBAC, menus, data scopes, and tenant isolation.
---

# Authorization and multi-tenancy

AppKernia separates visibility from authority:

- `sys.menus` describes Admin navigation.
- `iam.permissions.code`, such as `iam.user.read`, is a backend authorization fact.
- A hidden button improves UX but never replaces API authorization.
- `tenant_id` comes from a verified session context, never from an untrusted body or query.

Roles use `all`, `tenant`, `department`, `department_tree`, `self`, or `custom` data scopes. Repositories translate effective scope into SQL conditions; filtering a full dataset in Go or the client is prohibited.

| Client | Route prefix    | Audience    |
| ------ | --------------- | ----------- |
| Mobile | `/api/v1`       | `ak-mobile` |
| Admin  | `/admin-api/v1` | `ak-admin`  |

Even for the same user, Admin tokens cannot act as Mobile sessions and vice versa.
