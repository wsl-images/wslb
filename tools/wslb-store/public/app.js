const els = {
  console: document.getElementById("console"),
  profileList: document.getElementById("profileList"),
  profileFilter: document.getElementById("profileFilter"),
  btnCreateProfile: document.getElementById("btnCreateProfile"),
  btnSaveProfile: document.getElementById("btnSaveProfile"),
  btnDeleteProfile: document.getElementById("btnDeleteProfile"),
  btnApplyManifest: document.getElementById("btnApplyManifest"),
  btnDoctor: document.getElementById("btnDoctor"),
  btnFeatureRefresh: document.getElementById("btnFeatureRefresh"),
  btnIconRefresh: document.getElementById("btnIconRefresh"),
  featureSearch: document.getElementById("featureSearch"),
  featureSearchResults: document.getElementById("featureSearchResults"),
  iconSearch: document.getElementById("iconSearch"),
  iconSearchResults: document.getElementById("iconSearchResults"),
  iconPreview: document.getElementById("iconPreview"),
  selectedFeatures: document.getElementById("selectedFeatures"),
  rollbackTarget: document.getElementById("rollbackTarget"),
  engine: document.getElementById("engine"),
  nonInteractive: document.getElementById("nonInteractive"),
  fallbackWindowsDir: document.getElementById("fallbackWindowsDir"),
  btnSaveDefaults: document.getElementById("btnSaveDefaults"),

  profileName: document.getElementById("profileName"),
  profileManifestPath: document.getElementById("profileManifestPath"),
  manifestName: document.getElementById("manifestName"),
  manifestImage: document.getElementById("manifestImage"),
  manifestRemoteUser: document.getElementById("manifestRemoteUser"),
  manifestImageId: document.getElementById("manifestImageId"),
  manifestDistroName: document.getElementById("manifestDistroName"),
  manifestStateMode: document.getElementById("manifestStateMode"),
  manifestManaged: document.getElementById("manifestManaged"),
  manifestSystemd: document.getElementById("manifestSystemd"),
  manifestIconSimple: document.getElementById("manifestIconSimple"),
  manifestIconColor: document.getElementById("manifestIconColor"),
  manifestIconColorPicker: document.getElementById("manifestIconColorPicker"),
  manifestIconStyle: document.getElementById("manifestIconStyle"),
  manifestTerminalTheme: document.getElementById("manifestTerminalTheme"),
  manifestTerminalFont: document.getElementById("manifestTerminalFont"),
  manifestTerminalOpacity: document.getElementById("manifestTerminalOpacity"),
  manifestTerminalUseAcrylic: document.getElementById("manifestTerminalUseAcrylic"),
  manifestTerminalCursorShape: document.getElementById("manifestTerminalCursorShape"),

  defBaseImage: document.getElementById("defBaseImage"),
  defRemoteUser: document.getElementById("defRemoteUser"),
  defStateMode: document.getElementById("defStateMode"),
  defManaged: document.getElementById("defManaged"),
  defSystemd: document.getElementById("defSystemd"),
  defIconSimple: document.getElementById("defIconSimple"),
  defIconColor: document.getElementById("defIconColor"),
  defIconColorPicker: document.getElementById("defIconColorPicker"),
  defIconStyle: document.getElementById("defIconStyle"),
  defTerminalTheme: document.getElementById("defTerminalTheme"),
  defTerminalFont: document.getElementById("defTerminalFont"),
  defTerminalOpacity: document.getElementById("defTerminalOpacity"),
  defTerminalUseAcrylic: document.getElementById("defTerminalUseAcrylic"),
  defTerminalCursorShape: document.getElementById("defTerminalCursorShape"),
};

const state = {
  profiles: [],
  activeId: null,
  activeProfile: null,
  featureItems: [],
  iconItems: [],
  defaults: null,
};

function ts() {
  return new Date().toISOString();
}

function deepClone(value) {
  return JSON.parse(JSON.stringify(value));
}

function boolFromSelect(input) {
  return String(input.value) === "true";
}

function slugify(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "") || "profile";
}

