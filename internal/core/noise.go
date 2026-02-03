package core

import (
	"compress/flate"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	mrand "math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	sigLocal = 0x04034b50
	sigCDir  = 0x02014b50
	sigEOCD  = 0x06054b50
	sigDD    = 0x08074b50

	flagUTF8     = 1 << 11
	flagDataDesc = 1 << 3

	chunkSize = 1024 * 1024
)

type fileItem struct {
	index   int
	path    string
	rel     string
	modTime time.Time
}

type entry struct {
	name   []byte
	flags  uint16
	method uint16
	dosT   uint16
	dosD   uint16
	crc    uint32
	csize  uint32
	usize  uint32
	offset uint32
	tmp    string
}

type result struct {
	index int
	name  string
	entry entry
	err   error
}

type Config struct {
	SrcDir        string
	OutZip        string
	Compression   string
	Encoding      string
	OverwriteCentralDir bool
	CommentSize   int
	FixedTime     bool
	NoiseFiles    int
	NoiseSize     int
	Level         int
	Strategy      string
	DictSize      int
	Workers       int
	IncludeHidden bool
	Seed          int64
	HasSeed       bool
}

func RunEncrypt(cfg Config, progress func(done, total int, name string), log func(msg string)) (int, error) {
	if cfg.CommentSize < 0 || cfg.CommentSize > 0xffff {
		return 0, fmt.Errorf("comment-size must be in range 0..65535")
	}
	if cfg.NoiseFiles < 0 || cfg.NoiseSize < 0 {
		return 0, fmt.Errorf("noise-files and noise-size must be >= 0")
	}
	if cfg.Level < 0 || cfg.Level > 9 {
		return 0, fmt.Errorf("level must be in range 0..9")
	}
	if cfg.DictSize != 32768 {
		return 0, fmt.Errorf("dict-size must be 32768 (Go stdlib deflate uses fixed 32 KB window)")
	}

	comp := strings.ToLower(strings.TrimSpace(cfg.Compression))
	if comp != "deflate" && comp != "store" {
		return 0, fmt.Errorf("compression must be deflate or store")
	}
	cfg.Compression = comp

	strategyVal := strings.ToLower(strings.TrimSpace(cfg.Strategy))
	switch strategyVal {
	case "default", "filtered", "huffman", "rle", "fixed":
	default:
		return 0, fmt.Errorf("strategy must be one of: default, filtered, huffman, rle, fixed")
	}
	cfg.Strategy = strategyVal
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}

	items, err := listFiles(cfg.SrcDir, cfg.OutZip, cfg.IncludeHidden)
	if err != nil {
		return 0, fmt.Errorf("list files: %w", err)
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("no files found in source directory")
	}
	if log != nil {
		log(fmt.Sprintf("Files found: %d", len(items)))
	}

	encName, nameFlag, err := makeNameEncoder(cfg.Encoding)
	if err != nil {
		return 0, fmt.Errorf("encoding: %w", err)
	}

	useDeflate := cfg.Compression == "deflate"
	method := uint16(0)
	if useDeflate {
		method = 8
	}

	if strategyVal != "default" && strategyVal != "huffman" {
		if log != nil {
			log(fmt.Sprintf("Note: strategy %q is not supported by Go stdlib; ignored.", strategyVal))
		}
	}

	randReader := io.Reader(crand.Reader)
	if cfg.HasSeed {
		randReader = mrand.New(mrand.NewSource(cfg.Seed))
	}

	results := make([]entry, len(items))
	jobs := make(chan fileItem)
	out := make(chan result)
	var wg sync.WaitGroup

	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				ent, err := compressFile(item, encName, nameFlag, method, useDeflate, cfg.Level, strategyVal, cfg.FixedTime)
				out <- result{index: item.index, name: item.rel, entry: ent, err: err}
			}
		}()
	}

	go func() {
		for _, it := range items {
			jobs <- it
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()

	total := len(items) + cfg.NoiseFiles
	done := 0
	for res := range out {
		if res.err != nil {
			return 0, fmt.Errorf("compress: %w", res.err)
		}
		results[res.index] = res.entry
		done++
		if progress != nil {
			progress(done, total, res.name)
		}
	}

	for i := 0; i < cfg.NoiseFiles; i++ {
		name := fmt.Sprintf(".junk/%04d_%s.bin", i, randHex(randReader, 6))
		ent, err := makeNoiseEntry(randReader, name, encName, nameFlag, method, useDeflate, cfg.Level, strategyVal, cfg.FixedTime, cfg.NoiseSize)
		if err != nil {
			return 0, fmt.Errorf("noise: %w", err)
		}
		results = append(results, ent)
		done++
		if progress != nil {
			progress(done, total, name)
		}
	}

	if err := writeZip(randReader, cfg.OutZip, results, cfg.OverwriteCentralDir, cfg.CommentSize); err != nil {
		return 0, fmt.Errorf("write zip: %w", err)
	}

	return len(results), nil
}

func listFiles(srcDir, outZip string, includeHidden bool) ([]fileItem, error) {
	srcAbs, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, err
	}
	outAbs, _ := filepath.Abs(outZip)

	var files []fileItem
	err = filepath.WalkDir(srcAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == srcAbs {
			return nil
		}
		if !includeHidden {
			hidden, err := isHiddenPath(path, d, srcAbs)
			if err != nil {
				return err
			}
			if hidden {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		abs, _ := filepath.Abs(path)
		if abs == outAbs {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcAbs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		files = append(files, fileItem{
			index:   len(files),
			path:    path,
			rel:     rel,
			modTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].rel < files[j].rel
	})
	for i := range files {
		files[i].index = i
	}
	return files, nil
}

func compressFile(
	item fileItem,
	encName func(string) ([]byte, error),
	nameFlag uint16,
	method uint16,
	useDeflate bool,
	level int,
	strategy string,
	fixedTime bool,
) (entry, error) {
	nameBytes, err := encName(item.rel)
	if err != nil {
		return entry{}, fmt.Errorf("encode name %q: %w", item.rel, err)
	}
	dosT, dosD := dosTimeDate(item.modTime, fixedTime)
	tmp, err := os.CreateTemp("", "enczip_*")
	if err != nil {
		return entry{}, err
	}
	defer tmp.Close()

	src, err := os.Open(item.path)
	if err != nil {
		return entry{}, err
	}
	defer src.Close()

	var crc uint32
	var usize uint32
	var csize uint32

	if useDeflate {
		counter := &countingWriter{w: tmp}
		levelVal := level
		if strategy == "huffman" {
			levelVal = flate.HuffmanOnly
		}
		w, err := flate.NewWriter(counter, levelVal)
		if err != nil {
			return entry{}, err
		}
		crc, usize, err = copyDeflateWithCRC(w, src)
		if err != nil {
			w.Close()
			return entry{}, err
		}
		if err := w.Close(); err != nil {
			return entry{}, err
		}
		csize = uint32(counter.n)
	} else {
		crc, usize, err = copyStoreWithCRC(tmp, src)
		if err != nil {
			return entry{}, err
		}
		csize = usize
	}

	return entry{
		name:   nameBytes,
		flags:  nameFlag,
		method: method,
		dosT:   dosT,
		dosD:   dosD,
		crc:    crc,
		csize:  csize,
		usize:  usize,
		tmp:    tmp.Name(),
	}, nil
}

func makeNoiseEntry(
	randReader io.Reader,
	name string,
	encName func(string) ([]byte, error),
	nameFlag uint16,
	method uint16,
	useDeflate bool,
	level int,
	strategy string,
	fixedTime bool,
	size int,
) (entry, error) {
	nameBytes, err := encName(name)
	if err != nil {
		return entry{}, err
	}
	dosT, dosD := dosTimeDate(time.Unix(0, 0), fixedTime)
	tmp, err := os.CreateTemp("", "enczip_noise_*")
	if err != nil {
		return entry{}, err
	}
	defer tmp.Close()

	var crc uint32
	var usize uint32
	var csize uint32

	if useDeflate {
		counter := &countingWriter{w: tmp}
		levelVal := level
		if strategy == "huffman" {
			levelVal = flate.HuffmanOnly
		}
		w, err := flate.NewWriter(counter, levelVal)
		if err != nil {
			return entry{}, err
		}
		crc, usize, err = writeRandomWithCRC(randReader, w, size)
		if err != nil {
			w.Close()
			return entry{}, err
		}
		if err := w.Close(); err != nil {
			return entry{}, err
		}
		csize = uint32(counter.n)
	} else {
		crc, usize, err = writeRandomWithCRC(randReader, tmp, size)
		if err != nil {
			return entry{}, err
		}
		csize = usize
	}

	return entry{
		name:   nameBytes,
		flags:  nameFlag,
		method: method,
		dosT:   dosT,
		dosD:   dosD,
		crc:    crc,
		csize:  csize,
		usize:  usize,
		tmp:    tmp.Name(),
	}, nil
}

func writeZip(randReader io.Reader, outZip string, entries []entry, overwriteCentralDir bool, commentSize int) error {
	if err := os.MkdirAll(filepath.Dir(outZip), 0o755); err != nil {
		return err
	}
	out, err := os.Create(outZip)
	if err != nil {
		return err
	}
	defer out.Close()

	flags := uint16(0)
	if overwriteCentralDir {
		flags |= flagDataDesc
	}

	for i := range entries {
		ent := &entries[i]
		ent.flags |= flags

		offset, _ := out.Seek(0, io.SeekCurrent)
		ent.offset = uint32(offset)

		if overwriteCentralDir {
			if err := writeLocalHeader(out, ent, 0, 0, 0); err != nil {
				return err
			}
		} else {
			if err := writeLocalHeader(out, ent, ent.crc, ent.csize, ent.usize); err != nil {
				return err
			}
		}
		if _, err := out.Write(ent.name); err != nil {
			return err
		}
		if err := copyTemp(out, ent.tmp); err != nil {
			return err
		}
		if overwriteCentralDir {
			if err := patchCRC(out, int64(ent.offset), ent.crc); err != nil {
				return err
			}
			if err := writeDataDesc(out, ent); err != nil {
				return err
			}
		}
	}

	cdStart, _ := out.Seek(0, io.SeekCurrent)
	for _, ent := range entries {
		if err := writeCDir(out, ent); err != nil {
			return err
		}
		if _, err := out.Write(ent.name); err != nil {
			return err
		}
	}
	cdEnd, _ := out.Seek(0, io.SeekCurrent)
	cdSize := cdEnd - cdStart
	if err := writeEOCD(out, len(entries), cdSize, cdStart, commentSize); err != nil {
		return err
	}
	if commentSize > 0 {
		if err := writeRand(randReader, out, commentSize); err != nil {
			return err
		}
	}
	if overwriteCentralDir {
		if err := writePoisonTail(randReader, out); err != nil {
			return err
		}
	}

	for _, ent := range entries {
		_ = os.Remove(ent.tmp)
	}
	return nil
}

func writeLocalHeader(w io.Writer, ent *entry, crc, csize, usize uint32) error {
	buf := make([]byte, 30)
	binary.LittleEndian.PutUint32(buf[0:], sigLocal)
	binary.LittleEndian.PutUint16(buf[4:], 20)
	binary.LittleEndian.PutUint16(buf[6:], ent.flags)
	binary.LittleEndian.PutUint16(buf[8:], ent.method)
	binary.LittleEndian.PutUint16(buf[10:], ent.dosT)
	binary.LittleEndian.PutUint16(buf[12:], ent.dosD)
	binary.LittleEndian.PutUint32(buf[14:], crc)
	binary.LittleEndian.PutUint32(buf[18:], csize)
	binary.LittleEndian.PutUint32(buf[22:], usize)
	binary.LittleEndian.PutUint16(buf[26:], uint16(len(ent.name)))
	binary.LittleEndian.PutUint16(buf[28:], 0)
	_, err := w.Write(buf)
	return err
}

func writeCDir(w io.Writer, ent entry) error {
	buf := make([]byte, 46)
	binary.LittleEndian.PutUint32(buf[0:], sigCDir)
	binary.LittleEndian.PutUint16(buf[4:], 20)
	binary.LittleEndian.PutUint16(buf[6:], 20)
	binary.LittleEndian.PutUint16(buf[8:], ent.flags)
	binary.LittleEndian.PutUint16(buf[10:], ent.method)
	binary.LittleEndian.PutUint16(buf[12:], ent.dosT)
	binary.LittleEndian.PutUint16(buf[14:], ent.dosD)
	binary.LittleEndian.PutUint32(buf[16:], ent.crc)
	binary.LittleEndian.PutUint32(buf[20:], ent.csize)
	binary.LittleEndian.PutUint32(buf[24:], ent.usize)
	binary.LittleEndian.PutUint16(buf[28:], uint16(len(ent.name)))
	binary.LittleEndian.PutUint16(buf[30:], 0)
	binary.LittleEndian.PutUint16(buf[32:], 0)
	binary.LittleEndian.PutUint16(buf[34:], 0)
	binary.LittleEndian.PutUint16(buf[36:], 0)
	binary.LittleEndian.PutUint32(buf[38:], 0)
	binary.LittleEndian.PutUint32(buf[42:], ent.offset)
	_, err := w.Write(buf)
	return err
}

func writeEOCD(w io.Writer, count int, cdSize, cdStart int64, commentSize int) error {
	buf := make([]byte, 22)
	binary.LittleEndian.PutUint32(buf[0:], sigEOCD)
	binary.LittleEndian.PutUint16(buf[4:], 0)
	binary.LittleEndian.PutUint16(buf[6:], 0)
	binary.LittleEndian.PutUint16(buf[8:], uint16(count))
	binary.LittleEndian.PutUint16(buf[10:], uint16(count))
	binary.LittleEndian.PutUint32(buf[12:], uint32(cdSize))
	binary.LittleEndian.PutUint32(buf[16:], uint32(cdStart))
	binary.LittleEndian.PutUint16(buf[20:], uint16(commentSize))
	_, err := w.Write(buf)
	return err
}

func writeDataDesc(w io.Writer, ent *entry) error {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint32(buf[0:], sigDD)
	binary.LittleEndian.PutUint32(buf[4:], ent.crc)
	binary.LittleEndian.PutUint32(buf[8:], ent.csize)
	binary.LittleEndian.PutUint32(buf[12:], ent.usize)
	_, err := w.Write(buf)
	return err
}

func patchCRC(f *os.File, off int64, crc uint32) error {
	cur, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err := f.Seek(off+14, io.SeekStart); err != nil {
		return err
	}
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, crc)
	if _, err := f.Write(buf); err != nil {
		return err
	}
	_, err = f.Seek(cur, io.SeekStart)
	return err
}

func writePoisonTail(randReader io.Reader, w io.Writer) error {
	if err := writeRand(randReader, w, 32); err != nil {
		return err
	}
	buf := make([]byte, 22)
	binary.LittleEndian.PutUint32(buf[0:], sigEOCD)
	binary.LittleEndian.PutUint16(buf[4:], 0)
	binary.LittleEndian.PutUint16(buf[6:], 0)
	binary.LittleEndian.PutUint16(buf[8:], 0)
	binary.LittleEndian.PutUint16(buf[10:], 0)
	binary.LittleEndian.PutUint32(buf[12:], 0xffffffff)
	binary.LittleEndian.PutUint32(buf[16:], 0x80000000)
	binary.LittleEndian.PutUint16(buf[20:], 0)
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return writeRand(randReader, w, 96)
}

func copyTemp(out *os.File, tmpPath string) error {
	tmp, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	defer tmp.Close()
	_, err = io.CopyBuffer(out, tmp, make([]byte, chunkSize))
	return err
}

func copyDeflateWithCRC(w io.Writer, r io.Reader) (uint32, uint32, error) {
	hash := crc32.NewIEEE()
	var usize uint32
	buf := make([]byte, chunkSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			usize += uint32(n)
			if _, err := hash.Write(buf[:n]); err != nil {
				return 0, 0, err
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return 0, 0, err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, err
		}
	}
	return hash.Sum32(), usize, nil
}

func copyStoreWithCRC(w io.Writer, r io.Reader) (uint32, uint32, error) {
	hash := crc32.NewIEEE()
	var usize uint32
	buf := make([]byte, chunkSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			usize += uint32(n)
			if _, err := hash.Write(buf[:n]); err != nil {
				return 0, 0, err
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return 0, 0, err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0, err
		}
	}
	return hash.Sum32(), usize, nil
}

func writeRandomWithCRC(randReader io.Reader, w io.Writer, size int) (uint32, uint32, error) {
	hash := crc32.NewIEEE()
	var usize uint32
	buf := make([]byte, chunkSize)
	remaining := size
	for remaining > 0 {
		n := chunkSize
		if remaining < n {
			n = remaining
		}
		if _, err := randReader.Read(buf[:n]); err != nil {
			return 0, 0, err
		}
		usize += uint32(n)
		if _, err := hash.Write(buf[:n]); err != nil {
			return 0, 0, err
		}
		if _, err := w.Write(buf[:n]); err != nil {
			return 0, 0, err
		}
		remaining -= n
	}
	return hash.Sum32(), usize, nil
}

func writeRand(randReader io.Reader, w io.Writer, size int) error {
	buf := make([]byte, size)
	if _, err := randReader.Read(buf); err != nil {
		return err
	}
	_, err := w.Write(buf)
	return err
}

func randHex(randReader io.Reader, n int) string {
	buf := make([]byte, n)
	_, _ = randReader.Read(buf)
	const hexd = "0123456789abcdef"
	out := make([]byte, n*2)
	for i, b := range buf {
		out[i*2] = hexd[b>>4]
		out[i*2+1] = hexd[b&0x0f]
	}
	return string(out)
}

func dosTimeDate(t time.Time, fixed bool) (uint16, uint16) {
	if fixed || t.Year() < 1980 {
		t = time.Date(1980, 1, 1, 0, 0, 0, 0, time.Local)
	}
	dosTime := uint16(t.Hour()<<11 | t.Minute()<<5 | (t.Second() / 2))
	dosDate := uint16((t.Year()-1980)<<9 | int(t.Month())<<5 | t.Day())
	return dosTime, dosDate
}

type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

type nameEncoder func(string) ([]byte, error)

func makeNameEncoder(enc string) (nameEncoder, uint16, error) {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "utf-8", "utf8":
		return func(s string) ([]byte, error) {
			return []byte(s), nil
		}, flagUTF8, nil
	case "cp1251":
		return func(s string) ([]byte, error) {
			return encodeCP1251(s)
		}, 0, nil
	default:
		return nil, 0, fmt.Errorf("unsupported encoding %q", enc)
	}
}

func encodeCP1251(s string) ([]byte, error) {
	var out []byte
	for _, r := range s {
		if r < 0x80 {
			out = append(out, byte(r))
			continue
		}
		if b, ok := cp1251Encode[r]; ok {
			out = append(out, b)
			continue
		}
		return nil, fmt.Errorf("rune U+%04X is not representable in cp1251", r)
	}
	return out, nil
}

var cp1251Decode = [128]rune{
	0x0402, 0x0403, 0x201A, 0x0453, 0x201E, 0x2026, 0x2020, 0x2021,
	0x20AC, 0x2030, 0x0409, 0x2039, 0x040A, 0x040C, 0x040B, 0x040F,
	0x0452, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x0000, 0x2122, 0x0459, 0x203A, 0x045A, 0x045C, 0x045B, 0x045F,
	0x00A0, 0x040E, 0x045E, 0x0408, 0x00A4, 0x0490, 0x00A6, 0x00A7,
	0x0401, 0x00A9, 0x0404, 0x00AB, 0x00AC, 0x00AD, 0x00AE, 0x0407,
	0x00B0, 0x00B1, 0x0406, 0x0456, 0x0491, 0x00B5, 0x00B6, 0x00B7,
	0x0451, 0x2116, 0x0454, 0x00BB, 0x0458, 0x0405, 0x0455, 0x0457,
	0x0410, 0x0411, 0x0412, 0x0413, 0x0414, 0x0415, 0x0416, 0x0417,
	0x0418, 0x0419, 0x041A, 0x041B, 0x041C, 0x041D, 0x041E, 0x041F,
	0x0420, 0x0421, 0x0422, 0x0423, 0x0424, 0x0425, 0x0426, 0x0427,
	0x0428, 0x0429, 0x042A, 0x042B, 0x042C, 0x042D, 0x042E, 0x042F,
	0x0430, 0x0431, 0x0432, 0x0433, 0x0434, 0x0435, 0x0436, 0x0437,
	0x0438, 0x0439, 0x043A, 0x043B, 0x043C, 0x043D, 0x043E, 0x043F,
	0x0440, 0x0441, 0x0442, 0x0443, 0x0444, 0x0445, 0x0446, 0x0447,
	0x0448, 0x0449, 0x044A, 0x044B, 0x044C, 0x044D, 0x044E, 0x044F,
}

var cp1251Encode = func() map[rune]byte {
	m := make(map[rune]byte, 256)
	for i, r := range cp1251Decode {
		if r == 0 {
			continue
		}
		m[r] = byte(0x80 + i)
	}
	return m
}()
