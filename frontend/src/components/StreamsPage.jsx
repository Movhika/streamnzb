import React, { useState, useEffect, useCallback } from 'react'
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { StreamEditForm } from "@/components/StreamEditForm"
import { Loader2, Plus, Pencil, Trash2, Radio } from "lucide-react"
import { cn } from "@/lib/utils"

function getApiUrl(path) {
  const base = window.location.pathname.split('/').filter(Boolean)[0]
  const prefix = base && base !== 'api' ? `/${base}` : ''
  return `${prefix}${path}`
}

export function StreamsPage() {
  const [streams, setStreams] = useState([])
  const [loading, setLoading] = useState(true)
  const [view, setView] = useState('list') // 'list' | 'edit'
  const [editId, setEditId] = useState(null) // null = create new
  const [message, setMessage] = useState({ type: '', text: '' })
  const [deleteTarget, setDeleteTarget] = useState(null) // { id, name } for confirm dialog

  const fetchStreams = useCallback(() => {
    setLoading(true)
    const url = getApiUrl('/api/stream/configs')
    fetch(url, { credentials: 'include' })
      .then((res) => {
        if (res.status === 403) {
          setMessage({ type: 'error', text: 'Only admin can manage streams.' })
          return []
        }
        if (!res.ok) throw new Error(res.statusText)
        return res.json()
      })
      .then((data) => {
        setStreams(Array.isArray(data) ? data : [])
        setMessage({ type: '', text: '' })
      })
      .catch((err) => setMessage({ type: 'error', text: 'Failed to load streams: ' + err.message }))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (view === 'list') fetchStreams()
  }, [view, fetchStreams])

  const handleAdd = () => {
    setEditId(null)
    setView('edit')
  }

  const handleEdit = (id) => {
    setEditId(id)
    setView('edit')
  }

  const handleDone = () => {
    setView('list')
    setEditId(null)
  }

  const handleDeleteClick = (stream) => {
    setDeleteTarget({ id: stream.id, name: stream.name || stream.id })
  }

  const handleDeleteConfirm = () => {
    if (!deleteTarget) return
    const url = getApiUrl(`/api/stream/configs/${encodeURIComponent(deleteTarget.id)}`)
    fetch(url, { method: 'DELETE', credentials: 'include' })
      .then((res) => {
        if (res.status === 403) {
          setMessage({ type: 'error', text: 'Only admin can delete streams.' })
          return
        }
        if (res.status === 400) {
          return res.text().then((t) => {
            setMessage({ type: 'error', text: t || 'Cannot delete the last stream.' })
          })
        }
        if (!res.ok) throw new Error(res.statusText)
        setDeleteTarget(null)
        fetchStreams()
      })
      .catch((err) => setMessage({ type: 'error', text: err.message || 'Delete failed.' }))
  }

  if (view === 'edit') {
    return (
      <div className="space-y-6">
        <div className="flex items-center gap-2">
          <Button type="button" variant="ghost" size="sm" onClick={handleDone}>
            ← Back to streams
          </Button>
        </div>
        <StreamEditForm
          streamId={editId}
          onSaved={handleDone}
          onCancel={handleDone}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground">
          Configure multiple streams with different filters and sorting. Catalog and play use the default stream (Global).
        </p>
        <Button onClick={handleAdd}>
          <Plus className="mr-2 h-4 w-4" />
          Add stream
        </Button>
      </div>

      {message.text && (
        <p className={cn("text-sm", message.type === 'error' ? 'text-destructive' : 'text-primary')}>
          {message.text}
        </p>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : streams.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-center">
            <Radio className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground">No streams yet. Add one to get started.</p>
            <Button className="mt-4" onClick={handleAdd}>
              <Plus className="mr-2 h-4 w-4" />
              Add stream
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {streams.map((stream) => (
            <Card key={stream.id}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-base font-medium">{stream.name || stream.id}</CardTitle>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => handleEdit(stream.id)}
                    title="Edit"
                  >
                    <Pencil className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive hover:text-destructive"
                    onClick={() => handleDeleteClick(stream)}
                    title="Delete"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <CardDescription className="text-xs">
                  ID: {stream.id}
                </CardDescription>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete stream</DialogTitle>
            <DialogDescription>
              Delete &quot;{deleteTarget?.name}&quot;? This cannot be undone. You must keep at least one stream.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDeleteConfirm}>Delete</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
