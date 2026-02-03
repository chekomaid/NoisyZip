package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"noisyzip/internal/core"
)

func Main(args []string) int {
	if len(args) == 0 {
		printEncryptHelp(os.Stdout)
		return 2
	}
	mode := strings.ToLower(strings.TrimSpace(args[0]))
	if mode == "cli" {
		args = args[1:]
		if len(args) == 0 {
			printEncryptHelp(os.Stdout)
			return 2
		}
		mode = strings.ToLower(strings.TrimSpace(args[0]))
	}

	switch mode {
	case "help", "-h", "--help":
		printHelp(os.Stdout)
		return 0
	case "recover":
		return runRecover(args[1:])
	default:
		if strings.HasPrefix(mode, "-") {
			return runEncrypt(args)
		}
		fmt.Fprintln(os.Stderr, "Error: unknown command", mode)
		printHelp(os.Stderr)
		return 2
	}
}

type encryptOptions struct {
	help                bool
	configPath          string
	srcDir              string
	outZip              string
	compression         string
	encoding            string
	overwriteCentralDir bool
	commentSize         int
	fixedTime           bool
	noiseFiles          int
	noiseSize           int
	level               int
	strategy            string
	workers             int
	seed                string
	includeHidden       bool
}

func newEncryptFlagSet(output io.Writer) (*flag.FlagSet, *encryptOptions) {
	opts := &encryptOptions{
		compression:         "deflate",
		encoding:            "utf-8",
		overwriteCentralDir: true,
		level:               6,
		strategy:            "default",
		workers:             runtime.NumCPU(),
	}
	fs := flag.NewFlagSet("encrypt", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&opts.help, "h", false, "Show help")
	fs.BoolVar(&opts.help, "help", false, "Show help")
	fs.StringVar(&opts.configPath, "config", "", "Path to JSON config file")
	fs.StringVar(&opts.srcDir, "src", "", "Input directory")
	fs.StringVar(&opts.outZip, "out", "", "Output ZIP path")
	fs.StringVar(&opts.compression, "compression", opts.compression, "Compression method: deflate or store")
	fs.StringVar(&opts.compression, "method", opts.compression, "Alias for -compression")
	fs.StringVar(&opts.encoding, "encoding", opts.encoding, "Filename encoding: utf-8 or cp1251")
	fs.Var(&negatedBoolFlag{target: &opts.overwriteCentralDir}, "no-overwrite-cdir", "Do not overwrite central directory")
	fs.IntVar(&opts.commentSize, "comment-size", 0, "ZIP comment junk size (bytes)")
	fs.BoolVar(&opts.fixedTime, "fixed-time", false, "Overwrite file timestamps")
	fs.IntVar(&opts.noiseFiles, "noise-files", 0, "Number of noise files")
	fs.IntVar(&opts.noiseSize, "noise-size", 0, "Size of each noise file (bytes)")
	fs.IntVar(&opts.level, "level", opts.level, "Deflate level (0-9)")
	fs.StringVar(&opts.strategy, "strategy", opts.strategy, "Deflate strategy: default, huffman")
	fs.IntVar(&opts.workers, "workers", opts.workers, "Worker goroutines")
	fs.StringVar(&opts.seed, "seed", "", "Deterministic noise seed (integer)")
	fs.BoolVar(&opts.includeHidden, "include-hidden", false, "Include hidden files")
	return fs, opts
}

type recoverOptions struct {
	help          bool
	configPath    string
	inZip         string
	outZip        string
	compression   string
	encoding      string
	level         int
	strategy      string
	workers       int
	seed          string
	includeHidden bool
}

type negatedBoolFlag struct {
	target *bool
}

func (f *negatedBoolFlag) String() string {
	if f == nil || f.target == nil {
		return "false"
	}
	return strconv.FormatBool(!*f.target)
}

func (f *negatedBoolFlag) Set(val string) error {
	b, err := strconv.ParseBool(val)
	if err != nil {
		return err
	}
	if f.target != nil {
		*f.target = !b
	}
	return nil
}

func (f *negatedBoolFlag) IsBoolFlag() bool {
	return true
}

func newRecoverFlagSet(output io.Writer) (*flag.FlagSet, *recoverOptions) {
	opts := &recoverOptions{
		compression: "deflate",
		encoding:    "utf-8",
		level:       6,
		strategy:    "default",
		workers:     runtime.NumCPU(),
	}
	fs := flag.NewFlagSet("recover", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.BoolVar(&opts.help, "h", false, "Show help")
	fs.BoolVar(&opts.help, "help", false, "Show help")
	fs.StringVar(&opts.configPath, "config", "", "Path to JSON config file")
	fs.StringVar(&opts.inZip, "in", "", "Input ZIP path")
	fs.StringVar(&opts.outZip, "out", "", "Output ZIP path")
	fs.StringVar(&opts.compression, "compression", opts.compression, "Compression method: deflate or store")
	fs.StringVar(&opts.compression, "method", opts.compression, "Alias for -compression")
	fs.StringVar(&opts.encoding, "encoding", opts.encoding, "Filename encoding: utf-8 or cp1251")
	fs.IntVar(&opts.level, "level", opts.level, "Deflate level (0-9)")
	fs.StringVar(&opts.strategy, "strategy", opts.strategy, "Deflate strategy: default, filtered, huffman, rle, fixed")
	fs.IntVar(&opts.workers, "workers", opts.workers, "Worker goroutines")
	fs.StringVar(&opts.seed, "seed", "", "Deterministic noise seed (integer)")
	fs.BoolVar(&opts.includeHidden, "include-hidden", false, "Include hidden files")
	return fs, opts
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, "NoisyZip CLI")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  noisyzip -src <dir> -out <zip> [options]")
	fmt.Fprintln(w, "  noisyzip recover -in <zip> -out <zip> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Run noisyzip -h or noisyzip recover -h for options.")
}

