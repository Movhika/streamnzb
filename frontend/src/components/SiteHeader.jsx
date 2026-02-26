import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"

const pageTitles = {
  dashboard: "Dashboard",
  search: "Search",
  logs: "Logs",
  profile: "Profile",
  general: "General",
  indexers: "Indexers",
  providers: "Providers",
  streams: "Streams",
  devices: "Devices",
}

export function SiteHeader({ activePage }) {
  return (
    <header className="flex h-12 shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <h1 className="text-base font-medium">{pageTitles[activePage] || "Dashboard"}</h1>
      </div>
    </header>
  )
}