function setBoolSelect(input, value) {
  input.value = value === false ? "false" : "true";
}

function normalizeHexColor(raw, fallback) {
  const val = String(raw || "").trim().replace(/^#/, "").toUpperCase();
  if (/^[0-9A-F]{6}$/.test(val)) return val;
  return String(fallback || "E95420");
}

function syncHexInputs(textEl, pickerEl, fallback) {
  const normalized = normalizeHexColor(textEl.value, fallback);
  textEl.value = normalized;
  pickerEl.value = `#${normalized}`;
}

function readTerminalTemplateProfile(template) {
  if (!template || typeof template !== "object" || Array.isArray(template)) {
    return {};
  }
  let profile = template;
  if (Array.isArray(template.profiles) && template.profiles.length > 0 && typeof template.profiles[0] === "object") {
    profile = template.profiles[0] || {};
  }
  const opacityNum = Number(profile.opacity);
  return {
    colorScheme: String(profile.colorScheme || "").trim(),
    fontFace: String(profile?.font?.face || profile.fontFace || "").trim(),
    opacity: Number.isFinite(opacityNum) ? Math.max(30, Math.min(100, Math.round(opacityNum))) : "",
    useAcrylic: typeof profile.useAcrylic === "boolean" ? profile.useAcrylic : null,
    cursorShape: String(profile.cursorShape || "").trim(),
  };
}

function buildTerminalTemplate(theme, fontFace, opacityInput, useAcrylic, cursorShape) {
  const themeValue = String(theme || "").trim();
  const fontValue = String(fontFace || "").trim();
  const opacityNum = Number(opacityInput);
  const cursorValue = String(cursorShape || "").trim();

  const profile = {};
  if (themeValue) profile.colorScheme = themeValue;
  if (fontValue) profile.font = { face: fontValue };
  if (Number.isFinite(opacityNum)) {
    profile.opacity = Math.max(30, Math.min(100, Math.round(opacityNum)));
  }
  if (typeof useAcrylic === "boolean") profile.useAcrylic = useAcrylic;
  if (cursorValue) profile.cursorShape = cursorValue;
  if (Object.keys(profile).length === 0) return undefined;

  return {
    profiles: [profile],
  };
}

function renderIconPreview() {
  const slug = String(els.manifestIconSimple.value || "").trim().toLowerCase();
  if (!slug) {
    els.iconPreview.innerHTML = "<span class='hint'>Select a Simple Icon slug.</span>";
    return;
  }
  const color = normalizeHexColor(els.manifestIconColor.value, "E95420");
  const encodedURL = `url(\"https://simpleicons.org/icons/${encodeURIComponent(slug)}.svg\")`;
  els.iconPreview.innerHTML = `
    <span class="icon-mask large" style="--icon-url:${encodedURL};--icon-color:#${color};"></span>
    <div>
      <strong>${slug}</strong>
      <div class="hint">simpleIcon: ${slug} · color: #${color}</div>
    </div>
  `;
}

function appendConsole(message, data) {
  const chunk = [`[${ts()}] ${message}`];
  if (typeof data !== "undefined") {
    chunk.push(typeof data === "string" ? data : JSON.stringify(data, null, 2));
  }
  els.console.textContent += chunk.join("\n") + "\n";
  els.console.scrollTop = els.console.scrollHeight;
}

async function api(method, endpoint, body) {
  const res = await fetch(endpoint, {
    method,
    headers: {
      "Content-Type": "application/json",
    },
    body: typeof body === "undefined" ? undefined : JSON.stringify(body),
  });
  const payload = await res.json().catch(() => ({}));
  if (!res.ok) {
    const err = payload.error || payload.result?.stderr || `HTTP ${res.status}`;
    throw new Error(err);
  }
  return payload;
}

function ensureManifestShape(profile) {
  profile.manifest = profile.manifest || {};
  profile.manifest.features = profile.manifest.features || {};
  profile.manifest.wslb = profile.manifest.wslb || {};
  profile.manifest.wslb.state = profile.manifest.wslb.state || {};
  profile.manifest.wslb.wslconf = profile.manifest.wslb.wslconf || {};
  profile.manifest.wslb.wslconf.boot = profile.manifest.wslb.wslconf.boot || {};
  profile.manifest.wslb.distribution = profile.manifest.wslb.distribution || {};
  profile.manifest.wslb.distribution.shortcut = profile.manifest.wslb.distribution.shortcut || {};
  profile.manifest.wslb.distribution.shortcut.icon = profile.manifest.wslb.distribution.shortcut.icon || {};
  profile.manifest.wslb.distribution.windowsterminal = profile.manifest.wslb.distribution.windowsterminal || {};
}

function renderProfiles() {
  const q = String(els.profileFilter.value || "").toLowerCase().trim();
  const filtered = state.profiles.filter((p) => {
    const hay = `${p.name} ${p.imageId} ${p.distroName} ${p.baseImage}`.toLowerCase();
    return hay.includes(q);
  });

  els.profileList.innerHTML = "";
  if (filtered.length === 0) {
    const empty = document.createElement("div");
    empty.className = "profile-item";
    empty.innerHTML = "<p>No profiles yet.</p>";
    els.profileList.appendChild(empty);
    return;
  }

  for (const profile of filtered) {
    const item = document.createElement("button");
    item.type = "button";
    item.className = `profile-item ${profile.id === state.activeId ? "active" : ""}`;
    item.innerHTML = `
      <h3>${profile.name}</h3>
      <p>${profile.distroName || "distro"} · ${profile.baseImage || ""}</p>
      <p class="mono">${profile.id}</p>
    `;
    item.addEventListener("click", () => {
      void loadProfile(profile.id);
    });
    els.profileList.appendChild(item);
  }
}

function profileToForm(profile) {
  ensureManifestShape(profile);
  const m = profile.manifest;
  const w = m.wslb;
  const icon = w.distribution.shortcut.icon;
  const wtTemplate = readTerminalTemplateProfile(w.distribution.windowsterminal.template);

  els.profileName.value = profile.name || "";
  els.profileManifestPath.value = profile.manifestPath || "";
  els.manifestName.value = m.name || "";
  els.manifestImage.value = m.image || "";
  els.manifestRemoteUser.value = m.remoteUser || "";
  els.manifestImageId.value = w.id || w.imageId || "";
  els.manifestDistroName.value = w.distroName || "";
  els.manifestStateMode.value = w.state.mode || "windows-dir";
  setBoolSelect(els.manifestManaged, w.managed !== false);
  setBoolSelect(els.manifestSystemd, w.wslconf.boot.systemd !== false);
  els.manifestIconSimple.value = icon.simpleIcon || "ubuntu";
  els.manifestIconColor.value = normalizeHexColor(icon.color || "E95420", "E95420");
  syncHexInputs(els.manifestIconColor, els.manifestIconColorPicker, "E95420");
  els.manifestIconStyle.value = icon.style || "flat";
  els.manifestTerminalTheme.value = wtTemplate.colorScheme || "Campbell";
  els.manifestTerminalFont.value = wtTemplate.fontFace || "Cascadia Mono";
  els.manifestTerminalOpacity.value = String(wtTemplate.opacity || 100);
  setBoolSelect(els.manifestTerminalUseAcrylic, wtTemplate.useAcrylic === true);
  els.manifestTerminalCursorShape.value = wtTemplate.cursorShape || "bar";

  renderSelectedFeatures(m.features || {});
  renderIconPreview();
  renderIconResults(state.iconItems);
}

function selectedFeaturesFromEditor() {
  const out = {};
  const blocks = els.selectedFeatures.querySelectorAll(".selected-item");
  for (const block of blocks) {
    const ref = block.getAttribute("data-ref");
    const ta = block.querySelector("textarea");
    const text = String(ta.value || "").trim();
    if (!text) {
      out[ref] = {};
      ta.style.borderColor = "#c8d0dc";
      continue;
    }
    try {
      const parsed = JSON.parse(text);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        out[ref] = parsed;
      } else if (typeof parsed === "boolean") {
        out[ref] = parsed;
      } else {
        out[ref] = {};
      }
      ta.style.borderColor = "#c8d0dc";
    } catch {
      ta.style.borderColor = "#b42318";
      throw new Error(`Invalid JSON options for feature ${ref}`);
    }
  }
  return out;
}

