// Example data, the detail panel and the page bar, shared by all demo pages.
const ideas = {
  // still in the garden of frogly
  search: { bed: "search", text: "Searching takes forever on big projects", by: "Planted by Ana on the web, 3 weeks ago.", people: 4,
    metoo: [["search is really slow once there are a few hundred tickets", "Lea · email"], ["search lags once there are lots of tickets", "Tomás · web"], ["searching FROG takes 4 seconds", "Karo via Claude · MCP"]] },
  dropped: { bed: "search", text: "Search in dropped ideas too", by: "Planted by Karo via Claude through MCP, 2 weeks ago.", people: 2, metoo: [["also search the compost please", "Lea · web"]] },
  index: { bed: "search", text: "Index ticket titles for search", by: "Planted by Tomás on the web. Faded in March and came back last week, when Lea planted something similar.", people: 2, metoo: [["titles should be searchable", "Lea · web"]] },
  bytheidea: { bed: "search", text: "Find tickets by the idea they came from", by: "Planted by Lea on the web, a month ago. Fading.", people: 1, metoo: [] },
  enter: { bed: null, text: "Pressing Enter twice should keep the box open", by: "Planted by Tomás on the web, yesterday.", people: 2, metoo: [["keep the plant box open after Enter", "Karo · web"]] },
  shortcut: { bed: null, text: "A shortcut to plant from anywhere", by: "Planted by Lea on the web, last week.", people: 1, metoo: [] },
  dark: { bed: null, text: "Dark mode for the board", by: "Planted by Lea on the web. Fading.", people: 1, metoo: [] },
  json: { bed: null, text: "Export a project as JSON", by: "Planted by Karo from the CLI. Almost in the compost.", people: 1, metoo: [] },
  // harvested into tickets of frogly
  email: { bed: null, text: "Let people plant ideas by email", by: "Planted by Ana on the web, a month ago. Harvested into FROG-15.", people: 3, metoo: [["could I just mail my ideas in?", "Lea · email"], ["an address per project", "Tomás · web"]] },
  digest: { bed: null, text: "A weekly message instead of notifications", by: "Planted by Karo via Claude through MCP. Harvested into FROG-12.", people: 1, metoo: [] },
  ripeness: { bed: null, text: "Show how many people want an idea", by: "Planted by Lea on the web. Harvested into FROG-14.", people: 3, metoo: [["which ideas do people actually want?", "Ana · web"], ["count the me toos", "Karo · CLI"]] },
  suggest: { bed: null, text: "Suggest \"me too\" when ideas look alike", by: "Planted by Ana on the web. Harvested into FROG-13.", people: 2, metoo: [["we keep planting the same idea twice", "Tomás · web"]] },
  fade: { bed: null, text: "Old ideas should fade instead of piling up", by: "Planted by Tomás on the web. Harvested into FROG-11.", people: 2, metoo: [["the garden is getting crowded", "Karo · web"]] },
  findable: { bed: null, text: "Dropped ideas should stay findable", by: "Planted by Tomás via Claude through MCP. Harvested into FROG-10.", people: 2, metoo: [["didn't we decide this already?", "Lea · web"]] },
  // the Summer fair
  volunteers: { bed: null, text: "Not enough people for the evening shifts", by: "Planted by Lea by email, last month. Harvested into SUM-3.", people: 3, metoo: [["evening shifts were empty again", "Tomás · form"], ["can we ask the school parents?", "Ana · web"]] },
  hall: { bed: null, text: "Book the hall early this year", by: "Planted by Ana on the web, in August. Harvested into SUM-1.", people: 2, metoo: [["remember the hall!", "Karo · web"]] },
  parking: { bed: null, text: "Ask the school if we can use their parking", by: "Planted by Tomás through the form, last week.", people: 2, metoo: [["where do visitors park?", "Lea · email"]] },
  // in the compost
  gantt: { bed: null, text: "Gantt charts", by: "Planted by Lea on the web.", people: 1, compost: "dropped by Ana: \"a board with dates is enough for us\"", metoo: [] },
  points: { bed: null, text: "Story points on tickets", by: "Planted by Tomás on the web.", people: 2, compost: "dropped by Karo: \"a field has to earn its place\"", metoo: [["estimates would help planning", "Lea · email"]] },
  slack: { bed: null, text: "Post every event to the team chat", by: "Planted by Lea via Claude through MCP.", people: 1, compost: "faded in September, nobody came back to it", metoo: [] },
};
const tickets = {
  "SUM-3": { col: "doing · Karo · Summer fair · due Friday", title: "Draft the call for volunteers", body: "One short text for the newsletter and the notice board: what we need, when, and who to ask.",
    check: [[1, "List the shifts"], [0, "Write the text"], [0, "Ask Lea to read it"]], from: ["volunteers"], acts: ["move to done", "put on hold", "hand to someone"] },
  "SUM-1": { col: "ready · Summer fair · due in 9 days", title: "Book the hall", body: "The town hall for the fair in November, with chairs for 80.",
    check: [[0, "Call the town hall"], [0, "Pay the deposit"]], from: ["hall"], acts: ["take it", "hand to someone"] },
  "FROG-15": { col: "ready · next in line", title: "Plant an idea by email", body: "Every project gets an email address. A mail to it plants an idea, with the sender as the author.",
    check: [[0, "One address per project"], [0, "Plant the subject line as the idea"], [0, "Keep the mail as the origin"]], from: ["email"], acts: ["take it", "due date", "put on hold"] },
  "FROG-12": { col: "ready · second in line", title: "Weekly digest to the team chat", body: "Once a week, one calm message: what shipped, the ripest ideas, how work flowed, what went to the compost.",
    check: [[0, "Pick the chat"], [0, "Write the message"]], from: ["digest"], acts: ["take it", "put on hold"] },
  "FROG-14": { col: "doing · Karo, since Monday", title: "Show ripeness on ideas", body: "Each idea shows how many different people want it, as ripples. Agents count as their person.",
    check: [[1, "Count distinct people per idea"], [1, "Ripples in the garden"], [0, "Same ripples on the ticket"]], from: ["ripeness"], acts: ["move to review", "put on hold", "hand to someone"] },
  "FROG-13": { col: "doing · Karo via Claude · on hold", title: "Suggest \"me too\" with a local model", body: "On hold: which small model runs well on a Raspberry Pi?",
    check: [[1, "Try two small models"], [0, "Measure on a Pi"]], from: ["suggest"], acts: ["resume", "hand to someone"] },
  "FROG-11": { col: "review · Ana, waiting for Karo", title: "Fade ideas by project activity", body: "Ideas fade as the project moves on, not by the clock. A quiet project keeps its garden as it is.",
    check: [[1, "Fade on activity"], [1, "Compost when fully faded"]], from: ["fade"], acts: ["approve and move to done", "send back to doing"] },
  "FROG-10": { col: "review · Tomás via Claude, waiting for Ana", title: "Compost keeps dropped ideas searchable", body: "Dropped and faded ideas stay searchable and come back when someone plants something similar.",
    check: [[1, "Search the compost"], [1, "Suggest composted ideas while planting"]], from: ["findable"], acts: ["approve and move to done", "send back to doing"] },
};
const count = (n) => `${n} ${n === 1 ? "person wants" : "people want"} this`;
const ripples = (n, size) => `<span class="rip" role="img" aria-label="${count(n)}" style="--rs:${size || 30}px">${"<i></i>".repeat(Math.max(1, Math.min(4, n)))}</span>`;

