import React, { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from "@/components/ui/form"
import { PasswordInput } from "@/components/ui/password-input"
import { Trash2, Plus, ChevronDown, Search, Tv, Film, Info, Check, X } from "lucide-react"
import { cn } from "@/lib/utils"

const INDEXER_PRESETS = [
    { name: 'abNZB', url: 'https://abnzb.com', api_path: '/api', type: 'newznab' },
    { name: 'altHUB', url: 'https://api.althub.co.za', api_path: '/api', type: 'newznab' },
    { name: 'AnimeTosho (Usenet)', url: 'https://feed.animetosho.org', api_path: '/api', type: 'newznab' },
    { name: 'DOGnzb', url: 'https://api.dognzb.cr', api_path: '/api', type: 'newznab' },
    { name: 'DrunkenSlug', url: 'https://drunkenslug.com', api_path: '/api', type: 'newznab' },
    { name: 'GingaDADDY', url: 'https://www.gingadaddy.com', api_path: '/api', type: 'newznab' },
    { name: 'Miatrix', url: 'https://www.miatrix.com', api_path: '/api', type: 'newznab' },
    { name: 'Newz69', url: 'https://newz69.keagaming.com', api_path: '/api', type: 'newznab' },
    { name: 'NinjaCentral', url: 'https://ninjacentral.co.za', api_path: '/api', type: 'newznab' },
    { name: 'Nzb.life', url: 'https://api.nzb.life', api_path: '/api', type: 'newznab' },
    { name: 'NZBCat', url: 'https://nzb.cat', api_path: '/api', type: 'newznab' },
    { name: 'NZBFinder', url: 'https://nzbfinder.ws', api_path: '/api', type: 'newznab' },
    { name: 'NZBgeek', url: 'https://api.nzbgeek.info', api_path: '/api', type: 'newznab' },
    { name: 'NzbNoob', url: 'https://www.nzbnoob.com', api_path: '/api', type: 'newznab' },
    { name: 'NZBNDX', url: 'https://www.nzbndx.com', api_path: '/api', type: 'newznab' },
    { name: 'NzbPlanet', url: 'https://api.nzbplanet.net', api_path: '/api', type: 'newznab' },
    { name: 'NZBStars', url: 'https://nzbstars.com', api_path: '/api', type: 'newznab' },
    { name: 'SceneNZBs', url: 'https://scenenzbs.com', api_path: '/api', type: 'newznab' },
    { name: 'Tabula Rasa', url: 'https://www.tabula-rasa.pw', api_path: '/api/v1', type: 'newznab' },
    { name: 'Usenet Crawler', url: 'https://www.usenet-crawler.com', api_path: '/api', type: 'newznab' },
    //{ name: 'Easynews (Experimental)', url: '', api_path: '/api', type: 'easynews' },
    //{ name: 'Custom Newznab', url: '', api_path: '/api', type: 'newznab' }
]

function CapsInfo({ caps }) {
  if (!caps) return null
  return (
    <div className="space-y-2 text-xs">
      <div className="flex items-center gap-3">
        <span className="flex items-center gap-1">
          <Search className="h-3 w-3" />
          Search
          {caps.searching?.search ? <Check className="h-3 w-3 text-green-500" /> : <X className="h-3 w-3 text-muted-foreground" />}
        </span>
        <span className="flex items-center gap-1">
          <Film className="h-3 w-3" />
          Movie
          {caps.searching?.movie_search ? <Check className="h-3 w-3 text-green-500" /> : <X className="h-3 w-3 text-muted-foreground" />}
        </span>
        <span className="flex items-center gap-1">
          <Tv className="h-3 w-3" />
          TV
          {caps.searching?.tv_search ? <Check className="h-3 w-3 text-green-500" /> : <X className="h-3 w-3 text-muted-foreground" />}
        </span>
      </div>
      {caps.retention_days > 0 && (
        <div className="text-muted-foreground">Retention: {caps.retention_days} days</div>
      )}
    </div>
  )
}

function CategoriesHint({ caps, type }) {
  if (!caps?.categories?.length) return null
  const prefix = type === 'movie' ? '2' : '5'
  const relevant = caps.categories.filter(c => c.id.startsWith(prefix))
  if (relevant.length === 0) return null

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Info className="h-3.5 w-3.5 text-muted-foreground cursor-help hover:text-foreground shrink-0" />
        </TooltipTrigger>
        <TooltipContent className="max-w-xs p-3 z-[100]" side="bottom" sideOffset={5}>
          <div className="text-xs space-y-1">
            <div className="font-medium mb-1">Available {type === 'movie' ? 'Movie' : 'TV'} Categories</div>
            {relevant.map(cat => (
              <div key={cat.id}>
                <span className="font-mono">{cat.id}</span> - {cat.name}
                {cat.subcats?.map(sub => (
                  <div key={sub.id} className="ml-3">
                    <span className="font-mono">{sub.id}</span> - {sub.name}
                  </div>
                ))}
              </div>
            ))}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

function SearchSettings({ control, index, watch, indexerCaps }) {
  const [open, setOpen] = useState(false)
  const indexerName = watch(`indexers.${index}.name`)
  const caps = indexerCaps?.[indexerName]

  return (
    <div className="mt-3">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground hover:text-foreground transition-colors w-full"
      >
        <ChevronDown className={cn("h-3.5 w-3.5 transition-transform duration-200", open && "rotate-180")} />
        Search Settings
        {caps && <Badge variant="outline" className="ml-auto text-[9px] h-4 px-1">CAPS</Badge>}
      </button>
      {open && (
        <div className="mt-2 space-y-3">
          {caps && <CapsInfo caps={caps} />}

          <div className="grid grid-cols-2 gap-2">
            <FormField
              control={control}
              name={`indexers.${index}.movie_categories`}
              render={({ field }) => (
                <FormItem>
                  <div className="flex items-center gap-1">
                    <FormLabel className="text-[10px]">Movie Categories</FormLabel>
                    <CategoriesHint caps={caps} type="movie" />
                  </div>
                  <FormControl>
                    <Input placeholder="2000" className="h-8 text-xs" {...field} value={field.value || ''} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`indexers.${index}.tv_categories`}
              render={({ field }) => (
                <FormItem>
                  <div className="flex items-center gap-1">
                    <FormLabel className="text-[10px]">TV Categories</FormLabel>
                    <CategoriesHint caps={caps} type="tv" />
                  </div>
                  <FormControl>
                    <Input placeholder="5000" className="h-8 text-xs" {...field} value={field.value || ''} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <FormField
            control={control}
            name={`indexers.${index}.extra_search_terms`}
            render={({ field }) => (
              <FormItem>
                <FormLabel className="text-[10px]">Extra Search Terms</FormLabel>
                <FormControl>
                  <Input placeholder="(P73|LRO)" className="h-8 text-xs" {...field} value={field.value || ''} />
                </FormControl>
                <FormDescription className="text-[10px]">Appended to every search query for this indexer</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={control}
            name={`indexers.${index}.use_season_episode_params`}
            render={({ field }) => (
              <FormItem className="flex flex-row items-center space-x-2 space-y-0">
                <FormControl>
                  <Checkbox
                    checked={field.value !== false}
                    onCheckedChange={(v) => field.onChange(v === true ? undefined : false)}
                  />
                </FormControl>
                <FormLabel className="text-[10px]">Use season/episode in API</FormLabel>
                <FormDescription className="text-[10px]">Send season= and ep= to this indexer. Turn off if the indexer does not use them.</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="grid grid-cols-2 gap-2 pt-1 border-t border-border">
            <FormField
              control={control}
              name={`indexers.${index}.search_result_limit`}
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-[10px]">Search Result Limit</FormLabel>
                  <FormControl>
                    <Input type="number" min={0} max={5000} placeholder="0 = use global" className="h-8 text-xs" {...field} value={field.value === 0 || field.value == null ? '' : field.value} onChange={e => field.onChange(e.target.value === '' ? 0 : Number(e.target.value))} />
                  </FormControl>
                  <FormDescription className="text-[10px]">Max results from this indexer. 0 = use global.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={control}
              name={`indexers.${index}.include_year_in_search`}
              render={({ field }) => (
                <FormItem className="flex flex-row items-center space-x-2 space-y-0 pt-6">
                  <FormControl>
                    <Checkbox
                      checked={field.value === true}
                      onCheckedChange={(v) => field.onChange(v === true ? true : undefined)}
                    />
                  </FormControl>
                  <FormLabel className="text-[10px]">Include year in movie search</FormLabel>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={control}
            name={`indexers.${index}.search_title_language`}
            render={({ field }) => (
              <FormItem>
                <FormLabel className="text-[10px]">Search title language</FormLabel>
                <FormControl>
                  <Input placeholder="Use global or e.g. de-DE" className="h-8 text-xs" {...field} value={field.value || ''} />
                </FormControl>
                <FormDescription className="text-[10px]">TMDB language for movie title. Empty = use global.</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={control}
            name={`indexers.${index}.search_title_normalize`}
            render={({ field }) => (
              <FormItem className="flex flex-row items-center space-x-2 space-y-0">
                <FormControl>
                  <Checkbox
                    checked={field.value === true}
                    onCheckedChange={(v) => field.onChange(v === true ? true : undefined)}
                  />
                </FormControl>
                <FormLabel className="text-[10px]">Normalize title for search</FormLabel>
                <FormDescription className="text-[10px]">Apply umlaut→ascii to movie search query. Unset = use global.</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </div>
      )}
    </div>
  )
}

export function IndexerSettings({ control, indexerFields, appendIndexer, removeIndexer, watch, setValue, indexerCaps }) {
  return (
    <div className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {indexerFields.map((field, index) => {
                const currentType = watch(`indexers.${index}.type`) || 'newznab';
                const isEasynews = currentType === 'easynews';

                return (
                    <Card key={field.id} className="relative flex flex-col h-full">
                        <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="absolute right-1 top-1 h-8 w-8 text-destructive hover:text-destructive/90 z-10"
                            onClick={() => removeIndexer(index)}
                        >
                            <Trash2 className="h-4 w-4" />
                        </Button>
                        <CardHeader className="pb-3">
                            <CardTitle className="text-base truncate pr-8">
                                {watch(`indexers.${index}.name`) || `Indexer ${index + 1}`}
                            </CardTitle>
                        </CardHeader>
                        <CardContent className="space-y-3 flex-grow">
                            <FormField
                                control={control}
                                name={`indexers.${index}.enabled`}
                                render={({ field }) => (
                                    <FormItem className="flex flex-row items-center space-x-2 space-y-0">
                                        <FormControl>
                                            <Checkbox
                                                checked={field.value != null ? field.value : true}
                                                onCheckedChange={field.onChange}
                                            />
                                        </FormControl>
                                        <FormLabel className="text-xs">Enabled</FormLabel>
                                    </FormItem>
                                )}
                            />
                            <div className="space-y-2">
                                <Label className="text-sm font-medium">Preset / Type</Label>
                                <select
                                    className={cn(
                                      "flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm ring-offset-background",
                                      "placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                                    )}
                                    onChange={(e) => {
                                        const preset = INDEXER_PRESETS.find(p => p.name === e.target.value);
                                        if (preset) {
                                            setValue(`indexers.${index}.name`, preset.name);
                                            setValue(`indexers.${index}.url`, preset.url);
                                            setValue(`indexers.${index}.api_path`, preset.api_path || '/api');
                                            setValue(`indexers.${index}.type`, preset.type);
                                        }
                                    }}
                                    value={INDEXER_PRESETS.find(p => p.name === watch(`indexers.${index}.name`))?.name || (watch(`indexers.${index}.type`) === 'easynews' ? 'Easynews (Experimental)' : 'Custom Newznab')}
                                >
                                    {INDEXER_PRESETS.map(preset => (
                                        <option key={preset.name} value={preset.name}>{preset.name}</option>
                                    ))}
                                </select>
                            </div>
                            
                            {!isEasynews && (
                                <FormField
                                    control={control}
                                    name={`indexers.${index}.url`}
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>URL</FormLabel>
                                            <FormControl>
                                                <Input placeholder="https://api.nzbgeek.info" className="h-8 text-xs" {...field} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            )}
                            
                            {!isEasynews && (
                                <FormField
                                    control={control}
                                    name={`indexers.${index}.api_path`}
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>API Path</FormLabel>
                                            <FormControl>
                                                <Input placeholder="/api" className="h-8 text-xs" {...field} />
                                            </FormControl>
                                            <FormDescription className="text-[10px]">API endpoint path (default: /api, Tabula Rasa: /api/v1)</FormDescription>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            )}
                            
                            {!isEasynews ? (
                                <FormField
                                    control={control}
                                    name={`indexers.${index}.api_key`}
                                    render={({ field }) => (
                                        <FormItem>
                                            <FormLabel>API Key</FormLabel>
                                            <FormControl>
                                                <PasswordInput className="h-8 text-xs" {...field} />
                                            </FormControl>
                                            <FormMessage />
                                        </FormItem>
                                    )}
                                />
                            ) : (
                                <>
                                    <FormField
                                        control={control}
                                        name={`indexers.${index}.username`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel>Username</FormLabel>
                                                <FormControl>
                                                    <Input className="h-8 text-xs" {...field} />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                    <FormField
                                        control={control}
                                        name={`indexers.${index}.password`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel>Password</FormLabel>
                                                <FormControl>
                                                    <PasswordInput className="h-8 text-xs" {...field} />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                </>
                            )}

                            {!isEasynews && (
                                <div className="grid grid-cols-2 gap-2 mt-2">
                                    <FormField
                                        control={control}
                                        name={`indexers.${index}.api_hits_day`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel className="text-[10px]">Hits/Day</FormLabel>
                                                <FormControl>
                                                    <Input type="number" placeholder="100" className="h-8 text-xs" {...field} onChange={e => field.onChange(Number(e.target.value))} />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                    <FormField
                                        control={control}
                                        name={`indexers.${index}.downloads_day`}
                                        render={({ field }) => (
                                            <FormItem>
                                                <FormLabel className="text-[10px]">DLs/Day</FormLabel>
                                                <FormControl>
                                                    <Input type="number" placeholder="50" className="h-8 text-xs" {...field} onChange={e => field.onChange(Number(e.target.value))} />
                                                </FormControl>
                                                <FormMessage />
                                            </FormItem>
                                        )}
                                    />
                                </div>
                            )}

                            {!isEasynews && <SearchSettings control={control} index={index} watch={watch} indexerCaps={indexerCaps} />}
                        </CardContent>
                    </Card>
                )
            })}

            {/* Add Indexer Card */}
            <Button
                type="button"
                variant="outline"
                onClick={() => appendIndexer({ name: '', url: '', api_path: '/api', api_key: '', type: 'newznab', api_hits_day: 0, downloads_day: 0, enabled: true, username: '', password: '', movie_categories: '', tv_categories: '', extra_search_terms: '', use_season_episode_params: undefined, search_result_limit: 0, include_year_in_search: undefined, search_title_language: '', search_title_normalize: undefined })}
                className={cn(
                  "flex flex-col items-center justify-center p-4 h-auto min-h-[180px] border-2 border-dashed border-muted-foreground/25 hover:border-muted-foreground/50 hover:bg-accent/50 transition-all group"
                )}
            >
                <div className="flex items-center justify-center w-10 h-10 rounded-full bg-primary/10 group-hover:bg-primary/20 transition-colors mb-2">
                    <Plus className="w-6 h-6 text-primary" />
                </div>
                <span className="font-medium text-muted-foreground group-hover:text-foreground transition-colors">Add New Indexer</span>
                <span className="text-xs text-muted-foreground/80 mt-1">Configure another search source</span>
            </Button>
        </div>

        {indexerFields.length === 0 && (
            <div className="hidden">
                {/* This is handled by the skeleton card now */}
            </div>
        )}
    </div>
  )
}
