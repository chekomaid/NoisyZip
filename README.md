# NoisyZip

Windows desktop app (Wails v2) to build noisy ZIP archives and recover broken
ZIPs. The UI offers two modes: Encryptor (build ZIP with noise options) and
Decryptor (recover + rebuild).

## Features
- Encryptor: add noise files, overwrite central directory, overwrite timestamps
- Decryptor: recover files from damaged ZIPs and rebuild a clean ZIP
- Optional seed for deterministic noise
- Optional include hidden files

## Requirements
- Go 1.20+ (or your installed Go)
- Node.js + npm
- Wails CLI v2 (`wails`)
- Windows WebView2 runtime (usually already installed on Windows 10/11)

## Build (Windows)
From repo root:
```bash
cd frontend
npm install
npm run build
cd ..
wails build -clean
```

## Development
```bash
cd frontend
npm install
npm run dev
```
Then in another terminal:
```bash
wails dev
```
