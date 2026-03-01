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
import { DefaultResolutionOrder, DefaultQualityOrder } from "@/constants/pttOptions"

const defaultFilters = {
  quality_included: [], quality_required: [], quality_excluded: [],
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
  dubbed_excluded: undefined, hardcoded_excluded: undefined, proper_required: undefined,
  repack_required: undefined, repack_excluded: undefined, extended_required: undefined, unrated_required: undefined,
  min_size_gb: 0, max_size_gb: 0, min_year: 0, max_year: 0
}

const defaultSorting = {
  preferred_resolution: DefaultResolutionOrder,
  preferred_quality: DefaultQualityOrder,
  grab_weight: 0.5,
  age_weight: 1.0,
  preferred_codec: [],
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
  preferred_three_d: []
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
