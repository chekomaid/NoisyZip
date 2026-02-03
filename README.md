# NoisyZip

NoisyZip is a CLI tool and GUI app for creating "noisy" ZIP archives and recovering damaged ZIPs.

- Noise: builds a ZIP with noise options (junk files, comments, fixed timestamps, central directory overwrite).
- Recover: extracts files from a damaged ZIP and rebuilds a clean archive.

[![Download](https://img.shields.io/badge/releases-blue?label=download&style=for-the-badge&colorA=A0A0A0&colorB=ffffff)](https://github.com/chekomaid/NoisyZip/releases/latest/)⠀

## Supported systems

| OS             | Version         | CLI | GUI |
|----------------|-----------------|:---:|:---:|
| Windows        | 11              |  ✓  |  ✓  |
|                | 10              |  ✓  |  ✓  |
| Windows Server | 2022            |  ✓  |  ✓  |
|                | 2019            |  ✓  |  ✓  |
| Ubuntu         | 24.04           |  ✓  |  ✗  |
|                | 22.04           |  ✓  |  ✗  |
|                | 20.04           |  ✓  |  ✗  |
| Debian         | 12              |  ✓  |  ✗  |
|                | 11              |  ✓  |  ✗  |
| Fedora         | 40              |  ✓  |  ✗  |
|                | 39              |  ✓  |  ✗  |
|                | 38              |  ✓  |  ✗  |
| macOS          | 14              |  ✓  |  ✗  |
|                | 13              |  ✓  |  ✗  |
|                | 12              |  ✓  |  ✗  |

## How to use

### Command
Noise (default):
```bash
noisyzip -src <dir> -out <zip> [options]
```
Recover:
```bash
noisyzip recover -in <zip> -out <zip> [options]
```

### Flags
Common:
- -compression / -method — deflate or store.
- -encoding — utf-8 or cp1251.
- -level — compression level 0..9.
- -strategy — default or huffman.
- -workers — number of workers (>=1).
- -seed — fixed seed (integer).
- -include-hidden — include hidden files.
- -config — path to JSON config (optional).

Noise:
- -src, -out — input folder and output ZIP.
- -no-overwrite-cdir — do not overwrite the central directory (overwritten by default).
- -comment-size — ZIP comment junk size (0..65535).
- -fixed-time — overwrite file timestamps.
- -noise-files, -noise-size — number and size of noise files.

Recover:
- -in, -out — input ZIP and output ZIP.

### Config
Noise config (example):
```json
{
  "src": "C:\\path\\to\\folder",
  "out": "C:\\path\\out.zip",
  "compression": "deflate",
  "encoding": "utf-8",
  "no-overwrite-cdir": false,
  "comment-size": 0,
  "fixed-time": false,
  "noise-files": 0,
  "noise-size": 0,
  "level": 6,
  "strategy": "default",
  "workers": 8,
  "seed": 123,
  "include-hidden": false
}
```

Recover config (example):
```json
{
  "in": "C:\\path\\input.zip",
  "out": "C:\\path\\rebuilt.zip",
  "compression": "deflate",
  "encoding": "utf-8",
  "level": 6,
  "strategy": "default",
  "workers": 8,
  "seed": "42",
  "include-hidden": false
}
```

## Installation (Linux)
```bash
curl -L -o /usr/local/bin/noisyzip "https://github.com/chekomaid/NoisyZip/releases/latest/download/noisyzip-linux-$([[ "$(uname -m)" == "x86_64" ]] && echo "amd64" || echo "arm64")"
sudo chmod u+x /usr/local/bin/noisyzip
```

## Support
- Bitcoin: `bc1qtzsunr39p7vv3cxny34v6jfs0unhvskjh0m2dm`
- Ethereum: `0xD0e86B88449a4f197D90dcAdB9feeEd784aFb345`