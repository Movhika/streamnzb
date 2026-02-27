package seek

import (
	"encoding/binary"
)

// MP4 box header: 4 bytes size (big-endian) + 4 bytes type.
// Size 1 means 64-bit size in next 8 bytes (not used for our bounded parse).
const mp4BoxHeaderSize = 8

// moov and mvhd are ISO Base Media box types (fourcc).
var (
	moovType = [4]byte{'m', 'o', 'o', 'v'}
	mvhdType = [4]byte{'m', 'v', 'h', 'd'}
)

// durationFromMP4 parses data (typically the first 1–2 MB of the file) for
// moov → mvhd and returns duration in seconds. If moov is not fully present
// in data (e.g. moov at end of file), returns ok false.
func durationFromMP4(data []byte) (durationSec float64, ok bool) {
	for len(data) >= mp4BoxHeaderSize {
		size := binary.BigEndian.Uint32(data)
		typ := [4]byte{data[4], data[5], data[6], data[7]}
		payloadSize := int64(size) - mp4BoxHeaderSize
		if payloadSize < 0 {
			return 0, false
		}
		if size == 1 {
			// 64-bit size
			if len(data) < 8+8 {
				return 0, false
			}
			size64 := binary.BigEndian.Uint64(data[8:])
			payloadSize = int64(size64) - 8 - mp4BoxHeaderSize
			if payloadSize < 0 {
				return 0, false
			}
			data = data[16:]
		} else {
			data = data[mp4BoxHeaderSize:]
		}
		boxEnd := int64(len(data))
		if payloadSize > boxEnd {
			// Box extends past our buffer; we can't find moov in this slice
			if typ == moovType {
				return 0, false
			}
			return 0, false
		}
		switch typ {
		case moovType:
			return parseMvhdFrom(data[:payloadSize])
		default:
			data = data[payloadSize:]
		}
	}
	return 0, false
}

func parseMvhdFrom(moovPayload []byte) (durationSec float64, ok bool) {
	data := moovPayload
	for len(data) >= mp4BoxHeaderSize {
		size := binary.BigEndian.Uint32(data)
		typ := [4]byte{data[4], data[5], data[6], data[7]}
		payloadSize := int64(size) - mp4BoxHeaderSize
		if payloadSize < 0 {
			return 0, false
		}
		if size == 1 {
			if len(data) < 16 {
				return 0, false
			}
			size64 := binary.BigEndian.Uint64(data[8:])
			payloadSize = int64(size64) - 8 - mp4BoxHeaderSize
			if payloadSize < 0 {
				return 0, false
			}
			data = data[16:]
		} else {
			data = data[mp4BoxHeaderSize:]
		}
		if int64(len(data)) < payloadSize {
			return 0, false
		}
		if typ != mvhdType {
			data = data[payloadSize:]
			continue
		}
		// mvhd: version(1), flags(3), timescale(4), duration(4 or 8)
		if payloadSize < 8 {
			return 0, false
		}
		version := data[0]
		timescale := binary.BigEndian.Uint32(data[4:8])
		if timescale == 0 {
			return 0, false
		}
		var duration uint64
		if version == 1 {
			if payloadSize < 16 {
				return 0, false
			}
			duration = binary.BigEndian.Uint64(data[8:16])
		} else {
			duration = uint64(binary.BigEndian.Uint32(data[8:12]))
		}
		return float64(duration) / float64(timescale), true
	}
	return 0, false
}
