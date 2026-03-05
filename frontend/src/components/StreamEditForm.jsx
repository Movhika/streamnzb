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
  ChannelsOptions,
  BitDepthOptions,
  ContainerOptions,
  EditionOptions,
  NetworkOptions,
  RegionOptions,
  LanguageOptions,
  languageCodeToName,
} from "@/constants/pttOptions"

const defaultFilters = {
  quality_included: [], quality_required: [], quality_excluded: ['CAM', 'TeleSync', 'TeleCine', 'SCR'],
  resolution_included: [], resolution_required: [], resolution_excluded: [],
  codec_included: [], codec_required: [], codec_excluded: [],
  audio_included: [], audio_required: [], audio_excluded: [],
  channels_included: [], channels_required: [], channels_excluded: [],
  hdr_included: [], hdr_required: [], hdr_excluded: [],
  three_d_included: [], three_d_required: [], three_d_excluded: [],
  bit_depth_included: [], bit_depth_required: [], bit_depth_excluded: [],
  container_included: [], container_required: [], container_excluded: [],
  languages_included: [], languages_required: [], languages_excluded: [],
  edition_included: [], edition_required: [], edition_excluded: [],
  network_included: [], network_required: [], network_excluded: [],
  region_included: [], region_required: [], region_excluded: [],
  group_included: [], group_required: [], group_excluded: [],
  dubbed_excluded: true, hardcoded_excluded: undefined, proper_required: undefined,
  repack_required: undefined, repack_excluded: undefined, extended_required: undefined, unrated_required: undefined,
  min_size_gb: 0, max_size_gb: 0, min_year: 0, max_year: 0,
  min_age_hours: 0, max_age_hours: 0,
  keywords_excluded: [], keywords_required: [],
  availnzb_included: [], availnzb_required: [], availnzb_excluded: [],
  size_per_resolution: {},
  min_bitrate_kbps: 0, max_bitrate_kbps: 0
}

const defaultSortCriteriaOrder = ['availnzb', 'resolution', 'quality', 'codec', 'visual_tag', 'audio', 'channels', 'bit_depth', 'container', 'languages', 'group', 'edition', 'network', 'region', 'three_d', 'size', 'keywords', 'regex']

const defaultSorting = {
  sort_criteria_order: defaultSortCriteriaOrder,
  preferred_resolution: DefaultResolutionOrder,
  preferred_quality: [
    'BluRay REMUX', 'REMUX', 'BluRay', 'BRRip', 'BDRip', 'UHDRip', 'HDRip',
    'WEB-DL', 'WEBRip', 'WEB-DLRip', 'WEB',
    'HDTV', 'HDTVRip', 'PDTV', 'TVRip', 'SATRip',
    'DVD', 'DVDRip', 'PPVRip', 'R5', 'XviD', 'DivX',
  ],
  preferred_codec: [],
  grab_weight: 0.5,
  age_weight: 1.0,
  keywords_preferred: [],
  keywords_weight: 0,
  regex_preferred: [],
  regex_weight: 0,
  preferred_audio: [],
  preferred_visual_tag: [],
  preferred_channels: [],
  preferred_bit_depth: [],
  preferred_container: [],
  preferred_languages: [],
  preferred_group: [],
  preferred_edition: [],
  preferred_network: [],
  preferred_region: [],
  preferred_three_d: [],
  preferred_availnzb: ['available']
}

function toItems(options, labelMap = {}) {
  return options.map((key) => ({ key, label: labelMap[key] ?? key }))
}

