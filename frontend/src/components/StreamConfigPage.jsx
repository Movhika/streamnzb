import React, { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Form, FormField, FormItem, FormLabel, FormControl } from "@/components/ui/form"
import { FiltersSection } from "@/components/FiltersSection"
import { SortingSection } from "@/components/SortingSection"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { Loader2 } from "lucide-react"
import {
  DefaultResolutionOrder,
  DefaultCodecOrder,
  DefaultAudioOrder,
  DefaultQualityOrder,
  DefaultVisualTagOrder,
  DefaultChannelsOrder,
  DefaultBitDepthOrder,
  DefaultContainerOrder,
  DefaultLanguagesOrder,
  DefaultEditionOrder,
  DefaultNetworkOrder,
  DefaultRegionOrder,
  DefaultThreeDOrder,
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

const defaultSorting = {
  resolution_order: DefaultResolutionOrder,
  codec_order: DefaultCodecOrder,
  audio_order: DefaultAudioOrder,
  quality_order: DefaultQualityOrder,
  visual_tag_order: DefaultVisualTagOrder,
  channels_order: DefaultChannelsOrder,
  bit_depth_order: DefaultBitDepthOrder,
  container_order: DefaultContainerOrder,
  languages_order: DefaultLanguagesOrder,
  group_order: [],
  edition_order: DefaultEditionOrder,
  network_order: DefaultNetworkOrder,
  region_order: DefaultRegionOrder,
  three_d_order: DefaultThreeDOrder,
  grab_weight: 0.5,
  age_weight: 1.0,
  resolution_weight: 10,
  quality_weight: 8,
  codec_weight: 5,
  audio_weight: 3,
  visual_tag_weight: 5,
  channels_weight: 2,
  bit_depth_weight: 2,
  container_weight: 1,
  languages_weight: 2,
  group_weight: 3,
  edition_weight: 0.5,
  network_weight: 0.5,
  region_weight: 0.5,
  three_d_weight: 0.5,
  group_order_tier1: [],
  group_order_tier2: [],
  group_order_tier3: [],
  group_tier1_points: 30,
  group_tier2_points: 15,
  group_tier3_points: 5
}

function getApiUrl(path) {
  const base = window.location.pathname.split('/').filter(Boolean)[0]
  const prefix = base && base !== 'api' ? `/${base}` : ''
  return `${prefix}${path}`
}

export function StreamConfigPage() {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState({ type: '', text: '' })

  const form = useForm({
    defaultValues: {
      name: 'StreamNZB - Global',
      filters: defaultFilters,
      sorting: defaultSorting
    }
  })

  const { control, watch, reset, handleSubmit } = form

  useEffect(() => {
    const url = getApiUrl('/api/stream/config')
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
          reset({
            name: data.name ?? 'StreamNZB - Global',
            filters: { ...defaultFilters, ...(data.filters || {}) },
            sorting: { ...defaultSorting, ...(data.sorting || {}) }
          })
        }
      })
      .catch((err) => setMessage({ type: 'error', text: 'Failed to load stream config: ' + err.message }))
      .finally(() => setLoading(false))
  }, [reset])

  const onSave = handleSubmit(async (values) => {
    setSaving(true)
    setMessage({ type: '', text: '' })
    const url = getApiUrl('/api/stream/config')
    try {
      const res = await fetch(url, {
        method: 'PUT',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: values.name,
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
      setMessage({ type: 'success', text: 'Stream config saved.' })
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
              <CardTitle>Global stream</CardTitle>
              <CardDescription>
                One stream config (filters and sorting) is used for all catalog and play. Devices are tokens only.
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
                      <Input placeholder="StreamNZB - Global" {...field} />
                    </FormControl>
                  </FormItem>
                )}
              />
            </CardContent>
          </Card>

          <Tabs defaultValue="filters" className="w-full">
            <TabsList className="mb-4 grid w-full grid-cols-2">
              <TabsTrigger value="filters">Filters</TabsTrigger>
              <TabsTrigger value="sorting">Sorting</TabsTrigger>
            </TabsList>
            <TabsContent value="filters">
              <Card>
                <CardHeader>
                  <CardTitle>Filters</CardTitle>
                  <CardDescription>Quality, resolution, codec, and other release filters.</CardDescription>
                </CardHeader>
                <CardContent>
                  <FiltersSection control={control} watch={watch} fieldPrefix="filters" />
                </CardContent>
              </Card>
            </TabsContent>
            <TabsContent value="sorting">
              <Card>
                <CardHeader>
                  <CardTitle>Sorting</CardTitle>
                  <CardDescription>Weights for resolution, codec, audio, and quality.</CardDescription>
                </CardHeader>
                <CardContent>
                  <SortingSection control={control} watch={watch} fieldPrefix="sorting" />
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>

          {message.text && (
            <p className={`text-sm ${message.type === 'error' ? 'text-destructive' : 'text-primary'}`}>
              {message.text}
            </p>
          )}
          <div className="flex justify-end">
            <Button type="submit" disabled={saving}>
              {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save stream config
            </Button>
          </div>
        </form>
      </Form>
    </div>
  )
}
