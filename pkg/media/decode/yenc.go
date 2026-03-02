package decode

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"

	"github.com/javi11/rapidyenc"
)

var sizeMismatchRE = regexp.MustCompile(`expected size (\d+) but got (\d+)`)

type crlfReader struct {
	r    io.Reader
	buf  []byte
	last byte
	off  int
}

func (c *crlfReader) Read(p []byte) (int, error) {
	out := 0
	for out < len(p) {
		if c.off < len(c.buf) {
			b := c.buf[c.off]
			c.off++
			if b == '\n' && c.last != '\r' {
				p[out] = '\r'
				out++
				c.last = '\r'
				if out >= len(p) {
					c.off--
					return out, nil
				}
			}
			p[out] = b
			out++
			c.last = b
			continue
		}
		c.buf = make([]byte, 4096)
		n, err := c.r.Read(c.buf)
		c.buf = c.buf[:n]
		c.off = 0
		if n == 0 {
			return out, err
		}
	}
	return out, nil
}

func normalizeCRLF(r io.Reader) io.Reader { return &crlfReader{r: r} }

type Frame struct {
	Data     []byte
	FileName string
}

const maxDecodeSizeTolerance = 256

func DecodeToBytes(r io.Reader) (*Frame, error) {
	dec := rapidyenc.NewDecoder(normalizeCRLF(r))
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, dec)
	if err == nil || errors.Is(err, io.EOF) {
		return &Frame{Data: buf.Bytes(), FileName: dec.Meta.FileName}, nil
	}
	if sub := sizeMismatchRE.FindStringSubmatch(err.Error()); len(sub) == 3 {
		expected, _ := strconv.ParseInt(sub[1], 10, 64)
		got, _ := strconv.ParseInt(sub[2], 10, 64)
		shortfall := expected - got
		if shortfall > 0 && shortfall <= maxDecodeSizeTolerance && int64(buf.Len()) == got {
			return &Frame{Data: buf.Bytes(), FileName: dec.Meta.FileName}, nil
		}
	}
	return nil, err
}
