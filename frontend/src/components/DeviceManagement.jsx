import React, { useState, useEffect, useCallback } from 'react'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { AlertCircle, Plus, Trash2, RefreshCw, Copy, Check, Loader2 } from "lucide-react"
import { apiFetch, getApiUrl } from "@/api"

function DeviceManagement({ sendCommand, globalConfig }) {
  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(null)
  const [addDeviceLoading, setAddDeviceLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [newUsername, setNewUsername] = useState('')
  const [copiedToken, setCopiedToken] = useState('')
  const hasLoadedRef = React.useRef(false)

  // Fetch devices list (uses API via sendCommand)
  const fetchDevices = useCallback((showLoader = true) => {
    if (!sendCommand) {
      if (showLoader) setLoading(false)
      return
    }
    if (showLoader) setLoading(true)
    setError('')
    if (window.deviceManagementCallback) delete window.deviceManagementCallback
    window.deviceManagementCallback = (payload) => {
      if (payload && payload.error) {
        setError(payload.error)
        if (showLoader) setLoading(false)
      } else {
        setDevices(Array.isArray(payload) ? payload : [])
        if (showLoader) setLoading(false)
        hasLoadedRef.current = true
      }
      delete window.deviceManagementCallback
    }
    sendCommand('get_users', {})
  }, [sendCommand])

  // Initial load when sendCommand is available
  useEffect(() => {
    if (!hasLoadedRef.current && sendCommand) fetchDevices(true)
  }, [sendCommand])

  // Handle add device
  const handleAddDevice = async (e) => {
    e.preventDefault()
    e.stopPropagation()
    setError('')
    setSuccess('')
    setAddDeviceLoading(true)
    if (!sendCommand) {
      setError('Not connected')
      setAddDeviceLoading(false)
      return
    }
    if (window.deviceActionCallback) delete window.deviceActionCallback
    window.deviceActionCallback = (payload) => {
      setAddDeviceLoading(false)
      if (payload && payload.error) {
        setError(payload.error)
      } else {
        setSuccess(`Device "${newUsername}" created successfully`)
        setNewUsername('')
        setShowAddDialog(false)
        fetchDevices(false)
      }
      delete window.deviceActionCallback
    }
    sendCommand('create_user', { username: newUsername })
  }

  // Handle delete device
  const handleDeleteDevice = (username) => {
    if (username === 'admin') {
      setError('Cannot delete admin device')
      return
    }
    if (!confirm(`Are you sure you want to delete device "${username}"?`)) return
    if (!sendCommand) {
      setError('Not connected')
      return
    }
    setError('')
    setSuccess('')
    setActionLoading(`delete-${username}`)

    // Clean up any existing callback
    if (window.deviceActionCallback) {
      delete window.deviceActionCallback
    }

    window.deviceActionCallback = (payload) => {
      setActionLoading(null)
      if (payload.error) {
        setError(payload.error)
      } else {
        setSuccess(`Device "${username}" deleted successfully`)
        // Refresh list without showing loader (silent refresh)
        fetchDevices(false)
      }
      delete window.deviceActionCallback
    }

    sendCommand('delete_user', { username })
  }

  // Handle regenerate token
  const handleRegenerateToken = (username) => {
    if (!sendCommand) {
      setError('Not connected')
      return
    }
    setError('')
    setSuccess('')
    setActionLoading(`regenerate-${username}`)

    // Clean up any existing callback
    if (window.deviceActionCallback) {
      delete window.deviceActionCallback
    }

    window.deviceActionCallback = (payload) => {
      setActionLoading(null)
      if (payload.error) {
        setError(payload.error)
      } else {
        setSuccess(`Token regenerated for "${username}"`)
        setDevices(prev => prev.map(d => d.username === username ? { ...d, token: payload.token } : d))
      }
      delete window.deviceActionCallback
    }

    sendCommand('regenerate_token', { username })
  }


  // Get manifest URL
  const getManifestUrl = (token) => {
    const baseUrl = globalConfig?.addon_base_url 
      ? globalConfig.addon_base_url.replace(/\/$/, '')
      : window.location.origin
    return `${baseUrl}/${token}/manifest.json`
  }

  // Copy manifest URL
  const copyManifestUrl = (token) => {
    const url = getManifestUrl(token)
    navigator.clipboard.writeText(url)
    setCopiedToken(token)
    setTimeout(() => setCopiedToken(''), 2000)
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <CardDescription>
              Create device tokens for Stremio access.
            </CardDescription>
          </div>
          <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
            <DialogTrigger asChild>
              <Button type="button" className="w-full sm:w-auto">
                <Plus className="mr-2 h-4 w-4 shrink-0" />
                Add Device
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Add New Device</DialogTitle>
                <DialogDescription>
                  Create a new device account. Devices will access Stremio via their token in the URL.
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={handleAddDevice} className="space-y-4">
                {error && (
                  <div className="flex items-center gap-2 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
                    <AlertCircle className="h-4 w-4" />
                    <span>{error}</span>
                  </div>
                )}
                {success && (
                  <div className="flex items-center gap-2 p-3 text-sm text-green-600 bg-green-50 rounded-md">
                    <Check className="h-4 w-4" />
                    <span>{success}</span>
                  </div>
                )}
                <div className="space-y-2">
                  <Label htmlFor="new-username">Username</Label>
                  <Input
                    id="new-username"
                    type="text"
                    placeholder="Enter username"
                    value={newUsername}
                    onChange={(e) => setNewUsername(e.target.value)}
                    required
                    disabled={addDeviceLoading}
                  />
                  <p className="text-xs text-muted-foreground">
                    Devices access Stremio via their token in the URL: /{`{token}`}/manifest.json
                  </p>
                </div>
                <Button type="submit" className="w-full" disabled={addDeviceLoading}>
                  {addDeviceLoading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Creating...
                    </>
                  ) : (
                    'Create Device'
                  )}
                </Button>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        {error && !showAddDialog && (
          <div className="flex items-center gap-2 p-3 mb-4 text-sm text-destructive bg-destructive/10 rounded-md">
            <AlertCircle className="h-4 w-4" />
            <span>{error}</span>
          </div>
        )}
        {success && !showAddDialog && (
          <div className="flex items-center gap-2 p-3 mb-4 text-sm text-green-600 bg-green-50 rounded-md">
            <Check className="h-4 w-4" />
            <span>{success}</span>
          </div>
        )}

        {loading ? (
          <div className="flex items-center justify-center p-8">
            <Loader2 className="h-6 w-6 animate-spin" />
          </div>
        ) : devices.length === 0 ? (
          <div className="text-center p-8 text-muted-foreground">
            No devices found. Create your first device to get started.
          </div>
        ) : (
          <div className="space-y-4">
            {devices.map((device) => (
              <Card key={device.username}>
                <CardContent className="pt-6">
                  <div className="space-y-4">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2 mb-2">
                          <h3 className="font-semibold">{device.username}</h3>
                        </div>
                        <div className="space-y-2">
                          <Label className="text-xs text-muted-foreground block">Stremio URL:</Label>
                          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
                            <code className="text-xs bg-muted px-2 py-1.5 rounded break-all min-w-0">
                              {getManifestUrl(device.token)}
                            </code>
                            <Button
                              type="button"
                              variant="ghost"
                              size="sm"
                              onClick={() => copyManifestUrl(device.token)}
                              className="h-8 shrink-0 self-start sm:self-center"
                              title="Copy manifest URL"
                            >
                              {copiedToken === device.token ? (
                                <Check className="h-4 w-4 sm:mr-1" />
                              ) : (
                                <Copy className="h-4 w-4 sm:mr-1" />
                              )}
                              <span className="hidden sm:inline">
                                {copiedToken === device.token ? 'Copied' : 'Copy'}
                              </span>
                            </Button>
                          </div>
                        </div>
                      </div>
                      <div className="flex flex-wrap gap-2 sm:shrink-0 sm:ml-4">
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          onClick={() => handleRegenerateToken(device.username)}
                          disabled={actionLoading !== null || loading}
                          className="flex-1 min-w-[7rem] sm:flex-none"
                        >
                          {actionLoading === `regenerate-${device.username}` ? (
                            <Loader2 className="h-3 w-3 mr-1 shrink-0 animate-spin" />
                          ) : (
                            <RefreshCw className="h-3 w-3 mr-1 shrink-0" />
                          )}
                          <span className="truncate hidden sm:inline">Regenerate Token</span>
                          <span className="truncate sm:hidden">Regenerate</span>
                        </Button>
                        {device.username !== 'admin' && (
                          <Button
                            type="button"
                            variant="destructive"
                            size="sm"
                            onClick={() => handleDeleteDevice(device.username)}
                            disabled={actionLoading !== null || loading}
                            className="flex-1 min-w-[7rem] sm:flex-none"
                          >
                            {actionLoading === `delete-${device.username}` ? (
                              <Loader2 className="h-3 w-3 shrink-0 animate-spin" />
                            ) : (
                              <Trash2 className="h-3 w-3 shrink-0" />
                            )}
                            <span className="ml-1">Delete</span>
                          </Button>
                        )}
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export default DeviceManagement