const aioStyleCategories = [
  { key: 'resolution', label: 'Resolution', includedField: 'filters.resolution_included', orderField: 'sorting.preferred_resolution', excludedField: 'filters.resolution_excluded', items: toItems(ResolutionGroupOptions, resolutionGroupLabels) },
  { key: 'quality', label: 'Source quality', includedField: 'filters.quality_included', orderField: 'sorting.preferred_quality', excludedField: 'filters.quality_excluded', items: toItems(QualityOptions) },
  { key: 'codec', label: 'Codec', includedField: 'filters.codec_included', orderField: 'sorting.preferred_codec', excludedField: 'filters.codec_excluded', items: toItems(CodecOptions) },
  { key: 'audio', label: 'Audio', includedField: 'filters.audio_included', orderField: 'sorting.preferred_audio', excludedField: 'filters.audio_excluded', items: toItems(AudioOptions) },
  { key: 'channels', label: 'Channels', includedField: 'filters.channels_included', orderField: 'sorting.preferred_channels', excludedField: 'filters.channels_excluded', items: toItems(ChannelsOptions) },
  { key: 'visual_tag', label: 'Visual tags (HDR / SDR)', includedField: 'filters.hdr_included', orderField: 'sorting.preferred_visual_tag', excludedField: 'filters.hdr_excluded', items: toItems(HDROptions) },
  { key: 'three_d', label: '3D', includedField: 'filters.three_d_included', orderField: 'sorting.preferred_three_d', excludedField: 'filters.three_d_excluded', items: toItems(ThreeDOptions) },
  { key: 'bit_depth', label: 'Bit depth', includedField: 'filters.bit_depth_included', orderField: 'sorting.preferred_bit_depth', excludedField: 'filters.bit_depth_excluded', items: toItems(BitDepthOptions) },
  { key: 'container', label: 'Container', includedField: 'filters.container_included', orderField: 'sorting.preferred_container', excludedField: 'filters.container_excluded', items: toItems(ContainerOptions) },
  { key: 'languages', label: 'Languages', includedField: 'filters.languages_included', orderField: 'sorting.preferred_languages', excludedField: 'filters.languages_excluded', items: toItems(LanguageOptions, languageCodeToName) },
  { key: 'edition', label: 'Edition', includedField: 'filters.edition_included', orderField: 'sorting.preferred_edition', excludedField: 'filters.edition_excluded', items: toItems(EditionOptions) },
  { key: 'network', label: 'Network', includedField: 'filters.network_included', orderField: 'sorting.preferred_network', excludedField: 'filters.network_excluded', items: toItems(NetworkOptions) },
  { key: 'region', label: 'Region', includedField: 'filters.region_included', orderField: 'sorting.preferred_region', excludedField: 'filters.region_excluded', items: toItems(RegionOptions) },
  { type: 'freeText', key: 'group', label: 'Release groups', includedField: 'filters.group_included', orderField: 'sorting.preferred_group', excludedField: 'filters.group_excluded', firstColumnLabel: 'Included (bypass)' },
  { type: 'freeText', key: 'keywords', label: 'Keywords', includedField: 'filters.keywords_required', orderField: 'sorting.keywords_preferred', excludedField: 'filters.keywords_excluded', firstColumnLabel: 'Required' },
  { type: 'freeText', key: 'regex', label: 'Regex patterns', includedField: 'filters.regex_required', orderField: 'sorting.regex_preferred', excludedField: 'filters.regex_excluded', firstColumnLabel: 'Required' },
  { key: 'availnzb', label: 'AvailNZB status', includedField: 'filters.availnzb_included', orderField: 'sorting.preferred_availnzb', excludedField: 'filters.availnzb_excluded', items: [{ key: 'available', label: 'Available' }, { key: 'unavailable', label: 'Unavailable' }, { key: 'unknown', label: 'Unknown' }] },
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
          const savedOrder = Array.isArray(loadedSorting.sort_criteria_order) ? loadedSorting.sort_criteria_order : []
          if (savedOrder.length === 0) {
            loadedSorting.sort_criteria_order = defaultSortCriteriaOrder
          } else {
            // Merge in any categories from the full default list that are missing (e.g. after adding new sort criteria).
            const fullSet = new Set(defaultSortCriteriaOrder)
            const seen = new Set(savedOrder)
            const appended = defaultSortCriteriaOrder.filter((k) => !seen.has(k))
            if (appended.length > 0) {
              loadedSorting.sort_criteria_order = [...savedOrder, ...appended]
            }
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
