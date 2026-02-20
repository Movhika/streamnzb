import { useMemo } from 'react'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { ComposedChart, Area, XAxis, YAxis } from "recharts"
import { Activity, Globe, X, MonitorPlay } from "lucide-react"
import { cn } from "@/lib/utils"

const chartConfig = {
  speed: {
    label: "Speed",
    color: "hsl(var(--primary))",
  },
  conns: {
    label: "Connections",
    color: "hsl(var(--primary))",
  },
}

function formatDownloadedMb(mb) {
  const n = Number(mb) || 0
  if (n >= 1000) return { value: (n / 1000).toFixed(2), unit: 'GB' }
  return { value: n.toFixed(1), unit: 'MB' }
}

// Build sets of enabled provider hosts and indexer names from config (match by host/name)
function useEnabledSets(config) {
  return useMemo(() => {
    const providerHosts = new Set()
    const indexerNames = new Set()
    if (!config) return { providerHosts, indexerNames }
    ;(config.providers || []).forEach((p) => {
      if (p.enabled !== false) providerHosts.add((p.host || '').toLowerCase())
    })
    ;(config.indexers || []).forEach((i) => {
      if (i.enabled !== false) indexerNames.add((i.name || '').trim())
    })
    return { providerHosts, indexerNames }
  }, [config])
}