function formToProfile() {
  if (!state.activeProfile) throw new Error("no active profile");
  const next = deepClone(state.activeProfile);
  ensureManifestShape(next);
  const m = next.manifest;
  const w = m.wslb;
  const icon = w.distribution.shortcut.icon;

  next.name = els.profileName.value.trim();
  next.manifestPath = els.profileManifestPath.value.trim();

  m.name = els.manifestName.value.trim();
  m.image = els.manifestImage.value.trim();
  m.remoteUser = els.manifestRemoteUser.value.trim();

  w.id = els.manifestImageId.value.trim();
  delete w.imageId;
  w.distroName = els.manifestDistroName.value.trim();
  w.managed = boolFromSelect(els.manifestManaged);
  w.state.mode = els.manifestStateMode.value;
  w.wslconf.boot.systemd = boolFromSelect(els.manifestSystemd);
  icon.simpleIcon = els.manifestIconSimple.value.trim();
  icon.color = normalizeHexColor(els.manifestIconColor.value.trim(), "E95420");
  icon.style = els.manifestIconStyle.value;
  w.distribution.shortcut.enabled = true;
  w.distribution.windowsterminal.enabled = true;
  const terminalTemplate = buildTerminalTemplate(
    els.manifestTerminalTheme.value,
    els.manifestTerminalFont.value,
    els.manifestTerminalOpacity.value,
    boolFromSelect(els.manifestTerminalUseAcrylic),
    els.manifestTerminalCursorShape.value,
  );
  if (terminalTemplate) {
    w.distribution.windowsterminal.template = terminalTemplate;
  } else {
    delete w.distribution.windowsterminal.template;
  }

  m.features = selectedFeaturesFromEditor();
  return next;
}

