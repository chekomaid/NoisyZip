import "./style.css";
import { EventsOn } from "./wailsjs/runtime/runtime";
import {
  SelectSourceDir,
  SelectOutputZip,
  SelectInputZip,
  SelectOutputDir,
  RunEncrypt,
  RunDecrypt,
} from "./wailsjs/go/main/App";

const themeMedia = window.matchMedia("(prefers-color-scheme: light)");
const root = document.documentElement;

function applyTheme() {
  root.classList.toggle("light", themeMedia.matches);
}

applyTheme();
if (themeMedia.addEventListener) {
  themeMedia.addEventListener("change", applyTheme);
} else if (themeMedia.addListener) {
  themeMedia.addListener(applyTheme);
}

const enc = {
  view: document.getElementById("encryptorView"),
  srcDir: document.getElementById("srcDir"),
  outZip: document.getElementById("outZip"),
  browseSrc: document.getElementById("browseSrc"),
  browseOut: document.getElementById("browseOut"),
  includeHidden: document.getElementById("includeHidden"),
  method: document.getElementById("method"),
  encoding: document.getElementById("encoding"),
  level: document.getElementById("level"),
  strategy: document.getElementById("strategy"),
  workers: document.getElementById("workers"),
  seed: document.getElementById("seed"),
  breakCDir: document.getElementById("breakCDir"),
  fixedTime: document.getElementById("fixedTime"),
  noiseFiles: document.getElementById("noiseFiles"),
  noiseSize: document.getElementById("noiseSize"),
  commentSize: document.getElementById("commentSize"),
  progress: document.getElementById("progress"),
  status: document.getElementById("status"),
  start: document.getElementById("start"),
};

const dec = {
  view: document.getElementById("decryptorView"),
  inZip: document.getElementById("decInZip"),
  outZip: document.getElementById("decOutZip"),
  browseIn: document.getElementById("decBrowseIn"),
  browseOut: document.getElementById("decBrowseOut"),
  includeHidden: document.getElementById("decIncludeHidden"),
  method: document.getElementById("decMethod"),
  encoding: document.getElementById("decEncoding"),
  level: document.getElementById("decLevel"),
  strategy: document.getElementById("decStrategy"),
  workers: document.getElementById("decWorkers"),
  seed: document.getElementById("decSeed"),
  progress: document.getElementById("decProgress"),
  status: document.getElementById("decStatus"),
  start: document.getElementById("decStart"),
};

const modeEncrypt = document.getElementById("modeEncrypt");
const modeDecrypt = document.getElementById("modeDecrypt");
const ethCopy = document.getElementById("ethCopy");

const encLockables = document.querySelectorAll("#encryptorView [data-lock]");
const decLockables = document.querySelectorAll("#decryptorView [data-lock]");

function setMode(mode) {
  const isEncrypt = mode === "encrypt";
  enc.view.classList.toggle("hidden", !isEncrypt);
  dec.view.classList.toggle("hidden", isEncrypt);
  modeEncrypt.classList.toggle("active", isEncrypt);
  modeDecrypt.classList.toggle("active", !isEncrypt);
}

modeEncrypt.addEventListener("click", () => setMode("encrypt"));
modeDecrypt.addEventListener("click", () => setMode("decrypt"));
setMode("encrypt");

async function copyText(text) {
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    return;
  } catch (_) {
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.setAttribute("readonly", "");
    ta.style.position = "fixed";
    ta.style.top = "-9999px";
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand("copy");
    } finally {
      document.body.removeChild(ta);
    }
  }
}

if (ethCopy) {
  ethCopy.addEventListener("click", async () => {
    const address = ethCopy.dataset.address || ethCopy.textContent.trim();
    await copyText(address);
    ethCopy.classList.add("copied");
    setTimeout(() => ethCopy.classList.remove("copied"), 600);
  });
}

function setRunning(lockables, running) {
  lockables.forEach((el) => {
    el.disabled = running;
  });
}

function setStatus(el, text) {
  el.textContent = text;
}

function parseNumber(value, fallback = 0) {
  const n = Number.parseInt(String(value).trim(), 10);
  return Number.isFinite(n) ? n : fallback;
}

function updateDeflateControls(methodEl, levelEl, strategyEl) {
  const isDeflate = methodEl.value === "deflate";
  levelEl.disabled = !isDeflate;
  strategyEl.disabled = !isDeflate;
}

enc.method.addEventListener("change", () =>
  updateDeflateControls(enc.method, enc.level, enc.strategy),
);
updateDeflateControls(enc.method, enc.level, enc.strategy);

dec.method.addEventListener("change", () =>
  updateDeflateControls(dec.method, dec.level, dec.strategy),
);
updateDeflateControls(dec.method, dec.level, dec.strategy);

const cpuCount = navigator.hardwareConcurrency || 4;
enc.workers.value = cpuCount;
dec.workers.value = cpuCount;

enc.browseSrc.addEventListener("click", async () => {
  const path = await SelectSourceDir();
  if (path) {
    enc.srcDir.value = path;
  }
});

