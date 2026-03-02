import { useState, useEffect, useCallback, useRef } from 'react'
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Loader2, Search, Film, Tv, ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"

const DEBOUNCE_MS = 350

function useDebounce(value, delay) {
  const [debouncedValue, setDebouncedValue] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebouncedValue(value), delay)
    return () => clearTimeout(t)
  }, [value, delay])
  return debouncedValue
}

function formatSize(bytes) {
  if (bytes <= 0) return '—'
  const gb = bytes / (1024 ** 3)
  if (gb >= 1) return `${gb.toFixed(1)} GB`
  const mb = bytes / (1024 ** 2)
  return `${mb.toFixed(0)} MB`
}

const authHeader = (token) => (token ? { Authorization: `Bearer ${token}` } : {})

export function SearchPage({ authToken }) {
  const [query, setQuery] = useState('')
  const [tmdbResults, setTmdbResults] = useState([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [searchError, setSearchError] = useState(null)
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const [selected, setSelected] = useState(null)
  const [tvDetails, setTvDetails] = useState(null)
  const [tvDetailsLoading, setTvDetailsLoading] = useState(false)
  const [tvDetailsError, setTvDetailsError] = useState(null)
  const [selectedSeason, setSelectedSeason] = useState(null)
  const [episodes, setEpisodes] = useState([])
  const [episodesLoading, setEpisodesLoading] = useState(false)
  const [selectedEpisode, setSelectedEpisode] = useState(null)
  const [releasesData, setReleasesData] = useState(null)
  const [releasesLoading, setReleasesLoading] = useState(false)
  const [releasesError, setReleasesError] = useState(null)
  const [filterAvailability, setFilterAvailability] = useState(null)
  const [filterStreamId, setFilterStreamId] = useState(null)
  const [sortByStreamId, setSortByStreamId] = useState(null)
  const searchContainerRef = useRef(null)

  const debouncedQuery = useDebounce(query.trim(), DEBOUNCE_MS)

  const fetchTmdbSearch = useCallback(async (q) => {
    if (!q || !authToken) {
      setTmdbResults([])
      return
    }
    setSearchLoading(true)
    setSearchError(null)
    try {
      const res = await fetch(`/api/tmdb/search?q=${encodeURIComponent(q)}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      })
      if (!res.ok) {
        if (res.status === 401) setSearchError('Not authorized')
        else setSearchError('Search failed')
        setTmdbResults([])
        return
      }
      const data = await res.json()
      setTmdbResults(Array.isArray(data) ? data : [])
      setDropdownOpen(true)
    } catch {
      setSearchError('Search failed')
      setTmdbResults([])
    } finally {
      setSearchLoading(false)
    }
  }, [authToken])

  useEffect(() => {
    if (debouncedQuery) {
      fetchTmdbSearch(debouncedQuery)
    } else {
      setTmdbResults([])
      setDropdownOpen(false)
      setSearchLoading(false)
      setSearchError(null)
    }
  }, [debouncedQuery, fetchTmdbSearch])

  useEffect(() => {
    function handleClickOutside(e) {
      if (searchContainerRef.current && !searchContainerRef.current.contains(e.target)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const fetchReleases = useCallback(async (contentType, id) => {
    setReleasesLoading(true)
    setReleasesError(null)
    try {
      const res = await fetch(
        `/api/search/releases?type=${encodeURIComponent(contentType)}&id=${encodeURIComponent(id)}`,
        { headers: authHeader(authToken) }
      )
      if (!res.ok) {
        setReleasesError(res.status === 401 ? 'Not authorized' : 'Failed to load releases')
        setReleasesData(null)
        return
      }
      const data = await res.json()
      setReleasesData({ streams: data.streams || [], releases: data.releases || [] })
    } catch {
      setReleasesError('Failed to load releases')
      setReleasesData(null)
    } finally {
      setReleasesLoading(false)
    }
  }, [authToken])

  const handleResultClick = useCallback(async (item) => {
    if (!authToken) return
    setSelected(item)
    setReleasesData(null)
    setReleasesError(null)
    setTvDetails(null)
    setTvDetailsError(null)
    setSelectedSeason(null)
    setEpisodes([])
    setSelectedEpisode(null)
    setFilterAvailability(null)
    setFilterStreamId(null)
    setSortByStreamId(null)
    setDropdownOpen(false)

    if (item.media_type === 'movie') {
      const id = item.id || String(item.tmdb_id)
      fetchReleases('movie', id)
      return
    }

    // TV show: fetch seasons first
    const tmdbId = item.tmdb_id
    setTvDetailsLoading(true)
    try {
      const res = await fetch(`/api/tmdb/tv/${tmdbId}/details`, { headers: authHeader(authToken) })
      if (!res.ok) {
        setTvDetailsError(res.status === 401 ? 'Not authorized' : 'Failed to load show details')
        setTvDetails(null)
        return
      }
      const data = await res.json()
      setTvDetails({ name: data.name, seasons: data.seasons || [] })
    } catch {
      setTvDetailsError('Failed to load show details')
      setTvDetails(null)
    } finally {
      setTvDetailsLoading(false)
    }
  }, [authToken, fetchReleases])

  // When TV season is selected, fetch episodes
  useEffect(() => {
    if (!selected || selected.media_type !== 'tv' || selectedSeason == null) {
      setEpisodes([])
      setSelectedEpisode(null)
      return
    }
    setEpisodesLoading(true)
    setSelectedEpisode(null)
    fetch(`/api/tmdb/tv/${selected.tmdb_id}/seasons/${selectedSeason}`, { headers: authHeader(authToken) })
      .then((res) => (res.ok ? res.json() : Promise.reject(new Error('Failed to load episodes'))))
      .then((data) => setEpisodes(data.episodes || []))
      .catch(() => setEpisodes([]))
      .finally(() => setEpisodesLoading(false))
  }, [selected, selectedSeason, authToken])

  // When TV episode is selected, fetch releases
  useEffect(() => {
    if (!selected || selected.media_type !== 'tv' || selectedSeason == null || selectedEpisode == null) return
    const id = `tmdb:${selected.tmdb_id}:${selectedSeason}:${selectedEpisode}`
    fetchReleases('series', id)
  }, [selected, selectedSeason, selectedEpisode, fetchReleases])

  const streams = releasesData?.streams ?? []

  // Auto-select the first stream for sorting when data arrives
  const activeSortStream = sortByStreamId ?? streams[0]?.id ?? null

  const filteredAndSortedReleases = releasesData?.releases
    ? (() => {
        let list = [...releasesData.releases]
        if (filterAvailability) {
          list = list.filter((r) => r.availability === filterAvailability)
        }
        if (filterStreamId) {
          list = list.filter((r) => {
            const tag = r.stream_tags?.find((t) => t.stream_id === filterStreamId)
            return tag?.fits
          })
        }
        if (activeSortStream) {
          const availOrder = { Available: 2, Unknown: 1, Unavailable: 0 }
          list.sort((a, b) => {
            const sa = a.stream_tags?.find((t) => t.stream_id === activeSortStream)?.score ?? 0
            const sb = b.stream_tags?.find((t) => t.stream_id === activeSortStream)?.score ?? 0
            if (sb !== sa) return sb - sa
            // Tie-break by availability so Available sorts first when scores match
            return (availOrder[b.availability] ?? 0) - (availOrder[a.availability] ?? 0)
          })
        }
        return list
      })()
    : []

  return (
    <div className="py-4 md:py-6 px-4 lg:px-6">
      <div className="rounded-lg border bg-card text-card-foreground shadow-sm">
        <div className="p-4 md:p-6 flex flex-col gap-6">
        <div>
          <p className="text-sm text-muted-foreground mb-4">
            Search movies and TV shows. Pick a title to see all releases from indexers and AvailNZB. Use tags to filter by availability or stream, and to sort by stream priority.
          </p>

          <div className="relative max-w-xl" ref={searchContainerRef}>
            <div className="relative">
              <Input
                type="text"
                placeholder="Search movies or TV shows..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onFocus={() => tmdbResults.length > 0 && setDropdownOpen(true)}
                className="pr-10"
              />
              {searchLoading && (
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                </div>
              )}
              {!searchLoading && tmdbResults.length > 0 && (
                <button
                  type="button"
                  onClick={() => setDropdownOpen((o) => !o)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label="Toggle results"
                >
                  <ChevronDown className={cn("h-4 w-4 transition-transform", dropdownOpen && "rotate-180")} />
                </button>
              )}
            </div>
            {dropdownOpen && tmdbResults.length > 0 && (
              <ul
                className="absolute z-50 mt-1 max-h-[min(70vh,400px)] w-full overflow-auto rounded-md border bg-popover py-1 shadow-md"
                role="listbox"
              >
                {tmdbResults.map((item) => (
                  <li key={`${item.media_type}-${item.tmdb_id}`} role="option">
                    <button
                      type="button"
                      onClick={() => handleResultClick(item)}
                      className={cn(
                        "w-full flex items-center gap-3 rounded-sm px-3 py-2 text-left transition-colors hover:bg-muted/80 focus:bg-muted/80",
                        selected?.tmdb_id === item.tmdb_id && selected?.media_type === item.media_type && "bg-muted/60"
                      )}
                    >
                      <div className="h-12 w-10 shrink-0 overflow-hidden rounded bg-muted">
                        {item.poster_url ? (
                          <img src={item.poster_url} alt="" className="h-full w-full object-cover" loading="lazy" />
                        ) : (
                          <div className="flex h-full w-full items-center justify-center">
                            {item.media_type === 'movie' ? <Film className="h-5 w-5 text-muted-foreground" /> : <Tv className="h-5 w-5 text-muted-foreground" />}
                          </div>
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <span className="font-medium truncate block">{item.title}</span>
                        {item.year && <span className="text-muted-foreground text-sm">{item.year}</span>}
                      </div>
                      <Badge variant="secondary" className="shrink-0 text-xs">
                        {item.media_type === 'movie' ? 'Movie' : 'TV'}
                      </Badge>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
          {searchError && <p className="text-sm text-destructive mt-2">{searchError}</p>}
        </div>

        {selected && (
          <>
            <div className="flex gap-4 rounded-md border bg-muted/30 p-4">
              <div className="h-32 w-24 shrink-0 overflow-hidden rounded-md bg-muted">
                {selected.poster_url ? (
                  <img
                    src={selected.poster_url}
                    alt=""
                    className="h-full w-full object-cover"
                    loading="lazy"
                  />
                ) : (
                  <div className="flex h-full w-full items-center justify-center">
                    {selected.media_type === 'movie' ? (
                      <Film className="h-10 w-10 text-muted-foreground" />
                    ) : (
                      <Tv className="h-10 w-10 text-muted-foreground" />
                    )}
                  </div>
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2 flex-wrap">
                  <h3 className="text-lg font-semibold">{selected.title}</h3>
                  {selected.year && (
                    <span className="text-muted-foreground text-sm">({selected.year})</span>
                  )}
                  <Badge variant="secondary" className="text-xs">
                    {selected.media_type === 'movie' ? 'Movie' : 'TV'}
                  </Badge>
                </div>
                {selected.overview && (
                  <p className="text-sm text-muted-foreground mt-2 line-clamp-3">{selected.overview}</p>
                )}
                {selected.media_type === 'tv' && (
                  <div className="mt-3 flex flex-wrap items-center gap-3">
                    {tvDetailsLoading && (
                      <span className="text-sm text-muted-foreground flex items-center gap-1">
                        <Loader2 className="h-4 w-4 animate-spin" /> Loading seasons…
                      </span>
                    )}
                    {tvDetailsError && (
                      <p className="text-sm text-destructive">{tvDetailsError}</p>
                    )}
                    {tvDetails && tvDetails.seasons?.length > 0 && (
                      <>
                        <label className="text-sm font-medium">
                          Season
                          <select
                            value={selectedSeason ?? ''}
                            onChange={(e) => setSelectedSeason(e.target.value === '' ? null : parseInt(e.target.value, 10))}
                            className="ml-2 rounded-md border bg-background px-2 py-1 text-sm"
                          >
                            <option value="">Select season</option>
                            {tvDetails.seasons
                              .filter((se) => se.season_number >= 0)
                              .map((se) => (
                                <option key={se.season_number} value={se.season_number}>
                                  {se.season_number === 0 ? 'Specials' : `Season ${se.season_number}`}
                                  {se.episode_count > 0 ? ` (${se.episode_count} episodes)` : ''}
                                </option>
                              ))}
                          </select>
                        </label>
                        {selectedSeason != null && (
                          <label className="text-sm font-medium">
                            Episode
                            <select
                              value={selectedEpisode ?? ''}
                              onChange={(e) => setSelectedEpisode(e.target.value === '' ? null : parseInt(e.target.value, 10))}
                              disabled={episodesLoading}
                              className="ml-2 rounded-md border bg-background px-2 py-1 text-sm disabled:opacity-50"
                            >
                              <option value="">
                                {episodesLoading ? 'Loading…' : 'Select episode'}
                              </option>
                              {episodes.map((ep) => (
                                <option key={ep.episode_number} value={ep.episode_number}>
                                  E{ep.episode_number} {ep.name ? `– ${ep.name}` : ''}
                                </option>
                              ))}
                            </select>
                          </label>
                        )}
                      </>
                    )}
                  </div>
                )}
              </div>
            </div>

            {releasesLoading && (
              <div className="flex items-center gap-2 py-6">
                <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                <span className="text-sm text-muted-foreground">Loading releases from indexers and AvailNZB…</span>
              </div>
            )}
            {releasesError && <p className="text-sm text-destructive">{releasesError}</p>}

            {selected.media_type === 'tv' && !releasesData && !releasesLoading && tvDetails && (
              <p className="text-sm text-muted-foreground">Choose season and episode above to load releases.</p>
            )}

            {!releasesLoading && !releasesError && releasesData && (
              <>
                <div className="flex flex-wrap items-center gap-2 text-sm">
                  <span className="text-muted-foreground">Filters:</span>
                  <Badge
                    variant={filterAvailability === null ? "secondary" : "outline"}
                    className={cn("cursor-pointer", filterAvailability === null && "ring-2 ring-primary/20")}
                    onClick={() => setFilterAvailability(null)}
                  >
                    All availability
                  </Badge>
                  {['Available', 'Unavailable', 'Unknown'].map((av) => (
                    <Badge
                      key={av}
                      variant={filterAvailability === av ? "default" : "outline"}
                      className="cursor-pointer"
                      onClick={() => setFilterAvailability((prev) => (prev === av ? null : av))}
                    >
                      {av}
                    </Badge>
                  ))}
                  <span className="text-muted-foreground ml-2">Stream:</span>
                  <Badge
                    variant={filterStreamId === null ? "secondary" : "outline"}
                    className={cn("cursor-pointer", filterStreamId === null && "ring-2 ring-primary/20")}
                    onClick={() => setFilterStreamId(null)}
                  >
                    All streams
                  </Badge>
                  {streams.map((st) => (
                    <Badge
                      key={st.id}
                      variant={filterStreamId === st.id ? "default" : "outline"}
                      className="cursor-pointer"
                      onClick={() => setFilterStreamId((prev) => (prev === st.id ? null : st.id))}
                    >
                      {st.name} (fits)
                    </Badge>
                  ))}
                  <span className="text-muted-foreground ml-2">Sort by:</span>
                  {streams.map((st) => (
                    <Badge
                      key={st.id}
                      variant={activeSortStream === st.id ? "default" : "outline"}
                      className="cursor-pointer"
                      onClick={() => setSortByStreamId(st.id)}
                    >
                      {st.name} priority
                    </Badge>
                  ))}
                </div>

                <p className="text-sm text-muted-foreground">
                  {filteredAndSortedReleases.length} release{filteredAndSortedReleases.length !== 1 ? 's' : ''}. Click a tag to filter or sort.
                </p>

                {filteredAndSortedReleases.length === 0 ? (
                  <p className="text-sm text-muted-foreground">No releases match the current filters.</p>
                ) : (
                  <div className="rounded-md border overflow-x-auto -mx-1">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="text-left p-3 font-medium">Title</th>
                          <th className="text-right p-3 font-medium w-24">Size</th>
                          <th className="text-left p-3 font-medium w-28">Indexer</th>
                          <th className="text-left p-3 font-medium">Availability</th>
                          <th className="text-left p-3 font-medium">Streams</th>
                        </tr>
                      </thead>
                      <tbody>
                        {filteredAndSortedReleases.map((r, i) => (
                          <tr key={r.details_url || `${r.title}-${r.size}-${i}`} className="border-b last:border-0 hover:bg-muted/30">
                            <td className="p-3 font-mono text-xs truncate max-w-[280px]" title={r.title}>{r.title}</td>
                            <td className="p-3 text-right text-muted-foreground">{formatSize(r.size)}</td>
                            <td className="p-3 text-muted-foreground">{r.indexer || '—'}</td>
                            <td className="p-3">
                              <Badge
                                variant={r.availability === 'Available' ? 'default' : r.availability === 'Unavailable' ? 'destructive' : 'secondary'}
                                className="cursor-pointer"
                                onClick={() => setFilterAvailability((prev) => (prev === r.availability ? null : r.availability))}
                              >
                                {r.availability}
                              </Badge>
                            </td>
                            <td className="p-3">
                              <div className="flex flex-wrap gap-1">
                                {(r.stream_tags || []).map((t) => (
                                  <Badge
                                    key={t.stream_id}
                                    variant={t.fits ? 'secondary' : 'outline'}
                                    className={cn("cursor-pointer", !t.fits && "opacity-60")}
                                    onClick={() => setFilterStreamId((prev) => (prev === t.stream_id ? null : t.stream_id))}
                                    title={`Score: ${t.score}`}
                                  >
                                    {t.stream_name} {t.fits ? '✓' : '✗'}
                                  </Badge>
                                ))}
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </>
            )}
          </>
        )}

        {!authToken && (
          <p className="text-sm text-muted-foreground">Sign in to search and view releases.</p>
        )}
        </div>
      </div>
    </div>
  )
}
