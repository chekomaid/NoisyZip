import "./style.css";
import { EventsOn } from "./wailsjs/runtime/runtime";
import {
  SelectSourceDir,
  SelectOutputZip,
  SelectInputZip,
  SelectOutputDir,
  RunEncrypt,
  RunRecover,
} from "./wailsjs/go/gui/App";

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
  overwriteCentralDir: document.getElementById("overwriteCentralDir"),
  fixedTime: document.getElementById("fixedTime"),
  noiseFiles: document.getElementById("noiseFiles"),
  noiseSize: document.getElementById("noiseSize"),
  commentSize: document.getElementById("commentSize"),
  progress: document.getElementById("progress"),
  status: document.getElementById("status"),
  start: document.getElementById("start"),
};

const rec = {
  view: document.getElementById("recoverView"),
  inZip: document.getElementById("recInZip"),
  outZip: document.getElementById("recOutZip"),
  browseIn: document.getElementById("recBrowseIn"),
  browseOut: document.getElementById("recBrowseOut"),
  includeHidden: document.getElementById("recIncludeHidden"),
  method: document.getElementById("recMethod"),
  encoding: document.getElementById("recEncoding"),
  level: document.getElementById("recLevel"),
  strategy: document.getElementById("recStrategy"),
  workers: document.getElementById("recWorkers"),
  seed: document.getElementById("recSeed"),
  progress: document.getElementById("recProgress"),
  status: document.getElementById("recStatus"),
  start: document.getElementById("recStart"),
};

const modeEncrypt = document.getElementById("modeEncrypt");
const modeRecover = document.getElementById("modeRecover");
const ethCopy = document.getElementById("ethCopy");

const encLockables = document.querySelectorAll("#encryptorView [data-lock]");
const recLockables = document.querySelectorAll("#recoverView [data-lock]");

function setMode(mode) {
  const isEncrypt = mode === "encrypt";
  enc.view.classList.toggle("hidden", !isEncrypt);
  rec.view.classList.toggle("hidden", isEncrypt);
  modeEncrypt.classList.toggle("active", isEncrypt);
  modeRecover.classList.toggle("active", !isEncrypt);
}

modeEncrypt.addEventListener("click", () => setMode("encrypt"));
modeRecover.addEventListener("click", () => setMode("recover"));
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

rec.method.addEventListener("change", () =>
  updateDeflateControls(rec.method, rec.level, rec.strategy),
);
updateDeflateControls(rec.method, rec.level, rec.strategy);

const cpuCount = navigator.hardwareConcurrency || 4;
enc.workers.value = cpuCount;
rec.workers.value = cpuCount;

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

rec.browseIn.addEventListener("click", async () => {
  const path = await SelectInputZip();
  if (path) {
    rec.inZip.value = path;
  }
});

rec.browseOut.addEventListener("click", async () => {
  const path = await SelectOutputZip();
  if (path) {
    rec.outZip.value = path;
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

EventsOn("recover:log", (msg) => {
  if (typeof msg === "string" && msg.trim()) {
    setStatus(rec.status, msg);
  }
});

EventsOn("recover:progress", (payload) => {
  if (!payload) return;
  const done = payload.done ?? 0;
  const total = payload.total ?? 0;
  const name = payload.name ?? "";
  if (total > 0) {
    rec.progress.value = done / total;
  }
  setStatus(rec.status, `${done}/${total}: ${name}`);
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
    overwriteCentralDir: enc.overwriteCentralDir.checked,
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

rec.start.addEventListener("click", async () => {
  const inZip = rec.inZip.value.trim();
  const outZip = rec.outZip.value.trim();
  if (!inZip || !outZip) {
    setStatus(rec.status, "Choose input ZIP and output ZIP.");
    return;
  }

  const level = parseNumber(rec.level.value, 6);
  const workers = parseNumber(rec.workers.value, cpuCount);

  if (level < 0 || level > 9) {
    setStatus(rec.status, "Compression level must be 0..9.");
    return;
  }
  if (workers < 1) {
    setStatus(rec.status, "Workers must be >= 1.");
    return;
  }

  rec.progress.value = 0;
  setStatus(rec.status, "Starting...");
  setRunning(recLockables, true);

  const cfg = {
    inZip,
    outZip,
    compression: rec.method.value,
    encoding: rec.encoding.value,
    level,
    strategy: rec.strategy.value,
    dictSize: 32768,
    workers,
    seed: rec.seed.value.trim(),
    includeHidden: rec.includeHidden.checked,
  };

  try {
    const result = await RunRecover(cfg);
    setStatus(
      rec.status,
      `Recovered: ${result.recovered}, ZIP files: ${result.rebuilt}`,
    );
    rec.progress.value = 1;
  } catch (err) {
    const message = err?.message || String(err);
    setStatus(rec.status, `Error: ${message}`);
  } finally {
    setRunning(recLockables, false);
  }
});
