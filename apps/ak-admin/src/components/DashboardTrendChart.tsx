import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { init, use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { useEffect, useRef } from 'react'

import type { AdminDashboardTrendSeries } from '../generated/api/types.gen'

use([LineChart, GridComponent, TooltipComponent, CanvasRenderer])

export function DashboardTrendChart({ series, labels, ariaLabel }: {
  series: AdminDashboardTrendSeries[]
  labels: Record<string, string>
  ariaLabel: string
}) {
  const container = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!container.current) return
    const chart = init(container.current)
    const days = series[0]?.points.map((point) => point.day) ?? []
    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    chart.setOption({
      animation: !reducedMotion,
      grid: { left: 42, right: 24, top: 28, bottom: 44 },
      tooltip: { trigger: 'axis' },
      xAxis: { type: 'category', data: days, axisLabel: { hideOverlap: true } },
      yAxis: { type: 'value', minInterval: 1 },
      series: series.map((item) => ({
        name: labels[item.key] ?? item.key,
        type: 'line',
        data: item.points.map((point) => point.value),
        showSymbol: days.length <= 31,
        smooth: false,
      })),
    })
    const resize = () => {
      chart.resize()
    }
    window.addEventListener('resize', resize)
    return () => {
      window.removeEventListener('resize', resize)
      chart.dispose()
    }
  }, [labels, series])

  return <div aria-label={ariaLabel} className="ak-dashboard-chart" ref={container} role="img" />
}
