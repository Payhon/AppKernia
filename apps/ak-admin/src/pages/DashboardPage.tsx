import { Link, useNavigate, useSearch } from '@tanstack/react-router'
import type { TableColumnsType } from 'antd'
import { Alert, Button, Card, Col, Empty, Row, Segmented, Skeleton, Space, Statistic, Table, Tag, Typography } from 'antd'
import { lazy, Suspense, useEffect, useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { flattenMenuPages, resolveBackendMenus } from '../app/route-registry'
import { ArrowRightIcon } from '../app/icons'
import type { AdminDashboardTrendSeries } from '../generated/api/types.gen'
import type { DashboardRange } from '../features/auth/session'
import { useAuthStore } from '../features/auth/store'
import { useDashboardActivity, useDashboardSummary, useDashboardTrends } from '../features/dashboard/hooks'

const DashboardTrendChart = lazy(async () => import('../components/DashboardTrendChart').then((module) => ({ default: module.DashboardTrendChart })))

const metricKeys = ['users.total', 'users.new', 'sessions.active', 'jobs.failed', 'security.open', 'messages.published'] as const
const trendKeys = ['logins.success', 'logins.failure', 'users.new', 'jobs.failed', 'security.events'] as const

export function DashboardPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const { range } = useSearch({ from: '/dashboard' })
  const context = useAuthStore((state) => state.context)
  const permissions = useMemo(() => new Set(context?.permissions ?? []), [context?.permissions])
  const summary = useDashboardSummary(range)
  const trends = useDashboardTrends(range)
  const activity = useDashboardActivity(range)
  const quickAccess = useMemo(() => flattenMenuPages(resolveBackendMenus(context?.menus ?? [], permissions, context?.feature_flags ?? {})).filter((item) => item.componentKey !== 'dashboard').slice(0, 8), [context, permissions])
  const locale = i18n.resolvedLanguage === 'en-US' ? 'en-US' : 'zh-CN'
  const dateTimeFormatter = useMemo(() => new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }), [locale])
  const trendLabels = useMemo(() => Object.fromEntries(trendKeys.map((key) => [key, t(`dashboard.trends.series.${key}`)])), [t])

  useEffect(() => {
    document.title = `${t('routes.dashboard.title')} · ${t('app.name')}`
  }, [i18n.resolvedLanguage, t])

  const changeRange = (value: string | number) => {
    const next = value as DashboardRange
    void navigate({ to: '/dashboard', search: { range: next }, replace: true })
  }

  return (
    <div className="ak-page-container">
      <header className="ak-page-heading ak-dashboard-heading">
        <div>
          <Typography.Title level={1}>{t('routes.dashboard.title')}</Typography.Title>
          <Typography.Paragraph type="secondary">{t('dashboard.intro')}</Typography.Paragraph>
        </div>
        <div className="ak-dashboard-range">
          <Typography.Text id="dashboard-range-label" strong>{t('dashboard.range.label')}</Typography.Text>
          <Segmented
            aria-labelledby="dashboard-range-label"
            onChange={changeRange}
            options={(['7d', '30d', '90d'] as const).map((value) => ({ label: t(`dashboard.range.${value}`), value }))}
            value={range}
          />
        </div>
      </header>

      <section aria-labelledby="dashboard-summary-title" className="ak-dashboard-section">
        <Typography.Title id="dashboard-summary-title" level={2}>{t('dashboard.summary.title')}</Typography.Title>
        {summary.isPending ? <Row gutter={[16, 16]}>{metricKeys.slice(0, 4).map((key) => <Col key={key} lg={6} sm={12} xs={24}><Card><Skeleton active paragraph={false} /></Card></Col>)}</Row> : null}
        {summary.isError ? <DashboardError onRetry={() => { void summary.refetch() }} title={t('dashboard.summary.error')} /> : null}
        {summary.data?.metrics.length === 0 ? <Card><Empty description={t('dashboard.summary.empty')} /></Card> : null}
        {summary.data?.metrics.length ? (
          <Row gutter={[16, 16]}>
            {summary.data.metrics.map((metric) => (
              <Col key={metric.key} lg={6} sm={12} xs={24}>
                <Card className="ak-kpi-card"><Statistic title={t(`dashboard.metrics.${metric.key}`)} value={metric.value} /></Card>
              </Col>
            ))}
          </Row>
        ) : null}
      </section>

      <section aria-labelledby="dashboard-trends-title" className="ak-dashboard-section">
        <Typography.Title id="dashboard-trends-title" level={2}>{t('dashboard.trends.title')}</Typography.Title>
        {trends.isPending ? <Card><Skeleton active paragraph={{ rows: 7 }} /></Card> : null}
        {trends.isError ? <DashboardError onRetry={() => { void trends.refetch() }} title={t('dashboard.trends.error')} /> : null}
        {trends.data?.series.length === 0 ? <Card><Empty description={t('dashboard.trends.empty')} /></Card> : null}
        {trends.data?.series.length ? (
          <Card>
            <Suspense fallback={<Skeleton active paragraph={{ rows: 7 }} />}>
              <DashboardTrendChart ariaLabel={t('dashboard.trends.chart_label')} labels={trendLabels} series={trends.data.series} />
            </Suspense>
            <TrendTable labels={trendLabels} series={trends.data.series} />
          </Card>
        ) : null}
      </section>

      <section aria-labelledby="dashboard-activity-title" className="ak-dashboard-section">
        <Typography.Title id="dashboard-activity-title" level={2}>{t('dashboard.activity.title')}</Typography.Title>
        {activity.isPending ? <Row gutter={[16, 16]}>{[1, 2, 3].map((key) => <Col key={key} lg={8} xs={24}><Card><Skeleton active paragraph={{ rows: 5 }} /></Card></Col>)}</Row> : null}
        {activity.isError ? <DashboardError onRetry={() => { void activity.refetch() }} title={t('dashboard.activity.error')} /> : null}
        {activity.data ? (
          <Row gutter={[16, 16]}>
            {permissions.has('audit.operation.read') ? (
              <Col lg={8} xs={24}>
                <ActivityCard title={t('dashboard.activity.operations')} empty={activity.data.operations.length === 0}>
                  {activity.data.operations.map((item) => (
                    <li className="ak-activity-item" key={item.id}>
                      <Space wrap><Typography.Text strong>{item.action_name}</Typography.Text><Tag className={item.succeeded ? 'ak-status-tag ak-status-success' : 'ak-status-tag ak-status-error'}>{t(item.succeeded ? 'dashboard.activity.succeeded' : 'dashboard.activity.failed')}</Tag></Space>
                      <Typography.Text className="ak-activity-description">{dateTimeFormatter.format(new Date(item.occurred_at))}</Typography.Text>
                    </li>
                  ))}
                </ActivityCard>
              </Col>
            ) : null}
            {permissions.has('jobs.run.read') ? (
              <Col lg={8} xs={24}>
                <ActivityCard title={t('dashboard.activity.failed_jobs')} empty={activity.data.failed_jobs.length === 0}>
                  {activity.data.failed_jobs.map((item) => (
                    <li className="ak-activity-item" key={item.id}>
                      <Typography.Text strong>{item.schedule_name}</Typography.Text>
                      <Typography.Text className="ak-activity-description">{`${item.error_code || t('dashboard.activity.unknown_error')} · ${dateTimeFormatter.format(new Date(item.occurred_at))}`}</Typography.Text>
                    </li>
                  ))}
                </ActivityCard>
              </Col>
            ) : null}
            {permissions.has('audit.security.read') ? (
              <Col lg={8} xs={24}>
                <ActivityCard title={t('dashboard.activity.security_events')} empty={activity.data.security_events.length === 0}>
                  {activity.data.security_events.map((item) => (
                    <li className="ak-activity-item" key={item.id}>
                      <Space wrap><Typography.Text strong>{item.event_type}</Typography.Text><Tag className={`ak-status-tag ${severityClass(item.severity)}`}>{t(`dashboard.severity.${item.severity}`)}</Tag></Space>
                      <Typography.Text className="ak-activity-description">{`${item.source} · ${dateTimeFormatter.format(new Date(item.occurred_at))}`}</Typography.Text>
                    </li>
                  ))}
                </ActivityCard>
              </Col>
            ) : null}
          </Row>
        ) : null}
      </section>

      <section aria-labelledby="quick-access-title" className="ak-dashboard-section">
        <Typography.Title id="quick-access-title" level={2}>{t('dashboard.quick_access')}</Typography.Title>
        {quickAccess.length === 0 ? <Card><Empty description={t('common.states.empty')} /></Card> : (
          <div className="ak-quick-grid">
            {quickAccess.map((item) => <Link className="ak-quick-link" key={item.code} to={item.path as never}><span>{t(item.i18nKey)}</span><ArrowRightIcon /></Link>)}
          </div>
        )}
      </section>
    </div>
  )
}