function renderSelectedFeatures(features) {
  const refs = Object.keys(features || {}).sort();
  els.selectedFeatures.innerHTML = "";
  if (refs.length === 0) {
    els.selectedFeatures.innerHTML = "<div class='hint'>No features selected.</div>";
    return;
  }
  for (const ref of refs) {
    const options = features[ref];
    const wrap = document.createElement("div");
    wrap.className = "selected-item";
    wrap.setAttribute("data-ref", ref);
    wrap.innerHTML = `
      <div class="selected-row">
        <code>${ref}</code>
        <button type="button" class="btn danger btn-remove">Remove</button>
      </div>
      <textarea class="selected-options">${JSON.stringify(options || {}, null, 2)}</textarea>
    `;
    wrap.querySelector(".btn-remove").addEventListener("click", () => {
      wrap.remove();
    });
    els.selectedFeatures.appendChild(wrap);
  }
}

async function loadProfiles() {
  const payload = await api("GET", "/api/profiles");
  state.profiles = payload.profiles || [];
  if (!state.activeId && state.profiles.length) {
    state.activeId = state.profiles[0].id;
  }
  renderProfiles();
  if (state.activeId) {
    await loadProfile(state.activeId);
  }
}

async function loadProfile(id) {
  const payload = await api("GET", `/api/profiles/${encodeURIComponent(id)}`);
  state.activeId = id;
  state.activeProfile = payload.profile;
  renderProfiles();
  profileToForm(state.activeProfile);
}

async function saveCurrentProfile() {
  const payload = formToProfile();
  const saved = await api("PUT", `/api/profiles/${encodeURIComponent(state.activeId)}`, payload);
  state.activeProfile = saved.profile;
  await loadProfiles();
  appendConsole("Profile saved", {
    profileId: saved.profile.id,
    manifestPath: saved.profile.manifestPath,
  });
}

