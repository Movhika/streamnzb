import React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from "@/components/ui/form"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { X, Plus } from "lucide-react"
import { useFormContext } from "react-hook-form"
import { cn } from "@/lib/utils"
import {
  QualityOptions,
  ResolutionOptions,
  CodecOptions,
  AudioOptions,
  ChannelsOptions,
  BitDepthOptions,
  HDROptions,
  LanguageOptions,
  EditionOptions,
  NetworkOptions,
  RegionOptions,
  ThreeDOptions,
  ContainerOptions,
} from "@/constants/pttOptions"

/** One row: label + chips + Add dropdown (options not yet in value) */
function ChipRow({ label, value = [], onChange, options, variant = "include" }) {
  const available = options.filter(opt => !value.includes(opt))
  const add = (opt) => {
    if (opt && !value.includes(opt)) onChange([...value, opt])
  }
  const remove = (opt) => onChange(value.filter(v => v !== opt))
  return (
    <div className="space-y-2">
      <span className="text-sm font-medium text-muted-foreground">{label}</span>
      <div
        className={cn(
          "flex flex-wrap gap-2 min-h-[2.5rem] p-3 rounded-md border",
          variant === "include" && "border-green-500/30 bg-green-500/5",
          variant === "avoid" && "border-red-500/30 bg-red-500/5"
        )}
      >
        {value.map(item => (
          <Badge
            key={item}
            variant="secondary"
            className={cn(
              "gap-1 pr-1",
              variant === "include" && "bg-green-500/20 hover:bg-green-500/30",
              variant === "avoid" && "bg-red-500/20 hover:bg-red-500/30"
            )}
          >
            {item}
            <button
              type="button"
              className="rounded-full p-0.5 hover:bg-black/20"
              onClick={() => remove(item)}
              aria-label={`Remove ${item}`}
            >
              <X className="h-3 w-3" />
            </button>
          </Badge>
        ))}
        {available.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button type="button" variant="outline" size="sm" className="h-8 gap-1">
                <Plus className="h-3.5 w-3.5" />
                Add
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="max-h-[16rem] overflow-y-auto">
              {available.map(opt => (
                <button
                  key={opt}
                  type="button"
                  className="w-full cursor-pointer rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent"
                  onClick={() => add(opt)}
                >
                  {opt}
                </button>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </div>
  )
}

/** Include (only allow) + Avoid (exclude) block for a category with typed options */
function IncludeAvoidBlock({ title, options, includeValue, avoidValue, onIncludeChange, onAvoidChange, getFieldName, control, includeField, avoidField }) {
  return (
    <div className="space-y-4">
      <h4 className="font-medium">{title}</h4>
      <div className="grid gap-4 sm:grid-cols-2">
        <FormField
          control={control}
          name={getFieldName(includeField)}
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <ChipRow
                  label="Include (only allow these)"
                  value={field.value || []}
                  onChange={field.onChange}
                  options={options}
                  variant="include"
                />
              </FormControl>
              <FormDescription className="sr-only">Leave empty to allow all</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={getFieldName(avoidField)}
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <ChipRow
                  label="Avoid (exclude these)"
                  value={field.value || []}
                  onChange={field.onChange}
                  options={options}
                  variant="avoid"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
      </div>
    </div>
  )
}

/** Group filter: free-text chips for Include / Avoid (no fixed option list). Paste comma-separated list to add multiple. */
function GroupFilterRow({ label, value = [], onChange, placeholder }) {
  const [inputValue, setInputValue] = React.useState('')
  const add = (item) => {
    const v = (item || inputValue).trim()
    if (v && !value.includes(v)) {
      onChange([...value, v])
      setInputValue('')
    }
  }
  const addMany = (text) => {
    const parts = text.split(/[\s,]+/).map((s) => s.trim()).filter(Boolean)
    const seen = new Set(value.map((v) => v.toLowerCase()))
    const next = [...value]
    for (const p of parts) {
      if (p && !seen.has(p.toLowerCase())) {
        seen.add(p.toLowerCase())
        next.push(p)
      }
    }
    if (next.length !== value.length) onChange(next)
  }
  const remove = (item) => onChange(value.filter((v) => v !== item))
  const handlePaste = (e) => {
    const pasted = (e.clipboardData || window.clipboardData)?.getData('text')
    if (pasted && /[\s,]/.test(pasted)) {
      e.preventDefault()
      addMany(pasted)
    }
  }
  return (
    <div className="space-y-2">
      <span className="text-sm font-medium text-muted-foreground">{label}</span>
      <div className="flex flex-wrap gap-2 min-h-[2.5rem] p-3 rounded-md border border-border bg-background">
        {value.map((item) => (
          <Badge key={item} variant="secondary" className="gap-1 pr-1">
            {item}
            <button type="button" className="rounded-full p-0.5 hover:bg-black/20" onClick={() => remove(item)} aria-label={`Remove ${item}`}>
              <X className="h-3 w-3" />
            </button>
          </Badge>
        ))}
        <Input
          placeholder={placeholder}
          value={inputValue}
          onChange={(e) => setInputValue(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), add())}
          onPaste={handlePaste}
          className="w-32 min-w-0"
        />
      </div>
      <p className="text-xs text-muted-foreground">Paste a comma-separated list to add multiple groups.</p>
    </div>
  )
}

export function FiltersSection({ control, watch, fieldPrefix = '' }) {
  let formContext = null
  try {
    formContext = useFormContext()
  } catch (e) {}
  const actualControl = control || formContext?.control
  const getFieldName = (field) => {
    if (!fieldPrefix) return field
    if (field.startsWith('filters.')) {
      if (fieldPrefix === 'filters') return field
      return `${fieldPrefix}.${field}`
    }
    if (fieldPrefix === 'filters') return `filters.${field}`
    return `${fieldPrefix}.filters.${field}`
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Release Filters</CardTitle>
        <CardDescription>
          Filter by Include (only allow these) or Avoid (exclude these). Leave both empty to allow all.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <IncludeAvoidBlock
          title="Quality"
          options={QualityOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.quality_include"
          avoidField="filters.quality_avoid"
        />
        <IncludeAvoidBlock
          title="Resolution"
          options={ResolutionOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.resolution_include"
          avoidField="filters.resolution_avoid"
        />
        <IncludeAvoidBlock
          title="Codec"
          options={CodecOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.codec_include"
          avoidField="filters.codec_avoid"
        />
        <IncludeAvoidBlock
          title="Audio"
          options={AudioOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.audio_include"
          avoidField="filters.audio_avoid"
        />
        <IncludeAvoidBlock
          title="Channels"
          options={ChannelsOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.channels_include"
          avoidField="filters.channels_avoid"
        />
        <IncludeAvoidBlock
          title="Visual tags (HDR / SDR)"
          options={HDROptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.hdr_include"
          avoidField="filters.hdr_avoid"
        />
        <IncludeAvoidBlock
          title="3D"
          options={ThreeDOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.three_d_include"
          avoidField="filters.three_d_avoid"
        />
        <IncludeAvoidBlock
          title="Bit depth"
          options={BitDepthOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.bit_depth_include"
          avoidField="filters.bit_depth_avoid"
        />
        <IncludeAvoidBlock
          title="Container"
          options={ContainerOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.container_include"
          avoidField="filters.container_avoid"
        />
        <IncludeAvoidBlock
          title="Languages"
          options={LanguageOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.languages_include"
          avoidField="filters.languages_avoid"
        />
        <IncludeAvoidBlock
          title="Edition"
          options={EditionOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.edition_include"
          avoidField="filters.edition_avoid"
        />
        <IncludeAvoidBlock
          title="Network"
          options={NetworkOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.network_include"
          avoidField="filters.network_avoid"
        />
        <IncludeAvoidBlock
          title="Region"
          options={RegionOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includeField="filters.region_include"
          avoidField="filters.region_avoid"
        />

        {/* Release groups: free-text Include / Avoid */}
        <div className="space-y-4">
          <h4 className="font-medium">Release groups</h4>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              control={actualControl}
              name={getFieldName("filters.group_include")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Include (only these groups)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type group name..."
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.group_avoid")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Avoid (exclude these groups)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type group name..."
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>

        {/* Release flags (booleans) */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">Release flags</h4>
          <div className="flex flex-wrap gap-6">
            <FormField
              control={actualControl}
              name={getFieldName("filters.dubbed_avoid")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Avoid dubbed</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.hardcoded_avoid")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Avoid hardcoded subs</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.proper_include")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Prefer Proper</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.repack_include")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Prefer Repack</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.repack_avoid")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Avoid Repack</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.extended_include")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Prefer Extended</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.unrated_include")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Prefer Unrated</FormLabel>
                </FormItem>
              )}
            />
          </div>
        </div>

        {/* Size & year */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">File size & year</h4>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <FormField
              control={actualControl}
              name={getFieldName("filters.min_size_gb")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Min size (GB)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="0.1"
                      min={0}
                      placeholder="0"
                      value={field.value ?? ''}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.max_size_gb")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Max size (GB)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="0.1"
                      min={0}
                      placeholder="0"
                      value={field.value ?? ''}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.min_year")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Min year</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="0 = any"
                      value={field.value === 0 ? '' : (field.value ?? '')}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : parseInt(e.target.value, 10) || 0)}
                    />
                  </FormControl>
                  <FormDescription className="sr-only">0 = no minimum</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.max_year")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Max year</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      placeholder="0 = any"
                      value={field.value === 0 ? '' : (field.value ?? '')}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : parseInt(e.target.value, 10) || 0)}
                    />
                  </FormControl>
                  <FormDescription className="sr-only">0 = no maximum</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
