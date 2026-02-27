// Package seek provides time-to-byte conversion for HTTP Range requests
// when the client sends a start time via the t= query parameter (e.g. after
// failover or resume). The server uses this to set Range: bytes=offset- so
// http.ServeContent serves from the requested position.
//
// # Supported containers (t= → bytes)
//
// Server-side seek from t= is implemented for:
//   - MP4 (.mp4, .m4v, .mov) — duration from moov/mvhd
//   - Matroska / WebM (.mkv, .webm) — duration from Segment/Info (TimestampScale, Duration)
//
// # Unsupported containers (t= ignored for Range)
//
// For these formats, the server does not set Range from t=; playback starts
// from the beginning. The t= parameter is still forwarded on redirects for
// client-side use where the player supports it:
//   - AVI (.avi)
//   - MPEG-TS (.ts, .m2ts)
//   - VOB (.vob)
//   - WMV (.wmv)
//   - FLV (.flv)
//   - Other containers not listed in Supported
//
// If the format cannot be detected or duration parsing fails, the server
// serves from byte 0 and does not set Range from t=.
package seek
