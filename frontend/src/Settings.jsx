import React, { useEffect, useState, useMemo } from 'react'
import { useForm, useFieldArray } from 'react-hook-form'
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from "@/components/ui/form"
import { Switch } from "@/components/ui/switch"
import { PasswordInput } from "@/components/ui/password-input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Loader2, Info, AlertTriangle, Eye, EyeOff, Copy, Check, Settings as SettingsIcon, Server, Globe, MonitorSmartphone } from "lucide-react"
import { IndexerSettings } from "@/components/IndexerSettings"
import { ProviderSettings } from "@/components/ProviderSettings"
import DeviceManagement from "@/components/DeviceManagement"
import { apiFetch } from './api'
import { cn } from "@/lib/utils"

const TABS = [
  { id: 'general', label: 'General', icon: SettingsIcon },
  { id: 'indexers', label: 'Indexers', icon: Server },
  { id: 'providers', label: 'Providers', icon: Globe },
  { id: 'devices', label: 'Devices', icon: MonitorSmartphone },
]

/** Map a dotted field name (e.g. "indexers.0.url") to a tab id. */
function fieldToTab(fieldName) {
  if (fieldName.startsWith('indexers')) return 'indexers'
  if (fieldName.startsWith('providers')) return 'providers'
  return 'general'
}

function EnvOverrideNote({ show }) {
  if (!show) return null
  return (
    <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
      <AlertTriangle className="h-3.5 w-3 shrink-0" />
      Overwritten by environment variable on restart.
    </p>
  )
}

