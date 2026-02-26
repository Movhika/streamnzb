export function getApiUrl(path) {
  const base = window.location.pathname.split('/').filter(Boolean)[0]
  const prefix = base && base !== 'api' ? `/${base}` : ''
  return `${prefix}${path}`
}

export async function apiFetch(path, options = {}) {
  const url = getApiUrl(path)
  const res = await fetch(url, { credentials: 'include', ...options })
  let data = null
  const contentType = res.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    try {
      data = await res.json()
    } catch (_) {}
  }
  if (!res.ok) throw new Error((data && (data.error || data.message)) || res.statusText)
  return data
}
