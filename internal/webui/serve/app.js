// studio review UI — no framework, no build step. Keeps the manifest in memory,
// PATCHes each edit optimistically, and drives everything from the keyboard
// (plan §7.2).

const el = (id) => document.getElementById(id);
const state = { man: null, cam: "", i: 0 };

// Single source of truth for the shortcuts — feeds both the one-line hint under
// the form and the ? overlay, so they can never drift apart. `keys` are the
// glyphs to show; `hint` is the compact label; `desc` the fuller one.
const SHORTCUTS = [
  { keys: ["j", "k"], hint: "next·prev", desc: "Next / previous clip" },
  { keys: ["space"], hint: "play", desc: "Play / pause" },
  { keys: ["h", "l"], hint: "seek (⇧ ±10s)", desc: "Seek ∓2s (hold ⇧ for ∓10s)" },
  { keys: [",", "."], hint: "frame", desc: "Step one frame back / forward" },
  { keys: ["1", "–", "5"], hint: "rate", desc: "Rate the clip 1–5" },
  { keys: ["x"], hint: "reject", desc: "Reject clip (press again to un-reject)" },
  { keys: ["r"], hint: "desc", desc: "Edit the description" },
  { keys: ["enter"], hint: "save + keep", desc: "Save description and mark the clip kept" },
  { keys: ["u"], hint: "next pending", desc: "Jump to the next undecided clip" },
  { keys: ["?"], hint: "help", desc: "Show / hide this shortcuts panel" },
];

const video = el("video");
const descInput = el("desc");
const takeInput = el("take");

init();

async function init() {
  try {
    const res = await fetch("api/manifest");
    state.man = await res.json();
    state.cam = state.man.shoot.camCode || "CAM";
    el("shoot-title").textContent = state.man.shoot.title || "review";
  } catch (e) {
    return toast("failed to load manifest");
  }
  renderList();
  if (state.man.clips.length) select(0);
  wireForm();
  wireTheme();
  wireHelp();
  document.addEventListener("keydown", onKey);
}

// wireHelp builds the hint bar and the ? overlay from SHORTCUTS, and wires the
// button, backdrop, and close control. Toggling is also bound to the ? key in
// onKey so it works from anywhere.
function wireHelp() {
  const kbd = (k) => (k === "–" ? "–" : `<kbd>${k}</kbd>`);
  document.querySelector(".hint").innerHTML = SHORTCUTS.map(
    (s) => `${s.keys.map(kbd).join("/")} ${s.hint}`,
  ).join(" · ");
  el("help-table").innerHTML = SHORTCUTS.map(
    (s) => `<tr><td class="keys">${s.keys.map(kbd).join(" ")}</td><td>${s.desc}</td></tr>`,
  ).join("");
  el("help-toggle").addEventListener("click", () => toggleHelp());
  el("help-close").addEventListener("click", () => toggleHelp(false));
  el("help-overlay").addEventListener("click", (e) => {
    if (e.target === el("help-overlay")) toggleHelp(false); // click the backdrop
  });
}

// toggleHelp shows/hides the overlay. Pass a boolean to force a state.
function toggleHelp(force) {
  const o = el("help-overlay");
  o.hidden = force === undefined ? !o.hidden : !force;
}

// Theme toggle: flip data-theme on <html>, persist, label the *other* theme.
function wireTheme() {
  const btn = el("theme-toggle");
  const sync = () => {
    const dark = document.documentElement.getAttribute("data-theme") !== "light";
    btn.textContent = dark ? "Light" : "Dark";
  };
  sync();
  btn.addEventListener("click", () => {
    const next = document.documentElement.getAttribute("data-theme") === "light" ? "dark" : "light";
    document.documentElement.setAttribute("data-theme", next);
    try { localStorage.setItem("studio-theme", next); } catch (e) {}
    sync();
  });
}

// --- rendering ---

function clip() { return state.man.clips[state.i]; }

function pad(n, w) { return String(n).padStart(w, "0"); }

function fmtDur(s) {
  s = Math.round(s || 0);
  const m = Math.floor(s / 60);
  return `${m}:${pad(s % 60, 2)}`;
}

function finalName(c) {
  const d = new Date(c.media.createdAt);
  const date = `${d.getUTCFullYear()}${pad(d.getUTCMonth() + 1, 2)}${pad(d.getUTCDate(), 2)}`;
  let name = `${date}_${state.cam}${pad(c.seq, 3)}_${c.review.desc || "…"}`;
  if (c.review.take != null) name += `_t${c.review.take}`;
  return name;
}

