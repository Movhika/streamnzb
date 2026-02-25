# StreamNZB

[![Buy Me A Coffee](https://img.shields.io/badge/buy%20me%20a%20coffee-donate-yellow.svg)](https://buymeacoffee.com/gaisberg)
[![Discord](https://img.shields.io/badge/discord-join-7289DA.svg?logo=discord&logoColor=white)](https://snzb.stream/discord)

StreamNZB exists so me and my wife only see streams that can actually play. It checks availability before handing results to the client, so you don’t get a list of releases that then fail when you hit play. One app: Stremio addon, NNTP proxy, and indexer aggregation, all behind a single IP. No extra containers—just Usenet provider(s) and indexer(s).


## What it does

- **Stremio & Nuvio addon** – Add the manifest URL in [Stremio](https://www.stremio.com) or [Nuvio](https://nuvioapp.space). Search from your indexers and stream from Usenet; results are validated so only playable streams are offered. If a bad egg slips through we failover to the next possible stream.
- **NNTP proxy** – Standard NNTP (default port 119) for SABnzbd or NZBGet. Same provider pool as the addon.
- **AvailNZB** – [AvailNZB](https://check.snzb.stream) integration: reuse other users’ availability checks and report your own so the shared DB stays useful.
- **Devices** – Multiple “devices” (tokens), each with its own manifest URL and optional filters/sorting.
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

**First use:** Open `http://localhost:7000`. Default login is `admin` / `admin`; you’ll be asked to change the password. In **Settings** add at least one Usenet provider and one indexer. Create devices under **Settings → Devices** and use each device’s manifest URL in Stremio.


## AvailNZB

[AvailNZB](https://check.snzb.stream) is a community availability database. StreamNZB can skip re-checking releases that others have already verified and can report your playback success/failure so the data stays current. Official builds point at the project’s AvailNZB instance; to opt out, build the binary yourself.


## Troubleshooting

If you’re stuck, hit me up on [Discord](https://snzb.stream/discord) or [GitHub issues](https://github.com/Gaisberg/streamnzb/issues) (they sync via [GitThread](https://gitthreadsync.snzb.stream/)). Include relevant log snippets from **Settings → Logs** and strip any sensitive data (API keys, hostnames, etc.) before posting.


## Support

If StreamNZB is useful to you, you can support development here:

**[Buy Me A Coffee](https://buymeacoffee.com/gaisberg)**


## Credits

- [javi11](https://github.com/javi11) for Go-based RAR and 7z streaming ([altmount](https://github.com/javi11/altmount)).
- [Cursor](https://cursor.com) for helping with the project.
