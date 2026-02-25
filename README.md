# StreamNZB

[![Buy Me A Coffee](https://img.shields.io/badge/buy%20me%20a%20coffee-donate-yellow.svg)](https://buymeacoffee.com/gaisberg)
[![Discord](https://img.shields.io/badge/discord-join-7289DA.svg?logo=discord&logoColor=white)](https://snzb.stream/discord)

StreamNZB is a Stremio/Nuvio addon that streams from Usenet via your indexers. You see one row per **stream** (e.g. Global, 1080p)—each a named set of filters and sorting. No upfront NZB validation: we build an ordered list of releases from indexer + [AvailNZB](https://check.snzb.stream), try the first on play, and on failure report bad and fail over to the next. One app: addon, NNTP proxy, and indexer aggregation behind a single IP. No extra containers—just Usenet provider(s) and indexer(s).


## What it does

- **Stremio & Nuvio addon** – Add the manifest URL in [Stremio](https://www.stremio.com) or [Nuvio](https://nuvioapp.space). Open a title and you get one row per **stream config** (e.g. “Global”, “1080p”). Each row shows “StreamNZB [availNZB]” when the top release is known good, or “X possible releases”. Play uses that stream’s ordered list; if playback fails we report to AvailNZB and try the next release.
- **Streams** – In **Settings → Streams** you define multiple streams (name + filters + sorting). The **Global** stream is always first; others appear in stable order. Each stream gets its own play list for every title (same indexer/AvailNZB fetch, different filter/sort per stream). Optional “Next release” row per stream lets you advance through the list.
- **Devices** – **Settings → Devices** creates tokens (one manifest URL per token) for auth. All devices see the same stream list; streams are not per-device.
- **NNTP proxy** – Standard NNTP (default port 119) for SABnzbd or NZBGet. Same provider pool as the addon.
- **AvailNZB** – Reuse others’ availability checks and report your own so the shared DB stays useful. Bad releases are skipped when building play lists; good/bad is reported on play.
- **Single binary** – Docker image or native Windows/Linux/macOS. No other containers required.


## Release types we don’t support

Streaming is done on-the-fly from archive segments. That only works when the inner file is stored uncompressed:

- **Compressed RAR** – RAR must be STORE (no compression). Compressed RAR releases will not play.
- **Compressed 7z** – Same idea: only uncompressed (copy/store) 7z content is streamable.


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

**First use:** Open `http://localhost:7000`. Default login is `admin` / `admin`; you’ll be asked to change the password. In **Settings** add at least one Usenet provider and one indexer. The default **Global** stream (Settings → Streams) is enough to start; you can add more streams (e.g. “1080p”, “4K”) with different filters and sorting. Create devices under **Settings → Devices** and use each device’s manifest URL in Stremio—all devices see the same stream list (Global first, then your other streams).


## AvailNZB

[AvailNZB](https://check.snzb.stream) is a community availability database. We don’t download or validate NZBs before showing results—we build an ordered play list from indexer search plus AvailNZB (skipping releases already reported bad), then try on play. StreamNZB reports success/failure so the shared DB stays current. Official builds use the project’s AvailNZB instance; to opt out, build the binary yourself.


## Troubleshooting

If you’re stuck, hit me up on [Discord](https://snzb.stream/discord) or [GitHub issues](https://github.com/Gaisberg/streamnzb/issues) (they sync via [GitThread](https://gitthreadsync.snzb.stream/)). Include relevant log snippets from **Settings → Logs** and strip any sensitive data (API keys, hostnames, etc.) before posting.


## Support

If StreamNZB is useful to you, you can support development here:

**[Buy Me A Coffee](https://buymeacoffee.com/gaisberg)**


## Credits

- [javi11](https://github.com/javi11) for Go-based RAR and 7z streaming ([altmount](https://github.com/javi11/altmount)).
- [Cursor](https://cursor.com) for helping with the project.
