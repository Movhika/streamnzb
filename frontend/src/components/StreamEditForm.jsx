import React, { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Form, FormField, FormItem, FormLabel, FormControl, FormDescription } from "@/components/ui/form"
import { Checkbox } from "@/components/ui/checkbox"
import { ConfigComponent } from "@/components/ConfigComponent"
import { Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import {
  DefaultResolutionOrder,
  DefaultQualityOrder,
  ResolutionGroupOptions,
  resolutionGroupLabels,
  CodecOptions,
  QualityOptions,
  HDROptions,
  ThreeDOptions,
  AudioOptions,
} from "@/constants/pttOptions"

const defaultFilters = {
  quality_include: [], quality_avoid: [], resolution_include: [], resolution_avoid: [],
  codec_include: [], codec_avoid: [], audio_include: [], audio_avoid: [],
  channels_include: [], channels_avoid: [], hdr_include: [], hdr_avoid: [],
  three_d_include: [], three_d_avoid: [], bit_depth_include: [], bit_depth_avoid: [],
  container_include: [], container_avoid: [], languages_include: [], languages_avoid: [],
  edition_include: [], edition_avoid: [], network_include: [], network_avoid: [],
  region_include: [], region_avoid: [], group_include: [], group_avoid: [],
  dubbed_avoid: undefined, hardcoded_avoid: undefined, proper_include: undefined,
  repack_include: undefined, repack_avoid: undefined, extended_include: undefined, unrated_include: undefined,
  min_size_gb: 0, max_size_gb: 0, min_year: 0, max_year: 0
}

const defaultSortCriteriaOrder = ['resolution', 'quality', 'codec', 'visual_tag', 'audio', 'size']

const defaultSorting = {
  sort_criteria_order: defaultSortCriteriaOrder,
  resolution_order: DefaultResolutionOrder,
  quality_order: DefaultQualityOrder,
  codec_order: [],
  grab_weight: 0.5,
  age_weight: 1.0,
  audio_order: [],
  visual_tag_order: [],
  channels_order: [],
  bit_depth_order: [],
  container_order: [],
  languages_order: [],
  group_order: [],
  edition_order: [],
  network_order: [],
  region_order: [],
  three_d_order: []
}

function toItems(options, labelMap = {}) {
  return options.map((key) => ({ key, label: labelMap[key] ?? key }))
}

const aioStyleCategories = [
  { key: 'resolution', label: 'Resolution', orderField: 'sorting.resolution_order', avoidField: 'filters.resolution_avoid', items: toItems(ResolutionGroupOptions, resolutionGroupLabels) },
  { key: 'quality', label: 'Source quality', orderField: 'sorting.quality_order', avoidField: 'filters.quality_avoid', items: toItems(QualityOptions) },
  { key: 'codec', label: 'Codec', orderField: 'sorting.codec_order', avoidField: 'filters.codec_avoid', items: toItems(CodecOptions) },
  { key: 'visual_tag', label: 'Visual tags (HDR / 3D)', orderField: 'sorting.visual_tag_order', avoidField: 'filters.hdr_avoid', items: toItems([...HDROptions, ...ThreeDOptions]) },
  { key: 'audio', label: 'Audio', orderField: 'sorting.audio_order', avoidField: 'filters.audio_avoid', items: toItems(AudioOptions) },
  { type: 'size', minField: 'filters.min_size_gb', maxField: 'filters.max_size_gb' },
]

function getApiUrl(path) {
  const base = window.location.pathname.split('/').filter(Boolean)[0]
  const prefix = base && base !== 'api' ? `/${base}` : ''
  return `${prefix}${path}`
}

export function StreamEditForm({ streamId, onSaved, onCancel }) {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState({ type: '', text: '' })

  const isCreate = streamId == null

  const form = useForm({
    defaultValues: {
      name: isCreate ? 'New stream' : 'StreamNZB - Global',
      show_all_stream: false,
      filters: defaultFilters,
      sorting: defaultSorting
    }
  })

  const { control, reset, handleSubmit } = form

  useEffect(() => {
    if (isCreate) {
      reset({
        name: 'New stream',
        show_all_stream: false,
        filters: defaultFilters,
        sorting: defaultSorting
      })
      setLoading(false)
      return
    }
    const url = getApiUrl(`/api/stream/configs/${encodeURIComponent(streamId)}`)
    fetch(url, { credentials: 'include' })
      .then((res) => {
        if (res.status === 403) {
          setMessage({ type: 'error', text: 'Only admin can edit stream config.' })
          return null
        }
        if (!res.ok) throw new Error(res.statusText)
        return res.json()
      })
      .then((data) => {
        if (data) {
          const loadedSorting = { ...defaultSorting, ...(data.sorting || {}) }
          if (!Array.isArray(loadedSorting.sort_criteria_order) || loadedSorting.sort_criteria_order.length === 0) {
            loadedSorting.sort_criteria_order = defaultSortCriteriaOrder
          }
          reset({
            name: data.name ?? data.id ?? 'Stream',
            show_all_stream: data.show_all_stream === true,
            filters: { ...defaultFilters, ...(data.filters || {}) },
            sorting: loadedSorting
          })
        }
      })
      .catch((err) => setMessage({ type: 'error', text: 'Failed to load stream: ' + err.message }))
      .finally(() => setLoading(false))
  }, [streamId, isCreate, reset])

  const onSave = handleSubmit(async (values) => {
    setSaving(true)
    setMessage({ type: '', text: '' })
    const url = isCreate
      ? getApiUrl('/api/stream/configs')
      : getApiUrl(`/api/stream/configs/${encodeURIComponent(streamId)}`)
    const method = isCreate ? 'POST' : 'PUT'
    try {
      const res = await fetch(url, {
        method,
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: values.name,
          show_all_stream: values.show_all_stream,
          filters: values.filters,
          sorting: values.sorting
        })
      })
      if (res.status === 403) {
        setMessage({ type: 'error', text: 'Only admin can edit stream config.' })
        return
      }
      if (!res.ok) {
        const err = await res.text()
        throw new Error(err || res.statusText)
      }
      setMessage({ type: 'success', text: isCreate ? 'Stream created.' : 'Stream saved.' })
      onSaved?.()
    } catch (err) {
      setMessage({ type: 'error', text: err.message || 'Save failed.' })
    } finally {
      setSaving(false)
    }
  })

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <Form {...form}>
        <form onSubmit={onSave} className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{isCreate ? 'New stream' : 'Edit stream'}</CardTitle>
              <CardDescription>
                Display name and filters/sorting for this stream. Devices use the default stream (Global) unless configured otherwise.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <FormField
                control={control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Display name</FormLabel>
                    <FormControl>
                      <Input placeholder="Stream name" {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
              <FormField
                control={control}
                name="show_all_stream"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-start space-x-3 space-y-0">
                    <FormControl>
                      <Checkbox
                        checked={field.value === true}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <div className="space-y-1 leading-none">
                      <FormLabel className="font-normal">Show all stream</FormLabel>
                      <FormDescription>
                        Show every release (unknown or available) as a separate stream row so you can choose where to start. Failover to the next release still applies if playback fails.
                      </FormDescription>
                    </div>
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          <FormField
            control={control}
            name="sorting.sort_criteria_order"
            render={({ field }) => (
              <ConfigComponent
                control={control}
                criteriaOrderValue={field.value ?? defaultSortCriteriaOrder}
                onCriteriaOrderChange={field.onChange}
                categories={aioStyleCategories}
              />
            )}
          />

          {message.text && (
            <p className={cn("text-sm", message.type === 'error' ? 'text-destructive' : 'text-primary')}>
              {message.text}
            </p>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={saving}>
              {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {isCreate ? 'Create stream' : 'Save stream'}
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
