import { LineChart } from "echarts/charts";
import { GridComponent, LegendComponent, TooltipComponent } from "echarts/components";
import { init, use } from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import { useEffect, useRef } from "react";
import type { AdminNotificationTrendBucket } from "../generated/api/types.gen";

use([LineChart, GridComponent, LegendComponent, TooltipComponent, CanvasRenderer]);

const keys = ["accepted", "failed", "invalid_tokens", "opened", "skipped"] as const;

export function NotificationOperationsTrendChart({ items, labels, ariaLabel }: { items: AdminNotificationTrendBucket[]; labels: Record<string, string>; ariaLabel: string }) {
  const container = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!container.current) return;
    const chart = init(container.current);
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    chart.setOption({
      animation: !reducedMotion,
      color: ["#1677ff", "#d4380d", "#d48806", "#08979c", "#8c8c8c"],
      grid: { left: 48, right: 24, top: 52, bottom: 40 },
      legend: { top: 8 },
      tooltip: { trigger: "axis" },
      xAxis: { type: "category", data: items.map((item) => item.bucket.slice(0, 10)), axisLabel: { hideOverlap: true } },
      yAxis: { type: "value", minInterval: 1, splitLine: { lineStyle: { type: "dashed" } } },
      series: keys.map((key) => ({ name: labels[key], type: "line", data: items.map((item) => item[key]), showSymbol: items.length <= 31, symbolSize: 5 })),
    });
    const resize = () => { chart.resize(); };
    window.addEventListener("resize", resize);
    return () => { window.removeEventListener("resize", resize); chart.dispose(); };
  }, [ariaLabel, items, labels]);
  return <div aria-label={ariaLabel} className="ak-notification-operations-chart" ref={container} role="img" />;
}
