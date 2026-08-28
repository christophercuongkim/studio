// studio dashboard — read-only cockpit (slice 1). Lists projects with their
// pipeline stage and a detail view. No framework, no build; reads the same
// pipeline.State the CLI's `studio status` prints.

const el = (id) => document.getElementById(id);

init();

async function init() {
  wireTheme();
  el("back").addEventListener("click", showOverview);
  await loadProjects();
}

async function loadProjects() {
  let projects;
  try {
    projects = await (await fetch("/api/projects")).json();
  } catch (e) {
    return toast("failed to load projects");
  }
  const cards = el("cards");
  cards.innerHTML = "";
  el("empty").hidden = projects.length > 0;
  for (const p of projects) {
    cards.appendChild(projectCard(p));
  }
}

function projectCard(p) {
  const card = document.createElement("button");
  card.className = "card";
  card.onclick = () => showDetail(p.id);

  const h = document.createElement("h3");
  h.textContent = p.title;

  const strip = document.createElement("div");
  strip.className = "strip";
  for (const step of p.steps) {
    const pip = document.createElement("span");
    pip.className = "pip" + (step.done ? " done" : "") + (step.name === p.next ? " next" : "");
    pip.title = step.name + (step.done ? " ✓" : "");
    strip.appendChild(pip);
  }

  const badge = document.createElement("div");
  badge.className = "next-badge" + (p.next ? "" : " complete");
  badge.innerHTML = p.next ? `next: <span class="kw">${p.next}</span>` : "✓ complete";

  card.append(h, strip, badge);
  return card;
}

// Steps the dashboard can run, and which support a dry-run preview.
const RUNNABLE = new Set(["apply", "scaffold", "chapters", "qc", "thumbs", "archive"]);
const DRYABLE = new Set(["apply", "archive"]);

async function showDetail(id) {
  el("runlog").textContent = "";
  el("runlog").hidden = true;
  await renderDetail(id);
  el("overview").hidden = true;
  el("detail").hidden = false;
}

// renderDetail (re)draws the top of the detail view (checklist, actions, meta)
// without wiping the run log, so a step's output stays visible as the checklist
// advances.
async function renderDetail(id) {
  let d;
  try {
    d = await (await fetch(`/api/projects/${id}`)).json();
  } catch (e) {
    return toast("failed to load project");
  }
  el("detail-title").textContent = d.title;
  const top = el("detail-top");
  top.innerHTML = "";
  top.append(checklist(d), actions(d), metaPanels(d));
}

function actions(d) {
  const wrap = document.createElement("div");
  wrap.className = "actions";
  const step = d.next;
  if (!step) return wrap; // complete

  if (!RUNNABLE.has(step)) {
    const note = document.createElement("p");
    note.className = "note";
    note.textContent = nextNote(step);
    wrap.appendChild(note);
    return wrap;
  }
  if (DRYABLE.has(step)) {
    wrap.appendChild(button(`Preview ${step}`, () => runStep(d.id, step, true)));
  }
  wrap.appendChild(button(`Run ${step}`, () => {
    if (DRYABLE.has(step) && !confirm(`Run ${step}? This changes files in the project.`)) return;
    runStep(d.id, step, false);
  }, "primary"));
  return wrap;
}

function nextNote(step) {
  const via = { ingest: "studio ingest", review: "studio serve", upload: "studio upload" }[step];
  return via ? `Next: ${step} — coming to the dashboard soon; for now use \`${via}\`.` : `Next: ${step}`;
}

function button(label, onclick, kind) {
  const b = document.createElement("button");
  b.type = "button";
  b.className = "btn" + (kind === "primary" ? " primary" : "");
  b.textContent = label;
  b.onclick = onclick;
  return b;
}

async function runStep(id, step, dry) {
  const log = el("runlog");
  log.hidden = false;
  document.querySelectorAll(".actions .btn").forEach((b) => (b.disabled = true));
  try {
    const res = await fetch(`/api/projects/${id}/run/${step}${dry ? "?dry=1" : ""}`, { method: "POST" });
    const reader = res.body.getReader();
    const dec = new TextDecoder();
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      log.textContent += dec.decode(value, { stream: true });
      log.scrollTop = log.scrollHeight;
    }
  } catch (e) {
    log.textContent += "\n[connection error]\n";
  }
  // A real run may have advanced the pipeline — re-render (keeps the log).
  if (!dry) await renderDetail(id);
  else document.querySelectorAll(".actions .btn").forEach((b) => (b.disabled = false));
}

function showOverview() {
  el("detail").hidden = true;
  el("overview").hidden = false;
  loadProjects(); // refresh in case anything changed
}

function checklist(d) {
  const ul = document.createElement("ul");
  ul.className = "checklist";
  for (const step of d.steps) {
    const li = document.createElement("li");
    li.className = (step.done ? "done" : "") + (step.name === d.next ? " next" : "");
    li.innerHTML =
      `<span class="box">${step.done ? "[x]" : "[ ]"}</span>` +
      `<span class="name">${step.name}</span>` +
      `<span class="detail">${step.detail || ""}</span>`;
    ul.appendChild(li);
  }
  return ul;
}

function metaPanels(d) {
  const wrap = document.createElement("div");
  wrap.className = "meta";
  const v = d.video || {};
  const c = d.clips || {};
  wrap.appendChild(panel("Video", {
    Title: v.title || "—",
    Privacy: v.privacy || "—",
    Tags: (v.tags && v.tags.length) ? v.tags.join(", ") : "—",
    Render: v.render || "—",
    Thumbnail: v.thumbnail || "—",
    YouTube: v.videoId ? `youtu.be/${v.videoId}` : "—",
  }));
  wrap.appendChild(panel("Clips", {
    Total: c.total || 0,
    Kept: c.kept || 0,
    Rejected: c.rejected || 0,
    Pending: c.pending || 0,
    Proxies: `${c.camera || 0} camera · ${c.generated || 0} generated · ${c.noProxy || 0} none`,
  }));
  return wrap;
}

function panel(title, kv) {
  const p = document.createElement("div");
  p.className = "panel";
  const h = document.createElement("h2");
  h.textContent = title;
  const dl = document.createElement("dl");
  dl.className = "kv";
  for (const [k, val] of Object.entries(kv)) {
    const dt = document.createElement("dt");
    dt.textContent = k;
    const dd = document.createElement("dd");
    dd.textContent = String(val);
    dl.append(dt, dd);
  }
  p.append(h, dl);
  return p;
}

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

let toastTimer = null;
function toast(msg) {
  const t = el("toast");
  t.textContent = msg;
  t.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (t.hidden = true), 3000);
}
