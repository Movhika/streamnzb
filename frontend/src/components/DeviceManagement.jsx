import React, { useState, useEffect, useImperativeHandle, forwardRef, useCallback } from 'react'
import { useForm } from 'react-hook-form'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { AlertCircle, Plus, Trash2, RefreshCw, Copy, Check, Loader2, Settings, ChevronDown, ChevronUp } from "lucide-react"
import { Checkbox } from "@/components/ui/checkbox"
import { FiltersSection } from "@/components/FiltersSection"
import { SortingSection } from "@/components/SortingSection"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { FormField } from "@/components/ui/form"

// Build API override object from form values for one indexer (only include overridden fields)
function buildOverridePayload(o) {
  if (!o) return undefined
  const out = {}
  if (typeof o.search_result_limit === 'number' && o.search_result_limit > 0) out.search_result_limit = o.search_result_limit
  if (o.include_year_in_search !== 'use_indexer' && o.include_year_in_search !== undefined) out.include_year_in_search = o.include_year_in_search === true || o.include_year_in_search === 'true'
  if (o.search_title_language !== 'use_indexer' && o.search_title_language !== undefined && o.search_title_language !== '') out.search_title_language = o.search_title_language
  if (o.search_title_normalize !== 'use_indexer' && o.search_title_normalize !== undefined) out.search_title_normalize = o.search_title_normalize === true || o.search_title_normalize === 'true'
  if (o.movie_categories !== 'use_indexer' && o.movie_categories !== undefined && o.movie_categories !== '') out.movie_categories = o.movie_categories
  if (o.tv_categories !== 'use_indexer' && o.tv_categories !== undefined && o.tv_categories !== '') out.tv_categories = o.tv_categories
  if (o.extra_search_terms !== 'use_indexer' && o.extra_search_terms !== undefined && o.extra_search_terms !== '') out.extra_search_terms = o.extra_search_terms
  if (o.use_season_episode_params !== 'use_indexer' && o.use_season_episode_params !== undefined) out.use_season_episode_params = o.use_season_episode_params === true || o.use_season_episode_params === 'true'
  return Object.keys(out).length ? out : undefined
}