function renderList() {
  const ul = el("cliplist");
  ul.innerHTML = "";
  state.man.clips.forEach((c, idx) => {
    const li = document.createElement("li");
    li.className = c.review.status + (idx === state.i ? " active" : "");
    const stars = "★".repeat(c.review.rating) + "☆".repeat(5 - c.review.rating);
    li.innerHTML =
      `<span class="dot"></span>` +
      `<span class="name">${c.review.desc || c.stem}</span>` +
      `<span class="dur">${fmtDur(c.media.durationSec)}</span>` +
      `<span class="stars">${stars}</span>`;
    li.onclick = () => select(idx);
    ul.appendChild(li);
  });
  renderProgress();
}

function renderProgress() {
  let kept = 0, rejected = 0, pending = 0;
  for (const c of state.man.clips) {
    if (c.review.status === "kept") kept++;
    else if (c.review.status === "rejected") rejected++;
    else pending++;
  }
  el("progress").textContent = `kept ${kept} · rejected ${rejected} · pending ${pending}`;
}

function select(idx) {
  if (idx < 0 || idx >= state.man.clips.length) return;
  state.i = idx;
  const c = clip();

  const hasProxy = !!c.files.proxy;
  video.hidden = !hasProxy;
  el("noproxy").hidden = hasProxy;
  if (hasProxy) {
    video.src = `media/proxy/${c.id}`;
    video.play().catch(() => {});
  } else {
    video.removeAttribute("src");
  }

  el("meta").textContent =
    `${c.media.width}×${c.media.height} · ${c.media.fps || "?"}fps · ${c.media.vcodec} · proxy:${c.proxyInfo.source}`;
  descInput.value = c.review.desc || "";
  takeInput.value = c.review.take != null ? c.review.take : "";
  updatePreview();
  renderList();
  document.querySelector("#cliplist li.active")?.scrollIntoView({ block: "nearest" });
}

function updatePreview() {
  el("preview").textContent = finalName(clip());
}

// --- editing ---

async function patch(body) {
  const c = clip();
  try {
    const res = await fetch(`api/clips/${c.id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    if (!res.ok) return toast(await res.text());
    state.man.clips[state.i] = await res.json();
    renderList();
    updatePreview();
  } catch (e) {
    toast("save failed");
  }
}

function setRating(n) { patch({ rating: n }); }
function toggleReject() {
  patch({ status: clip().review.status === "rejected" ? "pending" : "rejected" });
}

function wireForm() {
  descInput.addEventListener("input", () => {
    // Normalize to the allowed slug charset as the user types.
    descInput.value = descInput.value.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "");
    clip().review.desc = descInput.value;
    updatePreview();
  });
  takeInput.addEventListener("input", () => {
    const v = parseInt(takeInput.value, 10);
    clip().review.take = Number.isFinite(v) && v > 0 ? v : null;
    updatePreview();
  });
}

// Save current desc/take and mark kept, then jump to the next pending clip.
async function saveKeepAdvance() {
  const v = parseInt(takeInput.value, 10);
  await patch({ desc: descInput.value, take: Number.isFinite(v) && v > 0 ? v : null, status: "kept" });
  descInput.blur();
  jumpNextPending();
}

function jumpNextPending() {
  const n = state.man.clips.length;
  for (let k = 1; k <= n; k++) {
    const idx = (state.i + k) % n;
    if (state.man.clips[idx].review.status === "pending") return select(idx);
  }
}

// --- keyboard ---

function onKey(e) {
  // The shortcuts overlay grabs keys first: Escape or ? closes it, and nothing
  // behind it should react while it's open.
  if (!el("help-overlay").hidden) {
    if (e.key === "Escape" || e.key === "?") toggleHelp(false);
    return;
  }

  const inInput = document.activeElement.tagName === "INPUT";
  if (inInput) {
    if (e.key === "Escape") document.activeElement.blur();
    else if (e.key === "Enter" && document.activeElement === descInput) {
      e.preventDefault();
      saveKeepAdvance();
    }
    return;
  }

  if (e.key === "?") {
    toggleHelp(true);
    return;
  }

  switch (e.key) {
    case "j": select(state.i + 1); break;
    case "k": select(state.i - 1); break;
    case " ": e.preventDefault(); video.paused ? video.play() : video.pause(); break;
    case "h": case "ArrowLeft": video.currentTime -= e.shiftKey ? 10 : 2; break;
    case "l": case "ArrowRight": video.currentTime += e.shiftKey ? 10 : 2; break;
    case ",": frameStep(-1); break;
    case ".": frameStep(1); break;
    case "1": case "2": case "3": case "4": case "5": setRating(+e.key); break;
    case "x": toggleReject(); break;
    case "r": e.preventDefault(); descInput.focus(); break;
    case "u": jumpNextPending(); break;
  }
}

function frameStep(dir) {
  video.pause();
  const fps = parseFloat(clip().media.fps) || 30;
  video.currentTime += dir / fps;
}

// --- toast ---

let toastTimer = null;
function toast(msg) {
  const t = el("toast");
  t.textContent = msg;
  t.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (t.hidden = true), 3000);
}
