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
  languageCodeToName,
  EditionOptions,
  NetworkOptions,
  RegionOptions,
  ThreeDOptions,
  ContainerOptions,
} from "@/constants/pttOptions"

const tierStyles = {
  included: { border: "border-blue-500/30 bg-blue-500/5", badge: "bg-blue-500/20 hover:bg-blue-500/30" },
  required: { border: "border-green-500/30 bg-green-500/5", badge: "bg-green-500/20 hover:bg-green-500/30" },
  excluded: { border: "border-red-500/30 bg-red-500/5", badge: "bg-red-500/20 hover:bg-red-500/30" },
}

/** One row: label + chips + Add dropdown (options not yet in value).
 * getOptionLabel(opt) optionally maps option value to display label (e.g. "en" -> "English"). */
function ChipRow({ label, value = [], onChange, options, variant = "required", getOptionLabel }) {
  const display = (opt) => (getOptionLabel ? getOptionLabel(opt) : opt) ?? opt
  const available = options.filter(opt => !value.includes(opt))
  const add = (opt) => {
    if (opt && !value.includes(opt)) onChange([...value, opt])
  }
  const remove = (opt) => onChange(value.filter(v => v !== opt))
  const styles = tierStyles[variant] || tierStyles.required
  return (
    <div className="space-y-2">
      <span className="text-sm font-medium text-muted-foreground">{label}</span>
      <div className={cn("flex flex-wrap gap-2 min-h-[2.5rem] p-3 rounded-md border", styles.border)}>
        {value.map(item => (
          <Badge
            key={item}
            variant="secondary"
            className={cn("gap-1 pr-1", styles.badge)}
          >
            {display(item)}
            <button
              type="button"
              className="rounded-full p-0.5 hover:bg-black/20"
              onClick={() => remove(item)}
              aria-label={`Remove ${display(item)}`}
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
                  {display(opt)}
                </button>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </div>
  )
}

/** 3-tier filter block: Included (bypass) + Required + Excluded. optionLabel: map or fn for display (e.g. language code -> full name). */
function FilterBlock({ title, options, control, getFieldName, includedField, requiredField, excludedField, optionLabel }) {
  const getOptionLabel = typeof optionLabel === 'function' ? optionLabel : (opt) => optionLabel?.[opt] ?? opt
  return (
    <div className="space-y-4">
      <h4 className="font-medium">{title}</h4>
      <div className="grid gap-4 sm:grid-cols-3">
        <FormField
          control={control}
          name={getFieldName(includedField)}
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <ChipRow
                  label="Included (bypass all filters)"
                  value={field.value || []}
                  onChange={field.onChange}
                  options={options}
                  variant="included"
                  getOptionLabel={getOptionLabel}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={getFieldName(requiredField)}
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <ChipRow
                  label="Required (must match one)"
                  value={field.value || []}
                  onChange={field.onChange}
                  options={options}
                  variant="required"
                  getOptionLabel={getOptionLabel}
                />
              </FormControl>
              <FormDescription className="sr-only">Leave empty to allow all</FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={control}
          name={getFieldName(excludedField)}
          render={({ field }) => (
            <FormItem>
              <FormControl>
                <ChipRow
                  label="Excluded (reject if matches)"
                  value={field.value || []}
                  onChange={field.onChange}
                  options={options}
                  variant="excluded"
                  getOptionLabel={getOptionLabel}
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

/** Group filter: free-text chips for Included / Required / Excluded. Paste comma-separated list to add multiple. */
function GroupFilterRow({ label, value = [], onChange, placeholder, variant = "required" }) {
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
  const styles = tierStyles[variant] || tierStyles.required
  return (
    <div className="space-y-2">
      <span className="text-sm font-medium text-muted-foreground">{label}</span>
      <div className={cn("flex flex-wrap gap-2 min-h-[2.5rem] p-3 rounded-md border", styles.border)}>
        {value.map((item) => (
          <Badge key={item} variant="secondary" className={cn("gap-1 pr-1", styles.badge)}>
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

const RESOLUTION_GROUPS = ['4k', '1080p', '720p', 'sd']
const RESOLUTION_GROUP_LABELS = { '4k': '4K / 2160p', '1080p': '1080p', '720p': '720p', 'sd': 'SD (480p and below)' }

function PerResolutionSizeInputs({ control, getFieldName }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {RESOLUTION_GROUPS.map((group) => (
        <div key={group} className="space-y-2">
          <span className="text-sm font-medium">{RESOLUTION_GROUP_LABELS[group]}</span>
          <div className="flex gap-2">
            <FormField
              control={control}
              name={getFieldName(`filters.size_per_resolution.${group}.min_gb`)}
              render={({ field }) => (
                <FormItem className="flex-1">
                  <FormControl>
                    <Input
                      type="number"
                      step="0.5"
                      min={0}
                      placeholder="Min GB"
                      value={field.value === 0 || field.value == null ? '' : field.value}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={getFieldName(`filters.size_per_resolution.${group}.max_gb`)}
              render={({ field }) => (
                <FormItem className="flex-1">
                  <FormControl>
                    <Input
                      type="number"
                      step="0.5"
                      min={0}
                      placeholder="Max GB"
                      value={field.value === 0 || field.value == null ? '' : field.value}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
          </div>
        </div>
      ))}
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
          <span className="font-medium text-blue-500">Included</span> = bypass all filters (whitelist).{' '}
          <span className="font-medium text-green-500">Required</span> = must match one.{' '}
          <span className="font-medium text-red-500">Excluded</span> = reject if matches.{' '}
          Leave all empty to allow everything.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <FilterBlock
          title="Quality"
          options={QualityOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.quality_included"
          requiredField="filters.quality_required"
          excludedField="filters.quality_excluded"
        />
        <FilterBlock
          title="Resolution"
          options={ResolutionOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.resolution_included"
          requiredField="filters.resolution_required"
          excludedField="filters.resolution_excluded"
        />
        <FilterBlock
          title="Codec"
          options={CodecOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.codec_included"
          requiredField="filters.codec_required"
          excludedField="filters.codec_excluded"
        />
        <FilterBlock
          title="Audio"
          options={AudioOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.audio_included"
          requiredField="filters.audio_required"
          excludedField="filters.audio_excluded"
        />
        <FilterBlock
          title="Channels"
          options={ChannelsOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.channels_included"
          requiredField="filters.channels_required"
          excludedField="filters.channels_excluded"
        />
        <FilterBlock
          title="Visual tags (HDR / SDR)"
          options={HDROptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.hdr_included"
          requiredField="filters.hdr_required"
          excludedField="filters.hdr_excluded"
        />
        <FilterBlock
          title="3D"
          options={ThreeDOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.three_d_included"
          requiredField="filters.three_d_required"
          excludedField="filters.three_d_excluded"
        />
        <FilterBlock
          title="Bit depth"
          options={BitDepthOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.bit_depth_included"
          requiredField="filters.bit_depth_required"
          excludedField="filters.bit_depth_excluded"
        />
        <FilterBlock
          title="Container"
          options={ContainerOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.container_included"
          requiredField="filters.container_required"
          excludedField="filters.container_excluded"
        />
        <FilterBlock
          title="Languages"
          options={LanguageOptions}
          optionLabel={languageCodeToName}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.languages_included"
          requiredField="filters.languages_required"
          excludedField="filters.languages_excluded"
        />
        <FilterBlock
          title="Edition"
          options={EditionOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.edition_included"
          requiredField="filters.edition_required"
          excludedField="filters.edition_excluded"
        />
        <FilterBlock
          title="Network"
          options={NetworkOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.network_included"
          requiredField="filters.network_required"
          excludedField="filters.network_excluded"
        />
        <FilterBlock
          title="Region"
          options={RegionOptions}
          getFieldName={getFieldName}
          control={actualControl}
          includedField="filters.region_included"
          requiredField="filters.region_required"
          excludedField="filters.region_excluded"
        />

        {/* Release groups: free-text Included / Required / Excluded */}
        <div className="space-y-4">
          <h4 className="font-medium">Release groups</h4>
          <div className="grid gap-4 sm:grid-cols-3">
            <FormField
              control={actualControl}
              name={getFieldName("filters.group_included")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Included (bypass)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type group name..."
                      variant="included"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.group_required")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Required (only these groups)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type group name..."
                      variant="required"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.group_excluded")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Excluded (reject these groups)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type group name..."
                      variant="excluded"
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
              name={getFieldName("filters.dubbed_excluded")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Exclude dubbed</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.hardcoded_excluded")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Exclude hardcoded subs</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.proper_required")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Require Proper</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.repack_required")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Require Repack</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.repack_excluded")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Exclude Repack</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.extended_required")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Require Extended</FormLabel>
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.unrated_required")}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                  <FormControl>
                    <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                  </FormControl>
                  <FormLabel className="font-normal">Require Unrated</FormLabel>
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

        {/* Per-resolution size ranges */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">Per-resolution size limits</h4>
          <p className="text-xs text-muted-foreground">Override global min/max size for specific resolution groups. Leave empty to use global limits.</p>
          <PerResolutionSizeInputs control={actualControl} getFieldName={getFieldName} />
        </div>

        {/* Age range */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">Release age</h4>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <FormField
              control={actualControl}
              name={getFieldName("filters.min_age_hours")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Min age (hours)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="1"
                      min={0}
                      placeholder="0 = any"
                      value={field.value === 0 ? '' : (field.value ?? '')}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.max_age_hours")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Max age (hours)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="1"
                      min={0}
                      placeholder="0 = any"
                      value={field.value === 0 ? '' : (field.value ?? '')}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>

        {/* Keywords */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">Keywords</h4>
          <p className="text-xs text-muted-foreground">Case-insensitive keyword matching against the full release title.</p>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              control={actualControl}
              name={getFieldName("filters.keywords_required")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Required (must contain one)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type keyword..."
                      variant="required"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.keywords_excluded")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Excluded (reject if contains)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="Type keyword..."
                      variant="excluded"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>

        {/* Regex patterns */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">Regex patterns</h4>
          <p className="text-xs text-muted-foreground">Regular expressions matched against the full release title (case-insensitive).</p>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField
              control={actualControl}
              name={getFieldName("filters.regex_required")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Required (must match one)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="e.g. REMUX|BluRay"
                      variant="required"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.regex_excluded")}
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <GroupFilterRow
                      label="Excluded (reject if matches)"
                      value={field.value || []}
                      onChange={field.onChange}
                      placeholder="e.g. CAM|TS"
                      variant="excluded"
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </div>

        {/* AvailNZB */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">AvailNZB</h4>
          <FormField
            control={actualControl}
            name={getFieldName("filters.availnzb_required")}
            render={({ field }) => (
              <FormItem className="flex flex-row items-center space-x-3 space-y-0">
                <FormControl>
                  <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                </FormControl>
                <FormLabel className="font-normal">Only show releases confirmed available by AvailNZB</FormLabel>
              </FormItem>
            )}
          />
        </div>

        {/* Bitrate */}
        <div className="space-y-4 pt-2 border-t">
          <h4 className="font-medium">Bitrate</h4>
          <p className="text-xs text-muted-foreground">Only applies to Easynews results (Newznab does not provide duration).</p>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
            <FormField
              control={actualControl}
              name={getFieldName("filters.min_bitrate_kbps")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Min bitrate (kbps)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="100"
                      min={0}
                      placeholder="0 = any"
                      value={field.value === 0 ? '' : (field.value ?? '')}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={actualControl}
              name={getFieldName("filters.max_bitrate_kbps")}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Max bitrate (kbps)</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      step="100"
                      min={0}
                      placeholder="0 = any"
                      value={field.value === 0 ? '' : (field.value ?? '')}
                      onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))}
                    />
                  </FormControl>
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
