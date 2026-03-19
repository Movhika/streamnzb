# StreamNZB

[![Buy Me A Coffee](https://img.shields.io/badge/buy%20me%20a%20coffee-donate-yellow.svg)](https://buymeacoffee.com/gaisberg)
[![Discord](https://img.shields.io/badge/discord-join-7289DA.svg?logo=discord&logoColor=white)](https://snzb.stream/discord)

StreamNZB is a Usenet streaming provider for [AIOStreams](https://github.com/Viren070/AIOStreams). It searches your indexers, checks availability via [AvailNZB](https://check.snzb.stream), and streams releases on-the-fly from your Usenet providers. One binary — addon, NNTP proxy, and indexer aggregation behind a single IP. No extra containers, just your Usenet provider(s) and indexer(s).


## What it does

- **Usenet provider for AIOStreams** — StreamNZB acts as a Stremio-compatible addon. Add the manifest URL to [AIOStreams](https://github.com/Viren070/AIOStreams) and let AIOStreams handle filtering, sorting, and formatting. StreamNZB returns all available releases so AIOStreams can do the triage.
- **NNTP proxy** — Standard NNTP (default port 119) for SABnzbd or NZBGet. Same provider pool as the addon.
- **AvailNZB** — Community availability database. Bad releases are skipped; success/failure is reported on play so the shared DB stays current.
- **Single binary** — Docker image or native Windows/Linux/macOS. No other containers required.


## Release types we don't support

Streaming is done on-the-fly from archive segments. That only works when the inner file is stored uncompressed:

- **Compressed RAR** — RAR must be STORE (no compression). Compressed RAR releases will not play.
- **Compressed 7z** — Same idea: only uncompressed (copy/store) 7z content is streamable.


## Run it

**Docker (recommended):**

```yaml
services:
  streamnzb:
    image: ghcr.io/gaisberg/streamnzb:latest
    container_name: streamnzb
    restart: unless-stopped
    ports:
      - "7000:7000"
      - "119:119"
    volumes:
      - /path/to/config:/app/data
```

Or run the binary from the [releases](https://github.com/Gaisberg/streamnzb/releases) page (Windows, Linux, macOS). See `.env.example` for config via environment variables.

**First use:**

1. Open `http://localhost:7000`. Default login is `admin` / `admin`; you'll be asked to change the password.
2. Go to **Settings → General** and set your base URL as an IP address or domain name AIOStreams can reach.
  - If using tailscale use the IP address of the machine running streamnzb. e.g. `http://100.64.0.1:7000`
  - If using a domain name make sure it is reachable from the machine running AIOStreams. e.g. `http://streamnzb.example.com:7000` or HTTPS `https://streamnzb.example.com`
3. Go to **Settings → Providers** and add at least one Usenet provider (host, port, username, password, connections).
4. Go to **Settings → Indexers** and add at least one Newznab-compatible indexer (URL + API key).
5. Click **Save Changes** at the bottom — validation will highlight any fields that need attention.
6. Click the **Install** button in the sidebar to copy your manifest URL.
7. Add that manifest URL to [AIOStreams](https://github.com/Viren070/AIOStreams) as a StreamNZB preset (see below).


## Using with AIOStreams

[AIOStreams](https://github.com/Viren070/AIOStreams) is THE way to use StreamNZB. It consolidates multiple Stremio addons into a single super-addon with advanced filtering, sorting, and formatting — all configured in one place.

**Setup:**

1. In the StreamNZB dashboard, click the **Install** button in the sidebar to copy your manifest URL.
2. In AIOStreams, add the StreamNZB preset and paste your manifest URL (e.g. `https://your-host:7000/<token>/manifest.json`).
3. **No Usenet service required in AIOStreams** — StreamNZB handles all Usenet provider connections, NZB fetching, and streaming internally. Skip the AIOStreams Usenet service configuration entirely.
4. Configure your filtering, sorting, and stream formatting rules in the AIOStreams UI. AIOStreams will aggregate StreamNZB results alongside any other addons you use and apply your rules uniformly.

StreamNZB automatically detects AIOStreams via its User-Agent and returns all available releases so AIOStreams can handle triage, scoring, and failover ordering on its side.


## AvailNZB

[AvailNZB](https://check.snzb.stream) is a community availability database. StreamNZB doesn't download or validate NZBs before showing results — it builds an ordered play list from indexer search plus AvailNZB (skipping releases already reported bad), then tries on play. Success/failure is reported so the shared DB stays current. Official builds can utilize the project's AvailNZB instance; you can change the mode in **Settings → General → AvailNZB**.


## Troubleshooting

If you're stuck, please either open a [GitHub issue](https://github.com/Gaisberg/streamnzb/issues) or report it in the [Discord](https://snzb.stream/discord) `#help` channel (they sync via [GitThread](https://gitthreadsync.snzb.stream/)). Include downloaded logs when relevant, and include the copied bad match report from **NZB History** when the issue is about a wrong or poor release match. Sensitive data should be automatically redacted but please double-check before posting.


## Support

If StreamNZB is useful to you, you can support development here:

**[Buy Me A Coffee](https://buymeacoffee.com/gaisberg)**


## Credits

- [javi11](https://github.com/javi11) for Go-based RAR and 7z streaming ([altmount](https://github.com/javi11/altmount)).
- [Augment](https://www.augmentcode.com/) for helping with the project.

