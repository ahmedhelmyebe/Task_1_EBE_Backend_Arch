// ==========================
// QUICK CONFIG (edit these)
// ==========================
const BASE_URL = "http://localhost:8080";
const API_PREFIX = "/api/v1";

const REGISTER_PATH = `${API_PREFIX}/auth/register`;
const LOGIN_PATH    = `${API_PREFIX}/auth/login`;
const TOKEN_RESPONSE_KEY = "token";

const RESOURCE = "users";
const RESOURCE_PATH = `${API_PREFIX}/${RESOURCE}`;
// 👇 match your Go JSON tags
const ITEM_FIELDS = ["name", "email", "password"];

const ID_FIELD = "id";


// ==================================
// STATE, UTILS, FETCH WRAPPER, TOAST
// ==================================
const qs  = (sel, root=document) => root.querySelector(sel);
const qsa = (sel, root=document) => Array.from(root.querySelectorAll(sel));

const state = {
  get token() { return localStorage.getItem("authToken") || ""; },
  set token(v) { v ? localStorage.setItem("authToken", v) : localStorage.removeItem("authToken"); },
};

function setAuthUI() {
  const isAuthed = !!state.token;
  qs("#btn-logout").style.display = isAuthed ? "inline-block" : "none";
  qs("#link-dashboard").style.display = isAuthed ? "inline-block" : "none";
  qs("#link-login").style.display = isAuthed ? "none" : "inline-block";
  qs("#link-register").style.display = isAuthed ? "none" : "inline-block";
}

function renderToast(message, type = "success", ttlMs = 2400) {
  const t = qs("#toast");
  t.textContent = message;
  t.hidden = false;
  t.className = `toast toast--show toast--${type}`;
  window.clearTimeout(renderToast._timer);
  renderToast._timer = setTimeout(() => {
    t.classList.remove("toast--show");
    setTimeout(() => (t.hidden = true), 250);
  }, ttlMs);
}

