// Minimal vanilla-JS frontend for paper-inator. It talks to the REST API and
// renders feeds and publications. No build step, no framework.

const api = {
  async get(path) {
    const res = await fetch(path);
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || res.statusText);
    return res.json();
  },
  async send(method, path, body) {
    const res = await fetch(path, {
      method,
      headers: { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || res.statusText);
    return res.status === 204 ? null : res.json();
  },
};

const el = (id) => document.getElementById(id);

function fmtDate(value) {
  if (!value) return "";
  const d = new Date(value);
  return isNaN(d) ? "" : d.toLocaleDateString();
}

async function loadFeeds() {
  const feeds = await api.get("/api/feeds");
  const list = el("feed-list");
  const select = el("filter-feed");

  list.innerHTML = "";
  // Reset the feed filter, keeping the "All feeds" option.
  select.length = 1;

  if (feeds.length === 0) {
    list.innerHTML = '<li class="empty">No feeds yet. Add one above.</li>';
  }

  for (const f of feeds) {
    const li = document.createElement("li");
    const row = document.createElement("div");
    row.className = "feed-row";

    const info = document.createElement("div");
    info.innerHTML = `<div>${escapeHTML(f.name)}</div>` +
      `<div class="feed-meta">${escapeHTML(f.url)}` +
      (f.last_fetched_at ? ` · last fetched ${fmtDate(f.last_fetched_at)}` : " · not fetched yet") +
      `</div>`;

    const del = document.createElement("button");
    del.className = "link";
    del.textContent = "Delete";
    del.onclick = async () => {
      if (!confirm(`Delete feed "${f.name}"? Its publications will be removed.`)) return;
      await api.send("DELETE", `/api/feeds/${f.id}`);
      await refresh();
    };

    row.append(info, del);
    li.append(row);
    list.append(li);

    const opt = document.createElement("option");
    opt.value = f.id;
    opt.textContent = f.name;
    select.append(opt);
  }
}

async function loadPublications() {
  const params = new URLSearchParams();
  const feedID = el("filter-feed").value;
  const q = el("filter-q").value.trim();
  const sort = el("filter-sort").value;
  if (feedID) params.set("feed_id", feedID);
  if (q) params.set("q", q);
  if (sort) params.set("sort", sort);

  const pubs = await api.get("/api/publications?" + params.toString());
  const list = el("publication-list");
  list.innerHTML = "";

  if (pubs.length === 0) {
    list.innerHTML = '<li class="empty">No publications yet.</li>';
    return;
  }

  for (const p of pubs) {
    const li = document.createElement("li");
    const title = p.link
      ? `<a href="${escapeAttr(p.link)}" target="_blank" rel="noopener">${escapeHTML(p.title)}</a>`
      : escapeHTML(p.title);
    li.innerHTML =
      `<div class="pub-title">${title}</div>` +
      `<div class="pub-meta">${escapeHTML(p.authors || "Unknown authors")}` +
      (p.published_at ? ` · ${fmtDate(p.published_at)}` : "") + `</div>`;
    list.append(li);
  }
}

async function refresh() {
  await loadFeeds();
  await loadPublications();
}

function escapeHTML(s) {
  return String(s ?? "").replace(/[&<>"']/g, (c) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}
function escapeAttr(s) { return escapeHTML(s); }

el("add-feed-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const err = el("feed-error");
  err.hidden = true;
  try {
    await api.send("POST", "/api/feeds", {
      name: el("feed-name").value.trim(),
      url: el("feed-url").value.trim(),
      enabled: true,
      fetch_interval_sec: 0,
    });
    el("feed-name").value = "";
    el("feed-url").value = "";
    await refresh();
  } catch (ex) {
    err.textContent = ex.message;
    err.hidden = false;
  }
});

["filter-feed", "filter-sort"].forEach((id) => el(id).addEventListener("change", loadPublications));
el("filter-q").addEventListener("input", debounce(loadPublications, 250));

function debounce(fn, ms) {
  let t;
  return (...args) => { clearTimeout(t); t = setTimeout(() => fn(...args), ms); };
}

refresh().catch((e) => console.error(e));
