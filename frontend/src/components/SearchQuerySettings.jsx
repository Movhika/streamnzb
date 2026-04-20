import React, { useEffect, useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, focusDialogCloseButton } from "@/components/ui/dialog"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { ConfirmDialog } from "@/components/ConfirmDialog"
import { apiFetch } from "@/api"
import { Copy, Plus, Settings, Trash2 } from "lucide-react"

const CACHE_CLEARED_SUFFIX = ' Search cache cleared.'

const TITLE_LANGUAGE_OPTIONS = [
  { value: '', label: 'Original' },
  { value: 'en-US', label: 'English' },
  { value: 'de-DE', label: 'German' },
  { value: 'fr-FR', label: 'French' },
  { value: 'es-ES', label: 'Spanish' },
  { value: 'it-IT', label: 'Italian' },
  { value: 'nl-NL', label: 'Dutch' },
  { value: 'pl-PL', label: 'Polish' },
  { value: 'pt-BR', label: 'Portuguese (Brazil)' },
  { value: 'pt-PT', label: 'Portuguese (Portugal)' },
  { value: 'sv-SE', label: 'Swedish' },
  { value: 'no-NO', label: 'Norwegian' },
  { value: 'da-DK', label: 'Danish' },
  { value: 'fi-FI', label: 'Finnish' },
  { value: 'cs-CZ', label: 'Czech' },
  { value: 'sk-SK', label: 'Slovak' },
  { value: 'hu-HU', label: 'Hungarian' },
  { value: 'ro-RO', label: 'Romanian' },
  { value: 'tr-TR', label: 'Turkish' },
  { value: 'ru-RU', label: 'Russian' },
  { value: 'uk-UA', label: 'Ukrainian' },
  { value: 'ja-JP', label: 'Japanese' },
  { value: 'ko-KR', label: 'Korean' },
  { value: 'zh-CN', label: 'Chinese (Simplified)' },
  { value: 'zh-TW', label: 'Chinese (Traditional)' },
]

const SERIES_SCOPE_OPTIONS = [
  { value: 'season_episode', label: 'Season/Episode' },
  { value: 'season', label: 'Season' },
  { value: 'none', label: 'None' },
]

const YEAR_HINT = 'Adds the metadata year to text searches and also checks it during result validation. With ID searches it only affects validation.'
const TV_SCOPE_HINT_ITEMS = [
  {
    label: 'Season/Episode',
    text: 'Targets one episode. ID mode uses season and episode params, and Text mode searches with an S01E01-style query.',
  },
  {
    label: 'Season',
    text: 'Broadens to the whole season. Validation still keeps only releases that can contain the requested episode.',
  },
  {
    label: 'None',
    text: 'Searches only by series title or ID. Validation trims the broader results back to releases that can contain the requested episode.',
  },
]
const TITLE_LANGUAGE_HINT_ITEMS = [
  {
    label: 'Text Search',
    text: 'Builds the outgoing search title from the selected metadata language.',
  },
  {
    label: 'ID Search',
    text: 'Does not change the ID request itself. It only changes which metadata title is used for result validation.',
  },
  {
    label: 'Normalization',
    text: 'Uses the same normalized title basis for search and validation. Examples: `König der Löwen` becomes `Koenig der Loewen`, and `Your Friends & Neighbors` becomes `Your Friends Neighbors`.',
  },
]
const SEARCH_LIMIT_HINT = '0 uses Max. For Newznab indexers, StreamNZB reads the max from caps. If caps are unavailable, it falls back to 2000. Any explicit value is sent as-is.'

function normalizeSeriesSearchScope(scope) {
  switch ((scope || '').trim().toLowerCase()) {
    case 'season_episode':
    case 'season':
    case 'none':
      return scope.trim().toLowerCase()
    case 'episode_param':
    case 'episode_query':
    case 'season_param':
    case 'season_query':
      return scope.trim().toLowerCase()
    default:
      return 'season_episode'
  }
}

function normalizeSeriesScopeSelection(scope) {
  const raw = (scope || '').trim().toLowerCase()
  switch (raw) {
    case 'season_episode':
      return 'season_episode'
    case 'season':
      return 'season'
    case 'none':
      return 'none'
    default:
      break
  }
  switch (normalizeSeriesSearchScope(scope)) {
    case 'season_param':
    case 'season_query':
      return 'season'
    case 'none':
      return 'none'
    default:
      return 'season_episode'
  }
}