// The panel is a modal dialog. While it's open, the page behind it is inert,
// Tab stays inside, and closing it gives focus back to what opened it.
const panel = document.createElement("aside");
panel.className = "panel";
panel.setAttribute("role", "dialog");
panel.setAttribute("aria-modal", "true");
panel.setAttribute("aria-labelledby", "panel-title");
document.body.appendChild(panel);
let opener = null;
const trail = [];
const behind = () => [...document.body.children].filter((el) => el !== panel && el.tagName !== "SCRIPT");
const show = (html) => {
  if (!panel.classList.contains("open")) opener = document.activeElement;
  const back = trail.length > 1 ? `<button class="btn back">back</button>` : "";
  panel.innerHTML = `<div class="panel-bar">${back}<button class="btn close">close</button></div>${html}`;
  panel.querySelector(".close").onclick = close;
  const b = panel.querySelector(".back");
  if (b) b.onclick = () => { trail.pop(); const prev = trail.pop(); prev(); };
  behind().forEach((el) => (el.inert = true));
  panel.classList.add("open");
  panel.querySelector("h1").focus();
};
const close = () => {
  if (!panel.classList.contains("open")) return;
  panel.classList.remove("open");
  trail.length = 0;
  behind().forEach((el) => (el.inert = false));
  if (opener && opener.isConnected) opener.focus();
};
const openIdea = (id, fromPanel) => {
  const d = ideas[id];
  if (!fromPanel) trail.length = 0;
  trail.push(() => openIdea(id, true));
  show(`<p class="label">idea${d.bed ? " · bed: " + d.bed : ""}</p><h1 id="panel-title" tabindex="-1">${d.text}</h1>
    <p class="quiet" style="display:flex;gap:12px;align-items:center">${ripples(d.people)} <span>${count(d.people)}</span></p>
    <p class="quiet">${d.by}</p>
    ${d.compost ? `<p class="quiet" style="color:var(--wait)">In the compost: ${d.compost}</p>` : ""}
    ${d.metoo.length ? `<h2 class="label" style="margin-top:26px">Me too</h2><ul class="plain">` + d.metoo.map(([t, m]) => `<li class="idea"><p>"${t}"</p><span class="meta">${m}</span></li>`).join("") + `</ul>` : ""}
    ${d.by.includes("Harvested into") ? "" : `<div class="acts">${d.compost ? `<button class="btn pond">bring it back to the garden</button>` : `<button class="btn pond">harvest into a ticket</button><button class="btn">move to a bed</button><button class="btn">drop</button>`}</div>`}`);
};
const openTicket = (key) => {
  const t = tickets[key];
  trail.length = 0;
  trail.push(() => openTicket(key));
  show(`<p class="mono faint">${key} · ${t.col}</p><h1 id="panel-title" tabindex="-1">${t.title}</h1><p class="quiet">${t.body}</p>
    <h2 class="label" style="margin-top:24px">Checklist</h2>
    <ul class="plain check">${t.check.map(([done, s]) => `<li><span aria-hidden="true">${done ? "☑" : "☐"}</span> <span class="vh">${done ? "done:" : "to do:"}</span> ${s}</li>`).join("")}</ul>
    <h2 class="label" style="margin-top:24px">Harvested from</h2>
    <ul class="plain">${t.from.map((id) => `<li><button class="idea from" data-idea="${id}">${ripples(ideas[id].people, 24)}<span><span class="t">${ideas[id].text}</span><span class="meta">${ideas[id].people} ${ideas[id].people === 1 ? "person" : "people"}</span></span></button></li>`).join("")}</ul>
    <div class="acts">${t.acts.map((a, i) => `<button class="btn${i ? "" : " pond"}">${a}</button>`).join("")}</div>`);
};

