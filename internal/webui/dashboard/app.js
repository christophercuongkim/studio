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

async function showDetail(id) {
  let d;
  try {
    d = await (await fetch(`/api/projects/${id}`)).json();
  } catch (e) {
    return toast("failed to load project");
  }
  el("detail-title").textContent = d.title;
  el("detail-body").innerHTML = "";
  el("detail-body").append(checklist(d), metaPanels(d));

  el("overview").hidden = true;
  el("detail").hidden = false;
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
