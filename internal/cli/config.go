package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

type configSeed struct {
	Value string
	Set   bool
}

func (s *configSeed) UnmarshalJSON(data []byte) error {
	if s == nil {
		return nil
	}
	if len(bytes.TrimSpace(data)) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}

	var asString string
	if err := json.Unmarshal(data, &asString); err == nil {
		s.Value = asString
		s.Set = true
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var num json.Number
	if err := dec.Decode(&num); err != nil {
		return fmt.Errorf("seed must be a string or number")
	}
	val, err := num.Int64()
	if err != nil {
		return fmt.Errorf("seed must be an integer")
	}
	s.Value = strconv.FormatInt(val, 10)
	s.Set = true
	return nil
}

type fileConfig struct {
	SrcDir                *string    `json:"src"`
	OutZip                *string    `json:"out"`
	InZip                 *string    `json:"in"`
	Compression           *string    `json:"compression"`
	Method                *string    `json:"method"`
	Encoding              *string    `json:"encoding"`
	NoOverwriteCentralDir *bool      `json:"no-overwrite-cdir"`
	CommentSize           *int       `json:"comment-size"`
	FixedTime             *bool      `json:"fixed-time"`
	NoiseFiles            *int       `json:"noise-files"`
	NoiseSize             *int       `json:"noise-size"`
	Level                 *int       `json:"level"`
	Strategy              *string    `json:"strategy"`
	Workers               *int       `json:"workers"`
	Seed                  configSeed `json:"seed"`
	IncludeHidden         *bool      `json:"include-hidden"`
}

func readConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg fileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

func collectVisitedFlags(fs *flag.FlagSet) map[string]bool {
	visited := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func flagWasSet(visited map[string]bool, names ...string) bool {
	for _, name := range names {
		if visited[name] {
			return true
		}
	}
	return false
}

func applyEncryptConfig(opts *encryptOptions, cfg *fileConfig, visited map[string]bool) {
	if cfg == nil || opts == nil {
		return
	}
	if !flagWasSet(visited, "src") && cfg.SrcDir != nil {
		opts.srcDir = *cfg.SrcDir
	}
	if !flagWasSet(visited, "out") && cfg.OutZip != nil {
		opts.outZip = *cfg.OutZip
	}
	if !flagWasSet(visited, "compression", "method") {
		if cfg.Compression != nil {
			opts.compression = *cfg.Compression
		} else if cfg.Method != nil {
			opts.compression = *cfg.Method
		}
	}
	if !flagWasSet(visited, "encoding") && cfg.Encoding != nil {
		opts.encoding = *cfg.Encoding
	}
	if !flagWasSet(visited, "no-overwrite-cdir") && cfg.NoOverwriteCentralDir != nil {
		opts.overwriteCentralDir = !*cfg.NoOverwriteCentralDir
	}
	if !flagWasSet(visited, "comment-size") && cfg.CommentSize != nil {
		opts.commentSize = *cfg.CommentSize
	}
	if !flagWasSet(visited, "fixed-time") && cfg.FixedTime != nil {
		opts.fixedTime = *cfg.FixedTime
	}
	if !flagWasSet(visited, "noise-files") && cfg.NoiseFiles != nil {
		opts.noiseFiles = *cfg.NoiseFiles
	}
	if !flagWasSet(visited, "noise-size") && cfg.NoiseSize != nil {
		opts.noiseSize = *cfg.NoiseSize
	}
	if !flagWasSet(visited, "level") && cfg.Level != nil {
		opts.level = *cfg.Level
	}
	if !flagWasSet(visited, "strategy") && cfg.Strategy != nil {
		opts.strategy = *cfg.Strategy
	}
	if !flagWasSet(visited, "workers") && cfg.Workers != nil {
		opts.workers = *cfg.Workers
	}
	if !flagWasSet(visited, "seed") && cfg.Seed.Set {
		opts.seed = cfg.Seed.Value
	}
	if !flagWasSet(visited, "include-hidden") && cfg.IncludeHidden != nil {
		opts.includeHidden = *cfg.IncludeHidden
	}
}

func applyRecoverConfig(opts *recoverOptions, cfg *fileConfig, visited map[string]bool) {
	if cfg == nil || opts == nil {
		return
	}
	if !flagWasSet(visited, "in") && cfg.InZip != nil {
		opts.inZip = *cfg.InZip
	}
	if !flagWasSet(visited, "out") && cfg.OutZip != nil {
		opts.outZip = *cfg.OutZip
	}
	if !flagWasSet(visited, "compression", "method") {
		if cfg.Compression != nil {
			opts.compression = *cfg.Compression
		} else if cfg.Method != nil {
			opts.compression = *cfg.Method
		}
	}
	if !flagWasSet(visited, "encoding") && cfg.Encoding != nil {
		opts.encoding = *cfg.Encoding
	}
	if !flagWasSet(visited, "level") && cfg.Level != nil {
		opts.level = *cfg.Level
	}
	if !flagWasSet(visited, "strategy") && cfg.Strategy != nil {
		opts.strategy = *cfg.Strategy
	}
	if !flagWasSet(visited, "workers") && cfg.Workers != nil {
		opts.workers = *cfg.Workers
	}
	if !flagWasSet(visited, "seed") && cfg.Seed.Set {
		opts.seed = cfg.Seed.Value
	}
	if !flagWasSet(visited, "include-hidden") && cfg.IncludeHidden != nil {
		opts.includeHidden = *cfg.IncludeHidden
	}
}