async function createProfile() {
  const defaults = state.defaults || {};
  const name = window.prompt("Profile name", "Ubuntu Developer");
  if (!name) return;
  const manifest = {
    $schema: "https://raw.githubusercontent.com/wsl-images/wslb/main/schemas/wslb-workspace.schema.json",
    name,
    image: defaults.baseImage || "ubuntu:24.04",
    remoteUser: defaults.remoteUser || "dev",
    features: {
      "wslb:feature/wsl-prereqs": {},
      "ghcr.io/devcontainers/features/common-utils:2": {},
    },
    wslb: {
      version: 1,
      id: slugify(name),
      distroName: name.replace(/[^A-Za-z0-9]/g, ""),
      managed: defaults.managed !== false,
      state: { mode: defaults.stateMode || "windows-dir" },
      wslconf: { boot: { systemd: defaults.systemd !== false } },
      distribution: {
        shortcut: {
          enabled: true,
          icon: {
            simpleIcon: defaults.iconSimpleIcon || "ubuntu",
            color: normalizeHexColor(defaults.iconColor || "E95420", "E95420"),
            style: defaults.iconStyle || "flat",
          },
        },
        windowsterminal: { enabled: true },
      },
    },
  };
  manifest.wslb.distribution.windowsterminal.template = buildTerminalTemplate(
    defaults.terminalTheme || "Campbell",
    defaults.terminalFont || "Cascadia Mono",
    defaults.terminalOpacity ?? 100,
    defaults.terminalUseAcrylic === true,
    defaults.terminalCursorShape || "bar",
  );
  const resp = await api("POST", "/api/profiles", {
    name,
    manifestPath: `.devcontainer/${slugify(name)}.json`,
    manifest,
  });
  state.activeId = resp.profile.id;
  await loadProfiles();
}

async function deleteCurrentProfile() {
  if (!state.activeProfile) return;
  const sure = window.confirm(`Delete profile "${state.activeProfile.name}"?`);
  if (!sure) return;
  await api("DELETE", `/api/profiles/${encodeURIComponent(state.activeId)}`);
  state.activeId = null;
  state.activeProfile = null;
  await loadProfiles();
}

async function writeManifest() {
  if (!state.activeProfile) return;
  await saveCurrentProfile();
  const resp = await api("POST", `/api/profiles/${encodeURIComponent(state.activeId)}/apply`);
  appendConsole("Manifest written", resp);
}

async function runProfileAction(action) {
  if (!state.activeProfile) return;
  await saveCurrentProfile();
  const body = {
    action,
    rollbackTo: els.rollbackTarget.value.trim(),
    engine: els.engine.value,
    nonInteractive: boolFromSelect(els.nonInteractive),
    fallbackWindowsDir: boolFromSelect(els.fallbackWindowsDir),
    writeManifest: true,
  };
  appendConsole(`Running action: ${action}`);
  const resp = await api("POST", `/api/profiles/${encodeURIComponent(state.activeId)}/action`, body);
  appendConsole(`Action completed: ${action}`, {
    ok: resp.ok,
    exitCode: resp.result.exitCode,
    durationMs: resp.result.durationMs,
  });
  if (resp.result.stdout) appendConsole("stdout", resp.result.stdout);
  if (resp.result.stderr) appendConsole("stderr", resp.result.stderr);
  if (resp.result.json) appendConsole("json", resp.result.json);
}

async function runDoctor() {
  appendConsole("Running doctor...");
  const resp = await api("POST", "/api/doctor", {});
  appendConsole("Doctor completed", {
    ok: resp.ok,
    exitCode: resp.result.exitCode,
  });
  if (resp.result.stdout) appendConsole("stdout", resp.result.stdout);
  if (resp.result.stderr) appendConsole("stderr", resp.result.stderr);
  if (resp.result.json) appendConsole("json", resp.result.json);
}

