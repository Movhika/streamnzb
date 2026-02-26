import React, { useState, useEffect } from 'react'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PasswordInput } from "@/components/ui/password-input"
import { Separator } from "@/components/ui/separator"
import { AlertCircle, Loader2, User } from "lucide-react"
import { cn } from "@/lib/utils"

function EnvOverrideNote({ show }) {
  if (!show) return null
  return (
    <p className="text-xs text-muted-foreground flex items-center gap-1 mt-1">
      <AlertCircle className="h-3.5 w-3 shrink-0" />
      Overwritten by environment variable on restart.
    </p>
  )
}

export function ProfilePage({
  currentUser,
  config,
  sendCommand,
  ws,
  onUsernameChanged,
  onPasswordChanged,
}) {
  const envOverrides = config?.env_overrides ?? []
  const currentUsername = config?.admin_username || 'admin' || currentUser

  const [username, setUsername] = useState(currentUsername)
  const [usernameSaving, setUsernameSaving] = useState(false)
  const [usernameMessage, setUsernameMessage] = useState({ type: '', text: '' })

  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordError, setPasswordError] = useState('')
  const [passwordLoading, setPasswordLoading] = useState(false)

  useEffect(() => {
    setUsername(config?.admin_username || 'admin' || currentUser)
  }, [config?.admin_username, currentUser])

  const handleSaveUsername = async (e) => {
    e.preventDefault()
    setUsernameMessage({ type: '', text: '' })
    const trimmed = (username || '').trim()
    if (!trimmed) {
      setUsernameMessage({ type: 'error', text: 'Username cannot be empty.' })
      return
    }
    if (trimmed === (config?.admin_username || 'admin')) {
      setUsernameMessage({ type: '', text: 'No change.' })
      return
    }
    if (!sendCommand) {
      setUsernameMessage({ type: 'error', text: 'Not connected.' })
      return
    }
    setUsernameSaving(true)
    window.profileUsernameCallback = (payload) => {
      setUsernameSaving(false)
      const ok = payload.status === 'success'
      if (!ok) {
        setUsernameMessage({ type: 'error', text: payload.message || payload.error || 'Save failed.' })
      } else {
        setUsernameMessage({ type: 'success', text: 'Username saved. Use it to log in next time.' })
        onUsernameChanged?.(trimmed)
      }
      delete window.profileUsernameCallback
    }
    sendCommand('save_config', { admin_username: trimmed })
  }

  const handleChangePassword = async (e) => {
    e.preventDefault()
    setPasswordError('')

    if (newPassword !== confirmPassword) {
      setPasswordError('New passwords do not match')
      return
    }
    if (newPassword.length < 6) {
      setPasswordError('Password must be at least 6 characters long')
      return
    }
    if (!sendCommand) {
      setPasswordError('Not connected. Please wait...')
      return
    }

    setPasswordLoading(true)
    try {
      const loginRes = await fetch('/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ username: currentUser, password: currentPassword }),
      })
      const loginData = await loginRes.json()
      if (!loginData.success) {
        setPasswordError('Current password is incorrect')
        setPasswordLoading(false)
        return
      }

      window.passwordChangeCallback = (payload) => {
        setPasswordLoading(false)
        if (payload.error) {
          setPasswordError(payload.error)
        } else {
          setCurrentPassword('')
          setNewPassword('')
          setConfirmPassword('')
          onPasswordChanged?.()
        }
        delete window.passwordChangeCallback
      }
      if (!sendCommand('update_password', { username: currentUser, password: newPassword })) {
        setPasswordError('Failed to send update request')
        setPasswordLoading(false)
        delete window.passwordChangeCallback
      }
    } catch {
      setPasswordError('Failed to connect to server')
      setPasswordLoading(false)
      delete window.passwordChangeCallback
    }
  }

  const usernameDisabled = envOverrides.includes('admin_username')

  return (
    <div className="space-y-8 max-w-2xl">
      {/* Profile hero: large avatar + title */}
      <div className="flex flex-col items-center gap-4 text-center">
        <div
          className={cn(
            "flex items-center justify-center rounded-full bg-muted border border-border",
            "w-24 h-24 sm:w-28 sm:h-28"
          )}
          aria-hidden
        >
          <User className="h-12 w-12 sm:h-14 sm:w-14 text-muted-foreground" strokeWidth={1.5} />
        </div>
        <div>
          <h2 className="text-xl font-semibold tracking-tight">Profile</h2>
          <p className="text-sm text-muted-foreground mt-1">
            Change your dashboard login username and password.
          </p>
        </div>
      </div>

      <Separator />

      {/* Username — minimal section */}
      <section className="space-y-3">
        <div>
          <h3 className="text-sm font-medium">Username</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            The username you use to log in. Save to apply.
          </p>
        </div>
        {usernameDisabled ? (
          <p className="text-sm text-muted-foreground">
            Username is set by environment variable and cannot be changed here.
          </p>
        ) : (
          <form onSubmit={handleSaveUsername} className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <div className="flex-1 space-y-2 min-w-0">
              <Label htmlFor="profile-username" className="sr-only">Username</Label>
              <Input
                id="profile-username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="admin"
                disabled={usernameSaving}
                className="max-w-xs"
              />
              <EnvOverrideNote show={envOverrides.includes('admin_username')} />
            </div>
            <Button type="submit" disabled={usernameSaving}>
              {usernameSaving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Save username
            </Button>
          </form>
        )}
        {usernameMessage.text && (
          <p className={cn("text-sm", usernameMessage.type === 'error' ? 'text-destructive' : 'text-muted-foreground')}>
            {usernameMessage.text}
          </p>
        )}
      </section>

      <Separator />

      {/* Password — minimal section */}
      <section className="space-y-4">
        <div>
          <h3 className="text-sm font-medium">Password</h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Change your password. You will stay logged in after changing it.
          </p>
        </div>
        <form onSubmit={handleChangePassword} className="space-y-4 max-w-sm">
          {passwordError && (
            <div className="flex items-center gap-2 p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{passwordError}</span>
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="profile-current-password">Current password</Label>
            <PasswordInput
              id="profile-current-password"
              placeholder="Enter current password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              disabled={passwordLoading}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="profile-new-password">New password</Label>
            <PasswordInput
              id="profile-new-password"
              placeholder="Min 6 characters"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              disabled={passwordLoading}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="profile-confirm-password">Confirm new password</Label>
            <PasswordInput
              id="profile-confirm-password"
              placeholder="Confirm new password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              disabled={passwordLoading}
            />
          </div>
          <Button type="submit" disabled={passwordLoading}>
            {passwordLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Change password
          </Button>
        </form>
      </section>
    </div>
  )
}