// Ideas and tickets on the page open with a click, Enter or Space.
document.querySelectorAll("[data-rip]").forEach((el) => el.insertAdjacentHTML("afterbegin", ripples(+el.dataset.rip, +el.dataset.size || 30)));
document.querySelectorAll("[data-idea], [data-ticket]").forEach((el) => {
  if (el.tagName === "BUTTON" || el.tagName === "A") return;
  el.tabIndex = 0;
  el.setAttribute("role", "button");
});
const activate = (target) => {
  const i = target.closest("[data-idea]"), t = target.closest("[data-ticket]");
  if (i) openIdea(i.dataset.idea, panel.contains(i));
  else if (t) openTicket(t.dataset.ticket);
};
document.addEventListener("click", (e) => activate(e.target));
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && panel.classList.contains("open")) { e.preventDefault(); close(); return; }
  if ((e.key === "Enter" || e.key === " ") && e.target.matches?.('[role="button"]')) { e.preventDefault(); activate(e.target); return; }
  if (e.key === "Tab" && panel.classList.contains("open")) {
    const f = [...panel.querySelectorAll("button, [href], [tabindex]:not([tabindex='-1'])")];
    if (!f.length) return;
    if (e.shiftKey && (document.activeElement === f[0] || document.activeElement === panel.querySelector("h1"))) { e.preventDefault(); f[f.length - 1].focus(); }
    else if (!e.shiftKey && document.activeElement === f[f.length - 1]) { e.preventDefault(); f[0].focus(); }
  }
});
document.querySelectorAll(".top nav a.on").forEach((a) => a.setAttribute("aria-current", "page"));

// light or dark, remembered across the demo pages
const root = document.documentElement;
try { const t = localStorage.getItem("frogly-theme"); if (t === "light" || t === "dark") root.dataset.theme = t; } catch (e) {}
const isDark = () => root.dataset.theme ? root.dataset.theme === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;

// the pager cycles through the pages
const pages = [["today.html", "today"], ["garden.html", "garden"], ["board.html", "board"], ["compost.html", "compost"], ["digest.html", "digest"]];
const here = location.pathname.split("/").pop();
const n = Math.max(0, pages.findIndex(([f]) => f === here));
const other = pages[(n + 1) % pages.length][0];
const bar = document.createElement("div");
bar.className = "pager";
bar.innerHTML = `<span class="what">${n + 1} / ${pages.length} · ${pages[n][1]} · click an idea or a ticket</span>` +
  `<button class="btn" id="theme" aria-live="polite"></button><a class="btn pond" href="${other}">next →</a>`;
document.body.appendChild(bar);
const themeBtn = bar.querySelector("#theme");
const label = () => (themeBtn.textContent = isDark() ? "switch to light" : "switch to dark");
label();
matchMedia("(prefers-color-scheme: dark)").addEventListener("change", label);
themeBtn.onclick = () => {
  root.dataset.theme = isDark() ? "light" : "dark";
  try { localStorage.setItem("frogly-theme", root.dataset.theme); } catch (e) {}
  label();
};