// Device config form: filters, sorting, and per-indexer overrides (indexer_overrides map)
function DeviceConfigForm({ username, initialFilters, initialSorting, initialIndexerOverrides, indexerNames, onConfigChange, formRef }) {
  const defaultOverrides = React.useMemo(() => {
    const map = {}
    ;(indexerNames || []).forEach((name) => {
      const init = initialIndexerOverrides?.[name] || {}
      map[name] = {
        search_result_limit: init.search_result_limit ?? 0,
        include_year_in_search: init.include_year_in_search === undefined ? 'use_indexer' : (init.include_year_in_search === true || init.include_year_in_search === 'true'),
        search_title_language: init.search_title_language === undefined || init.search_title_language === '' ? 'use_indexer' : (init.search_title_language ?? ''),
        search_title_normalize: init.search_title_normalize === undefined ? 'use_indexer' : (init.search_title_normalize === true || init.search_title_normalize === 'true'),
        movie_categories: init.movie_categories === undefined || init.movie_categories === '' ? 'use_indexer' : (init.movie_categories ?? ''),
        tv_categories: init.tv_categories === undefined || init.tv_categories === '' ? 'use_indexer' : (init.tv_categories ?? ''),
        extra_search_terms: init.extra_search_terms === undefined || init.extra_search_terms === '' ? 'use_indexer' : (init.extra_search_terms ?? ''),
        use_season_episode_params: init.use_season_episode_params === undefined ? 'use_indexer' : (init.use_season_episode_params === true || init.use_season_episode_params === 'true')
      }
    })
    return map
  }, [initialIndexerOverrides, indexerNames])

  const form = useForm({
    defaultValues: {
      filters: initialFilters || {},
      sorting: initialSorting || {},
      indexer_overrides: defaultOverrides
    }
  })

  const { watch, reset, getValues, control } = form
  const onConfigChangeRef = React.useRef(onConfigChange)
  const isInitialMount = React.useRef(true)

  React.useImperativeHandle(formRef, formRef ? () => ({
    getValues: () => {
      const all = getValues()
      const filters = all.filters || {}
      const sorting = all.sorting || {}
      const overrides = all.indexer_overrides || {}
      const indexer_overrides = {}
      Object.keys(overrides).forEach((name) => {
        const payload = buildOverridePayload(overrides[name])
        if (payload) indexer_overrides[name] = payload
      })
      return { filters: { ...filters }, sorting: { ...sorting }, indexer_overrides }
    }
  }) : undefined, [getValues, username, formRef])

  useEffect(() => { onConfigChangeRef.current = onConfigChange }, [onConfigChange])

  useEffect(() => {
    if (initialFilters || initialSorting || initialIndexerOverrides || indexerNames?.length) {
      reset({
        filters: initialFilters || {},
        sorting: initialSorting || {},
        indexer_overrides: defaultOverrides
      }, { keepDefaultValues: false })
      isInitialMount.current = true
    }
  }, [initialFilters, initialSorting, initialIndexerOverrides, indexerNames, defaultOverrides, reset, username])

  useEffect(() => {
    let timeoutId
    const sub = watch((value) => {
      if (isInitialMount.current) { isInitialMount.current = false; return }
      if (value?.filters || value?.sorting) {
        clearTimeout(timeoutId)
        timeoutId = setTimeout(() => onConfigChangeRef.current(username, value), 100)
      }
    })
    return () => { clearTimeout(timeoutId); sub.unsubscribe() }
  }, [watch, username])

  return (
    <div className="space-y-4">
      <Tabs defaultValue="filters" className="w-full">
        <TabsList className="mb-6 w-full grid grid-cols-3">
          <TabsTrigger value="filters" className="text-sm sm:text-base">Filters</TabsTrigger>
          <TabsTrigger value="sorting" className="text-sm sm:text-base">Sorting</TabsTrigger>
          <TabsTrigger value="indexers" className="text-sm sm:text-base">Indexer & Search</TabsTrigger>
        </TabsList>
        <TabsContent value="filters">
          <FiltersSection control={form.control} watch={watch} fieldPrefix="filters" />
        </TabsContent>
        <TabsContent value="sorting">
          <SortingSection control={form.control} watch={watch} fieldPrefix="sorting" />
        </TabsContent>
        <TabsContent value="indexers" className="space-y-4">
          <p className="text-xs text-muted-foreground">Override indexer search settings per indexer for this device. Empty = use indexer setting.</p>
          {(!indexerNames || indexerNames.length === 0) ? (
            <p className="text-sm text-muted-foreground">No indexers configured. Add indexers in Settings → Indexers.</p>
          ) : (
            indexerNames.map((indexerName) => (
              <PerIndexerOverrideSection
                key={indexerName}
                indexerName={indexerName}
                control={control}
              />
            ))
          )}
        </TabsContent>
      </Tabs>
    </div>
  )
}

