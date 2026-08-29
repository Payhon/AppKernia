# Decisions

## Shared adjacent-sibling rule

Use `.ak-page-container .ant-alert + * { margin-block-start: 16px; }`. It covers Alert-to-Alert, Alert-to-Card, Alert-to-table wrapper and Alert-to-description transitions, including conditional siblings. Existing 20px page-specific bottom margins remain the larger rhythm through normal margin behavior.

## Scope boundary

Do not add a global `.ant-alert` margin. Ant portals render Modal/Drawer content outside `.ak-page-container`, while `Space` wraps each Alert in `.ant-space-item`, so neither receives duplicate spacing from the shared selector.

## Audit result

The source audit found 27 page files containing 66 Alerts. Of these, 54 are page-scoped and not owned by Ant `Space`; the common rule covers that reusable functional-page boundary. The remaining Alerts are standalone states, Space-managed groups, authentication states or portal-local messages.

## Browser acceptance

App Upgrade Center was used as the representative real-rendered case from the reported defect. Chromium measured exactly 16px between the uni-app x capability Alert and the following Card at both 1440×900 and 375×812.
