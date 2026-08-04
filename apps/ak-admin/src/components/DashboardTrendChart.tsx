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
      color: ['#007CF0', '#00B8A9', '#6E56CF', '#EB367F', '#E58A00'],
      grid: { left: 46, right: 24, top: 32, bottom: 44 },
      tooltip: {
        trigger: 'axis',
        backgroundColor: 'rgba(23, 23, 23, 0.94)',
        borderWidth: 0,
        textStyle: { color: '#FFFFFF' },
      },
      xAxis: {
        type: 'category',
        data: days,
        axisLabel: { color: '#666666', hideOverlap: true },
        axisLine: { lineStyle: { color: '#D8D8D8' } },
        axisTick: { show: false },
      },
      yAxis: {
        type: 'value',
        minInterval: 1,
        axisLabel: { color: '#666666' },
        splitLine: { lineStyle: { color: '#EFEFEF', type: 'dashed' } },
      },
      series: series.map((item, index) => ({
        name: labels[item.key] ?? item.key,
        type: 'line',
        data: item.points.map((point) => point.value),
        lineStyle: { width: 2.5 },
        areaStyle: index === 0 ? { opacity: 0.06 } : undefined,
        emphasis: { focus: 'series' },
        showSymbol: days.length <= 31,
        symbolSize: 6,
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