function PerIndexerOverrideSection({ indexerName, control }) {
  const [open, setOpen] = useState(false)
  const prefix = `indexer_overrides.${indexerName}`

  return (
    <div className="rounded-lg border border-border bg-muted/30 overflow-hidden">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full px-3 py-2.5 text-left text-sm font-medium hover:bg-muted/50 transition-colors"
      >
        <ChevronDown className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} />
        {indexerName}
      </button>
      {open && (
        <div className="px-3 pb-3 pt-1 space-y-3 border-t border-border">
          <div className="grid grid-cols-2 gap-2">
            <FormField control={control} name={`${prefix}.movie_categories`} render={({ field }) => (
              <div className="space-y-1">
                <Label className="text-xs">Movie Categories</Label>
                <Input type="text" placeholder="Use indexer setting" className="h-8 text-xs" {...field} value={field.value === 'use_indexer' ? '' : (field.value ?? '')} onChange={e => field.onChange(e.target.value === '' ? 'use_indexer' : e.target.value)} />
              </div>
            )} />
            <FormField control={control} name={`${prefix}.tv_categories`} render={({ field }) => (
              <div className="space-y-1">
                <Label className="text-xs">TV Categories</Label>
                <Input type="text" placeholder="Use indexer setting" className="h-8 text-xs" {...field} value={field.value === 'use_indexer' ? '' : (field.value ?? '')} onChange={e => field.onChange(e.target.value === '' ? 'use_indexer' : e.target.value)} />
              </div>
            )} />
          </div>
          <FormField control={control} name={`${prefix}.extra_search_terms`} render={({ field }) => (
            <div className="space-y-1">
              <Label className="text-xs">Extra Search Terms</Label>
              <Input type="text" placeholder="Use indexer setting" className="h-8 text-xs" {...field} value={field.value === 'use_indexer' ? '' : (field.value ?? '')} onChange={e => field.onChange(e.target.value === '' ? 'use_indexer' : e.target.value)} />
            </div>
          )} />
          <FormField control={control} name={`${prefix}.use_season_episode_params`} render={({ field }) => (
            <div className="flex items-center gap-2">
              <select className="flex h-8 w-full max-w-[180px] rounded-md border border-input bg-transparent px-2 text-xs" value={field.value === true ? 'true' : field.value === false ? 'false' : 'use_indexer'} onChange={e => { const v = e.target.value; field.onChange(v === 'use_indexer' ? 'use_indexer' : v === 'true') }}>
                <option value="use_indexer">Use indexer setting</option>
                <option value="true">Yes</option>
                <option value="false">No</option>
              </select>
              <Label className="text-xs">Use season/episode in API</Label>
            </div>
          )} />
          <FormField control={control} name={`${prefix}.search_result_limit`} render={({ field }) => (
            <div className="space-y-1">
              <Label className="text-xs">Search Result Limit</Label>
              <Input type="number" min={0} max={5000} placeholder="0 = use indexer" className="h-8 text-xs" {...field} value={field.value === 0 ? '' : field.value} onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))} />
            </div>
          )} />
          <FormField control={control} name={`${prefix}.include_year_in_search`} render={({ field }) => (
            <div className="flex items-center gap-2">
              <select className="flex h-8 w-full max-w-[180px] rounded-md border border-input bg-transparent px-2 text-xs" value={field.value === true ? 'true' : field.value === false ? 'false' : 'use_indexer'} onChange={e => { const v = e.target.value; field.onChange(v === 'use_indexer' ? 'use_indexer' : v === 'true') }}>
                <option value="use_indexer">Use indexer setting</option>
                <option value="true">Yes</option>
                <option value="false">No</option>
              </select>
              <Label className="text-xs">Include year in movie search</Label>
            </div>
          )} />
          <FormField control={control} name={`${prefix}.search_title_language`} render={({ field }) => (
            <div className="space-y-1">
              <Label className="text-xs">Search title language</Label>
              <Input type="text" placeholder="Use indexer setting" className="h-8 text-xs" {...field} value={field.value === 'use_indexer' ? '' : (field.value ?? '')} onChange={e => field.onChange(e.target.value === '' ? 'use_indexer' : e.target.value)} />
            </div>
          )} />
          <FormField control={control} name={`${prefix}.search_title_normalize`} render={({ field }) => (
            <div className="flex items-center gap-2">
              <select className="flex h-8 w-full max-w-[180px] rounded-md border border-input bg-transparent px-2 text-xs" value={field.value === true ? 'true' : field.value === false ? 'false' : 'use_indexer'} onChange={e => { const v = e.target.value; field.onChange(v === 'use_indexer' ? 'use_indexer' : v === 'true') }}>
                <option value="use_indexer">Use indexer setting</option>
                <option value="true">Yes</option>
                <option value="false">No</option>
              </select>
              <Label className="text-xs">Normalize title for search</Label>
            </div>
          )} />
        </div>
      )}
    </div>
  )
}