func printEncryptHelp(w io.Writer) {
	fs, _ := newEncryptFlagSet(w)
	fmt.Fprintln(w, "Usage: noisyzip -src <dir> -out <zip> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fs.PrintDefaults()
}

func printRecoverHelp(w io.Writer) {
	fs, _ := newRecoverFlagSet(w)
	fmt.Fprintln(w, "Usage: noisyzip recover -in <zip> -out <zip> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Options:")
	fs.PrintDefaults()
}

func runEncrypt(args []string) int {
	fs, opts := newEncryptFlagSet(io.Discard)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		printEncryptHelp(os.Stderr)
		return 2
	}
	if opts.help {
		printEncryptHelp(os.Stdout)
		return 0
	}
	if cfgPath := strings.TrimSpace(opts.configPath); cfgPath != "" {
		cfg, err := readConfig(cfgPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: config:", err)
			return 2
		}
		applyEncryptConfig(opts, cfg, collectVisitedFlags(fs))
	}

	src := strings.TrimSpace(opts.srcDir)
	outZip := strings.TrimSpace(opts.outZip)
	if src == "" || outZip == "" {
		fmt.Fprintln(os.Stderr, "Error: -src and -out are required")
		printEncryptHelp(os.Stderr)
		return 2
	}
	if !strings.HasSuffix(strings.ToLower(outZip), ".zip") {
		outZip += ".zip"
	}

	cfg := core.Config{
		SrcDir:              src,
		OutZip:              outZip,
		Compression:         opts.compression,
		Encoding:            opts.encoding,
		OverwriteCentralDir: opts.overwriteCentralDir,
		CommentSize:         opts.commentSize,
		FixedTime:           opts.fixedTime,
		NoiseFiles:          opts.noiseFiles,
		NoiseSize:           opts.noiseSize,
		Level:               opts.level,
		Strategy:            opts.strategy,
		DictSize:            32768,
		Workers:             opts.workers,
		IncludeHidden:       opts.includeHidden,
	}

	seedText := strings.TrimSpace(opts.seed)
	if seedText != "" {
		seedVal, err := strconv.ParseInt(seedText, 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: seed must be an integer")
			return 2
		}
		cfg.Seed = seedVal
		cfg.HasSeed = true
	}

	progress := func(done, total int, name string) {
		fmt.Fprintf(os.Stderr, "%d/%d: %s\n", done, total, name)
	}
	logCb := func(msg string) {
		if strings.TrimSpace(msg) == "" {
			return
		}
		fmt.Fprintln(os.Stderr, msg)
	}

	total, err := core.RunEncrypt(cfg, progress, logCb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "Done. Files: %d\nOutput: %s\n", total, outZip)
	return 0
}

func runRecover(args []string) int {
	fs, opts := newRecoverFlagSet(io.Discard)
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		printRecoverHelp(os.Stderr)
		return 2
	}
	if opts.help {
		printRecoverHelp(os.Stdout)
		return 0
	}
	if cfgPath := strings.TrimSpace(opts.configPath); cfgPath != "" {
		cfg, err := readConfig(cfgPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: config:", err)
			return 2
		}
		applyRecoverConfig(opts, cfg, collectVisitedFlags(fs))
	}

	inZip := strings.TrimSpace(opts.inZip)
	outZip := strings.TrimSpace(opts.outZip)
	if inZip == "" || outZip == "" {
		fmt.Fprintln(os.Stderr, "Error: -in and -out are required")
		printRecoverHelp(os.Stderr)
		return 2
	}
	if !strings.HasSuffix(strings.ToLower(outZip), ".zip") {
		outZip += ".zip"
	}

	logCb := func(msg string) {
		if strings.TrimSpace(msg) == "" {
			return
		}
		fmt.Fprintln(os.Stderr, msg)
	}
	progress := func(done, total int, name string) {
		fmt.Fprintf(os.Stderr, "%d/%d: %s\n", done, total, name)
	}

	tmpDir, err := os.MkdirTemp("", "zip-recover-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	defer os.RemoveAll(tmpDir)

	recovered, err := core.RecoverZip(inZip, tmpDir, progress, logCb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	cfg := core.Config{
		SrcDir:              tmpDir,
		OutZip:              outZip,
		Compression:         opts.compression,
		Encoding:            opts.encoding,
		OverwriteCentralDir: false,
		CommentSize:         0,
		FixedTime:           false,
		NoiseFiles:          0,
		NoiseSize:           0,
		Level:               opts.level,
		Strategy:            opts.strategy,
		DictSize:            32768,
		Workers:             opts.workers,
		IncludeHidden:       opts.includeHidden,
	}

	seedText := strings.TrimSpace(opts.seed)
	if seedText != "" {
		seedVal, err := strconv.ParseInt(seedText, 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error: seed must be an integer")
			return 2
		}
		cfg.Seed = seedVal
		cfg.HasSeed = true
	}

	rebuilt, err := core.RunEncrypt(cfg, nil, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}

	fmt.Fprintf(os.Stdout, "Recovered: %d\nZIP files: %d\nOutput: %s\n", recovered, rebuilt, outZip)
	return 0
}