enc.browseOut.addEventListener("click", async () => {
  const path = await SelectOutputZip();
  if (path) {
    enc.outZip.value = path;
  }
});

dec.browseIn.addEventListener("click", async () => {
  const path = await SelectInputZip();
  if (path) {
    dec.inZip.value = path;
  }
});

dec.browseOut.addEventListener("click", async () => {
  const path = await SelectOutputZip();
  if (path) {
    dec.outZip.value = path;
  }
});

EventsOn("encrypt:log", (msg) => {
  if (typeof msg === "string" && msg.trim()) {
    setStatus(enc.status, msg);
  }
});

EventsOn("encrypt:progress", (payload) => {
  if (!payload) return;
  const done = payload.done ?? 0;
  const total = payload.total ?? 0;
  const name = payload.name ?? "";
  if (total > 0) {
    enc.progress.value = done / total;
  }
  setStatus(enc.status, `${done}/${total}: ${name}`);
});

EventsOn("decrypt:log", (msg) => {
  if (typeof msg === "string" && msg.trim()) {
    setStatus(dec.status, msg);
  }
});

EventsOn("decrypt:progress", (payload) => {
  if (!payload) return;
  const done = payload.done ?? 0;
  const total = payload.total ?? 0;
  const name = payload.name ?? "";
  if (total > 0) {
    dec.progress.value = done / total;
  }
  setStatus(dec.status, `${done}/${total}: ${name}`);
});

enc.start.addEventListener("click", async () => {
  const srcDir = enc.srcDir.value.trim();
  const outZip = enc.outZip.value.trim();
  if (!srcDir || !outZip) {
    setStatus(enc.status, "Choose input directory and output ZIP.");
    return;
  }

  const noiseFiles = parseNumber(enc.noiseFiles.value, 0);
  const noiseSize = parseNumber(enc.noiseSize.value, 0);
  const commentSize = parseNumber(enc.commentSize.value, 0);
  const level = parseNumber(enc.level.value, 6);
  const workers = parseNumber(enc.workers.value, cpuCount);

  if (commentSize < 0 || commentSize > 65535) {
    setStatus(enc.status, "ZIP comment junk must be 0..65535.");
    return;
  }
  if (noiseFiles < 0 || noiseSize < 0) {
    setStatus(enc.status, "Noise files and size must be >= 0.");
    return;
  }
  if (level < 0 || level > 9) {
    setStatus(enc.status, "Compression level must be 0..9.");
    return;
  }
  if (workers < 1) {
    setStatus(enc.status, "Workers must be >= 1.");
    return;
  }

  enc.progress.value = 0;
  setStatus(enc.status, "Starting...");
  setRunning(encLockables, true);

  const cfg = {
    srcDir,
    outZip,
    compression: enc.method.value,
    encoding: enc.encoding.value,
    breakCDir: enc.breakCDir.checked,
    commentSize,
    fixedTime: enc.fixedTime.checked,
    noiseFiles,
    noiseSize,
    level,
    strategy: enc.strategy.value,
    dictSize: 32768,
    workers,
    seed: enc.seed.value.trim(),
    includeHidden: enc.includeHidden.checked,
  };

  try {
    const result = await RunEncrypt(cfg);
    setStatus(enc.status, `Done. Files: ${result.total}`);
    enc.progress.value = 1;
  } catch (err) {
    const message = err?.message || String(err);
    setStatus(enc.status, `Error: ${message}`);
  } finally {
    setRunning(encLockables, false);
  }
});

dec.start.addEventListener("click", async () => {
  const inZip = dec.inZip.value.trim();
  const outZip = dec.outZip.value.trim();
  if (!inZip || !outZip) {
    setStatus(dec.status, "Choose input ZIP and output ZIP.");
    return;
  }

  const level = parseNumber(dec.level.value, 6);
  const workers = parseNumber(dec.workers.value, cpuCount);

  if (level < 0 || level > 9) {
    setStatus(dec.status, "Compression level must be 0..9.");
    return;
  }
  if (workers < 1) {
    setStatus(dec.status, "Workers must be >= 1.");
    return;
  }

  dec.progress.value = 0;
  setStatus(dec.status, "Starting...");
  setRunning(decLockables, true);

  const cfg = {
    inZip,
    outZip,
    compression: dec.method.value,
    encoding: dec.encoding.value,
    level,
    strategy: dec.strategy.value,
    dictSize: 32768,
    workers,
    seed: dec.seed.value.trim(),
    includeHidden: dec.includeHidden.checked,
  };

  try {
    const result = await RunDecrypt(cfg);
    setStatus(
      dec.status,
      `Recovered: ${result.recovered}, ZIP files: ${result.rebuilt}`,
    );
    dec.progress.value = 1;
  } catch (err) {
    const message = err?.message || String(err);
    setStatus(dec.status, `Error: ${message}`);
  } finally {
    setRunning(decLockables, false);
  }
});
