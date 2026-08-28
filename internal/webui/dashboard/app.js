// studio dashboard — read-only cockpit (slice 1). Lists projects with their
// pipeline stage and a detail view. No framework, no build; reads the same
// pipeline.State the CLI's `studio status` prints.

const el = (id) => document.getElementById(id);

init();

async function init() {
  wireTheme();
  el("back").addEventListener("click", showOverview);
  el("new-project").addEventListener("click", toggleNewForm);
  await loadProjects();
}

// Cache of external drives (label + path) for the pickers.
async function getDrives() {
  try {
    return (await (await fetch("/api/drives")).json()) || [];
  } catch (e) {
    return [];
  }
}

// --- New project form ---

async function toggleNewForm() {
  const c = el("newform-container");
  if (c.firstChild) {
    c.innerHTML = "";
    return;
  }
  const drives = await getDrives();
  const form = document.createElement("form");
  form.className = "form";
  form.innerHTML = `
    <label>Slug <input name="slug" placeholder="lake-trip" required></label>
    <label>Title <input name="title" placeholder="A weekend at the lake"></label>
    <label>Date <input name="date" placeholder="today" pattern="\\d{4}-\\d{2}-\\d{2}"></label>
    <label>Destination
      <select name="root">
        <option value="">Internal (default)</option>
        ${drives.map((d) => `<option value="${d.path}">${d.label} — ${d.path}</option>`).join("")}
      </select>
    </label>
    <div class="form-actions">
      <button type="submit" class="btn primary">Create</button>
      <button type="button" class="btn" data-cancel>Cancel</button>
    </div>
    <p class="form-error" hidden></p>`;
  form.querySelector("[data-cancel]").onclick = () => (el("newform-container").innerHTML = "");
  const slug = form.slug;
  slug.addEventListener("input", () => {
    slug.value = slug.value.toLowerCase().replace(/\s+/g, "-").replace(/[^a-z0-9-]/g, "");
  });
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const err = form.querySelector(".form-error");
    err.hidden = true;
    const body = { slug: form.slug.value, title: form.title.value, date: form.date.value, root: form.root.value };
    const res = await fetch("/api/projects", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
    if (!res.ok) {
      err.textContent = await res.text();
      err.hidden = false;
      return;
    }
    const { id } = await res.json();
    el("newform-container").innerHTML = "";
    showDetail(id);
  });
  el("newform-container").appendChild(form);
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
const RUNNABLE = new Set(["apply", "scaffold", "chapters", "qc", "archive"]);
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

  if (step === "ingest") {
    ingestForm(d.id).then((f) => wrap.appendChild(f));
    return wrap;
  }
  if (step === "review") {
    wrap.appendChild(button("Open review", () => openReview(d.id), "primary"));
    return wrap;
  }
  if (step === "thumbs") {
    wrap.appendChild(thumbsPicker(d.id));
    return wrap;
  }
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

// ingestForm builds the source picker for a project awaiting footage.
async function ingestForm(id) {
  const drives = await getDrives();
  const form = document.createElement("form");
  form.className = "form ingest";
  form.innerHTML = `
    <label>Card / source
      <select name="drive">
        <option value="">— pick a drive —</option>
        ${drives.map((d) => `<option value="${d.path}">${d.label} — ${d.path}</option>`).join("")}
      </select>
    </label>
    <label>Path <input name="source" placeholder="/run/media/you/CARD" required></label>
    <label class="check"><input type="checkbox" name="copy" checked> Copy (leave the card intact, safe to eject)</label>
    <label class="check"><input type="checkbox" name="clear"> Empty the card after copying (verified — deletes sources only if the copy checks out)</label>
    <div class="form-actions"><button type="submit" class="btn primary">Ingest</button></div>`;
  form.drive.addEventListener("change", () => {
    if (form.drive.value) form.source.value = form.drive.value;
  });
  // Clearing the card implies a copy.
  form.clear.addEventListener("change", () => {
    if (form.clear.checked) form.copy.checked = true;
  });
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const clearSource = form.clear.checked;
    const copy = form.copy.checked || clearSource;
    document.querySelectorAll(".actions .btn").forEach((b) => (b.disabled = true));
    const log = el("runlog");
    log.hidden = false;
    try {
      const res = await fetch(`/api/projects/${id}/ingest`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ source: form.source.value, copy, move: !copy, clearSource }),
      });
      const reader = res.body.getReader();
      const dec = new TextDecoder();
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        log.textContent += dec.decode(value, { stream: true });
        log.scrollTop = log.scrollHeight;
      }
    } catch (err) {
      log.textContent += "\n[connection error]\n";
    }
    await renderDetail(id); // advances to review once clips land
  });
  return form;
}