async function loadDefaults() {
  const payload = await api("GET", "/api/defaults");
  state.defaults = payload.defaults || {};
  els.defBaseImage.value = state.defaults.baseImage || "";
  els.defRemoteUser.value = state.defaults.remoteUser || "";
  els.defStateMode.value = state.defaults.stateMode || "windows-dir";
  setBoolSelect(els.defManaged, state.defaults.managed !== false);
  setBoolSelect(els.defSystemd, state.defaults.systemd !== false);
  els.defIconSimple.value = state.defaults.iconSimpleIcon || "";
  els.defIconColor.value = normalizeHexColor(state.defaults.iconColor || "E95420", "E95420");
  syncHexInputs(els.defIconColor, els.defIconColorPicker, "E95420");
  els.defIconStyle.value = state.defaults.iconStyle || "flat";
  els.defTerminalTheme.value = state.defaults.terminalTheme || "Campbell";
  els.defTerminalFont.value = state.defaults.terminalFont || "Cascadia Mono";
  els.defTerminalOpacity.value = String(state.defaults.terminalOpacity ?? 100);
  setBoolSelect(els.defTerminalUseAcrylic, state.defaults.terminalUseAcrylic === true);
  els.defTerminalCursorShape.value = state.defaults.terminalCursorShape || "bar";
}

async function saveDefaultsFromForm() {
  const defaults = {
    baseImage: els.defBaseImage.value.trim(),
    remoteUser: els.defRemoteUser.value.trim(),
    stateMode: els.defStateMode.value,
    managed: boolFromSelect(els.defManaged),
    systemd: boolFromSelect(els.defSystemd),
    iconSimpleIcon: els.defIconSimple.value.trim(),
    iconColor: normalizeHexColor(els.defIconColor.value.trim(), "E95420"),
    iconStyle: els.defIconStyle.value,
    terminalTheme: els.defTerminalTheme.value,
    terminalFont: els.defTerminalFont.value.trim(),
    terminalOpacity: Number(els.defTerminalOpacity.value || "100"),
    terminalUseAcrylic: boolFromSelect(els.defTerminalUseAcrylic),
    terminalCursorShape: els.defTerminalCursorShape.value,
  };
  const payload = await api("PUT", "/api/defaults", { defaults });
  state.defaults = payload.defaults;
  appendConsole("Defaults saved", payload.defaults);
}

function renderFeatureResults(items) {
  els.featureSearchResults.innerHTML = "";
  if (!items.length) {
    els.featureSearchResults.innerHTML = "<div class='hint'>No matching features.</div>";
    return;
  }
  for (const item of items) {
    const row = document.createElement("div");
    row.className = "feature-item";
    row.innerHTML = `
      <div class="feature-meta">
        <strong>${item.name}</strong>
        <div class="tag">${item.maintainer}</div>
        <code>${item.ref}</code>
      </div>
      <button type="button" class="btn subtle">Add</button>
    `;
    row.querySelector("button").addEventListener("click", () => {
      if (!state.activeProfile) return;
      const manifest = state.activeProfile.manifest || {};
      manifest.features = manifest.features || {};
      if (!(item.ref in manifest.features)) manifest.features[item.ref] = {};
      renderSelectedFeatures(manifest.features);
    });
    els.featureSearchResults.appendChild(row);
  }
}

async function searchFeatures(refresh) {
  const q = encodeURIComponent(els.featureSearch.value.trim());
  const refreshFlag = refresh ? "&refresh=1" : "";
  const payload = await api("GET", `/api/features/search?q=${q}&limit=120${refreshFlag}`);
  state.featureItems = payload.items || [];
  renderFeatureResults(state.featureItems);
}

function renderIconResults(items) {
  els.iconSearchResults.innerHTML = "";
  if (!items.length) {
    els.iconSearchResults.innerHTML = "<div class='hint'>No matching icons.</div>";
    return;
  }
  const color = normalizeHexColor(els.manifestIconColor.value, "E95420");
  for (const item of items) {
    const row = document.createElement("div");
    row.className = "icon-item";
    const encodedURL = `url("https://simpleicons.org/icons/${encodeURIComponent(item.slug)}.svg")`;
    row.innerHTML = `
      <span class="icon-mask" style="--icon-url:${encodedURL};--icon-color:#${color};"></span>
      <div class="icon-item-meta">
        <strong>${item.name}</strong>
        <code>${item.slug}</code>
      </div>
      <button type="button" class="btn subtle">Use</button>
    `;
    row.querySelector("button").addEventListener("click", () => {
      els.manifestIconSimple.value = item.slug;
      renderIconPreview();
      appendConsole("Selected icon", { slug: item.slug, color: `#${color}` });
    });
    els.iconSearchResults.appendChild(row);
  }
}

