import { useMemo } from 'react'
import { Button } from "@/components/ui/button"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart"
import { ComposedChart, Area, XAxis, YAxis } from "recharts"
import {
  Zap, Router, Network, BarChart3,
  PlayCircle, Cloud, Database, X, ShieldCheck, Settings,
} from "lucide-react"
import { cn } from "@/lib/utils"

/* ─── constants ─── */
const chartConfig = {
  speed: { label: "Speed (Mbps)", color: "hsl(var(--primary))" },
  conns: { label: "Connections", color: "hsl(var(--chart-2))" },
}

/* ─── helpers ─── */
function fmtMb(mb) {
  const n = Number(mb) || 0
  return n >= 1000
    ? { value: (n / 1000).toFixed(2), unit: "GB" }
    : { value: n.toFixed(1), unit: "MB" }
}

function useEnabledSets(config) {
  return useMemo(() => {
    const providerHosts = new Set()
    const indexerNames = new Set()
    if (!config) return { providerHosts, indexerNames }
    ;(config.providers || []).forEach((p) => {
      if (p.enabled !== false) providerHosts.add((p.host || "").toLowerCase())
    })
    ;(config.indexers || []).forEach((i) => {
      if (i.enabled !== false) indexerNames.add((i.name || "").trim())
    })
    return { providerHosts, indexerNames }
  }, [config])
}

/* ═══════════════════════════════════════════════════════
   DashboardPage  –  Obsidian Engine design system
   • Tonal depth via bg steps (surface → surface-container)
   • Space Grotesk for metrics, Inter for labels
   • No 1 px borders — color-shift only
   ═══════════════════════════════════════════════════════ */