function Settings({ initialConfig, sendCommand, saveStatus, isSaving, adminToken, indexerCaps }) {
  const [activeTab, setActiveTab] = useState('general')
  const [loading, setLoading] = useState(!initialConfig)
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [initialFormValues, setInitialFormValues] = useState(null)
  const [availNZBStatus, setAvailNZBStatus] = useState(null)
  const [availNZBStatusLoading, setAvailNZBStatusLoading] = useState(false)
  const [availNZBStatusError, setAvailNZBStatusError] = useState('')
  const [showAvailNZBRecoverySecret, setShowAvailNZBRecoverySecret] = useState(false)
  const [availNZBRecoverySecretCopied, setAvailNZBRecoverySecretCopied] = useState(false)

  const form = useForm({
    defaultValues: {
      addon_port: 7000,
      addon_base_url: '',
      log_level: 'INFO',
      proxy_port: 119,
      proxy_host: '',
      proxy_auth_user: '',
      proxy_auth_pass: '',
      movie_categories: '',
      tv_categories: '',
      extra_search_terms: '',
      use_season_episode_params: undefined,
      memory_limit_mb: 512,
      keep_log_files: 9,
      availnzb_mode: '',
      providers: [],
      indexers: []
    }
  })

  const envOverrides = initialConfig?.env_overrides ?? []
  const { control, handleSubmit, reset, setError, formState, setValue, watch, getValues } = form
  const { fields, append, remove } = useFieldArray({
    control,
    name: 'providers'
  })
  
  const { fields: indexerFields, append: appendIndexer, remove: removeIndexer } = useFieldArray({
    control,
    name: 'indexers'
  })

  useEffect(() => {
    if (initialConfig) {
      const { env_overrides: _envOverrides, admin_username: _au, admin_password: _ap, ...configForForm } = initialConfig
      const formattedData = {
        ...configForForm,
        addon_port: Number(initialConfig.addon_port),
        proxy_port: Number(initialConfig.proxy_port),
        movie_categories: initialConfig.movie_categories ?? '',
        tv_categories: initialConfig.tv_categories ?? '',
        extra_search_terms: initialConfig.extra_search_terms ?? '',
        use_season_episode_params: initialConfig.use_season_episode_params,
        memory_limit_mb: Number(initialConfig.memory_limit_mb || 0),
        keep_log_files: Number(initialConfig.keep_log_files ?? 9) || 9,
        providers: initialConfig.providers?.map((p, index) => ({
          ...p,
          priority: p.priority != null ? p.priority : index + 1,
          enabled: p.enabled != null ? p.enabled : true,
          port: Number(p.port),
          connections: Number(p.connections)
        })) || [],
        indexers: initialConfig.indexers?.map(idx => ({
          ...idx,
          enabled: idx.enabled != null ? idx.enabled : true,
          api_path: idx.api_path || '/api',
          api_hits_day: Number(idx.api_hits_day || 0),
          downloads_day: Number(idx.downloads_day || 0),
	          timeout_seconds: Number(idx.timeout_seconds || 0),
          username: idx.username || '',
          password: idx.password || '',
          use_season_episode_params: idx.use_season_episode_params,
	          include_year_in_search: idx.include_year_in_search,
          search_title_normalize: idx.search_title_normalize === true,
          disable_id_search: idx.disable_id_search === true,
          disable_string_search: idx.disable_string_search === true
        })) || []
      }
      reset(formattedData)
      setInitialFormValues(JSON.stringify(formattedData))
      setHasUnsavedChanges(false)
      setLoading(false)
    }
  }, [initialConfig, reset])

  useEffect(() => {
    const subscription = watch((value) => {
      const currentValues = JSON.stringify(value)
      if (initialFormValues && currentValues !== initialFormValues) {
        setHasUnsavedChanges(true)
      } else {
        setHasUnsavedChanges(false)
      }
    })
    return () => subscription.unsubscribe()
  }, [watch, initialFormValues])

  useEffect(() => {
      if (saveStatus.errors) {
          Object.keys(saveStatus.errors).forEach(key => {
              setError(key, { type: 'server', message: saveStatus.errors[key] });
          });
      }
  }, [saveStatus.errors, setError]);

  const fetchAvailNZBStatus = async () => {
    setAvailNZBStatusLoading(true)
    setAvailNZBStatusError('')
    setAvailNZBRecoverySecretCopied(false)
    try {
      const data = await apiFetch('/api/availnzb/status')
      setAvailNZBStatus(data || null)
    } catch (error) {
      setAvailNZBStatus(null)
      setAvailNZBStatusError(error.message || 'Failed to load AvailNZB key status.')
    } finally {
      setAvailNZBStatusLoading(false)
    }
  }

  const handleCopyAvailNZBRecoverySecret = async () => {
    const recoverySecret = availNZBStatus?.recovery_secret?.trim?.() || ''
    if (!recoverySecret) return
    if (!navigator?.clipboard?.writeText) {
      setAvailNZBStatusError('Clipboard access is unavailable in this browser.')
      return
    }
    try {
      await navigator.clipboard.writeText(recoverySecret)
      setAvailNZBRecoverySecretCopied(true)
      window.setTimeout(() => setAvailNZBRecoverySecretCopied(false), 1500)
    } catch (error) {
      setAvailNZBStatusError(error?.message || 'Failed to copy AvailNZB recovery secret.')
    }
  }

  useEffect(() => {
    if (activeTab === 'general') {
      fetchAvailNZBStatus()
    }
  }, [activeTab])

  // Compute which tabs have validation errors from server
  const tabsWithErrors = useMemo(() => {
    const errs = saveStatus.errors
    if (!errs) return new Set()
    const tabs = new Set()
    Object.keys(errs).forEach((key) => tabs.add(fieldToTab(key)))
    return tabs
  }, [saveStatus.errors])

  // When errors arrive, switch to the first failing tab
  useEffect(() => {
    if (tabsWithErrors.size > 0 && !tabsWithErrors.has(activeTab)) {
      const firstErrorTab = TABS.find((t) => tabsWithErrors.has(t.id))
      if (firstErrorTab) setActiveTab(firstErrorTab.id)
    }
  }, [tabsWithErrors])

  const onSubmit = async (data) => {
    try {
      const trimData = (obj) => {
        if (typeof obj !== 'object' || obj === null) return obj;
        if (Array.isArray(obj)) {
          return obj.map(item => trimData(item));
        }
        const newObj = {};
        for (const key in obj) {
          if (typeof obj[key] === 'string') {
            newObj[key] = obj[key].trim();
          } else if (typeof obj[key] === 'object') {
            newObj[key] = trimData(obj[key]);
          } else {
            newObj[key] = obj[key];
          }
        }
        return newObj;
      };

      const trimmedData = trimData(data);
      if (typeof trimmedData.memory_limit_mb !== 'number') {
        trimmedData.memory_limit_mb = Number(trimmedData.memory_limit_mb) || 0
      }
      const keepLog = Number(trimmedData.keep_log_files)
      trimmedData.keep_log_files = Math.min(50, Math.max(1, isNaN(keepLog) ? 9 : keepLog))

      sendCommand('save_config', trimmedData)
      setHasUnsavedChanges(false)
      setInitialFormValues(JSON.stringify(trimmedData))
    } catch (error) {
      console.error('Error saving configuration:', error)
      setError('root', { message: 'Failed to save configuration: ' + error.message })
    }
  }

  if (loading) return null

  const availNZBRecoverySecret = availNZBStatus?.recovery_secret || ''
  const availNZBStatusMessage = availNZBStatusError || availNZBStatus?.status_error || ''
	const availNZBMode = watch('availnzb_mode') || ''
	const availNZBModeDoesNotImproveScore = availNZBMode === 'status_only' || availNZBMode === 'disabled'
  const rawAvailNZBTrustScore = Number(availNZBStatus?.status?.trust_score)
	const maxAvailNZBTrustScore = 60
  const availNZBTrustScore = Number.isFinite(rawAvailNZBTrustScore)
	  ? (Math.max(0, Math.min(maxAvailNZBTrustScore, rawAvailNZBTrustScore)) / maxAvailNZBTrustScore) * 100
    : null
	const availNZBTrustBlocksStatusResults = availNZBTrustScore !== null && availNZBTrustScore < 100
	const availNZBBadgeClass = availNZBTrustBlocksStatusResults
	  ? 'border-red-500/40 bg-red-500/10 px-2 py-0.5 text-[10px] font-semibold text-red-900 dark:text-red-200'
	  : availNZBModeDoesNotImproveScore
	    ? 'border-amber-500/40 bg-amber-500/10 px-2 py-0.5 text-[10px] font-semibold text-amber-900 dark:text-amber-200'
	    : 'px-2 py-0.5 text-[10px] font-semibold'
  const availNZBTrustSummary = availNZBStatusLoading
    ? 'Loading…'
    : availNZBTrustScore !== null
      ? `${Math.round(availNZBTrustScore)}% trust`
      : availNZBStatusMessage
        ? 'Status unavailable'
        : 'Trust unavailable'
  const availNZBTrustBarClass = availNZBTrustScore === null
    ? 'bg-muted-foreground/20'
    : availNZBTrustScore < 34
      ? 'bg-destructive'
      : availNZBTrustScore < 67
        ? 'bg-chart-4'
        : 'bg-primary'

  const errorCount = saveStatus.errors ? Object.keys(saveStatus.errors).length : 0
  const saveFooterMsg = saveStatus.type === 'error' && errorCount > 0
    ? `Validation failed — ${errorCount} field${errorCount > 1 ? 's' : ''} need${errorCount === 1 ? 's' : ''} attention`
    : saveStatus.msg

  const saveFooter = (
    <div className="sticky bottom-0 z-10 -mx-4 mt-3 pb-4 flex flex-col-reverse sm:flex-row items-stretch sm:items-center justify-between gap-2 border-t bg-background px-4 pt-3 sm:px-0 sm:pt-4">
      <div className={`flex items-center gap-1.5 text-sm min-w-0 ${saveStatus.type === 'error' ? 'text-destructive' : saveStatus.type === 'success' ? 'text-emerald-500' : 'text-muted-foreground'}`}>
        {saveStatus.type === 'error' && errorCount > 0 && <AlertTriangle className="h-4 w-4 shrink-0" />}
        {saveStatus.type === 'success' && <Check className="h-4 w-4 shrink-0" />}
        {saveFooterMsg}
      </div>
      <div className="flex shrink-0 justify-end">
        <Button type="submit" variant="default" onClick={handleSubmit(onSubmit)} disabled={isSaving || formState.isSubmitting} className="w-full sm:w-auto">
          {(isSaving || formState.isSubmitting) && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
          Save Changes
        </Button>
      </div>
    </div>
  )
  
  return (
    <Form {...form}>
      <form onSubmit={handleSubmit(onSubmit)}>
        {/* Tab bar */}
        <div className="flex items-center gap-1 border-b border-border mb-6 -mt-1 overflow-x-auto">
          {TABS.map((tab) => {
            const Icon = tab.icon
            const hasError = tabsWithErrors.has(tab.id)
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveTab(tab.id)}
                className={cn(
                  'relative flex items-center gap-1.5 px-3 py-2 text-sm font-medium whitespace-nowrap transition-colors',
                  'hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring rounded-t-md',
                  activeTab === tab.id
                    ? 'text-foreground after:absolute after:bottom-0 after:inset-x-0 after:h-0.5 after:bg-primary'
                    : 'text-muted-foreground',
                  hasError && 'text-destructive'
                )}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
                {hasError && (
                  <span className="flex h-2 w-2 rounded-full bg-destructive" />
                )}
              </button>
            )
          })}
        </div>

        {activeTab === 'general' && (
          <>
            <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle>Addon Settings</CardTitle>
                <CardDescription>Configure how the Stremio addon listens and is accessed.</CardDescription>
              </CardHeader>
              <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField control={control} name="addon_base_url" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Base URL</FormLabel>
                    <FormControl><Input placeholder="http://localhost:7000" {...field} /></FormControl>
                    <FormMessage />
                    <EnvOverrideNote show={envOverrides.includes('addon_base_url')} />
                  </FormItem>
                )} />
                <FormField control={control} name="addon_port" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Port (Requires Restart)</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={e => field.onChange(e.target.valueAsNumber)} /></FormControl>
                    <FormMessage />
                    <EnvOverrideNote show={envOverrides.includes('addon_port')} />
                  </FormItem>
                )} />
              </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>NNTP Proxy Server</CardTitle>
                <CardDescription>Allow other apps (SABnzbd, NZBGet) to use StreamNZB as a localized news server.</CardDescription>
                {(envOverrides.includes('proxy_port') || envOverrides.includes('proxy_host') || envOverrides.includes('proxy_auth_user') || envOverrides.includes('proxy_auth_pass')) && (
                  <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
                    <AlertTriangle className="h-3.5 w-3 shrink-0" />
                    Some settings overwritten by environment variables (NNTP_PROXY_*) on restart.
                  </p>
                )}
              </CardHeader>
              <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField control={control} name="proxy_host" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Bind Host</FormLabel>
                    <FormControl><Input placeholder="0.0.0.0" {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="proxy_port" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Port</FormLabel>
                    <FormControl><Input type="number" {...field} onChange={e => field.onChange(e.target.valueAsNumber)} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="proxy_auth_user" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Proxy Username</FormLabel>
                    <FormControl><Input {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="proxy_auth_pass" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Proxy Password</FormLabel>
                    <FormControl><PasswordInput {...field} /></FormControl>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Advanced</CardTitle>
                <CardDescription>Log level, cache, and validation.</CardDescription>
              </CardHeader>
              <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField control={control} name="log_level" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Log Level</FormLabel>
                    <FormControl>
                      <select className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50" {...field}>
                        <option value="DEBUG">DEBUG</option>
                        <option value="INFO">INFO</option>
                        <option value="WARN">WARN</option>
                        <option value="ERROR">ERROR</option>
                      </select>
                    </FormControl>
                    <FormMessage />
                    <EnvOverrideNote show={envOverrides.includes('log_level')} />
                  </FormItem>
                )} />
                <FormField control={control} name="memory_limit_mb" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Memory limit (MB)</FormLabel>
                    <FormControl>
                      <Input type="number" min={0} {...field} value={field.value ?? ''} onChange={e => { const v = e.target.value; field.onChange(v === '' ? 0 : Number(v) || 0) }} />
                    </FormControl>
                    <FormDescription>Soft limit on total process memory (0 = no limit). Segment cache uses 80% of this. Restart required.</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="keep_log_files" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Keep log files</FormLabel>
                    <FormControl>
                      <Input type="number" min={1} max={50} {...field} value={field.value ?? ''} onChange={e => { const v = e.target.value; field.onChange(v === '' ? 9 : Math.min(50, Math.max(1, Number(v) || 9))) }} />
                    </FormControl>
                    <FormDescription>Number of log files to keep (current streamnzb.log plus rotated archives). Oldest rotated logs are purged on restart.</FormDescription>
                    <EnvOverrideNote show={envOverrides.includes('keep_log_files')} />
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="availnzb_mode" render={({ field }) => (
                  <FormItem className="space-y-2 col-span-full">
                    <FormLabel className="flex items-start justify-between gap-3 text-sm font-medium">
                      <span>AvailNZB mode</span>
                      <span className="flex flex-col items-end gap-1">
		                        <TooltipProvider delayDuration={150}>
		                          <span className="flex flex-wrap items-center justify-end gap-2">
		                            <Tooltip>
		                              <TooltipTrigger asChild>
		                                <span className="inline-flex cursor-help">
		                                  <Badge variant="outline" className={availNZBBadgeClass}>
		                                    {availNZBTrustBlocksStatusResults || availNZBModeDoesNotImproveScore ? (
		                                      <AlertTriangle className="mr-1 h-3 w-3" />
		                                    ) : (
		                                      <Info className="mr-1 h-3 w-3" />
		                                    )}
		                                  </Badge>
		                                </span>
		                              </TooltipTrigger>
		                              <TooltipContent side="bottom" align="end" className="max-w-xs p-3">
		                                <div className="space-y-1 text-xs font-normal">
		                                  <div className="font-medium">
		                                    {availNZBTrustBlocksStatusResults ? 'AvailNZB /status trust requirement' : 'AvailNZB scoring'}
		                                  </div>
		                                  {availNZBTrustBlocksStatusResults && (
		                                    <>
		                                      <div>AvailNZB /status availability results require 100 trust.</div>
		                                      <div>Below that, /status will not return available results.</div>
		                                    </>
		                                  )}
		                                  <div>Only "GET status + POST report" increases your AvailNZB score.</div>
		                                  <div>"GET status only" fetches availability data but does not report your playback results back to the community.</div>
		                                  <div>"Disabled" skips AvailNZB entirely.</div>
		                                </div>
		                              </TooltipContent>
		                            </Tooltip>

		                            <span className={`flex items-center gap-1 text-xs font-normal ${availNZBStatusError ? 'text-destructive/80' : 'text-muted-foreground'}`}>
                          {availNZBStatusLoading && <Loader2 className="h-3 w-3 animate-spin" />}
                          {availNZBTrustSummary}
                        </span>
		                          </span>
		                        </TooltipProvider>
                        {availNZBTrustScore !== null && !availNZBStatusLoading && !availNZBStatusError && (
                          <span className="h-1.5 w-20 overflow-hidden rounded-full bg-muted/70" aria-hidden="true">
                            <span
                              className={`block h-full rounded-full transition-all ${availNZBTrustBarClass}`}
                              style={{ width: `${availNZBTrustScore}%` }}
                            />
                          </span>
                        )}
                      </span>
	                    </FormLabel>
                    <FormControl>
                      <select className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50" {...field}>
                        <option value="">GET status + POST report</option>
                        <option value="status_only">GET status only</option>
                        <option value="disabled">Disabled</option>
                      </select>
                    </FormControl>
                    <FormDescription>
                      Controls how StreamNZB interacts with AvailNZB.
                    </FormDescription>
	                    {availNZBStatusMessage && (
	                      <p className="text-xs text-destructive/80">{availNZBStatusMessage}</p>
	                    )}
	                    {availNZBRecoverySecret && (
	                      <div className="rounded-md border border-border bg-muted/30 p-3 space-y-2">
	                        <div className="flex items-center justify-between gap-2">
	                          <span className="text-sm font-medium">Recovery secret</span>
	                          <span className="text-xs text-muted-foreground">
	                            {availNZBRecoverySecretCopied ? 'Copied' : 'Reveal to copy'}
	                          </span>
	                        </div>
	                        <div className="flex items-center gap-2">
	                          <Input
	                            type={showAvailNZBRecoverySecret ? 'text' : 'password'}
	                            value={availNZBRecoverySecret}
	                            readOnly
	                            className="font-mono"
	                          />
	                          <Button
	                            type="button"
	                            variant="outline"
	                            size="icon"
	                            aria-label={showAvailNZBRecoverySecret ? 'Hide AvailNZB recovery secret' : 'Show AvailNZB recovery secret'}
	                            onClick={() => setShowAvailNZBRecoverySecret((current) => !current)}
	                          >
	                            {showAvailNZBRecoverySecret ? <EyeOff /> : <Eye />}
	                          </Button>
	                          <Button
	                            type="button"
	                            variant="outline"
	                            size="icon"
	                            aria-label="Copy AvailNZB recovery secret"
	                            onClick={handleCopyAvailNZBRecoverySecret}
	                          >
	                            {availNZBRecoverySecretCopied ? <Check /> : <Copy />}
	                          </Button>
	                        </div>
	                        <p className="text-xs text-muted-foreground">
	                          Save this recovery secret somewhere safe. You can use it to recover the AvailNZB app key later.
	                        </p>
	                      </div>
	                    )}
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              </CardContent>
            </Card>
            </div>
          </>
        )}

        {activeTab === 'indexers' && (
          <div className="space-y-4">
            {envOverrides.includes('indexers') && (
              <div className="rounded-lg border border-border bg-muted/50 p-3">
                <p className="text-sm text-muted-foreground flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 shrink-0" />
                  Indexer list is overwritten by environment variables (INDEXER_1_*, etc.) on restart.
                </p>
              </div>
            )}
            <IndexerSettings
              control={control}
              indexerFields={indexerFields}
              appendIndexer={appendIndexer}
              removeIndexer={removeIndexer}
              watch={watch}
              setValue={setValue}
              indexerCaps={indexerCaps || {}}
            />
          </div>
        )}

        {activeTab === 'providers' && (
          <div className="space-y-4">
            {envOverrides.includes('providers') && (
              <div className="rounded-lg border border-border bg-muted/50 p-3">
                <p className="text-sm text-muted-foreground flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 shrink-0" />
                  Provider list is overwritten by environment variables (PROVIDER_1_*, etc.) on restart.
                </p>
              </div>
            )}
            <ProviderSettings
              control={control}
              fields={fields}
              append={append}
              remove={remove}
              watch={watch}
            />
          </div>
        )}

        {activeTab === 'devices' && (
          <div className="space-y-4">
            <DeviceManagement
              sendCommand={sendCommand}
              globalConfig={initialConfig}
            />
          </div>
        )}

        {activeTab !== 'devices' && saveFooter}
      </form>
    </Form>
  )
}

export default Settings
