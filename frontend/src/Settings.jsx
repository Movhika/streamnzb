import React, { useEffect, useState, useRef } from 'react'
import { useForm, useFieldArray } from 'react-hook-form'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from "@/components/ui/form"
import { PasswordInput } from "@/components/ui/password-input"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Loader2, Info, AlertTriangle } from "lucide-react"
import { FiltersSection } from "@/components/FiltersSection"
import { SortingSection } from "@/components/SortingSection"
import { IndexerSettings } from "@/components/IndexerSettings"
import { ProviderSettings } from "@/components/ProviderSettings"
import DeviceManagement from "@/components/DeviceManagement"

function EnvOverrideNote({ show }) {
  if (!show) return null
  return (
    <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
      <AlertTriangle className="h-3.5 w-3 shrink-0" />
      Overwritten by environment variable on restart.
    </p>
  )
}

function Settings({ initialConfig, sendCommand, saveStatus, isSaving, adminToken, activePage, indexerCaps }) {
  const [loading, setLoading] = useState(!initialConfig)
  const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false)
  const [initialFormValues, setInitialFormValues] = useState(null)
  const deviceManagementRef = useRef(null)
  const pendingSaveAfterPasswordRef = useRef(null)

  const form = useForm({
    defaultValues: {
      admin_username: 'admin',
      admin_password: '',
      addon_port: 7000,
      addon_base_url: '',
      log_level: 'INFO',
      proxy_port: 119,
      proxy_host: '',
      proxy_auth_user: '',
      proxy_auth_pass: '',
      cache_ttl_seconds: 300,
      validation_sample_size: 5,
      max_streams: 5,
      max_streams_per_resolution: 0,
      search_result_limit: 1000,
      include_year_in_search: true,
      search_title_language: '',
      search_title_normalize: false,
      movie_categories: '',
      tv_categories: '',
      extra_search_terms: '',
      use_season_episode_params: undefined,
      providers: [],
      indexers: [],
      filters: {
        allowed_qualities: [],
        blocked_qualities: [],
        min_resolution: '',
        max_resolution: '',
        allowed_codecs: [],
        blocked_codecs: [],
        required_audio: [],
        allowed_audio: [],
        min_channels: '',
        require_hdr: false,
        allowed_hdr: [],
        blocked_hdr: [],
        block_sdr: false,
        required_languages: [],
        allowed_languages: [],
        block_dubbed: false,
        block_cam: false,
        require_proper: false,
        allow_repack: true,
        block_hardcoded: false,
        min_bit_depth: '',
        min_size_gb: 0,
        max_size_gb: 0,
        blocked_groups: []
      },
      sorting: {
        resolution_weights: {
          '4k': 4000000,
          '1080p': 3000000,
          '720p': 2000000,
          'sd': 1000000
        },
        codec_weights: {
          'HEVC': 1000,
          'x265': 1000,
          'x264': 500,
          'AVC': 500
        },
        audio_weights: {
          'Atmos': 1500,
          'TrueHD': 1200,
          'DTS-HD': 1000,
          'DTS-X': 1000,
          'DTS': 500,
          'DD+': 400,
          'DD': 300,
          'AC3': 200,
          '5.1': 500,
          '7.1': 1000
        },
        quality_weights: {
          'BluRay': 2000,
          'WEB-DL': 1500,
          'WEBRip': 1200,
          'HDTV': 1000,
          'Blu-ray': 2000
        },
        visual_tag_weights: {
          'DV': 1500,
          'HDR10+': 1200,
          'HDR': 1000,
          '3D': 800
        },
        grab_weight: 0.5,
        age_weight: 1.0,
        preferred_groups: [],
        preferred_languages: []
      }
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
      const defaultSorting = {
        resolution_weights: { '4k': 4000000, '1080p': 3000000, '720p': 2000000, 'sd': 1000000 },
        codec_weights: { 'HEVC': 1000, 'x265': 1000, 'x264': 500, 'AVC': 500 },
        audio_weights: { 'Atmos': 1500, 'TrueHD': 1200, 'DTS-HD': 1000, 'DTS-X': 1000, 'DTS': 500, 'DD+': 400, 'DD': 300, 'AC3': 200, '5.1': 500, '7.1': 1000 },
        quality_weights: { 'BluRay': 2000, 'WEB-DL': 1500, 'WEBRip': 1200, 'HDTV': 1000, 'Blu-ray': 2000 },
        visual_tag_weights: { 'DV': 1500, 'HDR10+': 1200, 'HDR': 1000, '3D': 800 },
        grab_weight: 0.5, age_weight: 1.0, preferred_groups: [], preferred_languages: []
      }

      const defaultFilters = {
        allowed_qualities: [], blocked_qualities: [], min_resolution: '', max_resolution: '',
        allowed_codecs: [], blocked_codecs: [], required_audio: [], allowed_audio: [],
        min_channels: '', require_hdr: false, allowed_hdr: [], blocked_hdr: [], block_sdr: false,
        required_languages: [], allowed_languages: [], block_dubbed: false, block_cam: false,
        require_proper: false, allow_repack: true, block_hardcoded: false, min_bit_depth: '',
        min_size_gb: 0, max_size_gb: 0, blocked_groups: []
      }

      const { env_overrides: _envOverrides, ...configForForm } = initialConfig
      const formattedData = {
        ...configForForm,
        admin_username: initialConfig.admin_username || 'admin',
        admin_password: '',
        addon_port: Number(initialConfig.addon_port),
        proxy_port: Number(initialConfig.proxy_port),
        cache_ttl_seconds: Number(initialConfig.cache_ttl_seconds),
        validation_sample_size: Number(initialConfig.validation_sample_size),
        max_concurrent_validations: Number(initialConfig.max_concurrent_validations),
        max_streams_per_resolution: Number(initialConfig.max_streams_per_resolution || 0),
        search_result_limit: Number(initialConfig.search_result_limit || 1000),
        include_year_in_search: initialConfig.include_year_in_search !== false,
        search_title_language: initialConfig.search_title_language ?? '',
        search_title_normalize: initialConfig.search_title_normalize === true,
        movie_categories: initialConfig.movie_categories ?? '',
        tv_categories: initialConfig.tv_categories ?? '',
        extra_search_terms: initialConfig.extra_search_terms ?? '',
        use_season_episode_params: initialConfig.use_season_episode_params,
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
          username: idx.username || '',
          password: idx.password || ''
        })) || [],
        sorting: {
          ...defaultSorting,
          ...(initialConfig.sorting || {}),
          resolution_weights: { ...defaultSorting.resolution_weights, ...(initialConfig.sorting?.resolution_weights || {}) },
          codec_weights: { ...defaultSorting.codec_weights, ...(initialConfig.sorting?.codec_weights || {}) },
          audio_weights: { ...defaultSorting.audio_weights, ...(initialConfig.sorting?.audio_weights || {}) },
          quality_weights: { ...defaultSorting.quality_weights, ...(initialConfig.sorting?.quality_weights || {}) },
          visual_tag_weights: { ...defaultSorting.visual_tag_weights, ...(initialConfig.sorting?.visual_tag_weights || {}) },
          grab_weight: initialConfig.sorting?.grab_weight ?? defaultSorting.grab_weight,
          age_weight: initialConfig.sorting?.age_weight ?? defaultSorting.age_weight,
          preferred_groups: initialConfig.sorting?.preferred_groups || defaultSorting.preferred_groups,
          preferred_languages: initialConfig.sorting?.preferred_languages || defaultSorting.preferred_languages
        },
        filters: {
          ...defaultFilters,
          ...(initialConfig.filters || {}),
          allowed_qualities: initialConfig.filters?.allowed_qualities || defaultFilters.allowed_qualities,
          blocked_qualities: initialConfig.filters?.blocked_qualities || defaultFilters.blocked_qualities,
          allowed_codecs: initialConfig.filters?.allowed_codecs || defaultFilters.allowed_codecs,
          blocked_codecs: initialConfig.filters?.blocked_codecs || defaultFilters.blocked_codecs,
          required_audio: initialConfig.filters?.required_audio || defaultFilters.required_audio,
          allowed_audio: initialConfig.filters?.allowed_audio || defaultFilters.allowed_audio,
          required_languages: initialConfig.filters?.required_languages || defaultFilters.required_languages,
          allowed_languages: initialConfig.filters?.allowed_languages || defaultFilters.allowed_languages,
          allowed_hdr: initialConfig.filters?.allowed_hdr || defaultFilters.allowed_hdr,
          blocked_hdr: initialConfig.filters?.blocked_hdr || defaultFilters.blocked_hdr,
          blocked_groups: initialConfig.filters?.blocked_groups || defaultFilters.blocked_groups
        }
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

  const onSubmit = async (data) => {
    try {
      if (deviceManagementRef.current) {
        const deviceConfigs = deviceManagementRef.current.getDeviceConfigs()
        if (Object.keys(deviceConfigs).length > 0) {
          const configsToSave = {}
          const adminUsername = data.admin_username || 'admin'
          for (const [username, deviceConfig] of Object.entries(deviceConfigs)) {
            if (username === adminUsername || !deviceConfig) continue
            configsToSave[username] = {
              filters: deviceConfig.filters || {},
              sorting: deviceConfig.sorting || {},
              indexer_overrides: deviceConfig.indexer_overrides ?? {}
            }
          }
          if (Object.keys(configsToSave).length > 0) {
            sendCommand('save_user_configs', configsToSave)
          }
        }
      }
      
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
      const newPassword = trimmedData.admin_password
      delete trimmedData.admin_password

      if (newPassword) {
        pendingSaveAfterPasswordRef.current = trimmedData
        const prevCb = window.deviceActionCallback
        window.deviceActionCallback = (payload) => {
          const isPasswordResponse = payload.success !== undefined || (payload.error && !payload.token && !payload.user)
          if (isPasswordResponse) {
            if (prevCb) prevCb(payload)
            if (payload.error) {
              setError('admin_password', { type: 'server', message: payload.error })
            } else {
              setValue('admin_password', '')
              sendCommand('save_config', pendingSaveAfterPasswordRef.current)
              setHasUnsavedChanges(false)
              setInitialFormValues(JSON.stringify(pendingSaveAfterPasswordRef.current))
            }
            pendingSaveAfterPasswordRef.current = null
            window.deviceActionCallback = prevCb || undefined
            if (!prevCb) delete window.deviceActionCallback
            return
          }
          if (prevCb) prevCb(payload)
        }
        sendCommand('update_password', { username: trimmedData.admin_username || 'admin', password: newPassword })
        return
      }

      sendCommand('save_config', trimmedData)
      setHasUnsavedChanges(false)
      setInitialFormValues(JSON.stringify(trimmedData))
    } catch (error) {
      console.error('Error saving configuration:', error)
      setError('root', { message: 'Failed to save configuration: ' + error.message })
    }
  }

  if (loading) return null

  const saveFooter = (
    <div className="sticky bottom-0 z-10 -mx-4 mt-3 pb-4 flex flex-col-reverse sm:flex-row items-stretch sm:items-center justify-between gap-2 border-t bg-background px-4 pt-3 sm:px-0 sm:pt-4">
      <div className={`flex items-center text-sm min-w-0 ${saveStatus.type === 'error' ? 'text-destructive' : saveStatus.type === 'success' ? 'text-primary' : 'text-muted-foreground'}`}>
        {saveStatus.msg}
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
        {activePage === 'general' && (
          <div className="space-y-8">
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
                <CardTitle>Dashboard credentials</CardTitle>
                <CardDescription>Username and password to log in to this dashboard. Save to apply. Leave password blank to keep the current one.</CardDescription>
              </CardHeader>
              <CardContent>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <FormField control={control} name="admin_username" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">Username</FormLabel>
                    <FormControl><Input placeholder="admin" {...field} /></FormControl>
                    <FormMessage />
                    <EnvOverrideNote show={envOverrides.includes('admin_username')} />
                  </FormItem>
                )} />
                <FormField control={control} name="admin_password" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium">New password</FormLabel>
                    <FormControl><PasswordInput placeholder="Leave blank to keep current" {...field} /></FormControl>
                    <FormMessage />
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
                <CardDescription>Log level, cache, validation, and stream limits.</CardDescription>
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
                <FormField control={control} name="cache_ttl_seconds" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium flex items-center gap-2">
                      Cache TTL (seconds)
                      <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-4 w-4 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent className="max-w-xs"><p>How long to cache validation results.</p></TooltipContent></Tooltip></TooltipProvider>
                    </FormLabel>
                    <FormControl><Input type="number" min="60" max="3600" {...field} onChange={e => field.onChange(e.target.valueAsNumber)} /></FormControl>
                    <FormDescription>Cache duration in seconds (default: 300)</FormDescription>
                    <FormMessage />
                    <EnvOverrideNote show={envOverrides.includes('cache_ttl_seconds')} />
                  </FormItem>
                )} />
                <FormField control={control} name="validation_sample_size" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium flex items-center gap-2">
                      Validation Sample Size
                      <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-4 w-4 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent className="max-w-xs"><p>Number of segments to check per NZB. Higher = more accurate, slower.</p></TooltipContent></Tooltip></TooltipProvider>
                    </FormLabel>
                    <FormControl><Input type="number" min="1" max="20" {...field} onChange={e => field.onChange(e.target.valueAsNumber)} /></FormControl>
                    <FormDescription>Segments to check per NZB (default: 5)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="max_streams" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium flex items-center gap-2">
                      Max Streams
                      <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-4 w-4 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent className="max-w-xs"><p>Maximum streams to return per search.</p></TooltipContent></Tooltip></TooltipProvider>
                    </FormLabel>
                    <FormControl><Input type="number" min="1" max="20" {...field} onChange={e => field.onChange(e.target.valueAsNumber)} /></FormControl>
                    <FormDescription>Number of streams to return (default: 6)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
                <FormField control={control} name="max_streams_per_resolution" render={({ field }) => (
                  <FormItem className="space-y-2">
                    <FormLabel className="text-sm font-medium flex items-center gap-2">
                      Max Streams per Resolution
                      <TooltipProvider><Tooltip><TooltipTrigger asChild><Info className="h-4 w-4 text-muted-foreground cursor-help" /></TooltipTrigger><TooltipContent className="max-w-xs"><p>Limit streams per resolution. 0 = disabled.</p></TooltipContent></Tooltip></TooltipProvider>
                    </FormLabel>
                    <FormControl><Input type="number" min="0" max="10" {...field} onChange={e => field.onChange(e.target.valueAsNumber)} /></FormControl>
                    <FormDescription>Max streams per resolution (0 = disabled, default: 0)</FormDescription>
                    <FormMessage />
                  </FormItem>
                )} />
              </div>
              </CardContent>
            </Card>
            </div>
          </div>
        )}

        {activePage === 'indexers' && (
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

        {activePage === 'providers' && (
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

        {activePage === 'filters' && (
          <div className="space-y-4">
            <FiltersSection control={control} watch={form.watch} />
          </div>
        )}

        {activePage === 'sorting' && (
          <div className="space-y-4">
            <SortingSection control={control} watch={form.watch} />
          </div>
        )}

        {activePage === 'devices' && (
          <div className="space-y-4">
            <DeviceManagement
              ref={deviceManagementRef}
              globalFilters={getValues('filters')}
              globalSorting={getValues('sorting')}
              sendCommand={sendCommand}
              ws={window.ws}
            />
          </div>
        )}

        {saveFooter}
      </form>
    </Form>
  )
}

export default Settings