export function DashboardPage({ stats, chartData, sendCommand, config, onNavigate }) {
  const { providerHosts, indexerNames } = useEnabledSets(config)
  const dl = fmtMb(stats.total_downloaded_mb)

  return (
    <div className="flex flex-col gap-6 py-6 px-4 lg:px-6 max-w-[1600px] mx-auto w-full">

      {/* ── KPI row ── */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <KPI icon={Zap}       label="Total Speed"       value={(stats.total_speed_mbps ?? 0).toFixed(1)} unit="Mbps" />
        <KPI icon={Router}    label="Active Connections" value={stats.active_sessions?.length ?? 0}       unit="Current" />
        <KPI icon={Network}   label="Pool Connections"   value={`${stats.active_connections ?? 0}/${stats.total_connections ?? 0}`} unit="Capacity utilized" />
        <KPI icon={BarChart3} label="Downloaded Today"   value={dl.value}                                 unit={dl.unit} />
      </div>

      {/* ── Network Activity ── */}
      <section className="rounded-xl bg-[hsl(var(--card))] p-6">
        <div className="flex items-center justify-between mb-5">
          <div>
            <h2 className="text-base font-headline font-semibold tracking-tight">Network Activity</h2>
            <p className="text-xs text-muted-foreground mt-0.5">Real-time throughput analysis</p>
          </div>
          <div className="flex items-center gap-1.5 text-[11px] font-label font-medium">
            <span className="px-2.5 py-1 rounded-md bg-[hsl(var(--secondary))] text-muted-foreground select-none">1H</span>
            <span className="px-2.5 py-1 rounded-md bg-primary/15 text-primary select-none">Live</span>
          </div>
        </div>

        <ChartContainer config={chartConfig} className="h-[220px] w-full">
          <ComposedChart data={chartData} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="gSpeed" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%"   stopColor="hsl(var(--primary))" stopOpacity={0.35} />
                <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="gConns" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%"   stopColor="hsl(var(--chart-2))" stopOpacity={0.2} />
                <stop offset="100%" stopColor="hsl(var(--chart-2))" stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="time" tick={{ fontSize: 10 }} stroke="hsl(var(--muted-foreground))" tickLine={false} axisLine={false} />
            <YAxis yAxisId="left"  tick={{ fontSize: 10 }} width={32} stroke="hsl(var(--muted-foreground))" tickLine={false} axisLine={false} />
            <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 10 }} allowDecimals={false} width={28} stroke="hsl(var(--muted-foreground))" tickLine={false} axisLine={false} />
            <ChartTooltip content={<ChartTooltipContent />} />
            <Area yAxisId="left"  type="monotone" dataKey="speed" stroke="hsl(var(--primary))" strokeWidth={2} fill="url(#gSpeed)" dot={false} isAnimationActive={false} name="speed" />
            <Area yAxisId="right" type="monotone" dataKey="conns" stroke="hsl(var(--chart-2))" strokeWidth={1.5} strokeDasharray="4 3" fill="url(#gConns)" dot={false} isAnimationActive={false} name="conns" />
          </ComposedChart>
        </ChartContainer>
      </section>

      {/* ── Active Streams ── */}
      <section className="rounded-xl bg-[hsl(var(--card))] p-6">
        <SectionHeader icon={PlayCircle} title="Active Streams" subtitle={`${stats.active_sessions?.length ?? 0} active`} />

        {stats.active_sessions?.length > 0 ? (
          <div className="space-y-2 mt-4">
            {stats.active_sessions.map((sess) => (
              <div key={sess.id} className="flex items-center gap-4 rounded-lg bg-[hsl(var(--background))]/60 px-4 py-3 group transition-colors hover:bg-[hsl(var(--background))]/80">
                <PlayCircle className="h-5 w-5 shrink-0 text-primary/70" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{sess.title}</p>
                  <p className="text-xs text-muted-foreground truncate mt-0.5">{sess.clients?.join(", ")}</p>
                </div>
                <Button
                  variant="ghost" size="icon"
                  className="shrink-0 h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10 opacity-0 group-hover:opacity-100 transition-opacity"
                  onClick={() => sendCommand("close_session", { id: sess.id })}
                  title="End stream"
                >
                  <X className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <p className="py-10 text-center text-sm text-muted-foreground">No active streams right now.</p>
        )}
      </section>

      {/* ── Providers & Indexers ── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">

        {/* Usenet Providers */}
        <section className="rounded-xl bg-[hsl(var(--card))] p-6">
          <SectionHeader icon={Cloud} title="Usenet Providers" action={onNavigate && (
            <button onClick={() => onNavigate('settings', 'providers')} className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors font-label">
              <Settings className="h-3.5 w-3.5" /> Manage
            </button>
          )} />
          <div className="space-y-2 mt-4">
            {stats.providers?.map((p) => {
              const on = providerHosts.has((p.host || "").toLowerCase())
              const dl = fmtMb(p.downloaded_mb ?? 0)
              return (
                <div key={p.name} className={cn("flex items-center justify-between rounded-lg bg-[hsl(var(--background))]/60 px-4 py-3", !on && "opacity-40")}>
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{p.name}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate">
                      <span className="font-headline">{dl.value}</span> {dl.unit}
                    </p>
                  </div>
                </div>
              )
            })}
            {(!stats.providers || stats.providers.length === 0) && (
              <p className="py-8 text-center text-sm text-muted-foreground">No providers configured.</p>
            )}
          </div>
        </section>

        {/* Indexers */}
        <section className="rounded-xl bg-[hsl(var(--card))] p-6">
          <SectionHeader icon={Database} title="Indexers" action={onNavigate && (
            <button onClick={() => onNavigate('settings', 'indexers')} className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors font-label">
              <Settings className="h-3.5 w-3.5" /> Manage
            </button>
          )} />
          <div className="space-y-2 mt-4">
            {stats.indexers?.map((idx) => {
              const on = indexerNames.has((idx.name || "").trim())
              return (
                <div key={idx.name} className={cn("flex items-center justify-between rounded-lg bg-[hsl(var(--background))]/60 px-4 py-3", !on && "opacity-40")}>
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{idx.name}</p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      API: {idx.api_hits_used ?? 0}{idx.api_hits_limit > 0 ? `/${idx.api_hits_limit}` : ""}{" "}
                      • DL: {idx.downloads_used ?? 0}{idx.downloads_limit > 0 ? `/${idx.downloads_limit}` : ""}
                    </p>
                  </div>
                </div>
              )
            })}
            {(!stats.indexers || stats.indexers.length === 0) && (
              <p className="py-8 text-center text-sm text-muted-foreground">No indexers configured.</p>
            )}
          </div>
        </section>
      </div>
    </div>
  )
}

/* ─── Sub-components ─── */

function KPI({ icon: Icon, label, value, unit }) {
  return (
    <div className="rounded-xl bg-[hsl(var(--card))] p-5 flex flex-col gap-3">
      <div className="flex items-center gap-2.5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon className="h-4 w-4" />
        </div>
        <span className="text-xs text-muted-foreground font-label">{label}</span>
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-3xl font-headline font-semibold tracking-tight">{value}</span>
        {unit && <span className="text-xs text-muted-foreground font-label">{unit}</span>}
      </div>
    </div>
  )
}

function SectionHeader({ icon: Icon, title, subtitle, action }) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2.5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon className="h-4 w-4" />
        </div>
        <div>
          <h2 className="text-base font-headline font-semibold tracking-tight">{title}</h2>
          {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
        </div>
      </div>
      {action}
    </div>
  )
}