const DeviceManagement = forwardRef(function DeviceManagement({ globalFilters, globalSorting, sendCommand, ws }, ref) {
  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(true) // Start as true to show loader initially
  const [actionLoading, setActionLoading] = useState(null) // Track which action is loading
  const [addDeviceLoading, setAddDeviceLoading] = useState(false) // Separate loading for add device dialog
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [expandedDevice, setExpandedDevice] = useState(null)
  const [newUsername, setNewUsername] = useState('')
  const [copiedToken, setCopiedToken] = useState('')
  const [globalConfig, setGlobalConfig] = useState(null)
  
  // Store device configs - keyed by username
  const [deviceConfigs, setDeviceConfigs] = useState({})
  // Store form refs for each device so we can get current values directly
  const formRefs = React.useRef({})
  // Track if initial load has happened
  const hasLoadedRef = React.useRef(false)

  // Expose getDeviceConfigs to parent via ref
  useImperativeHandle(ref, () => ({
    getDeviceConfigs: () => {
      // Get current values directly from forms to ensure we have the latest data
      const configs = {}
      
      for (const [username, formRef] of Object.entries(formRefs.current)) {
        if (formRef?.current?.getValues) {
          const formValues = formRef.current.getValues()
          if (formValues && (formValues.filters || formValues.sorting || formValues.indexer_overrides)) {
            configs[username] = formValues
          }
        } else if (deviceConfigs[username]) {
          configs[username] = deviceConfigs[username]
        }
      }
      return configs
    },
    getUserConfigs: () => {
      const configs = {}
      for (const [username, formRef] of Object.entries(formRefs.current)) {
        if (formRef?.current?.getValues) {
          const formValues = formRef.current.getValues()
          if (formValues && (formValues.filters || formValues.sorting || formValues.indexer_overrides)) {
            configs[username] = formValues
          }
        } else if (deviceConfigs[username]) {
          configs[username] = deviceConfigs[username]
        }
      }
      return configs
    }
  }))

  // Fetch devices list
  const fetchDevices = useCallback((showLoader = true) => {
    if (!sendCommand || !ws || ws.readyState !== WebSocket.OPEN) {
      setError('WebSocket not connected')
      if (showLoader) {
        setLoading(false)
      }
      return
    }

    if (showLoader) {
      setLoading(true)
    }
    setError('')
    
    // Clean up any existing callback
    if (window.deviceManagementCallback) {
      delete window.deviceManagementCallback
    }
    
    window.deviceManagementCallback = (payload) => {
      if (payload.error) {
        setError(payload.error)
        if (showLoader) {
          setLoading(false)
        }
        delete window.deviceManagementCallback
        return
      }
      setDevices(payload)
      if (showLoader) {
        setLoading(false)
      }
      hasLoadedRef.current = true
      delete window.deviceManagementCallback
    }

    sendCommand('get_users', {})
  }, [sendCommand, ws])

  // Fetch global config
  const fetchGlobalConfig = useCallback(() => {
    if (sendCommand && ws && ws.readyState === WebSocket.OPEN) {
      // Clean up any existing callback
      if (window.globalConfigCallback) {
        delete window.globalConfigCallback
      }
      window.globalConfigCallback = (config) => {
        setGlobalConfig(config)
        delete window.globalConfigCallback
      }
      sendCommand('get_config', {})
    }
  }, [sendCommand, ws])

  // Initial load - only run once when component mounts
  useEffect(() => {
    if (!hasLoadedRef.current && ws && ws.readyState === WebSocket.OPEN) {
      fetchDevices(true)
      fetchGlobalConfig()
    }
  }, []) // Empty deps - only run on mount

  // Also fetch when WebSocket becomes available
  useEffect(() => {
    if (!hasLoadedRef.current && ws && ws.readyState === WebSocket.OPEN && sendCommand) {
      fetchDevices(true)
      fetchGlobalConfig()
    }
  }, [ws?.readyState, sendCommand]) // Only depend on WebSocket state

  // Track loaded devices to avoid re-fetching
  const loadedDevicesRef = React.useRef(new Set())

  // Load device config when expanded
  const loadDeviceConfig = useCallback((username) => {
    if (loadedDevicesRef.current.has(username)) {
      return // Already loaded
    }

    if (!sendCommand || !ws || ws.readyState !== WebSocket.OPEN) {
      const defaultConfig = {
        filters: globalFilters || globalConfig?.filters || {},
        sorting: globalSorting || globalConfig?.sorting || {},
        indexer_overrides: {}
      }
      setDeviceConfigs(prev => ({ ...prev, [username]: defaultConfig }))
      loadedDevicesRef.current.add(username)
      if (!formRefs.current[username]) formRefs.current[username] = React.createRef()
      return
    }

    window.deviceResponseCallback = (payload) => {
      const configData = payload.error ? {
        filters: globalFilters || globalConfig?.filters || {},
        sorting: globalSorting || globalConfig?.sorting || {},
        indexer_overrides: {}
      } : {
        filters: payload.filters || {},
        sorting: payload.sorting || {},
        indexer_overrides: payload.indexer_overrides || {}
      }
      setDeviceConfigs(prev => ({ ...prev, [username]: configData }))
        loadedDevicesRef.current.add(username)
        // Create form ref if it doesn't exist
        if (!formRefs.current[username]) {
          formRefs.current[username] = React.createRef()
        }
        delete window.deviceResponseCallback
      }

    sendCommand('get_user', { username })
  }, [sendCommand, ws, globalFilters, globalSorting, globalConfig])

  // Handle toggle config expansion
  const handleToggleConfig = useCallback((username) => {
    setExpandedDevice(prev => {
      if (prev === username) {
        return null
      } else {
        loadDeviceConfig(username)
        return username
      }
    })
  }, [loadDeviceConfig])

  // Handle config changes
  const handleConfigChange = useCallback((username, config) => {
    setDeviceConfigs(prev => ({
      ...prev,
      [username]: config
    }))
  }, [])

  // Handle add device
  const handleAddDevice = async (e) => {
    e.preventDefault()
    e.stopPropagation()
    setError('')
    setSuccess('')
    setAddDeviceLoading(true)

    if (!sendCommand || !ws || ws.readyState !== WebSocket.OPEN) {
      setError('WebSocket not connected')
      setAddDeviceLoading(false)
      return
    }

    // Clean up any existing callback
    if (window.deviceActionCallback) {
      delete window.deviceActionCallback
    }

    window.deviceActionCallback = (payload) => {
      setAddDeviceLoading(false)
      if (payload.error) {
        setError(payload.error)
      } else {
        setSuccess(`Device "${newUsername}" created successfully`)
        setNewUsername('')
        setShowAddDialog(false)
        // Refresh list without showing loader (silent refresh)
        fetchDevices(false)
      }
      delete window.deviceActionCallback
    }

    sendCommand('create_user', { username: newUsername })
  }

  // Handle delete device
  const handleDeleteDevice = (username) => {
    if (username === 'admin') {
      setError('Cannot delete admin device')
      return
    }

    if (!confirm(`Are you sure you want to delete device "${username}"?`)) {
      return
    }

    if (!sendCommand || !ws || ws.readyState !== WebSocket.OPEN) {
      setError('WebSocket not connected')
      return
    }

    setError('')
    setSuccess('')
    setActionLoading(`delete-${username}`)

    // Clean up any existing callback
    if (window.deviceActionCallback) {
      delete window.deviceActionCallback
    }

    window.deviceActionCallback = (payload) => {
      setActionLoading(null)
      if (payload.error) {
        setError(payload.error)
      } else {
        setSuccess(`Device "${username}" deleted successfully`)
        // Clean up configs
        setDeviceConfigs(prev => {
          const next = { ...prev }
          delete next[username]
          return next
        })
        loadedDevicesRef.current.delete(username)
        delete formRefs.current[username]
        setExpandedDevice(prev => prev === username ? null : prev)
        // Refresh list without showing loader (silent refresh)
        fetchDevices(false)
      }
      delete window.deviceActionCallback
    }

    sendCommand('delete_user', { username })
  }

  // Handle regenerate token
  const handleRegenerateToken = (username) => {
    if (!sendCommand || !ws || ws.readyState !== WebSocket.OPEN) {
      setError('WebSocket not connected')
      return
    }

    setError('')
    setSuccess('')
    setActionLoading(`regenerate-${username}`)

    // Clean up any existing callback
    if (window.deviceActionCallback) {
      delete window.deviceActionCallback
    }

    window.deviceActionCallback = (payload) => {
      setActionLoading(null)
      if (payload.error) {
        setError(payload.error)
      } else {
        setSuccess(`Token regenerated for "${username}"`)
        setDevices(prev => prev.map(d => d.username === username ? { ...d, token: payload.token } : d))
      }
      delete window.deviceActionCallback
    }

    sendCommand('regenerate_token', { username })
  }


  // Get manifest URL
  const getManifestUrl = (token) => {
    const baseUrl = globalConfig?.addon_base_url 
      ? globalConfig.addon_base_url.replace(/\/$/, '')
      : window.location.origin
    return `${baseUrl}/${token}/manifest.json`
  }

  // Copy manifest URL
  const copyManifestUrl = (token) => {
    const url = getManifestUrl(token)
    navigator.clipboard.writeText(url)
    setCopiedToken(token)
    setTimeout(() => setCopiedToken(''), 2000)
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardDescription>
              Manage devices and their authentication tokens
            </CardDescription>
          </div>
          <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
            <DialogTrigger asChild>
              <Button type="button" className="w-full sm:w-auto">
                <Plus className="mr-2 h-4 w-4 shrink-0" />
                Add Device
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add New Device</DialogTitle>
                <DialogDescription>
                  Create a new device account. Devices will access Stremio via their token in the URL.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleAddDevice} className="space-y-4">
                {error && (
                  <div className="flex items-center gap-2 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
                    <AlertCircle className="h-4 w-4" />
                    <span>{error}</span>
                  </div>
                )}
                {success && (
                  <div className="flex items-center gap-2 p-3 text-sm text-green-600 bg-green-50 rounded-md">
                    <Check className="h-4 w-4" />
                    <span>{success}</span>
                  </div>
                )}
                <div className="space-y-2">
                  <Label htmlFor="new-username">Username</Label>
                  <Input
                    id="new-username"
                    type="text"
                    placeholder="Enter username"
                    value={newUsername}
                    onChange={(e) => setNewUsername(e.target.value)}
                    required
                    disabled={addDeviceLoading}
                  />
                  <p className="text-xs text-muted-foreground">
                    Devices access Stremio via their token in the URL: /{`{token}`}/manifest.json
                  </p>
                </div>
                <Button type="submit" className="w-full" disabled={addDeviceLoading}>
                  {addDeviceLoading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    'Create Device'
                  )}
                </Button>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {error && !showAddDialog && (
          <div className="flex items-center gap-2 p-3 mb-4 text-sm text-destructive bg-destructive/10 rounded-md">
            <AlertCircle className="h-4 w-4" />
            <span>{error}</span>
          </div>
        )}
        {success && !showAddDialog && (
          <div className="flex items-center gap-2 p-3 mb-4 text-sm text-green-600 bg-green-50 rounded-md">
            <Check className="h-4 w-4" />
            <span>{success}</span>
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center p-8">
            <Loader2 className="h-6 w-6 animate-spin" />
          </div>
        ) : devices.length === 0 ? (
          <div className="text-center p-8 text-muted-foreground">
            No devices found. Create your first device to get started.
          </div>
        ) : (
          <div className="space-y-4">
            {devices.map((device) => (
              <Card key={device.username}>
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          <h3 className="font-semibold">{device.username}</h3>
                        </div>
                        <div className="space-y-2">
                          <Label className="text-xs text-muted-foreground block">Stremio URL:</Label>
                          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                            <code className="text-xs bg-muted px-2 py-1.5 rounded break-all min-w-0">
                              {getManifestUrl(device.token)}
                            </code>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => copyManifestUrl(device.token)}
                              className="h-8 shrink-0 self-start sm:self-center"
                              title="Copy manifest URL"
                            >
                              {copiedToken === device.token ? (
                                <Check className="h-4 w-4 sm:mr-1" />
                              ) : (
                                <Copy className="h-4 w-4 sm:mr-1" />
                              )}
                              <span className="hidden sm:inline">
                                {copiedToken === device.token ? 'Copied' : 'Copy'}
                              </span>
                            </Button>
                          </div>
                        </div>
                      </div>
                      <div className="flex flex-wrap gap-2 sm:shrink-0 sm:ml-4">
                        {device.username !== 'admin' && (
                          <Button
                            type="button"
                            variant={expandedDevice === device.username ? "default" : "outline"}
                            size="sm"
                            onClick={() => handleToggleConfig(device.username)}
                            disabled={actionLoading !== null || loading}
                            className="flex-1 min-w-[7rem] sm:flex-none"
                          >
                            {expandedDevice === device.username ? (
                              <>
                                <ChevronUp className="h-3 w-3 mr-1 shrink-0" />
                                <span className="truncate">Hide Config</span>
                              </>
                            ) : (
                              <>
                                <Settings className="h-3 w-3 mr-1 shrink-0" />
                                <span className="truncate">Configure</span>
                              </>
                            )}
                          </Button>
                        )}
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => handleRegenerateToken(device.username)}
                          disabled={actionLoading !== null || loading}
                          className="flex-1 min-w-[7rem] sm:flex-none"
                        >
                          {actionLoading === `regenerate-${device.username}` ? (
                            <Loader2 className="h-3 w-3 mr-1 shrink-0 animate-spin" />
                          ) : (
                            <RefreshCw className="h-3 w-3 mr-1 shrink-0" />
                          )}
                          <span className="truncate hidden sm:inline">Regenerate Token</span>
                          <span className="truncate sm:hidden">Regenerate</span>
                        </Button>
                        {device.username !== 'admin' && (
                          <Button
                            type="button"
                            variant="destructive"
                            size="sm"
                            onClick={() => handleDeleteDevice(device.username)}
                            disabled={actionLoading !== null || loading}
                            className="flex-1 min-w-[7rem] sm:flex-none"
                          >
                            {actionLoading === `delete-${device.username}` ? (
                              <Loader2 className="h-3 w-3 shrink-0 animate-spin" />
                            ) : (
                              <Trash2 className="h-3 w-3 shrink-0" />
                            )}
                            <span className="ml-1">Delete</span>
                          </Button>
                        )}
                      </div>
                    </div>
                    
                    {device.username !== 'admin' && expandedDevice === device.username && deviceConfigs[device.username] && (() => {
                      // Ensure form ref exists for this device
                      if (!formRefs.current[device.username]) {
                        formRefs.current[device.username] = React.createRef()
                      }
                      return (
                        <div className="pt-4 border-t">
                          <DeviceConfigForm
                            username={device.username}
                            initialFilters={deviceConfigs[device.username]?.filters}
                            initialSorting={deviceConfigs[device.username]?.sorting}
                            initialIndexerOverrides={deviceConfigs[device.username]?.indexer_overrides}
                            indexerNames={globalConfig?.indexers?.map(i => i.name) ?? []}
                            onConfigChange={handleConfigChange}
                            formRef={formRefs.current[device.username]}
                          />
                        </div>
                      )
                    })()}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
})

export default DeviceManagement