async function apiFetch(path, { method = "GET", body, auth = true } = {}) {
  const headers = { "Content-Type": "application/json" };
  if (auth && state.token) headers["Authorization"] = `Bearer ${state.token}`;

  const res = await fetch(`${BASE_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });

  const isJSON = res.headers.get("content-type")?.includes("application/json");
  const data = isJSON ? await res.json().catch(() => ({})) : await res.text();

  if (!res.ok) {
    const msg = (isJSON && (data.error || data.message)) || (typeof data === "string" ? data : "Request failed");
    const err = new Error(msg);
    err.status = res.status;
    err.payload = data;
    // Auto-logout on 401
    if (res.status === 401) {
      state.token = "";
      setAuthUI();
      if (location.hash !== "#/login") location.hash = "#/login";
    }
    throw err;
  }
  return data;
}

function requireAuth() {
  if (!state.token) {
    location.hash = "#/login";
    throw new Error("Not authenticated");
  }
}

function disableWhileLoading(el, fn) {
  return async (...args) => {
    const prev = el.textContent;
    el.disabled = true;
    el.textContent = "Working…";
    try { return await fn(...args); }
    finally { el.disabled = false; el.textContent = prev; }
  };
}

function serializeForm(form) {
  const out = {};
  qsa("input, select, textarea", form).forEach((inp) => {
    if (!inp.name) return;
    if (inp.type === "number" && inp.value !== "") out[inp.name] = Number(inp.value);
    else if (inp.type === "checkbox") out[inp.name] = !!inp.checked;
    else if (inp.value !== "") out[inp.name] = inp.value;
  });
  return out;
}

function buildFields(formEl, fields) {
  formEl.innerHTML = "";
  fields.forEach((f) => {
    const wrap = document.createElement("div");
    wrap.className = "field";
    wrap.innerHTML = `
      <label>${f}</label>
      <input name="${f}" type="text" />
    `;
    formEl.appendChild(wrap);
  });
}

// ====================
// RENDERING / ROUTING
// ====================
function showRoute(hash) {
  const views = qsa("section[data-route]");
  views.forEach(v => v.hidden = true);
  const target = qs(`section[data-route="${hash}"]`);
  if (!target) {
    // default route
    if (state.token) return showRoute("#/dashboard");
    return showRoute("#/login");
  }
  target.hidden = false;

  // Small per-view init
  if (hash === "#/dashboard") {
    try {
      requireAuth();
      renderDashboard();
    } catch { /* handled in requireAuth */ }
  }
  if (hash === "#/login") {
    qs("#err-login").textContent = "";
  }
  if (hash === "#/register") {
    qs("#err-register").textContent = "";
  }
}

// ==============
// AUTH HANDLERS
// ==============
async function handleRegister(ev) {
  ev.preventDefault();
  const errEl = qs("#err-register");
  errEl.textContent = "";
  const body = serializeForm(qs("#form-register"));
  try {
    await apiFetch(`${REGISTER_PATH}`, { method: "POST", body, auth: false });
    // Auto-login
    await handleLoginCore(body.email, body.password, false);
    renderToast("Registered & logged in", "success");
    location.hash = "#/dashboard";
  } catch (e) {
    errEl.textContent = e.message;
    renderToast(e.message, "error");
  }
}

async function handleLogin(ev) {
  ev.preventDefault();
  const f = qs("#form-login");
  const body = serializeForm(f);
  const errEl = qs("#err-login");
  errEl.textContent = "";
  try {
    await handleLoginCore(body.email, body.password, true);
    renderToast("Logged in", "success");
    location.hash = "#/dashboard";
  } catch (e) {
    errEl.textContent = e.message;
    renderToast(e.message, "error");
  }
}

async function handleLoginCore(email, password, showErrors) {
  const data = await apiFetch(`${LOGIN_PATH}`, {
    method: "POST",
    body: { email, password },
    auth: false,
  });
  const tok = data?.[TOKEN_RESPONSE_KEY];
  if (!tok) throw new Error("Token missing in response");
  state.token = tok;
  setAuthUI();
}

// ==============
// DASHBOARD UI
// ==============
async function renderDashboard() {
  // Build Create/Update dynamic fields
  buildFields(qs("#form-create"), ITEM_FIELDS);
  buildFields(qs("#form-update"), ITEM_FIELDS);

  // Build table headers
  const thead = qs("#table-list thead");
  thead.innerHTML = `
    <tr>
      <th>${ID_FIELD}</th>
      ${ITEM_FIELDS.map(f => `<th>${f}</th>`).join("")}
      <th>created_at</th>
      <th>updated_at</th>
    </tr>
  `;

  // First load
  await refreshList();
}

async function refreshList() {
  try {
    const list = await apiFetch(`${RESOURCE_PATH}`, { method: "GET" });
    const tbody = qs("#table-list tbody");
    tbody.innerHTML = (Array.isArray(list) ? list : list.items || []).map(row => {
      const id = row[ID_FIELD] ?? "";
      const cols = ITEM_FIELDS.map(f => `<td>${sanitize(row[f])}</td>`).join("");
      const ca = sanitize(row.created_at || "");
      const ua = sanitize(row.updated_at || "");
      return `<tr>
        <td>${sanitize(id)}</td>${cols}<td>${ca}</td><td>${ua}</td>
      </tr>`;
    }).join("") || `<tr><td colspan="${ITEM_FIELDS.length+3}"><em>No data</em></td></tr>`;
  } catch (e) {
    renderToast(e.message, "error");
  }
}

function sanitize(v) {
  if (v === null || v === undefined) return "";
  return String(v).replace(/[&<>"']/g, s => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[s]));
}

// =====================
// CRUD EVENT HANDLERS
// =====================
async function onCreate() {
  requireAuth();
  const body = serializeForm(qs("#form-create"));
  const data = await apiFetch(`${RESOURCE_PATH}`, { method: "POST", body });
  renderToast(`${RESOURCE} created (id=${data?.[ID_FIELD] ?? "?"})`, "success");
  await refreshList();
}

async function onGetById() {
  requireAuth();
  const id = qs("#get-id").value.trim();
  if (!id) return renderToast("Provide an id", "error");
  const data = await apiFetch(`${RESOURCE_PATH}/${encodeURIComponent(id)}`, { method: "GET" });
  qs("#get-json").textContent = JSON.stringify(data, null, 2);
  renderToast(`Fetched ${RESOURCE} ${id}`, "success");
}

async function onUpdate() {
  requireAuth();
  const id = qs("#update-id").value.trim();
  if (!id) return renderToast("Provide an id", "error");
  const body = serializeForm(qs("#form-update"));
  const data = await apiFetch(`${RESOURCE_PATH}/${encodeURIComponent(id)}`, { method: "PUT", body });
  renderToast(`Updated ${RESOURCE} ${data?.[ID_FIELD] ?? id}`, "success");
  await refreshList();
}

async function onDelete() {
  requireAuth();
  const id = qs("#delete-id").value.trim();
  if (!id) return renderToast("Provide an id", "error");
  await apiFetch(`${RESOURCE_PATH}/${encodeURIComponent(id)}`, { method: "DELETE" });
  renderToast(`Deleted ${RESOURCE} ${id}`, "success");
  await refreshList();
}

// ========
// WIRING
// ========
window.addEventListener("DOMContentLoaded", () => {
  setAuthUI();
  // Nav actions
  qs("#btn-logout").addEventListener("click", () => {
    state.token = "";
    setAuthUI();
    location.hash = "#/login";
  });

  // Forms
  qs("#form-register").addEventListener("submit", disableWhileLoading(qs("#btn-register"), handleRegister));
  qs("#form-login").addEventListener("submit",   disableWhileLoading(qs("#btn-login"),    handleLogin));

  // Dashboard buttons
  qs("#btn-create").addEventListener("click", disableWhileLoading(qs("#btn-create"), onCreate));
  qs("#btn-refresh").addEventListener("click", disableWhileLoading(qs("#btn-refresh"), refreshList));
  qs("#btn-get").addEventListener("click", disableWhileLoading(qs("#btn-get"), onGetById));
  qs("#btn-update").addEventListener("click", disableWhileLoading(qs("#btn-update"), onUpdate));
  qs("#btn-delete").addEventListener("click", disableWhileLoading(qs("#btn-delete"), onDelete));

  // Hash router
  window.addEventListener("hashchange", () => showRoute(location.hash));
  if (!location.hash) location.hash = state.token ? "#/dashboard" : "#/login";
  showRoute(location.hash);
});

// ============================
// DEV CORS BYPASS (Optional):
// ============================
// If you face CORS while opening from file:// or a different port, you can run a quick proxy
// and point BASE_URL to it (e.g., http://localhost:5050). Two tiny options:
//
// 1) Node (Express):
//   npm i express http-proxy-middleware
//   -- proxy.js --
//   const express = require('express');
//   const { createProxyMiddleware } = require('http-proxy-middleware');
//   const app = express();
//   app.use('/', createProxyMiddleware({ target: 'http://localhost:8080', changeOrigin: true }));
//   app.listen(5050);
//   // Then set BASE_URL = "http://localhost:5050"
//
// 2) Go net/http reverse proxy:
//   package main
//   import ( "log"; "net/http"; "net/http/httputil"; "net/url" )
//   func main() {
//     u,_ := url.Parse("http://localhost:8080")
//     p := httputil.NewSingleHostReverseProxy(u)
//     log.Fatal(http.ListenAndServe(":5050", p))
//   }
//   // Then set BASE_URL = "http://localhost:5050"
