import { useRef, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { FileText } from "lucide-react"
import { cn } from "@/lib/utils"

export function LogsPage({ logs = [] }) {
  const logsEndRef = useRef(null)

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs.length])

  return (
    <div className={cn("flex flex-col gap-4 py-4 md:gap-6 md:py-6 px-4 lg:px-6")}>
      <Card className="flex flex-col overflow-hidden flex-1 min-h-0">
        <CardHeader className="pb-2">
          <CardTitle className="flex items-center gap-2 text-xl font-semibold tracking-tight">
            <FileText className="h-5 w-5 text-muted-foreground" />
            System Logs
          </CardTitle>
          <CardDescription className="text-muted-foreground">
            Live application logs. New entries appear at the bottom.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex-1 p-0 overflow-hidden flex flex-col min-h-0">
          <ScrollArea className="flex-1 min-h-[360px] px-4 pb-4">
            <div className="font-mono text-xs space-y-1 pr-4">
              {logs.length === 0 && (
                <div className="text-muted-foreground italic py-4">Waiting for logs...</div>
              )}
              {logs.map((log, i) => (
                <div
                  key={i}
                  className={cn(
                    "whitespace-pre-wrap break-all border-b border-border/40 pb-0.5 mb-0.5 last:border-0",
                    "text-foreground"
                  )}
                >
                  {log}
                </div>
              ))}
              <div ref={logsEndRef} />
            </div>
          </ScrollArea>
        </CardContent>
      </Card>
    </div>
  )
}
