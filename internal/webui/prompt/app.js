// studio teleprompter — server-authoritative state, polled once a second so a
// laptop and an iPad stay in sync. Keys and tap zones POST actions; rendering
// is driven entirely by /api/state (plan §10.2).

const el = (id) => document.getElementById(id);
let sections = [];
let st = { section: 0, bullet: 0, take: 1, blackout: false };
let lastTake = 1;
let lastSection = -1;
let sectionStart = Date.now();
const startTime = Date.now();

init();

async function init() {
  const sc = await (await fetch("/api/script")).json();
  sections = sc.sections || [];
  document.title = sc.title ? `prompter — ${sc.title}` : "prompter";
  await poll();
  setInterval(poll, 1000);
  setInterval(tickClocks, 250);

  document.addEventListener("keydown", onKey);
  el("tapback").addEventListener("click", () => act("back"));
  el("tapfwd").addEventListener("click", () => act("next"));
}

async function act(a) {
  st = await (await fetch(`/api/action?a=${a}`, { method: "POST" })).json();
  render();
}

async function poll() {
  try {
    st = await (await fetch("/api/state")).json();
    render();
  } catch (e) { /* keep last state on a blip */ }
}

function onKey(e) {
  switch (e.key) {
    case "ArrowRight": case " ": e.preventDefault(); act("next"); break;
    case "ArrowLeft": act("back"); break;
    case "n": act("nextSection"); break;
    case "p": act("prevSection"); break;
    case "t": act("take"); break;
    case ".": act("blackout"); break;
  }
}

function render() {
  el("blackout").hidden = !st.blackout;

  // Take change → flash the bar and reset the section timer.
  if (st.take !== lastTake) {
    lastTake = st.take;
    sectionStart = Date.now();
    const bar = el("bar");
    bar.classList.remove("flash");
    void bar.offsetWidth; // restart animation
    bar.classList.add("flash");
  }
  if (st.section !== lastSection) {
    lastSection = st.section;
    sectionStart = Date.now();
  }

  el("take").textContent = st.take;

  const sec = sections[st.section];
  if (!sec) return;
  el("section").textContent = sec.title;

  const ul = el("bullets");
  ul.innerHTML = "";
  (sec.bullets || []).forEach((b, i) => {
    const li = document.createElement("li");
    li.className = i === st.bullet ? "current" : (i < st.bullet ? "done" : "");
    li.textContent = b.text;
    if (b.children && b.children.length) {
      const sub = document.createElement("ul");
      for (const c of b.children) {
        const cli = document.createElement("li");
        cli.textContent = c;
        sub.appendChild(cli);
      }
      li.appendChild(sub);
    }
    ul.appendChild(li);
  });

  const nextSec = sections[st.section + 1];
  el("nextsection").textContent = nextSec ? nextSec.title : "— end —";

  const pct = sections.length ? ((st.section + 1) / sections.length) * 100 : 0;
  el("progressfill").style.width = pct + "%";
}

function tickClocks() {
  el("clock").textContent = fmt(Date.now() - startTime);
  el("sectiontimer").textContent = fmt(Date.now() - sectionStart);
}

function fmt(ms) {
  const s = Math.floor(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}
