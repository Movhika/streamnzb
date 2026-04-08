import { Fragment, useState, useEffect, useCallback, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { History, Loader2, ExternalLink, RefreshCw, Copy, Check, ChevronDown, ChevronRight, Info, Search as SearchIcon, SlidersHorizontal } from 'lucide-react'
import { getApiUrl, apiFetch } from '../api'
import { cn } from '@/lib/utils'

function formatSize(bytes) {
  if (bytes <= 0) return '—'
  const gb = bytes / (1024 * 1024 * 1024)
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  const mb = bytes / (1024 * 1024)
  if (mb >= 1) return `${mb.toFixed(0)} MB`
  const kb = bytes / 1024
  return `${kb.toFixed(0)} KB`
}

function formatAttemptResult(attempt) {
  if (attempt.preload) return 'Pending'
  return attempt.success ? 'OK' : 'Failed'
}

function shortReason(reason) {
  const value = (reason || '').toLowerCase()
  if (!value) return ''
  if (value.includes('download limit reached') || value.includes('api limit reached') || value.includes('request limit reached')) return 'Limit'
  if (value.includes('playback startup timeout') || value.includes('timed out') || value.includes('context deadline exceeded')) return 'Timeout'
  if (value.includes('probe inspect') || value.includes('probe:') || value.includes('invalid container header')) return 'Probe'
  if (value.includes('episode target not found') || value.includes('no file')) return 'No file'
  if (value.includes('430') || value.includes('segment unavailable') || value.includes('not found')) return 'Segment'
  if (value.includes('eof')) return 'EOF'
  if (value.includes('corrupt') || value.includes('rapidyenc') || value.includes('yenc')) return 'Corrupt'
  if (value.includes('compressed')) return 'Compressed'
  if (value.includes('encrypted')) return 'Encrypted'
  return 'Error'
}

function attemptBadgeClass(attempt, reasonLabel) {
  if (attempt.success) return 'bg-green-600 text-white hover:bg-green-600 hover:text-white dark:text-black'
  if (attempt.preload) return 'bg-muted text-foreground hover:bg-muted'
  if (reasonLabel === 'Limit') return 'bg-slate-500 text-white hover:bg-slate-500 hover:text-white dark:bg-slate-400 dark:text-black'
  return 'bg-red-500 text-white hover:bg-red-500 hover:text-white dark:bg-red-500 dark:text-black'
}

function formatDateTime(value) {
  return new Date(value).toLocaleString()
}

function formatDateOnly(value) {
  return new Date(value).toLocaleDateString()
}

function formatTimeOnly(value) {
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatContentTypeLabel(contentType) {
  if (contentType === 'series') return 'Series'
  if (contentType === 'movie') return 'Movie'
  return '—'
}

function formatContentTitle(title, contentType, contentID) {
  const baseTitle = (title || '').trim()
  if (!baseTitle) return '—'
  if (contentType !== 'series' || !contentID) return baseTitle

  const parts = String(contentID).split(':')
  if (parts.length < 3) return baseTitle

  const season = Number(parts[parts.length - 2])
  const episode = Number(parts[parts.length - 1])
  if (!Number.isInteger(season) || !Number.isInteger(episode)) return baseTitle

  return `${baseTitle} S${String(season).padStart(2, '0')}E${String(episode).padStart(2, '0')}`
}

function buildBadMatchReport(attempt) {
  const details = [
    ['Time', attempt.tried_at ? new Date(attempt.tried_at).toISOString() : '—'],
    ['Content type', attempt.content_type || '—'],
    ['Content title', attempt.content_title || '—'],
    ['Content ID', attempt.content_id || '—'],
    ['Indexer', attempt.indexer_name || '—'],
    ['Release title', attempt.release_title || '—'],
    ['Served file', attempt.served_file || '—'],
    ['Release size', formatSize(attempt.release_size)],
    ['Result', formatAttemptResult(attempt)],
    ['Failure reason', attempt.failure_reason || '—'],
    ['Release URL', attempt.release_url || '—'],
    ['Slot path', attempt.slot_path || '—'],
  ].map(([label, value]) => `- ${label}: ${value}`)

  return [
    'Bad match report',
    '',
    'Why this is a bad match:',
    '- ',
    '',
    'Metadata:',
    ...details,
  ].join('\n')
}

function withinTimeframe(attempt, timeframe) {
  if (timeframe === 'all') return true
  const triedAt = new Date(attempt.tried_at).getTime()
  const now = Date.now()
  const dayMs = 24 * 60 * 60 * 1000
  if (timeframe === 'today') {
    const start = new Date()
    start.setHours(0, 0, 0, 0)
    return triedAt >= start.getTime()
  }
  if (timeframe === '7d') return triedAt >= now - 7 * dayMs
  if (timeframe === '30d') return triedAt >= now - 30 * dayMs
  return true
}

function matchesResult(attempt, result) {
  if (result === 'all') return true
  if (result === 'preload') return Boolean(attempt.preload)
  if (result === 'ok') return !attempt.preload && Boolean(attempt.success)
  if (result === 'failed') return !attempt.preload && !attempt.success
  return true
}

function matchesSearch(attempt, search) {
  if (!search) return true
  const haystack = [
    attempt.content_title,
    attempt.content_id,
    attempt.release_title,
    attempt.indexer_name,
    attempt.provider_name,
    attempt.served_file,
    attempt.failure_reason,
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
  return haystack.includes(search.toLowerCase())
}

function matchesStream(attempt, streamName) {
  if (!streamName || streamName === 'all') return true
  return (attempt.stream_name || 'default') === streamName
}

function buildContentKey(attempt) {
  const identity = attempt.content_id || attempt.content_title || ''
  return [attempt.content_type || '', identity].join('::')
}

function buildRequestGroups(attempts) {
  const byContent = new Map()
  attempts.forEach((attempt) => {
    const key = buildContentKey(attempt)
    const list = byContent.get(key) || []
    list.push(attempt)
    byContent.set(key, list)
  })

  const requestGroups = []
  const requestWindowMs = 15 * 60 * 1000

  byContent.forEach((contentAttempts, contentKey) => {
    const sorted = [...contentAttempts].sort((a, b) => new Date(b.tried_at) - new Date(a.tried_at))
    let cluster = []

    sorted.forEach((attempt) => {
      if (cluster.length === 0) {
        cluster = [attempt]
        return
      }

      const previous = cluster[cluster.length - 1]
      const gap = Math.abs(new Date(previous.tried_at).getTime() - new Date(attempt.tried_at).getTime())
      if (gap <= requestWindowMs) {
        cluster.push(attempt)
        return
      }

      requestGroups.push({ contentKey, attempts: cluster })
      cluster = [attempt]
    })

    if (cluster.length > 0) {
      requestGroups.push({ contentKey, attempts: cluster })
    }
  })

  return requestGroups
    .map((group, index) => {
      const attemptsSorted = [...group.attempts].sort((a, b) => new Date(b.tried_at) - new Date(a.tried_at))
      const latest = attemptsSorted[0]
      const oldest = attemptsSorted[attemptsSorted.length - 1]
      const active = attemptsSorted.find((a) => a.preload) || latest
      const okCount = attemptsSorted.filter((a) => !a.preload && a.success).length
      const failedCount = attemptsSorted.filter((a) => !a.preload && !a.success).length
      const preloadCount = attemptsSorted.filter((a) => a.preload).length

      return {
        key: `${group.contentKey}::${index}::${oldest?.id || latest?.id || index}`,
        contentType: latest?.content_type || '',
        contentID: latest?.content_id || '',
        title: formatContentTitle(latest?.content_title, latest?.content_type, latest?.content_id),
        attempts: attemptsSorted,
        latest,
        active,
        requestTime: oldest?.tried_at || latest?.tried_at,
        okCount,
        failedCount,
        preloadCount,
      }
    })
    .sort((a, b) => new Date(b.requestTime || 0) - new Date(a.requestTime || 0))
}

function formatGroupStatus(group) {
  if (group.okCount > 0) return 'OK'
  return 'Failed'
}

function statusTone(group) {
  return group.okCount > 0 ? 'success' : 'destructive'
}

function groupRequestsByDay(groups) {
  const dayMap = new Map()
  groups.forEach((group) => {
    const dayKey = formatDateOnly(group.requestTime)
    const list = dayMap.get(dayKey) || []
    list.push(group)
    dayMap.set(dayKey, list)
  })
  return Array.from(dayMap.entries()).map(([day, items]) => ({
    day,
    items,
  }))
}

function SummaryCard({ label, value, tone }) {
  return (
    <div className="rounded-lg border border-border/60 bg-muted/30 px-3 py-2">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">{label}</div>
      <div
        className={cn(
          'mt-1 text-lg font-semibold',
          tone === 'success' && 'text-green-600',
          tone === 'destructive' && 'text-destructive'
        )}
      >
        {value}
      </div>
    </div>
  )
}

function formatTimeframeLabel(value) {
  if (value === 'today') return 'Today'
  if (value === '7d') return '7 days'
  if (value === '30d') return '30 days'
  if (value === 'all') return 'All time'
  return value
}

function formatResultFilterLabel(value) {
  if (value === 'ok') return 'OK'
  if (value === 'failed') return 'Failed'
  if (value === 'preload') return 'Pending'
  return 'All'
}

export function NZBHistoryPage({ refreshTrigger }) {
  const [attempts, setAttempts] = useState([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState(null)
  const [copyError, setCopyError] = useState(null)
  const [copiedAttemptId, setCopiedAttemptId] = useState(null)
  const [expandedGroups, setExpandedGroups] = useState({})
  const [selectedAttemptId, setSelectedAttemptId] = useState(null)
  const [timeframe, setTimeframe] = useState('7d')
  const [resultFilter, setResultFilter] = useState('all')
  const [streamFilter, setStreamFilter] = useState('all')
  const [search, setSearch] = useState('')
  const [filtersDialogOpen, setFiltersDialogOpen] = useState(false)

  const fetchAttempts = useCallback((showLoadingSpinner = true) => {
    if (showLoadingSpinner) setLoading(true)
    else setRefreshing(true)
    setError(null)
    const url = getApiUrl('/api/nzb-attempts?limit=200')
    apiFetch(url)
      .then((data) => {
        if (Array.isArray(data)) setAttempts(data)
      })
      .catch((err) => {
        setError(err.message || 'Failed to load NZB history')
      })
      .finally(() => {
        setLoading(false)
        setRefreshing(false)
      })
  }, [])

  useEffect(() => {
    fetchAttempts(true)
  }, [fetchAttempts])

  useEffect(() => {
    if (refreshTrigger == null || refreshTrigger === 0) return
    fetchAttempts(false)
  }, [refreshTrigger, fetchAttempts])

  const streamOptions = useMemo(() => {
    return Array.from(new Set(attempts.map((attempt) => attempt.stream_name || 'default'))).sort((a, b) => a.localeCompare(b))
  }, [attempts])

  const filteredAttempts = useMemo(() => {
    return attempts.filter((attempt) => (
      withinTimeframe(attempt, timeframe) &&
      matchesResult(attempt, resultFilter) &&
      matchesStream(attempt, streamFilter) &&
      matchesSearch(attempt, search)
    ))
  }, [attempts, timeframe, resultFilter, streamFilter, search])

  const requestGroups = useMemo(() => buildRequestGroups(filteredAttempts), [filteredAttempts])

  const summary = useMemo(() => ({
    requests: requestGroups.length,
    attempts: filteredAttempts.length,
    ok: filteredAttempts.filter((attempt) => !attempt.preload && attempt.success).length,
    failed: filteredAttempts.filter((attempt) => !attempt.preload && !attempt.success).length,
    preload: filteredAttempts.filter((attempt) => attempt.preload).length,
  }), [filteredAttempts, requestGroups])

  const toggleGroup = useCallback((key) => {
    setExpandedGroups((current) => ({ ...current, [key]: !current[key] }))
  }, [])

  const requestsByDay = useMemo(() => groupRequestsByDay(requestGroups), [requestGroups])

  const activeFilterChips = useMemo(() => {
    const chips = []
    if (timeframe !== '7d') {
      chips.push({ key: 'timeframe', label: formatTimeframeLabel(timeframe) })
    }
    if (streamFilter !== 'all') {
      chips.push({ key: 'stream', label: streamFilter })
    }
    if (resultFilter !== 'all') {
      chips.push({ key: 'status', label: formatResultFilterLabel(resultFilter) })
    }
    return chips
  }, [timeframe, streamFilter, resultFilter])

  const selectedAttempt = useMemo(
    () => filteredAttempts.find((attempt) => attempt.id === selectedAttemptId) || null,
    [filteredAttempts, selectedAttemptId]
  )

  const resetFilters = useCallback(() => {
    setTimeframe('7d')
    setStreamFilter('all')
    setResultFilter('all')
  }, [])

  const handleCopyBadMatch = useCallback(async (attempt) => {
    if (!navigator?.clipboard?.writeText) {
      setCopyError('Clipboard access is unavailable in this browser.')
      return
    }
    try {
      await navigator.clipboard.writeText(buildBadMatchReport(attempt))
      setCopyError(null)
      setCopiedAttemptId(attempt.id)
      setTimeout(() => {
        setCopiedAttemptId((current) => (current === attempt.id ? null : current))
      }, 2000)
    } catch {
      setCopyError('Failed to copy bad match details.')
    }
  }, [])

  return (
    <div className={cn('flex min-w-0 flex-1 min-h-0 flex-col gap-4 overflow-x-hidden px-4 py-4 md:gap-6 md:py-6 lg:px-6')}>
      <Card className="flex min-w-0 flex-1 min-h-0 flex-col overflow-hidden">
        <CardHeader>
          <div className="flex items-start justify-between gap-4">
            <div className="min-w-0 flex-1 max-w-[42rem] space-y-0.5">
              <CardTitle className="flex items-center gap-2">
                <History className="size-5" />
                NZB play attempts
              </CardTitle>
              <CardDescription>
                Browse recent play attempts grouped by requested movie or episode. Filters and summary reflect the currently visible set.
              </CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchAttempts(false)}
              disabled={refreshing || loading}
              className="shrink-0"
            >
              {refreshing ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
              Refresh
            </Button>
          </div>
        </CardHeader>
        <CardContent className="flex min-w-0 flex-1 min-h-0 flex-col gap-4 overflow-hidden">
          <div className="grid grid-cols-2 gap-3 rounded-lg border border-border/60 bg-muted/20 p-3 md:grid-cols-4">
            <SummaryCard label="Requests" value={summary.requests} />
            <SummaryCard label="Attempts" value={summary.attempts} />
            <SummaryCard label="OK" value={summary.ok} tone="success" />
            <SummaryCard label="Failed" value={summary.failed} tone="destructive" />
          </div>

          <div className="flex flex-col gap-3 rounded-lg border border-border/60 bg-muted/20 p-3">
            <label className="space-y-1.5">
              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Search</span>
              <div className="flex items-center gap-2">
                <div className="flex h-11 min-w-0 flex-1 items-center gap-2 rounded-lg border border-border/60 bg-background px-3 shadow-xs">
                  <SearchIcon className="size-4 shrink-0 text-muted-foreground" />
                  <Input
                    value={search}
                    onChange={(event) => setSearch(event.target.value)}
                    placeholder="Request, release, provider, ID or indexer"
                    className="h-auto border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
                  />
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="icon"
                  className="relative h-11 w-11 shrink-0 rounded-lg"
                  onClick={() => setFiltersDialogOpen(true)}
                  aria-label="Open filters"
                >
                  <SlidersHorizontal className="size-4" />
                  {activeFilterChips.length > 0 ? (
                    <span className="absolute right-2 top-2 size-2 rounded-full bg-primary" />
                  ) : null}
                </Button>
              </div>
            </label>
            {activeFilterChips.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {activeFilterChips.map((chip) => (
                  <Badge key={chip.key} variant="secondary" className="rounded-full px-2.5 py-0.5 text-xs font-medium">
                    {chip.label}
                  </Badge>
                ))}
              </div>
            ) : null}
          </div>

          {loading && (
            <div className="flex items-center justify-center gap-2 py-12 text-muted-foreground">
              <Loader2 className="size-5 animate-spin" />
              Loading…
            </div>
          )}
          {error && <div className="px-2 text-destructive">{error}</div>}
          {copyError && !error && <div className="px-2 text-destructive">{copyError}</div>}

          {!loading && !error && (
            requestGroups.length === 0 ? (
              <div className="flex flex-1 min-h-[320px] items-center justify-center rounded-lg border border-dashed px-6 py-8 text-center text-sm text-muted-foreground">
                No matching NZB attempts found for the current filters.
              </div>
            ) : (
              <div className="flex min-w-0 flex-1 min-h-0 flex-col overflow-y-auto rounded-lg border border-border/60">
                <table className="w-full table-fixed border-collapse text-sm">
                  <thead className="sticky top-0 z-10 bg-background">
                      <tr className="border-b border-border/60 text-left">
                        <th className="w-[84px] px-3 py-3 font-medium text-muted-foreground">
                          <div className="flex justify-center">Time</div>
                        </th>
                        <th className="px-3 py-3 font-medium text-muted-foreground">Request</th>
                        <th className="w-[120px] px-2 py-3 font-medium text-muted-foreground text-center">
                          <div className="mx-auto w-fit pl-5 text-center">Status</div>
                        </th>
                        <th className="w-[72px] px-2 py-3 font-medium text-muted-foreground" />
                      </tr>
                  </thead>
                  {requestsByDay.map((section) => (
                    <tbody key={section.day}>
                      <tr className="bg-background">
                        <td colSpan={4} className="px-0 py-0">
                          <div className="grid grid-cols-[84px_minmax(0,1fr)_120px_72px] border-y border-border/40 bg-muted/45 px-0 py-2">
                            <div className="flex justify-center">
                              <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                {section.day}
                              </span>
                            </div>
                            <div />
                            <div />
                            <div />
                          </div>
                        </td>
                      </tr>
                      {section.items.map((group) => {
                        const expanded = Boolean(expandedGroups[group.key])
                        const activeAttempt = group.active
                        return (
                          <Fragment key={group.key}>
                            <tr className="border-b border-border/50 bg-background/80 hover:bg-muted/20">
                              <td className="px-3 py-3 align-middle">
                                <div className="flex justify-center">
                                  <span className="text-xs font-normal text-muted-foreground">
                                    {formatTimeOnly(group.requestTime)}
                                  </span>
                                </div>
                              </td>
                              <td className="px-3 py-3 align-top">
                                <div className="min-w-0">
                                  <div className="truncate font-medium">{group.title}</div>
                                  <div className="truncate text-xs text-muted-foreground">
                                    {[group.latest?.stream_name || 'default', formatContentTypeLabel(group.contentType), group.latest?.content_id || '—'].join(' • ')}
                                  </div>
                                </div>
                              </td>
                              <td className="px-2 py-3 align-middle">
                                <div className="flex w-full justify-center pl-5">
                                  <Badge
                                    variant={statusTone(group) === 'destructive' ? 'destructive' : 'secondary'}
                                    className={cn(
                                      statusTone(group) === 'success' && 'bg-green-600 text-white hover:bg-green-600 hover:text-white dark:text-black',
                                      statusTone(group) === 'destructive' && 'bg-red-500 text-white hover:bg-red-500 hover:text-white dark:bg-red-500 dark:text-black',
                                      statusTone(group) === 'default' && 'bg-blue-600 text-white hover:bg-blue-600'
                                    )}
                                  >
                                    {formatGroupStatus(group)}
                                  </Badge>
                                </div>
                              </td>
                              <td className="w-[56px] px-2 py-3 align-middle text-right">
                                <div className="inline-flex min-w-[40px] items-center justify-end gap-1.5">
                                  {!expanded ? (
                                    <span className="text-xs font-medium text-muted-foreground">
                                      {group.attempts.length}
                                    </span>
                                  ) : null}
                                  <button
                                    type="button"
                                    onClick={() => toggleGroup(group.key)}
                                    aria-label={`${expanded ? 'Collapse' : 'Expand'} attempts for ${group.title}`}
                                    aria-expanded={expanded}
                                    className="inline-flex items-center justify-center rounded-sm text-muted-foreground transition-colors hover:text-foreground"
                                  >
                                    {expanded ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
                                  </button>
                                </div>
                              </td>
                            </tr>
                            {expanded && (
                              <tr className="border-b border-border/50 bg-muted/10">
                                <td colSpan={4} className="bg-muted/25 px-4 py-5">
                                  <div className="animate-in slide-in-from-top-1 fade-in-0 rounded-lg border border-border/70 bg-background/95 shadow-sm duration-200">
                                    <table className="w-full table-fixed border-collapse text-sm">
                                      <tbody>
                                        {group.attempts.map((attempt) => {
                                          const reasonLabel = shortReason(attempt.failure_reason)
                                          const attemptBadgeLabel = attempt.success ? 'OK' : (reasonLabel || (attempt.preload ? 'Pending' : 'Failed'))
                                          return (
                                            <Fragment key={attempt.id}>
                                              <tr className="border-b border-border/50">
                                                <td className="px-3 py-3 align-top">
                                                  <div className="truncate font-medium">{attempt.release_title || '—'}</div>
                                                  <div className="mt-1 truncate text-xs text-muted-foreground">
                                                    {[attempt.indexer_name || '—', attempt.provider_name || '—', formatSize(attempt.release_size)].join(' • ')}
                                                  </div>
                                                </td>
                                                <td className="w-[120px] px-2 py-3 align-middle">
                                                  <div className="flex items-center justify-center pl-5">
                                                    <Badge
                                                      variant={attempt.success ? 'default' : attempt.preload ? 'secondary' : 'destructive'}
                                                      className={attemptBadgeClass(attempt, reasonLabel)}
                                                    >
                                                      {attemptBadgeLabel}
                                                    </Badge>
                                                  </div>
                                                </td>
                                                <td className="w-[72px] px-3 py-3 align-middle text-right">
                                                  <div className="flex w-full justify-end">
                                                    <button
                                                      type="button"
                                                      onClick={() => setSelectedAttemptId(attempt.id)}
                                                      className="inline-flex items-center justify-center rounded-sm text-muted-foreground hover:text-foreground"
                                                      aria-label="Show attempt details"
                                                    >
                                                      <Info className="size-4" />
                                                    </button>
                                                  </div>
                                                </td>
                                              </tr>
                                            </Fragment>
                                          )
                                        })}
                                      </tbody>
                                    </table>
                                  </div>
                                </td>
                              </tr>
                            )}
                          </Fragment>
                        )
                      })}
                    </tbody>
                  ))}
                </table>
              </div>
            )
          )}

          <Dialog open={Boolean(selectedAttempt)} onOpenChange={(open) => {
            if (!open) setSelectedAttemptId(null)
          }}>
            <DialogContent className="max-h-[85vh] w-[calc(100vw-2rem)] max-w-3xl overflow-x-hidden overflow-y-auto rounded-2xl px-5 sm:px-6">
              {selectedAttempt ? (
                <>
                  <DialogHeader>
                    <DialogTitle className="pr-10 text-left text-xl leading-tight [overflow-wrap:anywhere] sm:text-2xl">
                      {selectedAttempt.release_title || 'Attempt details'}
                    </DialogTitle>
                    <DialogDescription className="text-left">
                      {formatDateTime(selectedAttempt.tried_at)}
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-2 text-sm">
                    <div><span className="text-muted-foreground">Stream:</span> {selectedAttempt.stream_name || 'default'}</div>
                    <div><span className="text-muted-foreground">Content:</span> {selectedAttempt.content_title || '—'}</div>
                    <div><span className="text-muted-foreground">Content ID:</span> {selectedAttempt.content_id || '—'}</div>
                    <div><span className="text-muted-foreground">Indexer:</span> {selectedAttempt.indexer_name || '—'}</div>
                    <div><span className="text-muted-foreground">Provider:</span> {selectedAttempt.provider_name || '—'}</div>
                    <div><span className="text-muted-foreground">Served file:</span> {selectedAttempt.served_file || '—'}</div>
                    <div><span className="text-muted-foreground">Size:</span> {formatSize(selectedAttempt.release_size)}</div>
                    <div><span className="text-muted-foreground">Reason:</span> {selectedAttempt.failure_reason || '—'}</div>
                    <div className="[overflow-wrap:anywhere]"><span className="text-muted-foreground">Slot path:</span> {selectedAttempt.slot_path || '—'}</div>
                  </div>
                  <DialogFooter className="grid min-w-0 grid-cols-2 gap-2 sm:justify-start">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      className="w-full min-w-0"
                      onClick={() => handleCopyBadMatch(selectedAttempt)}
                    >
                      {copiedAttemptId === selectedAttempt.id ? <Check className="size-4" /> : <Copy className="size-4" />}
                      {copiedAttemptId === selectedAttempt.id ? 'Copied' : 'Copy bad match'}
                    </Button>
                    {selectedAttempt.release_url ? (
                      <a
                        href={selectedAttempt.release_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex w-full min-w-0 items-center justify-center gap-2 rounded-md border border-input bg-background px-3 py-2 text-sm font-medium shadow-xs hover:bg-accent hover:text-accent-foreground"
                      >
                        <ExternalLink className="size-4" />
                        Open release
                      </a>
                    ) : null}
                  </DialogFooter>
                </>
              ) : null}
            </DialogContent>
          </Dialog>

          <Dialog open={filtersDialogOpen} onOpenChange={setFiltersDialogOpen}>
            <DialogContent className="w-[calc(100vw-2rem)] max-w-lg rounded-2xl px-5 sm:px-6">
              <DialogHeader>
                <DialogTitle className="text-left text-xl">Filters</DialogTitle>
                <DialogDescription className="text-left">
                  Refine the currently visible NZB history entries.
                </DialogDescription>
              </DialogHeader>
              <div className="rounded-md border border-border/60">
                <div className="p-4">
                  <div className="flex items-center justify-between gap-4">
                    <div className="text-sm font-medium">Timeframe</div>
                    <select
                      aria-label="Timeframe"
                      value={timeframe}
                      onChange={(event) => setTimeframe(event.target.value)}
                      className="flex h-9 w-40 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    >
                      <option value="today">Today</option>
                      <option value="7d">7 days</option>
                      <option value="30d">30 days</option>
                      <option value="all">All</option>
                    </select>
                  </div>
                </div>
                <div className="relative p-4">
                  <div className="absolute left-4 right-4 top-0 border-t border-border/60" />
                  <div className="flex items-center justify-between gap-4">
                    <div className="text-sm font-medium">Stream</div>
                    <select
                      aria-label="Stream"
                      value={streamFilter}
                      onChange={(event) => setStreamFilter(event.target.value)}
                      className="flex h-9 w-40 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    >
                      <option value="all">All streams</option>
                      {streamOptions.map((streamName) => (
                        <option key={streamName} value={streamName}>{streamName}</option>
                      ))}
                    </select>
                  </div>
                </div>
                <div className="relative p-4">
                  <div className="absolute left-4 right-4 top-0 border-t border-border/60" />
                  <div className="flex items-center justify-between gap-4">
                    <div className="text-sm font-medium">Status</div>
                    <select
                      aria-label="Status"
                      value={resultFilter}
                      onChange={(event) => setResultFilter(event.target.value)}
                      className="flex h-9 w-40 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                    >
                      <option value="all">All</option>
                      <option value="ok">OK</option>
                      <option value="failed">Failed</option>
                      <option value="preload">Pending</option>
                    </select>
                  </div>
                </div>
              </div>
              <DialogFooter className="flex-row justify-end gap-2">
                <Button type="button" variant="outline" onClick={resetFilters}>
                  Reset
                </Button>
                <Button type="button" variant="destructive" onClick={() => setFiltersDialogOpen(false)}>
                  Save
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </CardContent>
      </Card>
    </div>
  )
}