// thumbsPicker builds the extract form + candidate gallery. Extraction streams
// into the run log; then the candidates render as clickable tiles, and clicking
// one sets it as the thumbnail (advancing the checklist).
function thumbsPicker(id) {
  const wrap = document.createElement("div");
  wrap.className = "thumbs";

  const form = document.createElement("form");
  form.className = "form ingest";
  form.innerHTML = `
    <label>Sample from
      <select name="from">
        <option value="render">Final render</option>
        <option value="clips">Rating-5 clips</option>
      </select>
    </label>
    <label>Count <input name="count" type="number" min="1" max="24" value="12"></label>
    <div class="form-actions"><button type="submit" class="btn primary">Extract candidates</button></div>`;

  const gallery = document.createElement("div");
  gallery.className = "thumb-gallery";

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const log = el("runlog");
    log.hidden = false;
    log.textContent = "";
    form.querySelector("button").disabled = true;
    try {
      const res = await fetch(`/api/projects/${id}/thumbs`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ from: form.from.value, count: parseInt(form.count.value, 10) || 12 }),
      });
      const reader = res.body.getReader();
      const dec = new TextDecoder();
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        log.textContent += dec.decode(value, { stream: true });
        log.scrollTop = log.scrollHeight;
      }
    } catch (err) {
      log.textContent += "\n[connection error]\n";
    }
    form.querySelector("button").disabled = false;
    loadCandidates(id, gallery);
  });

  wrap.append(form, gallery);
  loadCandidates(id, gallery); // show any candidates from a previous run
  return wrap;
}

async function loadCandidates(id, gallery) {
  let data;
  try {
    data = await (await fetch(`/api/projects/${id}/thumbs/candidates`)).json();
  } catch (e) {
    return;
  }
  gallery.innerHTML = "";
  if (!data.candidates.length) return;

  for (const file of data.candidates) {
    const tile = document.createElement("button");
    tile.type = "button";
    tile.className = "thumb-tile" + (file === data.current ? " current" : "");
    const img = document.createElement("img");
    img.src = `/media/thumb/${id}/${file}`;
    img.alt = file;
    img.loading = "lazy";
    const cap = document.createElement("span");
    cap.className = "thumb-cap";
    cap.textContent = file === data.current ? "✓ selected" : "Use this";
    tile.append(img, cap);
    tile.onclick = () => setThumbnail(id, file, gallery);
    gallery.appendChild(tile);
  }
  if (data.contactSheet) {
    const link = document.createElement("a");
    link.className = "contact-link";
    link.href = `/media/thumb/${id}/${data.contactSheet}`;
    link.target = "_blank";
    link.textContent = "Open contact sheet";
    gallery.appendChild(link);
  }
}

async function setThumbnail(id, file, gallery) {
  try {
    const res = await fetch(`/api/projects/${id}/thumbnail`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ file }),
    });
    if (!res.ok) return toast(await res.text());
  } catch (e) {
    return toast("failed to set thumbnail");
  }
  await loadCandidates(id, gallery); // re-mark the current pick
  renderDetail(id); // thumbs step is now done
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

// openReview embeds the full review UI (served per-project under
// /projects/{id}/review/) in an iframe, so rating/naming/keeping clips happens
// inside the cockpit. "Done reviewing" tears it down and re-renders the detail
// so the checklist reflects the new keep/reject counts.
function openReview(id) {
  const panel = el("review-panel");
  panel.hidden = false;
  panel.innerHTML = "";

  const bar = document.createElement("div");
  bar.className = "review-bar";
  bar.appendChild(button("Done reviewing", () => {
    panel.hidden = true;
    panel.innerHTML = "";
    renderDetail(id);
  }, "primary"));

  const frame = document.createElement("iframe");
  frame.className = "review-frame";
  frame.src = `/projects/${id}/review/`;
  frame.title = "review";

  panel.append(bar, frame);
  panel.scrollIntoView({ block: "start" });
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
