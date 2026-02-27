import React, { useState, useEffect } from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { FormField, FormItem, FormLabel, FormControl, FormDescription, FormMessage } from "@/components/ui/form"
import { PriorityList, PrioritySubsetList, MultiplierSlider } from "@/components/ui/priority-list"
import { useFormContext } from "react-hook-form"
import {
  ResolutionGroupOptions,
  resolutionGroupLabels,
  CodecOptions,
  AudioOptions,
  QualityOptions,
  HDROptions,
  ThreeDOptions,
  ChannelsOptions,
  BitDepthOptions,
  ContainerOptions,
  LanguageOptions,
  EditionOptions,
  NetworkOptions,
  RegionOptions,
} from "@/constants/pttOptions"

function CommaSeparatedInput({ value = [], onChange, placeholder }) {
  const [rawValue, setRawValue] = useState(value?.join(', ') || '')
  useEffect(() => {
    setRawValue(value?.join(', ') || '')
  }, [value?.join?.()])
  return (
    <Input
      placeholder={placeholder}
      value={rawValue}
      onChange={(e) => setRawValue(e.target.value)}
      onBlur={() => {
        const arr = rawValue.split(',').map(s => s.trim()).filter(Boolean)
        onChange(arr)
        setRawValue(arr.join(', '))
      }}
    />
  )
}

/** Build { key, label } items from option array; optional label map */
function toItems(options, labelMap = {}) {
  return options.map(key => ({ key, label: labelMap[key] ?? key }))
}

const resolutionItems = toItems(ResolutionGroupOptions, resolutionGroupLabels)
const codecItems = toItems(CodecOptions)
const audioItems = toItems(AudioOptions)
const qualityItems = toItems(QualityOptions)
const visualTagItems = toItems([...HDROptions, ...ThreeDOptions])
const channelsItems = toItems(ChannelsOptions)
const bitDepthItems = toItems(BitDepthOptions)
const containerItems = toItems(ContainerOptions)
const languagesItems = toItems(LanguageOptions)
const editionItems = toItems(EditionOptions)
const networkItems = toItems(NetworkOptions)
const regionItems = toItems(RegionOptions)
const threeDItems = toItems(ThreeDOptions)