export function DashboardPage({ stats, chartData, sendCommand, config }) {
  const { providerHosts, indexerNames } = useEnabledSets(config)

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 px-4 lg:px-6">
      {/* KPI cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <Card>
          <CardHeader>
            <CardDescription>Total Speed</CardDescription>
            <CardTitle className="flex items-baseline gap-1.5 tabular-nums">
              <span className="text-primary">{(stats.total_speed_mbps ?? 0).toFixed(1)}</span>
              <span className="text-sm font-normal text-muted-foreground">Mbps</span>
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>Active Connections</CardDescription>
            <CardTitle className="tabular-nums text-primary">{stats.active_sessions?.length ?? 0}</CardTitle>
            <p className="text-xs text-muted-foreground">streaming</p>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>Pool Connections</CardDescription>
            <CardTitle className="flex items-baseline gap-1.5 tabular-nums">
              <span className="text-primary">{stats.active_connections}</span>
              <span className="text-sm font-normal text-muted-foreground">/ {stats.total_connections}</span>
            </CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader>
            <CardDescription>Downloaded Today</CardDescription>
            <CardTitle className="flex items-baseline gap-1.5 tabular-nums">
              {(() => {
                const { value, unit } = formatDownloadedMb(stats.total_downloaded_mb)
                return <><span className="text-primary">{value}</span><span className="text-sm font-normal text-muted-foreground">{unit}</span></>
              })()}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Network chart */}
      <Card className="overflow-hidden">
        <CardHeader>
          <CardTitle>Network activity</CardTitle>
          <CardDescription>Speed (Mbps) and active connections over time</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <ChartContainer config={chartConfig} className="h-[200px] w-full">
            <ComposedChart data={chartData} margin={{ top: 8, right: 8, bottom: 8, left: 32 }}>
              <defs>
                <linearGradient id="chartSpeed" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity={0.4} />
                  <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="chartConns" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity={0.25} />
                  <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis dataKey="time" tick={{ fontSize: 10 }} />
              <YAxis yAxisId="left" tick={{ fontSize: 10 }} width={28} />
              <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 10 }} allowDecimals={false} width={28} />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Area
                yAxisId="left"
                type="monotone"
                dataKey="speed"
                stroke="hsl(var(--primary))"
                strokeWidth={2}
                fill="url(#chartSpeed)"
                dot={false}
                isAnimationActive={false}
                name="speed"
              />
              <Area
                yAxisId="right"
                type="monotone"
                dataKey="conns"
                stroke="hsl(var(--primary))"
                strokeWidth={2}
                strokeOpacity={0.7}
                fill="url(#chartConns)"
                dot={false}
                isAnimationActive={false}
                name="conns"
              />
            </ComposedChart>
          </ChartContainer>
        </CardContent>
      </Card>

      {/* Active sessions */}
      {stats.active_sessions?.length > 0 && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Activity className="h-4 w-4 text-primary" />
            <h2 className="text-sm font-semibold">Active streams</h2>
          </div>
          <div className="grid gap-2 md:grid-cols-2 lg:grid-cols-3">
            {stats.active_sessions.map(sess => (
              <Card key={sess.id} className="group relative min-w-0 pr-10">
                <CardContent className="p-3">
                  <div className="text-sm font-medium truncate pr-2 min-w-0" title={sess.title}>{sess.title}</div>
                  <div className="text-xs text-muted-foreground truncate min-w-0">{sess.clients.join(', ')}</div>
                </CardContent>
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute right-2 top-1/2 -translate-y-1/2 h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={() => sendCommand('close_session', { id: sess.id })}
                  title="End stream"
                >
                  <X className="h-4 w-4" />
                </Button>
              </Card>
            ))}
          </div>
        </div>
      )}

      {/* Providers & Indexers */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Providers */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <Globe className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold tracking-tight">Usenet Providers</h2>
          </div>
          <div className="grid gap-3 grid-cols-1 sm:grid-cols-2">
            {stats.providers.map((p) => {
              const loadPct = (p.active_conns / (p.max_conns || 1)) * 100
              const isEnabled = providerHosts.has((p.host || '').toLowerCase())
              return (
                <Card
                  key={p.name}
                  className={cn(!isEnabled && "opacity-60 grayscale")}
                >
                  <CardHeader className="p-3 pb-1">
                    <div className="flex items-center gap-2">
                      <CardTitle className="text-sm font-bold truncate leading-tight" title={p.name}>{p.name}</CardTitle>
                      <Badge variant="outline" className="text-[10px] py-0 h-4 ml-auto">{p.max_conns}</Badge>
                    </div>
                    <p className="text-[10px] text-muted-foreground truncate" title={p.host}>{p.host}</p>
                  </CardHeader>
                  <CardContent className="p-3 pt-0">
                    <div className="flex items-center justify-between mt-2">
                      <div className="flex flex-col">
                        <span className="text-[10px] uppercase text-muted-foreground font-medium">Load</span>
                        <span className="text-sm font-bold">{loadPct.toFixed(0)}%</span>
                      </div>
                      <div className="flex flex-col text-right">
                        <span className="text-[10px] uppercase text-muted-foreground font-medium">Speed</span>
                        <span className="text-sm font-bold text-primary">{(p.current_speed_mbps ?? 0).toFixed(1)} <span className="text-[10px]">Mbps</span></span>
                      </div>
                    </div>
                    <div className="w-full bg-muted h-1.5 rounded-full mt-2 overflow-hidden">
                      <div className="bg-primary h-full transition-all duration-500 rounded-full" style={{ width: `${loadPct}%` }} />
                    </div>
                    <p className="text-[10px] text-muted-foreground mt-2">Downloaded: {(p.downloaded_mb ?? 0).toFixed(1)} MB</p>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        </div>

        {/* Indexers */}
        <div className="space-y-3">
          <div className="flex items-center gap-2">
            <MonitorPlay className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold tracking-tight">Indexers</h2>
          </div>
          <div className="grid gap-3 grid-cols-1 sm:grid-cols-2">
            {stats.indexers?.map((idx) => {
              const apiUsedPct = idx.api_hits_limit > 0 ? ((idx.api_hits_limit - idx.api_hits_remaining) / idx.api_hits_limit) * 100 : 0
              const dlUsedPct = idx.downloads_limit > 0 ? ((idx.downloads_limit - idx.downloads_remaining) / idx.downloads_limit) * 100 : 0
              const barColor = (pct) => pct >= 90 ? 'bg-destructive' : pct >= 75 ? 'bg-chart-4' : 'bg-primary'
              const hasApiLimit = idx.api_hits_limit > 0
              const hasDlLimit = idx.downloads_limit > 0
              const isEnabled = indexerNames.has((idx.name || '').trim())
              return (
                <Card
                  key={idx.name}
                  className={cn("relative overflow-hidden", !isEnabled && "opacity-60 grayscale")}
                >
                  <CardHeader className="p-4 pb-2">
                    <CardTitle className="text-base font-semibold truncate leading-tight" title={idx.name}>{idx.name}</CardTitle>
                  </CardHeader>
                  <CardContent className="p-4 pt-0">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="space-y-1.5">
                        <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">API hits</p>
                        <p className="text-lg font-bold tabular-nums text-primary">{idx.api_hits_used}</p>
                        <p className="text-xs text-muted-foreground">
                          {hasApiLimit ? `of ${idx.api_hits_limit} today` : 'Unlimited'}
                        </p>
                        {hasApiLimit && (
                          <div className="w-full bg-muted h-2 rounded-full overflow-hidden mt-1">
                            <div className={cn("h-full transition-all duration-500 rounded-full", barColor(apiUsedPct))} style={{ width: `${apiUsedPct}%` }} />
                          </div>
                        )}
                        <p className="text-[11px] text-muted-foreground">All-time: {idx.api_hits_used_all_time ?? 0}</p>
                      </div>
                      <div className="space-y-1.5">
                        <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Downloads</p>
                        <p className="text-lg font-bold tabular-nums text-primary">{idx.downloads_used}</p>
                        <p className="text-xs text-muted-foreground">
                          {hasDlLimit ? `of ${idx.downloads_limit} today` : 'Unlimited'}
                        </p>
                        {hasDlLimit && (
                          <div className="w-full bg-muted h-2 rounded-full overflow-hidden mt-1">
                            <div className={cn("h-full transition-all duration-500 rounded-full", barColor(dlUsedPct))} style={{ width: `${dlUsedPct}%` }} />
                          </div>
                        )}
                        <p className="text-[11px] text-muted-foreground">All-time: {idx.downloads_used_all_time ?? 0}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )
            })}
            {(!stats.indexers || stats.indexers.length === 0) && (
              <div className="col-span-full py-8 text-center rounded-lg border border-dashed text-muted-foreground text-sm">
                No internal indexers configured.
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
