import { useState, useEffect, useCallback, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Loader2, Search, Film, Tv, ExternalLink, ChevronDown } from "lucide-react"
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

function dedupeStreamsByUrl(streams) {
  if (!Array.isArray(streams)) return []
  const seen = new Set()
  return streams.filter((s) => {
    const key = s.url || s.name || ''
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
}

export function SearchPage({ authToken, config, sendCommand, ws }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [searchError, setSearchError] = useState(null)
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const [selected, setSelected] = useState(null)
  const [availStreams, setAvailStreams] = useState([])
  const [validatedStreams, setValidatedStreams] = useState([])
  const [streamSearchDone, setStreamSearchDone] = useState(false)
  const [streamsLoading, setStreamsLoading] = useState(false)
  const [streamsError, setStreamsError] = useState(null)
  const [loadingMore, setLoadingMore] = useState(false)
  const searchContainerRef = useRef(null)
  const currentSearchRef = useRef(null) // { type, id } to match stream_result/done

  const debouncedQuery = useDebounce(query.trim(), DEBOUNCE_MS)

  const fetchSearch = useCallback(async (q) => {
    if (!q || !authToken) {
      setResults([])
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
        setResults([])
        return
      }
      const data = await res.json()
      setResults(Array.isArray(data) ? data : [])
      setDropdownOpen(true)
    } catch (err) {
      setSearchError('Search failed')
      setResults([])
    } finally {
      setSearchLoading(false)
    }
  }, [authToken])

  useEffect(() => {
    if (debouncedQuery) {
      fetchSearch(debouncedQuery)
    } else {
      setResults([])
      setDropdownOpen(false)
      setSearchLoading(false)
      setSearchError(null)
    }
  }, [debouncedQuery, fetchSearch])

  useEffect(() => {
    const onMessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data)
        if (msg.type === 'stream_result' && msg.payload && currentSearchRef.current) {
          setValidatedStreams((prev) => dedupeStreamsByUrl([...prev, msg.payload]))
        } else if (msg.type === 'stream_search_done' && currentSearchRef.current) {
          setStreamSearchDone(true)
          setStreamsLoading(false)
          setLoadingMore(false)
          currentSearchRef.current = null
        }
      } catch (_) {}
    }
    if (!ws) return
    ws.addEventListener('message', onMessage)
    return () => ws.removeEventListener('message', onMessage)
  }, [ws])

  useEffect(() => {
    function handleClickOutside(e) {
      if (searchContainerRef.current && !searchContainerRef.current.contains(e.target)) {
        setDropdownOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const handleResultClick = useCallback(async (item) => {
    if (!authToken) return
    const contentType = item.media_type === 'movie' ? 'movie' : 'series'
    const id = item.id || (item.media_type === 'movie' ? String(item.tmdb_id) : `tmdb:${item.tmdb_id}`)
    setSelected(item)
    setAvailStreams([])
    setValidatedStreams([])
    setStreamSearchDone(false)
    setStreamsError(null)
    setDropdownOpen(false)
    setStreamsLoading(true)
    currentSearchRef.current = { type: contentType, id }

    try {
      const availRes = await fetch(
        `/api/streams/avail?type=${encodeURIComponent(contentType)}&id=${encodeURIComponent(id)}`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      )
      if (availRes.ok) {
        const availData = await availRes.json()
        setAvailStreams(availData.streams || [])
      }
    } catch (_) {}

    if (sendCommand && ws && ws.readyState === WebSocket.OPEN) {
      sendCommand('stream_search', { type: contentType, id })
    } else {
      setStreamsLoading(false)
      setStreamSearchDone(true)
      currentSearchRef.current = null
    }
  }, [authToken, sendCommand, ws])

  const handleLoadMore = useCallback(() => {
    if (!selected || !authToken || !currentSearchRef.current) return
    const contentType = selected.media_type === 'movie' ? 'movie' : 'series'
    const id = selected.id || (selected.media_type === 'movie' ? String(selected.tmdb_id) : `tmdb:${selected.tmdb_id}`)
    setLoadingMore(true)
    currentSearchRef.current = { type: contentType, id }
    if (sendCommand && ws && ws.readyState === WebSocket.OPEN) {
      sendCommand('stream_search', { type: contentType, id })
    } else {
      setLoadingMore(false)
    }
  }, [selected, authToken, sendCommand, ws])

  const allStreams = dedupeStreamsByUrl([...availStreams, ...validatedStreams])
  const showLoadMore = selected && streamSearchDone && !streamsLoading && !loadingMore && allStreams.length > 0

  const addonBaseUrl = config?.addon_base_url ? config.addon_base_url.replace(/\/$/, '') : window.location.origin
  const manifestUrl = authToken ? `${addonBaseUrl}/${authToken}/manifest.json` : addonBaseUrl

  return (
    <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6 px-4 lg:px-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            Search
          </CardTitle>
          <CardDescription>
            Search movies and TV shows. Results show a dropdown with posters. Click a result to see AvailNZB streams first, then validated streams arrive via WebSocket.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="relative" ref={searchContainerRef}>
            <div className="relative">
              <Input
                type="text"
                placeholder="Search movies or TV shows..."
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onFocus={() => results.length > 0 && setDropdownOpen(true)}
                className="pr-10"
              />
              {searchLoading && (
                <div className="absolute right-3 top-1/2 -translate-y-1/2">
                  <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                </div>
              )}
              {!searchLoading && results.length > 0 && (
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
            {dropdownOpen && results.length > 0 && (
              <ul
                className="absolute z-50 mt-1 max-h-[min(70vh,400px)] w-full overflow-auto rounded-md border bg-popover py-1 shadow-md"
                role="listbox"
              >
                {results.map((item) => (
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
                          <img
                            src={item.poster_url}
                            alt=""
                            className="h-full w-full object-cover"
                            loading="lazy"
                          />
                        ) : (
                          <div className="flex h-full w-full items-center justify-center">
                            {item.media_type === 'movie' ? (
                              <Film className="h-5 w-5 text-muted-foreground" />
                            ) : (
                              <Tv className="h-5 w-5 text-muted-foreground" />
                            )}
                          </div>
                        )}
                      </div>
                      <div className="min-w-0 flex-1">
                        <span className="font-medium truncate block">{item.title}</span>
                        {item.year && (
                          <span className="text-muted-foreground text-sm">{item.year}</span>
                        )}
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
          {searchError && (
            <p className="text-sm text-destructive">{searchError}</p>
          )}
        </CardContent>
      </Card>

      {selected && (
        <Card>
          <CardHeader>
            <CardTitle>{selected.title} {selected.year && `(${selected.year})`}</CardTitle>
            <CardDescription>
              AvailNZB results appear first; validated streams stream in below (up to max_streams × 2). Open the addon in Stremio to play.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {(streamsLoading || loadingMore) && (
              <div className="flex items-center gap-2 py-2">
                <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                <span className="text-sm text-muted-foreground">
                  {loadingMore ? 'Loading more...' : 'Validating streams...'}
                </span>
              </div>
            )}
            {streamsError && (
              <p className="text-sm text-destructive">{streamsError}</p>
            )}
            {!streamsLoading && !streamsError && allStreams.length === 0 && streamSearchDone && (
              <p className="text-sm text-muted-foreground">No streams found.</p>
            )}
            {allStreams.length > 0 && (
              <>
                <ul className="space-y-2">
                  {allStreams.map((s, i) => (
                    <li
                      key={s.url || i}
                      className="rounded-lg border p-3 text-sm"
                    >
                      <div className="font-medium">{s.name || s.title || 'Stream'}</div>
                      {s.description && (
                        <div className="mt-1 text-muted-foreground line-clamp-2">{s.description}</div>
                      )}
                    </li>
                  ))}
                </ul>
                {showLoadMore && (
                  <div className="pt-2">
                    <Button variant="outline" size="sm" onClick={handleLoadMore} disabled={loadingMore}>
                      {loadingMore ? (
                        <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      ) : null}
                      Load more
                    </Button>
                  </div>
                )}
                <div className="pt-2">
                  <Button variant="outline" size="sm" asChild>
                    <a href={manifestUrl} target="_blank" rel="noopener noreferrer">
                      <ExternalLink className="h-4 w-4 mr-2" />
                      Open addon in Stremio
                    </a>
                  </Button>
                </div>
              </>
            )}
          </CardContent>
        </Card>
      )}

      {!authToken && (
        <p className="text-sm text-muted-foreground">Sign in to search and load streams.</p>
      )}
    </div>
  )
}
