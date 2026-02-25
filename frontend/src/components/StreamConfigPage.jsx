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

const defaultFilters = {
  allowed_qualities: [], blocked_qualities: [], min_resolution: '', max_resolution: '',
  allowed_codecs: [], blocked_codecs: [], required_audio: [], allowed_audio: [],
  min_channels: '', require_hdr: false, allowed_hdr: [], blocked_hdr: [], block_sdr: false,
  required_languages: [], allowed_languages: [], block_dubbed: false, block_cam: false,
  require_proper: false, allow_repack: true, block_hardcoded: false, min_bit_depth: '',
  min_size_gb: 0, max_size_gb: 0, blocked_groups: []
}

const defaultSorting = {
  resolution_weights: { '4k': 4000000, '1080p': 3000000, '720p': 2000000, 'sd': 1000000 },
  codec_weights: { 'HEVC': 1000, 'x265': 1000, 'x264': 500, 'AVC': 500 },
  audio_weights: { 'Atmos': 1500, 'TrueHD': 1200, 'DTS-HD': 1000, 'DTS-X': 1000, 'DTS': 500, 'DD+': 400, 'DD': 300, 'AC3': 200, '5.1': 500, '7.1': 1000 },
  quality_weights: { 'BluRay': 2000, 'WEB-DL': 1500, 'WEBRip': 1200, 'HDTV': 1000, 'Blu-ray': 2000 },
  visual_tag_weights: { 'DV': 1500, 'HDR10+': 1200, 'HDR': 1000, '3D': 800 },
  grab_weight: 0.5, age_weight: 1.0, preferred_groups: [], preferred_languages: []
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
            sorting: {
              ...defaultSorting,
              ...(data.sorting || {}),
              resolution_weights: { ...defaultSorting.resolution_weights, ...(data.sorting?.resolution_weights || {}) },
              codec_weights: { ...defaultSorting.codec_weights, ...(data.sorting?.codec_weights || {}) },
              audio_weights: { ...defaultSorting.audio_weights, ...(data.sorting?.audio_weights || {}) },
              quality_weights: { ...defaultSorting.quality_weights, ...(data.sorting?.quality_weights || {}) },
              visual_tag_weights: { ...defaultSorting.visual_tag_weights, ...(data.sorting?.visual_tag_weights || {}) }
            }
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
