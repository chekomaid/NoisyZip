package main

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const (
	zipSigLocal     = 0x04034B50
	zipFlagUTF8     = 1 << 11
	zipFlagDataDesc = 1 << 3
)

type localHeader struct {
	off     int
	flags   uint16
	comp    uint16
	csize   uint32
	fname   string
	dataOff int
}

func scoreName(s string) int {
	score := 0
	for _, ch := range s {
		o := int(ch)
		switch {
		case unicode.IsLetter(ch) || unicode.IsDigit(ch):
			score += 2
		case strings.ContainsRune(" ._-()[]{}", ch):
			score += 1
		case ch == '/' || ch == '\\':
			score += 1
		case ch == '\t' || ch == '\r' || ch == '\n':
			score -= 5
		case o >= 0x2500 && o <= 0x257F:
			score -= 3
		case ch == '\uFFFD':
			score -= 5
		case unicode.IsPrint(ch):
			score += 0
		default:
			score -= 3
		}
		if strings.ContainsRune("A?NA", ch) {
			score -= 2
		}
	}
	return score
}

func decodeWith(enc *charmap.Charmap, b []byte) (string, bool) {
	dec := enc.NewDecoder()
	out, err := dec.Bytes(b)
	if err != nil {
		return "", false
	}
	return string(out), true
}

func decodeFilename(name []byte, flags uint16) (string, bool) {
	if flags&zipFlagUTF8 != 0 {
		if utf8.Valid(name) {
			return string(name), true
		}
		return "", false
	}

	candidates := make([]struct {
		score int
		name  string
	}, 0, 4)

	if utf8.Valid(name) {
		decoded := string(name)
		candidates = append(candidates, struct {
			score int
			name  string
		}{scoreName(decoded), decoded})
	}
	if decoded, ok := decodeWith(charmap.CodePage866, name); ok {
		candidates = append(candidates, struct {
			score int
			name  string
		}{scoreName(decoded), decoded})
	}
	if decoded, ok := decodeWith(charmap.Windows1251, name); ok {
		candidates = append(candidates, struct {
			score int
			name  string
		}{scoreName(decoded), decoded})
	}
	if decoded, ok := decodeWith(charmap.CodePage437, name); ok {
		candidates = append(candidates, struct {
			score int
			name  string
		}{scoreName(decoded), decoded})
	}

	if len(candidates) == 0 {
		return "", false
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}
	return best.name, true
}

func safeRelPath(name string) (string, bool) {
	n := strings.ReplaceAll(name, "\\", "/")
	n = regexp.MustCompile(`^\.*/+`).ReplaceAllString(n, "")
	parts := strings.Split(n, "/")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." || p == ".." {
			continue
		}
		clean = append(clean, p)
	}
	if len(clean) == 0 {
		return "", false
	}
	return filepath.Join(clean...), true
}

func isJunkPath(rel string) bool {
	rel = strings.ReplaceAll(rel, "\\", "/")
	if rel == ".junk" {
		return true
	}
	return strings.HasPrefix(rel, ".junk/")
}

func parseLocalHeader(buf []byte, off int) (localHeader, bool) {
	if off+30 > len(buf) {
		return localHeader{}, false
	}
	if binary.LittleEndian.Uint32(buf[off:off+4]) != zipSigLocal {
		return localHeader{}, false
	}
	flags := binary.LittleEndian.Uint16(buf[off+6 : off+8])
	comp := binary.LittleEndian.Uint16(buf[off+8 : off+10])
	csize := binary.LittleEndian.Uint32(buf[off+18 : off+22])
	fnlen := binary.LittleEndian.Uint16(buf[off+26 : off+28])
	exlen := binary.LittleEndian.Uint16(buf[off+28 : off+30])

	nameStart := off + 30
	nameEnd := nameStart + int(fnlen)
	extraEnd := nameEnd + int(exlen)
	if extraEnd > len(buf) || nameEnd > len(buf) {
		return localHeader{}, false
	}

	nameBytes := buf[nameStart:nameEnd]
	fname, ok := decodeFilename(nameBytes, flags)
	if !ok {
		return localHeader{}, false
	}

	return localHeader{
		off:     off,
		flags:   flags,
		comp:    comp,
		csize:   csize,
		fname:   fname,
		dataOff: extraEnd,
	}, true
}

func inflateRaw(data []byte) ([]byte, error) {
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	return io.ReadAll(r)
}

func inflateIncremental(buf []byte, start int, positions []int, i int) ([]byte, error) {
	endIndex := i + 1
	tries := 0
	for endIndex < len(positions) {
		end := positions[endIndex]
		if end > start {
			out, err := inflateRaw(buf[start:end])
			if err == nil {
				return out, nil
			}
		}
		endIndex++
		tries++
		if tries > 20000 {
			break
		}
	}
	if start < len(buf) {
		if out, err := inflateRaw(buf[start:]); err == nil {
			return out, nil
		}
	}
	return nil, fmt.Errorf("failed to locate end of deflate stream")
}

func recoverZip(zipPath string, outDir string, progressCb func(done, total int, name string), logCb func(string)) (int, error) {
	buf, err := os.ReadFile(zipPath)
	if err != nil {
		return 0, err
	}

	positions := make([]int, 0)
	for i := 0; i+4 <= len(buf); i++ {
		if buf[i] == 'P' && buf[i+1] == 'K' && buf[i+2] == 3 && buf[i+3] == 4 {
			positions = append(positions, i)
		}
	}

	if logCb != nil {
		logCb(fmt.Sprintf("Found local headers: %d", len(positions)))
	}

	recovered := 0
	total := len(positions)
	for idx, off := range positions {
		h, ok := parseLocalHeader(buf, off)
		nameForProgress := ""
		if ok {
			nameForProgress = h.fname
		}
		if progressCb != nil {
			progressCb(idx+1, total, nameForProgress)
		}
		if !ok {
			continue
		}

		rel, ok := safeRelPath(h.fname)
		if !ok {
			continue
		}
		if isJunkPath(rel) {
			continue
		}

		var content []byte
		if h.comp == 8 {
			content, err = inflateIncremental(buf, h.dataOff, positions, idx)
			if err != nil {
				continue
			}
		} else if h.comp == 0 && h.flags&zipFlagDataDesc == 0 {
			end := h.dataOff + int(h.csize)
			if end <= len(buf) {
				content = buf[h.dataOff:end]
			}
		}

		if content == nil {
			continue
		}

		target := filepath.Join(outDir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			continue
		}
		if err := os.WriteFile(target, content, 0o644); err != nil {
			continue
		}
		recovered++
	}

	return recovered, nil
}