function DashboardError({ title, onRetry }: { title: string; onRetry: () => void }) {
  const { t } = useTranslation()
  return <Alert action={<Button onClick={onRetry} size="small">{t('common.actions.retry')}</Button>} role="alert" showIcon title={title} type="error" />
}

function ActivityCard({ title, empty, children }: { title: string; empty: boolean; children: ReactNode }) {
  const { t } = useTranslation()
  return <Card className="ak-activity-card" title={title}>{empty ? <Empty description={t('dashboard.activity.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> : <ul className="ak-activity-list">{children}</ul>}</Card>
}

function TrendTable({ series, labels }: { series: AdminDashboardTrendSeries[]; labels: Record<string, string> }) {
  const { t } = useTranslation()
  const rows = (series[0]?.points ?? []).map((point, index) => {
    const row: Record<string, string | number> = { day: point.day }
    for (const item of series) row[item.key] = item.points[index]?.value ?? 0
    return row
  })
  const columns: TableColumnsType<Record<string, string | number>> = [
    { dataIndex: 'day', key: 'day', title: t('dashboard.trends.day') },
    ...series.map((item) => ({ dataIndex: item.key, key: item.key, title: labels[item.key] ?? item.key })),
  ]
  return (
    <details className="ak-dashboard-table-alternative">
      <summary>{t('dashboard.trends.table_toggle')}</summary>
      <Table columns={columns} dataSource={rows} pagination={false} rowKey="day" scroll={{ x: 'max-content' }} size="small" />
    </details>
  )
}

function severityClass(severity: string): string {
  if (severity === 'critical' || severity === 'high') return 'ak-status-error'
  if (severity === 'medium') return 'ak-status-warning'
  return 'ak-status-neutral'
}
