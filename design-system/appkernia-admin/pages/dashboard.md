# Dashboard Page Override

- Route: `/dashboard`
- Task: `AKADM-070`
- Keep the Admin shell, system typography, spacing scale, focus treatment and high-contrast secondary text from `MASTER.md`.
- Use a dense responsive KPI grid: 1 column at 375 px, 2 columns from 768 px, and up to 4 columns on desktop.
- KPI 卡使用 hairline + stacked shadow，顶部仅使用 2px 蓝青绿品牌光谱线；数字使用 ink、600 weight 与紧凑字距。
- 图表使用 DESIGN 光谱中的可区分序列色、虚线网格和 ink tooltip；第一序列可使用不超过 6% 的面积填充。
- Put the 7/30/90-day range control beside the page heading on desktop and below it on mobile. Selection updates URL search params.
- Hide unauthorized KPI cards and trend series completely; never substitute zero for missing permission.
- Use per-section skeleton, guided empty and retryable `role=alert` error states.
- Lazy-load ECharts, disable animation for reduced-motion users, and provide an expandable table alternative with horizontal scrolling.
- Lay out operation, failed-job and security activity as three cards on desktop and a single column on small screens.
- Do not display audit detail JSON, job payload/output, IP addresses, user agents or server error messages.