function resolveSeriesSearchScope(selection) {
  switch ((selection || '').trim().toLowerCase()) {
    case 'season':
      return 'season'
    case 'none':
      return 'none'
    case 'season_episode':
    default:
      return 'season_episode'
  }
}

function normalizeName(value) {
  return (value || '').trim().toLowerCase()
}

function truncateCompactValue(value, maxLength = 28) {
  const text = String(value || '').trim()
  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength - 3)}...`
}

function normalizeQueryYearSetting(searchMode, includeYear, legacyIncludeYearInTextSearch) {
  if (includeYear != null) return includeYear === true
  if (legacyIncludeYearInTextSearch != null) return legacyIncludeYearInTextSearch === true
  return String(searchMode || '').trim().toLowerCase() !== 'id'
}

function titleLanguageLabel(value) {
  return TITLE_LANGUAGE_OPTIONS.find((option) => option.value === value)?.label || 'Original'
}

function assignedStreamsForQuery(streamsByName, kind, queryName) {
  const field = kind === 'movie' ? 'movie_search_queries' : 'series_search_queries'
  const target = normalizeName(queryName)
  if (!target || !streamsByName) return []
  return Object.values(streamsByName)
    .filter(Boolean)
    .filter((stream) => Array.isArray(stream[field]) && stream[field].some((name) => normalizeName(name) === target))
    .map((stream) => stream.username)
}

function mapStreamsByUsername(streams) {
  return (Array.isArray(streams) ? streams : []).reduce((acc, stream) => {
    if (!stream?.username) return acc
    acc[stream.username] = stream
    return acc
  }, {})
}

function emptyDraft(kind) {
  return {
    name: kind === 'movie' ? 'MovieQuery01' : 'TVQuery01',
    search_mode: 'id',
    movie_categories: kind === 'movie' ? '2000' : undefined,
    tv_categories: kind === 'series' ? '5000' : undefined,
    extra_search_terms: '',
    search_result_limit: 0,
    search_title_language: '',
    include_year: false,
    series_search_scope: kind === 'series' ? 'season_episode' : undefined,
  }
}

function normalizeDraft(kind, draft) {
  const value = draft || {}
  const searchMode = value.search_mode || 'id'
  return {
    name: (value.name || '').trim(),
    search_mode: searchMode,
    extra_search_terms: value.extra_search_terms || '',
    search_result_limit: value.search_result_limit ?? 0,
    movie_categories: kind === 'movie' ? (value.movie_categories ?? '2000') : undefined,
    tv_categories: kind === 'series' ? (value.tv_categories ?? '5000') : undefined,
    search_title_language: value.search_title_language || '',
    include_year: normalizeQueryYearSetting(searchMode, value.include_year, value.include_year_in_text_search),
    series_search_scope: kind === 'series'
      ? normalizeSeriesScopeSelection(value.series_search_scope)
      : undefined,
  }
}

function persistableDraft(kind, draft) {
  const next = normalizeDraft(kind, draft)
  if (kind === 'series') {
    const resolvedScope = resolveSeriesSearchScope(next.series_search_scope)
    next.series_search_scope = resolvedScope
  }
  return next
}

function comparableQuerySignature(kind, draft) {
  const value = normalizeDraft(kind, draft)
  return JSON.stringify({
    search_mode: value.search_mode || 'id',
    movie_categories: kind === 'movie' ? String(value.movie_categories ?? '').trim() : '',
    tv_categories: kind === 'series' ? String(value.tv_categories ?? '').trim() : '',
    extra_search_terms: String(value.extra_search_terms || '').trim(),
    search_result_limit: Number(value.search_result_limit || 0),
    search_title_language: String(value.search_title_language || '').trim(),
    include_year: value.include_year !== false,
    series_search_scope: kind === 'series' ? normalizeSeriesScopeSelection(value.series_search_scope) : undefined,
  })
}

function findDuplicateQueryName(kind, draft, queries) {
  const signature = comparableQuerySignature(kind, draft)
  const match = (Array.isArray(queries) ? queries : []).find((query) => comparableQuerySignature(kind, query) === signature)
  return match?.name || ''
}

function extractScopedQueryFieldErrors(fieldErrors, kind, index) {
  if (!fieldErrors || typeof fieldErrors !== 'object') return {}
  const prefix = `${kind === 'movie' ? 'movie_search_queries' : 'series_search_queries'}.${index}.`
  return Object.entries(fieldErrors).reduce((acc, [path, message]) => {
    if (!path.startsWith(prefix) || typeof message !== 'string' || message.trim() === '') {
      return acc
    }
    const field = path.slice(prefix.length)
    if (field) {
      acc[field] = message
    }
    return acc
  }, {})
}

function summarizeQuery(query, kind) {
  const search = []
  if (query.search_mode) search.push(`Mode: ${query.search_mode.toUpperCase()}`)
  if (kind === 'movie' && query.movie_categories) search.push(`Movie: ${query.movie_categories}`)
  if (kind === 'series' && query.tv_categories) search.push(`TV: ${query.tv_categories}`)
  search.push(`Limit: ${Number(query.search_result_limit || 0) === 0 ? 'Max' : query.search_result_limit}`)
  if (kind === 'series') {
    const scope = normalizeSeriesScopeSelection(query.series_search_scope)
    const scopeLabel = SERIES_SCOPE_OPTIONS.find((option) => option.value === scope)?.label || 'Season/Episode'
    search.push(`Scope: ${scopeLabel}`)
  }
  search.push(`Year: ${query.include_year === false ? 'Off' : 'On'}`)
  search.push(`Title: ${titleLanguageLabel(query.search_title_language || '')}`)
  if (query.extra_search_terms) search.push(`Extra: ${truncateCompactValue(query.extra_search_terms)}`)
  return search
}


function QueryDraftFields({ kind, draft, setDraft, editing = false, fieldErrors = {} }) {
  const normalizedScope = kind === 'series'
    ? normalizeSeriesScopeSelection(draft.series_search_scope)
    : ''

  const update = (key, value) => {
    setDraft((current) => {
      if (key === 'search_mode') {
        return {
          ...current,
          search_mode: value,
          include_year: value === 'text' ? true : current.include_year,
        }
      }
      if (key === 'series_search_scope') {
        return {
          ...current,
          series_search_scope: normalizeSeriesScopeSelection(value),
        }
      }
      const next = { ...current, [key]: value }
      if (kind === 'series' && !next.series_search_scope) {
        next.series_search_scope = normalizeSeriesScopeSelection(current.series_search_scope)
      }
      return next
    })
  }
  const fieldClass = (key) => fieldErrors[key] ? "border-destructive focus-visible:ring-destructive" : ""
  const categoryField = kind === 'movie' ? 'movie_categories' : 'tv_categories'
  const rowClass = "space-y-3"
  const inlineRowClass = "flex flex-col gap-3 min-[360px]:flex-row min-[360px]:items-center min-[360px]:gap-4"
  const inlineLabelClass = "min-w-0 min-[360px]:flex-1"
  const controlBaseClass = "w-full min-[360px]:ml-auto min-[360px]:shrink-0"
  const controlWideClass = `${controlBaseClass} min-[360px]:w-[14rem]`
  const controlMediumClass = `${controlBaseClass} min-[360px]:w-[13rem]`
  const controlNarrowClass = `${controlBaseClass} min-[360px]:w-[9rem]`

  return (
    <div className="space-y-4">
      <div className="rounded-md border border-border/60 p-3">
        <div className={rowClass}>
          <div className={inlineRowClass}>
            <div className={inlineLabelClass}>
              <Label className="text-sm font-medium">Name</Label>
            </div>
            <div className={controlWideClass}>
              <Input className={`h-9 ${fieldClass('name')}`} value={draft.name || ''} onChange={(event) => update('name', event.target.value)} placeholder={kind === 'movie' ? 'MovieQuery01' : 'TVQuery01'} disabled={editing} />
            </div>
          </div>
        </div>
      </div>

      <div className="rounded-md border border-border/60">
        <div className="p-3">
          <div className={rowClass}>
            <div className={inlineRowClass}>
              <div className={inlineLabelClass}>
                <Label className="text-sm font-medium">Search Mode</Label>
              </div>
              <div className={controlMediumClass}>
                <select
                  className={`flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${fieldClass('search_mode')}`}
                  value={draft.search_mode || 'id'}
                  onChange={(event) => update('search_mode', event.target.value)}
                >
                  <option value="id">ID Search</option>
                  <option value="text">Text Search</option>
                </select>
              </div>
            </div>
          </div>
        </div>
        <div className="relative p-3">
          <div className="absolute left-3 right-3 top-0 border-t border-border/60" />
          <div className={rowClass}>
            <div className={inlineRowClass}>
              <div className={inlineLabelClass}>
                <Label className="text-sm font-medium">Category</Label>
              </div>
              <div className={controlNarrowClass}>
                <Input className={`h-9 ${fieldClass(categoryField)}`} value={kind === 'movie' ? (draft.movie_categories ?? '') : (draft.tv_categories ?? '')} onChange={(event) => update(categoryField, event.target.value)} placeholder={kind === 'movie' ? '2000' : '5000'} />
              </div>
            </div>
          </div>
        </div>
        <div className="relative p-3">
          <div className="absolute left-3 right-3 top-0 border-t border-border/60" />
          <div className={rowClass}>
            <div className={inlineRowClass}>
              <div className={inlineLabelClass}>
                <Label className="text-sm font-medium">Limit</Label>
              </div>
              <div className={controlNarrowClass}>
                <Input
                  type="number"
                  min={0}
                  max={5000}
                  placeholder="Max"
                  className={`h-9 ${fieldClass('search_result_limit')}`}
                  value={Number(draft.search_result_limit || 0) === 0 ? '' : draft.search_result_limit}
                  onChange={(event) => update('search_result_limit', event.target.value === '' ? 0 : Number(event.target.value))}
                />
              </div>
            </div>
            <p className="text-sm text-muted-foreground">{SEARCH_LIMIT_HINT}</p>
          </div>
        </div>
        {kind === 'series' && (
          <div className="relative p-3">
            <div className="absolute left-3 right-3 top-0 border-t border-border/60" />
            <div className={rowClass}>
              <div className={inlineRowClass}>
                <div className={inlineLabelClass}>
                  <Label className="text-sm font-medium">Scope</Label>
                </div>
                <div className={controlMediumClass}>
                  <select
                    className={`flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${fieldClass('series_search_scope')}`}
                    value={normalizedScope}
                    onChange={(event) => update('series_search_scope', event.target.value)}
                  >
                    {SERIES_SCOPE_OPTIONS.map((option) => (
                      <option key={option.value} value={option.value}>{option.label}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div className="space-y-1 text-sm text-muted-foreground">
                {TV_SCOPE_HINT_ITEMS.map((item) => (
                  <p key={item.label}>
                    <span className="font-medium text-foreground/80">{item.label}:</span>{' '}
                    {item.text}
                  </p>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>

      <div className="rounded-md border border-border/60">
        <div className="p-3">
          <div className={rowClass}>
            <div className={inlineRowClass}>
              <div className={inlineLabelClass}>
                <Label className="text-sm font-medium">Year</Label>
              </div>
              <div className={controlMediumClass}>
                <select
                  className={`flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${fieldClass('include_year')}`}
                  value={draft.include_year === false ? 'off' : 'on'}
                  onChange={(event) => update('include_year', event.target.value === 'on')}
                >
                  <option value="on">{draft.search_mode === 'text' ? 'Search + Validation' : 'Validation'}</option>
                  <option value="off">Ignore</option>
                </select>
              </div>
            </div>
            <p className="text-sm text-muted-foreground">{YEAR_HINT}</p>
          </div>
        </div>
        <div className="relative p-3">
          <div className="absolute left-3 right-3 top-0 border-t border-border/60" />
          <div className={rowClass}>
            <div className={inlineRowClass}>
              <div className={inlineLabelClass}>
                <Label className="text-sm font-medium">Title Language</Label>
              </div>
              <div className={controlMediumClass}>
                <select
                  className={`flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 ${fieldClass('search_title_language')}`}
                  value={draft.search_title_language || ''}
                  onChange={(event) => update('search_title_language', event.target.value)}
                >
                  {TITLE_LANGUAGE_OPTIONS.map((option) => (
                    <option key={option.value || 'original'} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className="space-y-1 text-sm text-muted-foreground">
              {TITLE_LANGUAGE_HINT_ITEMS.map((item) => (
                <p key={item.label}>
                  <span className="font-medium text-foreground/80">{item.label}:</span>{' '}
                  {item.text}
                </p>
              ))}
            </div>
          </div>
        </div>
      </div>

      <div className="rounded-md border border-border/60 p-3">
        <div className={rowClass}>
          <div className="space-y-3">
            <div>
              <Label className="text-sm font-medium">Extra Terms</Label>
            </div>
            <div className="w-full">
              <Input className={`h-9 ${fieldClass('extra_search_terms')}`} placeholder={'"The Walking Dead" !cam (1080p|720p)'} value={draft.extra_search_terms || ''} onChange={(event) => update('extra_search_terms', event.target.value)} />
            </div>
          </div>
          <p className="text-sm text-muted-foreground">Optional terms for text and ID searches. Use quotes for exact phrases, `!term` to exclude words, `*` as wildcard, `|` or `OR` for alternatives, and parentheses to group like `(1080p|720p)`.</p>
        </div>
      </div>
    </div>
  )
}

function dialogTitle(kind, editing) {
  if (editing) return kind === 'movie' ? 'Change Movie Query' : 'Change TV Query'
  return kind === 'movie' ? 'Add Movie Query' : 'Add TV Query'
}

function dialogDescription(kind) {
  return kind === 'movie'
    ? 'Build your search requests for movies.'
    : 'Build your search requests for TV.'
}

function defaultQueryName(kind, index) {
  return kind === 'movie' ? `MovieQuery${String(index + 1).padStart(2, '0')}` : `TVQuery${String(index + 1).padStart(2, '0')}`
}

function QueryDialog({ open, onOpenChange, kind, initialValue, existingNames = [], existingQueries = [], onSave, saveLabel, editing = false, nextIndex = 0, onClearStatus }) {
  const [draft, setDraft] = useState(() => normalizeDraft(kind, initialValue))
  const [wasOpen, setWasOpen] = useState(open)
  const [validationError, setValidationError] = useState('')
  const [fieldErrors, setFieldErrors] = useState({})
  const [showDiscardConfirm, setShowDiscardConfirm] = useState(false)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (open && !wasOpen) {
      const nextDraft = normalizeDraft(kind, initialValue)
      if (!editing && (!nextDraft.name || nextDraft.name === 'MovieQuery01' || nextDraft.name === 'TVQuery01')) {
        nextDraft.name = defaultQueryName(kind, nextIndex)
      }
      setDraft(nextDraft)
      setValidationError('')
      setFieldErrors({})
    }
    setWasOpen(open)
  }, [open, wasOpen, kind, initialValue, editing, nextIndex])

  const normalizedInitial = JSON.stringify(normalizeDraft(kind, initialValue))
  const normalizedCurrent = JSON.stringify(normalizeDraft(kind, draft))
  const isDirty = normalizedInitial !== normalizedCurrent
  const duplicateName = existingNames.some((name) => normalizeName(name) === normalizeName(draft.name))
  const duplicateQueryName = findDuplicateQueryName(kind, draft, existingQueries)
  const duplicateQuery = Boolean(duplicateQueryName)

  const requestClose = () => {
    if (saving) return
    if (isDirty) {
      setShowDiscardConfirm(true)
      return
    }
    onClearStatus?.()
    onOpenChange(false)
  }

  const handleSave = async () => {
    const next = normalizeDraft(kind, draft)
    if (kind === 'series') {
      next.series_search_scope = resolveSeriesSearchScope(next.series_search_scope)
    }
    const nextFieldErrors = {}
    if (!next.name) nextFieldErrors.name = 'Name is required.'
    if (duplicateName) nextFieldErrors.name = 'Name already exists.'
    if (duplicateQuery) nextFieldErrors.name = `An identical search request already exists: "${duplicateQueryName}".`
    const category = kind === 'movie' ? String(next.movie_categories ?? '').trim() : String(next.tv_categories ?? '').trim()
    const limit = Number(next.search_result_limit)
    if (!category || category === '0') {
      nextFieldErrors[kind === 'movie' ? 'movie_categories' : 'tv_categories'] = 'Category is required.'
    }
    if (Number.isNaN(limit) || limit < 0) {
      nextFieldErrors.search_result_limit = 'Limit must be 0 or greater.'
    }
    if (Object.keys(nextFieldErrors).length > 0) {
      setFieldErrors(nextFieldErrors)
      setValidationError(Object.values(nextFieldErrors)[0])
      return
    }
    setFieldErrors({})
    setValidationError('')
    next.search_result_limit = limit
    setSaving(true)
    try {
      await onSave(next)
      onOpenChange(false)
    } catch (error) {
      const scopedFieldErrors = extractScopedQueryFieldErrors(error?.fieldErrors, kind, nextIndex)
      if (Object.keys(scopedFieldErrors).length > 0) {
        setFieldErrors(scopedFieldErrors)
        setValidationError(Object.values(scopedFieldErrors)[0] || error?.message || 'Save failed.')
      } else {
        const firstFieldError = error?.fieldErrors && Object.values(error.fieldErrors).find((message) => typeof message === 'string' && message.trim() !== '')
        setFieldErrors({})
        setValidationError(firstFieldError || error?.message || 'Save failed.')
      }
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => {
      if (nextOpen) {
        onOpenChange(true)
        return
      }
      requestClose()
    }}>
      <DialogContent className="flex max-h-[85vh] max-w-3xl flex-col overflow-hidden" onOpenAutoFocus={focusDialogCloseButton}>
        <DialogHeader>
          <DialogTitle>{dialogTitle(kind, editing)}</DialogTitle>
          <DialogDescription>{dialogDescription(kind)}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <QueryDraftFields kind={kind} draft={draft} setDraft={setDraft} editing={editing} fieldErrors={fieldErrors} />
        </div>
        <DialogFooter className="flex items-center justify-between gap-3">
          <div className="min-h-9 flex-1">
            {validationError && (
              <div className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">{validationError}</div>
            )}
          </div>
          <div className="flex flex-row items-center justify-end gap-2">
            <Button type="button" variant="outline" onClick={requestClose} disabled={saving}>Cancel</Button>
            <Button type="button" variant="destructive" onClick={() => void handleSave()} disabled={saving}>
              {saving ? 'Saving...' : saveLabel}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
      <ConfirmDialog
        open={showDiscardConfirm}
        onOpenChange={setShowDiscardConfirm}
        title="Discard changes?"
        description="Your unsaved search request changes will be lost."
        confirmLabel="Discard"
        onConfirm={() => {
          setShowDiscardConfirm(false)
          onClearStatus?.()
          onOpenChange(false)
        }}
      />
    </Dialog>
  )
}

function QuerySection({ title, description, kind, items, names, update, remove, watch, streamsByName, onPersist, onCreate, onStatus, onClearStatus }) {
  const [editingId, setEditingId] = useState(null)
  const [copyDraft, setCopyDraft] = useState(null)
  const [deleteTarget, setDeleteTarget] = useState(null)
  const [deleteBlockedName, setDeleteBlockedName] = useState('')
  const existingQueries = items.map((item) => normalizeDraft(kind, watch(item.prefix) || item.field))
  const buildPersistPayload = (nextQueries) => (
    kind === 'movie'
      ? { movie_search_queries: nextQueries.map((query) => persistableDraft(kind, query)) }
      : { series_search_queries: nextQueries.map((query) => persistableDraft(kind, query)) }
  )

  const handleDelete = async (queryName, index) => {
    let assignedStreams = []
    try {
      const liveStreams = await apiFetch('/api/streams')
      assignedStreams = assignedStreamsForQuery(mapStreamsByUsername(liveStreams), kind, queryName)
    } catch {
      assignedStreams = assignedStreamsForQuery(streamsByName, kind, queryName)
    }

    if (assignedStreams.length > 0) {
      setDeleteBlockedName(queryName || '')
      onStatus?.({
        type: 'error',
        message: `Query "${queryName}" cannot be deleted while assigned to stream(s): ${assignedStreams.join(', ')}`
      })
      return
    }

    setDeleteBlockedName('')
    const nextQueries = items
      .filter((_, currentIndex) => currentIndex !== index)
      .map((item) => normalizeDraft(kind, watch(item.prefix) || item.field))
    try {
      await onPersist?.(buildPersistPayload(nextQueries))
      remove(index)
      onStatus?.({
        type: 'success',
        message: `${kind === 'movie' ? 'Movie' : 'Show'} query "${queryName}" deleted successfully.${CACHE_CLEARED_SUFFIX}`
      })
    } catch (error) {
      onStatus?.({
        type: 'error',
        message: error?.message || `Failed to delete query "${queryName}".`,
      })
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-start gap-3">
          <div className="min-w-0 space-y-0.5">
            <CardTitle>{title}</CardTitle>
            <CardDescription className="break-words">{description}</CardDescription>
          </div>
          <AddQueryButton
            kind={kind}
            title={kind === 'movie' ? 'Add Movie Query' : 'Add Show Query'}
            existingNames={names}
            existingQueries={existingQueries}
            onCreate={onCreate}
            onPersist={onPersist}
            onStatus={onStatus}
            onClearStatus={onClearStatus}
          />
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">No queries configured yet.</p>
        ) : (
          items.map(({ field, index, prefix }) => {
            const query = normalizeDraft(kind, watch(prefix) || field)
            const summary = summarizeQuery(query, kind)
            const editNames = names.filter((name, nameIndex) => nameIndex !== index)
            const editQueries = items
              .filter((item) => item.field.id !== field.id)
              .map((item) => normalizeDraft(kind, watch(item.prefix) || item.field))
            return (
              <Card className={deleteBlockedName && deleteBlockedName === query.name ? 'border-destructive/60 ring-1 ring-destructive/30' : ''} key={field.id}>
                <CardContent className="pt-6">
                  <div className="space-y-3">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex items-center gap-2 self-end sm:order-2">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button type="button" variant="outline" size="icon" className="h-9 w-9" onClick={() => {
                              setDeleteBlockedName('')
                              onClearStatus?.()
                              setEditingId(field.id)
                            }}>
                              <Settings className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Edit query</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              type="button"
                              variant="outline"
                              size="icon"
                              className="h-9 w-9"
                              onClick={() => setCopyDraft({
                                ...query,
                                name: defaultQueryName(kind, names.length),
                              })}
                            >
                              <Copy className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Copy query</TooltipContent>
                        </Tooltip>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button type="button" variant="destructive" size="icon" className="h-9 w-9" onClick={() => setDeleteTarget({ name: query.name, index })}>
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>Delete query</TooltipContent>
                        </Tooltip>
                      </div>
                      <div className="min-w-0 font-semibold sm:order-1">{query.name || defaultQueryName(kind, index)}</div>
                    </div>
                    {summary.length === 0 ? (
                      <p className="text-sm text-muted-foreground">No values set.</p>
                    ) : (
                      <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                        {summary.map((part) => (
                          <span key={part} className="rounded-full border border-border px-2 py-1">{part}</span>
                        ))}
                      </div>
                    )}
                  </div>
                </CardContent>
                <QueryDialog
                  open={editingId === field.id}
                  onOpenChange={(nextOpen) => {
                    if (!nextOpen) {
                      setDeleteBlockedName('')
                    }
                    setEditingId(nextOpen ? field.id : null)
                  }}
                  kind={kind}
                  initialValue={query}
                  existingNames={editNames}
                  existingQueries={editQueries}
                  saveLabel="Save"
                  editing
                  nextIndex={index}
                  onClearStatus={onClearStatus}
                  onSave={async (next) => {
                    const nextQueries = items.map((item, currentIndex) => (
                      currentIndex === index
                        ? normalizeDraft(kind, next)
                        : normalizeDraft(kind, watch(item.prefix) || item.field)
                    ))
                    await onPersist?.(buildPersistPayload(nextQueries))
                    update(index, next)
                    setDeleteBlockedName('')
                    onStatus?.({
                      type: 'success',
                      message: `${kind === 'movie' ? 'Movie' : 'Show'} query "${next.name}" saved successfully.${CACHE_CLEARED_SUFFIX}`
                    })
                  }}
                />
              </Card>
            )
          })
        )}
      </CardContent>
      <QueryDialog
        open={copyDraft !== null}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setCopyDraft(null)
          }
        }}
        kind={kind}
        initialValue={copyDraft || emptyDraft(kind)}
        existingNames={names}
        existingQueries={existingQueries}
        saveLabel="Save"
        nextIndex={names.length}
        onClearStatus={onClearStatus}
        onSave={async (next) => {
          const nextQueries = [...existingQueries, normalizeDraft(kind, next)]
          await onPersist?.(buildPersistPayload(nextQueries))
          onCreate(next)
          setDeleteBlockedName('')
          onStatus?.({
            type: 'success',
            message: `${kind === 'movie' ? 'Movie' : 'Show'} query "${next.name}" created successfully.${CACHE_CLEARED_SUFFIX}`
          })
          setCopyDraft(null)
        }}
      />
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setDeleteTarget(null)
        }}
        title="Delete search request?"
        description={deleteTarget ? `Are you sure you want to delete query "${deleteTarget.name}"?` : ''}
        confirmLabel="Delete"
        onConfirm={() => {
          const target = deleteTarget
          setDeleteTarget(null)
          if (target) {
            void handleDelete(target.name, target.index)
          }
        }}
      />
    </Card>
  )
}

function AddQueryButton({ kind, title, existingNames, existingQueries, onCreate, onPersist, onStatus, onClearStatus }) {
  const [open, setOpen] = useState(false)

  return (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button type="button" variant="destructive" size="icon" className="h-9 w-9 shrink-0" onClick={() => setOpen(true)}>
            <Plus className="h-4 w-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{title}</TooltipContent>
      </Tooltip>
      <QueryDialog
        open={open}
        onOpenChange={(nextOpen) => {
          setOpen(nextOpen)
        }}
        kind={kind}
        initialValue={emptyDraft(kind)}
        existingNames={existingNames}
        existingQueries={existingQueries}
        saveLabel="Save"
        nextIndex={existingNames.length}
        onClearStatus={onClearStatus}
        onSave={async (next) => {
          const nextQueries = [...existingQueries, normalizeDraft(kind, next)]
          await onPersist?.(
            kind === 'movie'
              ? { movie_search_queries: nextQueries }
              : { series_search_queries: nextQueries }
          )
          onCreate(next)
          onStatus?.({
            type: 'success',
            message: `${kind === 'movie' ? 'Movie' : 'Show'} query "${next.name}" created successfully.${CACHE_CLEARED_SUFFIX}`
          })
        }}
      />
    </>
  )
}

export function SearchQuerySettings({
  watch,
  movieFields,
  seriesFields,
  appendMovie,
  appendSeries,
  updateMovie,
  updateSeries,
  removeMovie,
  removeSeries,
  streamsByName = {},
  onPersist,
  onStatus,
  onClearStatus,
}) {
  const movieItems = useMemo(() => movieFields.map((field, index) => ({ field, index, prefix: `movie_search_queries.${index}` })), [movieFields])
  const seriesItems = useMemo(() => seriesFields.map((field, index) => ({ field, index, prefix: `series_search_queries.${index}` })), [seriesFields])
  const movieNames = useMemo(() => movieFields.map((field, index) => (watch(`movie_search_queries.${index}.name`) || field.name || '')).filter(Boolean), [movieFields, watch])
  const seriesNames = useMemo(() => seriesFields.map((field, index) => (watch(`series_search_queries.${index}.name`) || field.name || '')).filter(Boolean), [seriesFields, watch])

  useEffect(() => () => {
    onClearStatus?.()
  }, [onClearStatus])

  return (
    <TooltipProvider delayDuration={100}>
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <QuerySection
          title="Movie"
          description="Build your search requests for movies."
          kind="movie"
          items={movieItems}
          names={movieNames}
          update={updateMovie}
          remove={removeMovie}
          watch={watch}
          streamsByName={streamsByName}
          onPersist={onPersist}
          onCreate={(query) => appendMovie(query)}
          onStatus={onStatus}
          onClearStatus={onClearStatus}
        />
        <QuerySection
          title="TV"
          description="Build your search requests for TV."
          kind="series"
          items={seriesItems}
          names={seriesNames}
          update={updateSeries}
          remove={removeSeries}
          watch={watch}
          streamsByName={streamsByName}
          onPersist={onPersist}
          onCreate={(query) => appendSeries(query)}
          onStatus={onStatus}
          onClearStatus={onClearStatus}
        />
      </div>
    </div>
    </TooltipProvider>
  )
}
