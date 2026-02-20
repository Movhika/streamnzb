import { useState, useEffect, useRef } from 'react'
import Settings from './Settings'
import Login from './components/Login'
import ChangePassword from './components/ChangePassword'
import { SidebarProvider, SidebarInset } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/AppSidebar"
import { SiteHeader } from "@/components/SiteHeader"
import { DashboardPage } from "@/components/DashboardPage"
import { LogsPage } from "@/components/LogsPage"
import { AlertCircle, Loader2 } from "lucide-react"

const MAX_HISTORY = 60
const MAX_LOGS = 200

function App() {
  const [authenticated, setAuthenticated] = useState(false)
  const [currentUser, setCurrentUser] = useState(null)
  const [authToken, setAuthToken] = useState(localStorage.getItem('auth_token') || '')
  const [mustChangePassword, setMustChangePassword] = useState(false)
  const [stats, setStats] = useState(null)
  const [config, setConfig] = useState(null)
  const [saveStatus, setSaveStatus] = useState({ type: '', msg: '', errors: null })
  const [isSaving, setIsSaving] = useState(false)
  const [isRestarting, setIsRestarting] = useState(false)
  const isRestartingRef = useRef(false)
  const [error, setError] = useState(null)
  const [history, setHistory] = useState([])
  const [connHistory, setConnHistory] = useState([])
  const [wsStatus, setWsStatus] = useState('connecting')
  const [ws, setWs] = useState(null)
  const [theme, setTheme] = useState(localStorage.getItem('theme') || 'system')
  const [version, setVersion] = useState(null)
  const hasLoggedOutRef = useRef(false)
  const authCheckTimeoutRef = useRef(null)
  const [logs, setLogs] = useState([])
  const [copied, setCopied] = useState(false)
  const [activePage, setActivePage] = useState('dashboard')

  const chartData = history.map((h, i) => ({
    time: h.time,
    speed: h.speed,
    conns: connHistory[i]?.conns ?? 0,
  }))

  useEffect(() => {
    fetch('/api/info')
      .then(res => res.ok ? res.json() : null)
      .then(data => data?.version && setVersion(data.version))
      .catch(() => {})
  }, [])

  useEffect(() => {
    const token = localStorage.getItem('auth_token')
    if (!token) {
      const pathParts = window.location.pathname.split('/').filter(p => p !== '')
      if (pathParts.length > 0 && pathParts[0] !== 'api') {
        hasLoggedOutRef.current = false
        setAuthenticated(true)
        setCurrentUser('legacy')
      } else {
        setAuthenticated(false)
      }
    } else {
      hasLoggedOutRef.current = false
      setAuthToken(token)
      setAuthenticated(true)
      authCheckTimeoutRef.current = setTimeout(() => {
        if (authenticated && !currentUser && wsStatus !== 'connected') {
          setAuthenticated(false)
          setAuthToken('')
          localStorage.removeItem('auth_token')
        }
      }, 5000)
    }
    return () => {
      if (authCheckTimeoutRef.current) {
        clearTimeout(authCheckTimeoutRef.current)
      }
    }
  }, [])

  const handleLogin = (username, token, mustChange) => {
    hasLoggedOutRef.current = false
    setAuthenticated(true)
    setCurrentUser(username)
    setAuthToken(token)
    setMustChangePassword(mustChange)
    localStorage.setItem('auth_token', token)
  }

  const handleLogout = () => {
    hasLoggedOutRef.current = true
    setAuthenticated(false)
    setCurrentUser(null)
    setAuthToken('')
    localStorage.removeItem('auth_token')
    if (ws) {
      ws.close()
    }
    setWs(null)
    window.ws = null
  }

  useEffect(() => {
    const root = window.document.documentElement;
    root.classList.remove("light", "dark");

    if (theme === "system") {
      const systemTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
      root.classList.add(systemTheme);
    } else {
      root.classList.add(theme);
    }
    localStorage.setItem('theme', theme);
  }, [theme]);

  useEffect(() => {
    if (!authenticated) return
    if (hasLoggedOutRef.current) return

    let socket;
    let reconnectTimeout;

    const connect = () => {
      if (hasLoggedOutRef.current) return
      
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      const pathParts = window.location.pathname.split('/').filter(p => p !== '');
      const tokenPrefix = pathParts.length > 0 && pathParts[0] !== 'api' ? `/${pathParts[0]}` : '';
      const wsToken = authToken || (pathParts.length > 0 && pathParts[0] !== 'api' ? pathParts[0] : '');
      const wsUrl = `${protocol}//${host}${tokenPrefix}/api/ws${wsToken ? `?token=${wsToken}` : ''}`;
      socket = new WebSocket(wsUrl);

      socket.onopen = () => {
        if (isRestartingRef.current) {
            window.location.reload();
            return;
        }
        if (hasLoggedOutRef.current) {
          socket.close();
          return;
        }
        setWsStatus('connected');
        setError(null);
        setWs(socket);
        window.ws = socket;
        setLogs([]);
      };

      socket.onmessage = (event) => {
        if (hasLoggedOutRef.current) return
        
        const msg = JSON.parse(event.data);
        
        switch (msg.type) {
          case 'stats': {
            const data = msg.payload;
            setStats(data);
            const timestamp = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
            setHistory(prev => [...prev, { time: timestamp, speed: data.total_speed_mbps }].slice(-MAX_HISTORY));
            setConnHistory(prev => [...prev, { time: timestamp, conns: data.active_connections }].slice(-MAX_HISTORY));
            break;
          }
          case 'config': {
            setConfig(msg.payload);
            if (window.globalConfigCallback) {
              window.globalConfigCallback(msg.payload);
            }
            break;
          }
          case 'log_entry': {
             setLogs(prev => [...prev, msg.payload].slice(-MAX_LOGS));
             break;
          }
          case 'log_history': {
             setLogs(msg.payload.slice(-MAX_LOGS));
             break;
          }
          case 'save_status': {
            setSaveStatus({
                type: msg.payload.status === 'success' ? 'success' : 'error',
                msg: msg.payload.message,
                errors: msg.payload.errors
            });
            setIsSaving(false);
            break;
          }
          case 'auth_info': {
            if (msg.payload?.version) setVersion(msg.payload.version)
            if (hasLoggedOutRef.current) {
              socket.close();
              return;
            }
            if (msg.payload.authenticated) {
              if (authCheckTimeoutRef.current) {
                clearTimeout(authCheckTimeoutRef.current)
                authCheckTimeoutRef.current = null
              }
              setAuthenticated(true);
              setCurrentUser(msg.payload.username);
              setMustChangePassword(msg.payload.must_change_password || false);
            } else {
              if (authCheckTimeoutRef.current) {
                clearTimeout(authCheckTimeoutRef.current)
                authCheckTimeoutRef.current = null
              }
              hasLoggedOutRef.current = false
              setAuthenticated(false);
              setCurrentUser(null);
              setAuthToken('')
              localStorage.removeItem('auth_token');
              socket.close();
            }
            break;
          }
          case 'users_response': {
            if (window.deviceManagementCallback) {
              window.deviceManagementCallback(msg.payload);
            }
            break;
          }
          case 'user_response': {
            if (window.deviceResponseCallback) {
              window.deviceResponseCallback(msg.payload);
            }
            break;
          }
          case 'user_action_response': {
            if (window.deviceActionCallback) {
              window.deviceActionCallback(msg.payload);
            }
            if (window.passwordChangeCallback) {
              window.passwordChangeCallback(msg.payload);
            }
            break;
          }
        }
      };

      socket.onclose = () => {
        setWsStatus('disconnected');
        setWs(null);
        window.ws = null;
        if (!hasLoggedOutRef.current) {
          reconnectTimeout = setTimeout(() => {
            if (authenticated && !hasLoggedOutRef.current) {
              connect();
            }
          }, 3000);
        }
      };

      socket.onerror = () => {
        setError("Network Error: Could not connect to API");
        if (authToken && authenticated && !currentUser) {
          setAuthenticated(false);
          setAuthToken('')
          localStorage.removeItem('auth_token');
        }
        socket.close();
      };
    };

    connect();
    return () => {
      if (socket) socket.close();
      if (reconnectTimeout) clearTimeout(reconnectTimeout);
    }
  }, [authenticated, authToken]);

  const sendCommand = (type, payload) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
          if (type === 'save_config' || type === 'save_user_configs') {
              setSaveStatus({ type: 'normal', msg: 'Validating and saving...', errors: null });
              setIsSaving(true);
          } else if (type === 'restart') {
              setIsRestarting(true);
              isRestartingRef.current = true;
          }
          ws.send(JSON.stringify({ type, payload }));
      }
  };

  const getHTTPSLink = () => {
      if (!config) return '#';
      let baseUrl = config.addon_base_url || window.location.origin;
      let url = baseUrl.replace(/\/$/, '');
      if (currentUser && currentUser !== 'legacy' && authToken) {
        return `${url}/${authToken}/manifest.json`;
      }
      return `${url}/manifest.json`;
  }

  const handleInstallClick = (type) => {
      const httpsLink = getHTTPSLink();
      if (type === 'web') {
          const encodedManifest = encodeURIComponent(httpsLink);
          window.open(`https://web.stremio.com/#/addons?addon=${encodedManifest}`, '_blank');
      } else if (type === 'copy') {
          navigator.clipboard.writeText(httpsLink).then(() => {
              setCopied(true);
              setTimeout(() => setCopied(false), 2000);
          });
      }
  };

  if (!authenticated) {
    return <Login onLogin={handleLogin} version={version} />
  }

  if (mustChangePassword && currentUser) {
    return <ChangePassword username={currentUser} onPasswordChanged={() => {
      setMustChangePassword(false)
    }} />
  }

  if (error && wsStatus === 'disconnected') {
      return (
        <div className="flex flex-col h-screen items-center justify-center gap-4">
            <AlertCircle className="h-12 w-12 text-destructive animate-pulse" />
            <div className="text-xl font-semibold text-destructive">{error}</div>
            <p className="text-muted-foreground">Retrying connection...</p>
        </div>
      )
  }

  if (!stats || isRestarting) return (
    <div className="fixed inset-0 z-50 flex flex-col items-center justify-center bg-background/80 backdrop-blur-sm gap-4">
        <Loader2 className="h-12 w-12 text-primary animate-spin" />
        <div className="text-xl font-semibold tracking-tight">
            {isRestarting ? "Restarting StreamNZB..." : "Initializing StreamNZB Dashboard..."}
        </div>
        {isRestarting && <p className="text-muted-foreground animate-pulse">Redirecting to home shortly...</p>}
    </div>
  )

  const settingsPages = ['general', 'indexers', 'providers', 'filters', 'sorting', 'devices']
  const isSettingsPage = settingsPages.includes(activePage)

  return (
    <SidebarProvider>
      <AppSidebar
        activePage={activePage}
        onNavigate={setActivePage}
        version={version}
        currentUser={currentUser}
        onLogout={handleLogout}
        theme={theme}
        onThemeChange={setTheme}
        config={config}
        onInstallClick={handleInstallClick}
        copied={copied}
      />
      <SidebarInset>
        <SiteHeader activePage={activePage} />
        <div className="flex flex-1 flex-col">
          {activePage === 'dashboard' && (
            <DashboardPage
              stats={stats}
              chartData={chartData}
              sendCommand={sendCommand}
              config={config}
            />
          )}
          {activePage === 'logs' && (
            <LogsPage logs={logs} />
          )}
          {isSettingsPage && (
            <div className="py-4 md:py-6 px-4 lg:px-6">
              <Settings
                initialConfig={config}
                sendCommand={sendCommand}
                saveStatus={saveStatus}
                isSaving={isSaving}
                adminToken={currentUser && currentUser !== 'legacy' ? authToken : null}
                activePage={activePage}
              />
            </div>
          )}
        </div>
      </SidebarInset>
    </SidebarProvider>
  )
}

export default App