async function searchIcons(refresh) {
  const q = encodeURIComponent(els.iconSearch.value.trim());
  const refreshFlag = refresh ? "&refresh=1" : "";
  const payload = await api("GET", `/api/icons/search?q=${q}&limit=220${refreshFlag}`);
  state.iconItems = payload.items || [];
  renderIconResults(state.iconItems);
}

let featureSearchTimer = null;
function queueFeatureSearch(refresh) {
  clearTimeout(featureSearchTimer);
  featureSearchTimer = setTimeout(() => {
    void searchFeatures(refresh).catch((err) => appendConsole("Feature search error", err.message));
  }, 180);
}

let iconSearchTimer = null;
function queueIconSearch(refresh) {
  clearTimeout(iconSearchTimer);
  iconSearchTimer = setTimeout(() => {
    void searchIcons(refresh).catch((err) => appendConsole("Icon search error", err.message));
  }, 180);
}

function wireActions() {
  els.profileFilter.addEventListener("input", renderProfiles);
  els.btnCreateProfile.addEventListener("click", () => {
    void createProfile().catch((err) => appendConsole("Create profile failed", err.message));
  });
  els.btnSaveProfile.addEventListener("click", () => {
    void saveCurrentProfile().catch((err) => appendConsole("Save failed", err.message));
  });
  els.btnDeleteProfile.addEventListener("click", () => {
    void deleteCurrentProfile().catch((err) => appendConsole("Delete failed", err.message));
  });
  els.btnApplyManifest.addEventListener("click", () => {
    void writeManifest().catch((err) => appendConsole("Write manifest failed", err.message));
  });
  els.btnDoctor.addEventListener("click", () => {
    void runDoctor().catch((err) => appendConsole("Doctor failed", err.message));
  });
  els.btnSaveDefaults.addEventListener("click", () => {
    void saveDefaultsFromForm().catch((err) => appendConsole("Save defaults failed", err.message));
  });
  els.featureSearch.addEventListener("input", () => queueFeatureSearch(false));
  els.btnFeatureRefresh.addEventListener("click", () => queueFeatureSearch(true));
  els.iconSearch.addEventListener("input", () => queueIconSearch(false));
  els.btnIconRefresh.addEventListener("click", () => queueIconSearch(true));
  els.manifestIconSimple.addEventListener("input", () => renderIconPreview());
  els.manifestIconColor.addEventListener("input", () => {
    syncHexInputs(els.manifestIconColor, els.manifestIconColorPicker, "E95420");
    renderIconPreview();
    renderIconResults(state.iconItems);
  });
  els.manifestIconColorPicker.addEventListener("input", () => {
    els.manifestIconColor.value = normalizeHexColor(els.manifestIconColorPicker.value, "E95420");
    renderIconPreview();
    renderIconResults(state.iconItems);
  });
  els.defIconColor.addEventListener("input", () => {
    syncHexInputs(els.defIconColor, els.defIconColorPicker, "E95420");
  });
  els.defIconColorPicker.addEventListener("input", () => {
    els.defIconColor.value = normalizeHexColor(els.defIconColorPicker.value, "E95420");
  });

  document.querySelectorAll("[data-action]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const action = btn.getAttribute("data-action");
      void runProfileAction(action).catch((err) => appendConsole(`Action ${action} failed`, err.message));
    });
  });
}

async function init() {
  wireActions();
  await loadDefaults();
  await loadProfiles();
  if (!state.profiles.length) {
    appendConsole("No profiles found. Create one from the sidebar.");
  }
  await searchFeatures(false);
  await searchIcons(false);
  renderIconPreview();
  appendConsole("WSLB Store ready");
}

void init().catch((err) => {
  appendConsole("Initialization failed", err.message || String(err));
});