export function SortingSection({ control, watch, fieldPrefix = '' }) {
  let formContext = null
  try {
    formContext = useFormContext()
  } catch (e) {}
  const actualControl = control || formContext?.control
  const getFieldName = (field) => {
    if (!fieldPrefix) return field
    if (field.startsWith('sorting.')) {
      if (fieldPrefix === 'sorting') return field
      return `${fieldPrefix}.${field}`
    }
    if (fieldPrefix === 'sorting') return `sorting.${field}`
    return `${fieldPrefix}.sorting.${field}`
  }

  const categoryWeights = [
    { key: 'resolution_weight', label: 'Resolution' },
    { key: 'quality_weight', label: 'Quality' },
    { key: 'codec_weight', label: 'Codec' },
    { key: 'audio_weight', label: 'Audio' },
    { key: 'visual_tag_weight', label: 'Visual (HDR/3D)' },
    { key: 'channels_weight', label: 'Channels' },
    { key: 'bit_depth_weight', label: 'Bit depth' },
    { key: 'container_weight', label: 'Container' },
    { key: 'languages_weight', label: 'Languages' },
    { key: 'group_weight', label: 'Group' },
    { key: 'edition_weight', label: 'Edition' },
    { key: 'network_weight', label: 'Network' },
    { key: 'region_weight', label: 'Region' },
    { key: 'three_d_weight', label: '3D' }
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Sorting Priority</CardTitle>
        <CardDescription>
          Drag items to reorder. Top of each list = highest priority (10 pts), bottom = lowest (0 pts).
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4 rounded-lg border p-4">
          <h4 className="font-medium">Category weights</h4>
          <p className="text-sm text-muted-foreground">
            Weight each category from 0 to 10. Higher values (e.g. resolution 10) matter more; lower (e.g. edition 0.5) matter less.
          </p>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3">
            {categoryWeights.map(({ key, label }) => (
              <FormField
                key={key}
                control={actualControl}
                name={getFieldName(`sorting.${key}`)}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="text-xs">{label}</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        max={10}
                        step={0.5}
                        value={field.value ?? 1}
                        onChange={(e) => {
                          const v = parseFloat(e.target.value);
                          if (!Number.isFinite(v)) { field.onChange(undefined); return; }
                          field.onChange(Math.min(10, Math.max(0, v)));
                        }}
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
            ))}
          </div>
        </div>
        <FormField
          control={actualControl}
          name={getFieldName("sorting.resolution_order")}
          render={({ field }) => (
            <PriorityList
              items={resolutionItems}
              value={field.value}
              onChange={field.onChange}
              title="Resolution"
              description="Preferred resolution order."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.codec_order")}
          render={({ field }) => (
            <PriorityList
              items={codecItems}
              value={field.value}
              onChange={field.onChange}
              title="Codec"
              description="Preferred video codecs."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.audio_order")}
          render={({ field }) => (
            <PriorityList
              items={audioItems}
              value={field.value}
              onChange={field.onChange}
              title="Audio"
              description="Preferred audio formats."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.quality_order")}
          render={({ field }) => (
            <PriorityList
              items={qualityItems}
              value={field.value}
              onChange={field.onChange}
              title="Source quality"
              description="Preferred source types."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.visual_tag_order")}
          render={({ field }) => (
            <PriorityList
              items={visualTagItems}
              value={field.value}
              onChange={field.onChange}
              title="Visual tags (HDR / 3D)"
              description="Preferred visual tags."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.channels_order")}
          render={({ field }) => (
            <PriorityList
              items={channelsItems}
              value={field.value}
              onChange={field.onChange}
              title="Channels"
              description="Preferred channel layout."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.bit_depth_order")}
          render={({ field }) => (
            <PriorityList
              items={bitDepthItems}
              value={field.value}
              onChange={field.onChange}
              title="Bit depth"
              description="Preferred bit depth."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.container_order")}
          render={({ field }) => (
            <PriorityList
              items={containerItems}
              value={field.value}
              onChange={field.onChange}
              title="Container"
              description="Preferred container format."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.languages_order")}
          render={({ field }) => (
            <PrioritySubsetList
              allItems={languagesItems}
              value={field.value}
              onChange={field.onChange}
              title="Languages"
              description="Add only the languages you care about; order = priority."
              showScoreHint={false}
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.edition_order")}
          render={({ field }) => (
            <PrioritySubsetList
              allItems={editionItems}
              value={field.value}
              onChange={field.onChange}
              title="Edition"
              description="Add only the edition types you care about."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.network_order")}
          render={({ field }) => (
            <PrioritySubsetList
              allItems={networkItems}
              value={field.value}
              onChange={field.onChange}
              title="Network"
              description="Add only the streaming/broadcast sources you care about."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.region_order")}
          render={({ field }) => (
            <PrioritySubsetList
              allItems={regionItems}
              value={field.value}
              onChange={field.onChange}
              title="Region"
              description="Add only the regions you care about."
            />
          )}
        />
        <FormField
          control={actualControl}
          name={getFieldName("sorting.three_d_order")}
          render={({ field }) => (
            <PrioritySubsetList
              allItems={threeDItems}
              value={field.value}
              onChange={field.onChange}
              title="3D"
              description="Add only the 3D formats you care about."
            />
          )}
        />

        <div className="space-y-4 pt-4 border-t">
          <h4 className="font-medium">Group & language order</h4>
          <FormField
            control={actualControl}
            name={getFieldName("sorting.group_order")}
            render={({ field }) => (
              <FormItem>
                <FormLabel>Group priority (ordered list)</FormLabel>
                <FormControl>
                  <CommaSeparatedInput
                    value={field.value || []}
                    onChange={field.onChange}
                    placeholder="e.g. FLUX, NTb, SWTYBLZ (first = highest)"
                  />
                </FormControl>
                <FormDescription>First group in list gets highest score.</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>

        <div className="space-y-4 pt-4 border-t">
          <h4 className="font-medium">Score multipliers</h4>
          <FormField
            control={actualControl}
            name={getFieldName("sorting.grab_weight")}
            render={({ field }) => (
              <MultiplierSlider
                label="Grab weight"
                value={field.value ?? 0.5}
                onChange={field.onChange}
                min={0}
                max={2}
                step={0.1}
                description="Prioritize popular releases (higher grabs = more popular)"
              />
            )}
          />
          <FormField
            control={actualControl}
            name={getFieldName("sorting.age_weight")}
            render={({ field }) => (
              <MultiplierSlider
                label="Age weight"
                value={field.value ?? 1.0}
                onChange={field.onChange}
                min={0}
                max={2}
                step={0.1}
                description="Prioritize newer releases (higher = prefer newer)"
              />
            )}
          />
        </div>
      </CardContent>
    </Card>
  )
}
