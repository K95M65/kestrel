import camelcaseKeys from "camelcase-keys";
import type { NetworkItem, SetupRequest } from "@/types";
import { publishRobotName } from "./robotName";

const API_BASE =
  import.meta.env.VITE_API_BASE ??
  import.meta.env.VITE_NETWORK_API ??
  import.meta.env.VITE_API_URL ??
  "";

/** 0 = error, 1 = success (matches backend JSONReponseStatus) */
export type JSONResponseStatus = 0 | 1;

export interface JSONResponse<T = unknown> {
  status: JSONResponseStatus;
  message: string | null;
  data: T;
}

// Legacy Bearer fallback. Browsers normally authenticate via the
// `os_session` cookie set by POST /api/login, but scripted callers and
// shareable dev links may still pass an explicit token. Cleared on logout;
// not persisted on first load (cookie auth makes sessionStorage unnecessary).
const TOKEN_STORAGE_KEY = "device_api_token";
let apiToken: string =
  typeof window !== "undefined" ? sessionStorage.getItem(TOKEN_STORAGE_KEY) ?? "" : "";

export function setApiToken(token: string): void {
  apiToken = token ?? "";
  if (typeof window === "undefined") return;
  if (apiToken) sessionStorage.setItem(TOKEN_STORAGE_KEY, apiToken);
  else sessionStorage.removeItem(TOKEN_STORAGE_KEY);
}

export function getApiToken(): string {
  return apiToken;
}

/** Append ?token=<key> to a URL only when a legacy Bearer token is in play.
 *  After login, cookies attach automatically — callers can pass URLs through
 *  this helper unchanged and the URL stays clean. */
export function withApiToken(url: string): string {
  if (!apiToken) return url;
  const sep = url.includes("?") ? "&" : "?";
  return `${url}${sep}token=${encodeURIComponent(apiToken)}`;
}

/** Build a `/api/hardware/<path>` URL. Cookie auto-attaches for same-origin
 *  requests, so this is now just a prefix builder — no token leaks into the
 *  URL, DOM, or browser history. Legacy Bearer fallback still rides along
 *  when a token is set (dev/scripted callers). */
export function hwUrl(path: string): string {
  return withApiToken(`/api/hardware${path}`);
}

/** Build a `GET /api/agent/file` URL for a DEVICE-LOCAL path the agent named in
 *  a reply (a camera snapshot, a generated report). The path is validated
 *  server-side against an allow-list of roots and served types — this helper
 *  only builds the URL, it makes no claim that the file is servable, so callers
 *  must handle 403/404 (an <img> onError, a link that just fails). */
export function agentFileUrl(devicePath: string): string {
  return withApiToken(`${API_BASE}/api/agent/file?path=${encodeURIComponent(devicePath)}`);
}

/** Base64-encode a File for JSON bodies (e.g. face enroll). Uses FileReader
 *  instead of `btoa(String.fromCharCode(...new Uint8Array(buf)))`: spreading a
 *  full-resolution JPEG's bytes into a function call blows the call stack
 *  (RangeError) on large photos, silently failing the upload. Strips the
 *  `data:<mime>;base64,` prefix so the result is the raw base64 HAL expects. */
export function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve((reader.result as string).split(",")[1]);
    reader.onerror = () => reject(new Error("Failed to read file"));
    reader.readAsDataURL(file);
  });
}

// Setup query params that may carry secrets. When a redirect or shareable
// link preserves window.location.search, these must be stripped so the
// token doesn't propagate to a new origin, browser history, proxy log, or
// any clipboard the user pastes the URL into.
const SECRET_QUERY_KEYS = [
  "tele_token",
  "slack_bot_token",
  "slack_app_token",
  "discord_bot_token",
  "llm_api_key",
  "deepgram_api_key",
  "stt_api_key",
  "tts_api_key",
  "mqtt_password",
  "password",
  "admin_password",
];

/** Return window.location.search (or the given query string) with every
 *  known secret key removed. Preserves harmless params like `debug=true`. */
export function safeSearch(search?: string): string {
  const raw = search ?? (typeof window !== "undefined" ? window.location.search : "");
  if (!raw) return "";
  const p = new URLSearchParams(raw);
  let changed = false;
  for (const k of SECRET_QUERY_KEYS) {
    if (p.has(k)) {
      p.delete(k);
      changed = true;
    }
  }
  if (!changed) return raw;
  const out = p.toString();
  return out ? `?${out}` : "";
}

// Session-storage key that outlives the scrub. Setup's useSetupUrlParams
// module reads from this key when window.location.search is empty (post-scrub
// reload), so the Setup form can still ship the operator-provided secrets. Key
// MUST match the one in hooks/setup/useSetupUrlParams.ts.
const SETUP_URL_SEARCH_STORE_KEY = "autonomous.setup_url_search.v1";

/** Scrub secret query params from window.location without a navigation.
 *  Called once on every page mount so a `?llm_api_key=…` link doesn't survive
 *  in browser history / address bar / clipboard after the page reads it.
 *
 *  EXCEPTION — the /setup route is deliberately left untouched: the operator
 *  flow requires that an F5 on Setup keeps the full URL (secrets included) on
 *  the address bar, so a reload re-reads them straight from the query string
 *  rather than depending on sessionStorage rehydration (which does not survive
 *  the AP→STA origin change: 192.168.100.1 → the device's LAN IP). This is an
 *  accepted trade-off: secrets stay visible in Setup's history / address bar.
 *
 *  F5-reload survival (all OTHER routes except Login): persist the raw pre-scrub search to
 *  sessionStorage BEFORE wiping the URL. That way a reload (which reloads the
 *  scrubbed URL, losing everything the module-load snapshot in useSetupUrlParams
 *  would have captured) can still rehydrate the operator's secrets.
 *  sessionStorage is per-tab and cleared on tab close, so this stays a safer
 *  resting place than the URL — not shown in the address bar, not
 *  screenshot-captured, not walked by "back" history. Doing this here (rather
 *  than only in useSetupUrlParams) covers the cache-transitional case: a cached
 *  OLD JS bundle that runs scrub before the NEW JS bundle has ever loaded still
 *  seeds sessionStorage, so a subsequent F5 into NEW JS can rehydrate. */
export function scrubLocationSecrets(): void {
  if (typeof window === "undefined") return;
  // Keep the full URL (secrets included) on /setup — see doc comment above.
  if (window.location.pathname === "/setup") return;
  const raw = window.location.search;
  const cleaned = safeSearch(raw);
  if (cleaned === raw) return;
  try {
    // /login reads ?password during its first render and submits it straight
    // away. Do not retain that credential in sessionStorage after scrubbing.
    if (raw && window.location.pathname !== "/login") {
      sessionStorage.setItem(SETUP_URL_SEARCH_STORE_KEY, raw);
    }
  } catch {
    /* private-mode / storage disabled — URL scrub still proceeds */
  }
  const next = `${window.location.pathname}${cleaned}${window.location.hash}`;
  window.history.replaceState(null, "", next);
}

// Patched window.fetch: ensures every same-origin /api/* request rides the
// session cookie (credentials: include) and attaches a legacy Bearer header
// when one is in play. Browsers default fetch to credentials: 'same-origin'
// for same-origin requests, but Vite's dev server can confuse the heuristic
// and the explicit setting is cheap insurance.
if (typeof window !== "undefined" && !(window as unknown as { __osFetchPatched?: boolean }).__osFetchPatched) {
  const origFetch = window.fetch.bind(window);
  window.fetch = function patchedFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    let url = "";
    if (typeof input === "string") url = input;
    else if (input instanceof URL) url = input.toString();
    else url = (input as Request).url;

    const isApiCall = url.startsWith("/api/") || url.includes("/api/");
    if (!isApiCall) return origFetch(input, init);

    // `mode: "no-cors"` fetches (the mDNS probe in useSetupStatusPolling is
    // the only intentional caller) must stay as the operator wrote them —
    // both the Authorization header (not in the CORS safelist) and the
    // forced `credentials: "include"` flip Chrome into preflight / private-
    // network restriction behaviour that throws before the request leaves
    // the page. Pass-through preserves the original "send raw ping, don't
    // care about response body" semantics.
    if (init?.mode === "no-cors") return origFetch(input, init);

    const headers = new Headers(init?.headers);
    if (apiToken && !headers.has("Authorization")) {
      headers.set("Authorization", `Bearer ${apiToken}`);
    }
    return origFetch(input, { ...init, headers, credentials: "include" });
  };
  (window as unknown as { __osFetchPatched?: boolean }).__osFetchPatched = true;
}

async function apiRequest<T>(url: string, options?: RequestInit): Promise<T> {
  const headers = new Headers(options?.headers);
  if (apiToken && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${apiToken}`);
  }
  const res = await fetch(url, { credentials: "include", ...options, headers });
  const json = (await res.json()) as JSONResponse<T>;
  if (json.status !== 1) {
    const msg =
      typeof json.message === "string" ? json.message : res.ok ? "Request failed" : res.statusText;
    const err = new Error(msg) as Error & { status?: number };
    err.status = res.status;
    throw err;
  }
  return json.data;
}

/**
 * Converts object keys from snake_case to camelCase (uses camelcase-keys).
 * Use for API responses that return snake_case keys.
 */
export function parseSnakeToCamel<T = Record<string, unknown>>(
  raw: Record<string, unknown>,
  options?: { deep?: boolean }
): T {
  return camelcaseKeys(raw as Record<string, unknown>, { deep: options?.deep ?? false }) as T;
}

export async function getNetworks(): Promise<NetworkItem[]> {
  return apiRequest<NetworkItem[]>(`${API_BASE}/api/network`);
}

/** The Wi-Fi network wlan0 is currently associated with (from `iwgetid -r`),
 *  or null when the interface isn't associated with any station network.
 *  Public (no admin auth) so the reloaded Setup page — served from the new
 *  LAN IP after the AP→STA join — can confirm the device is actually on home
 *  Wi-Fi and mark the Wi-Fi step done without reading admin-gated config. */
export interface CurrentNetwork {
  ssid: string;
  signal: number;
  linkRate: number;
}

/** GET /api/network/current — the SSID the device is presently joined to.
 *  Returns null when wlan0 isn't associated (e.g. still running the setup AP). */
export async function getCurrentNetwork(): Promise<CurrentNetwork | null> {
  return apiRequest<CurrentNetwork | null>(`${API_BASE}/api/network/current`);
}

export async function setupNetwork(ssid: string, password: string): Promise<string> {
  return apiRequest<string>(`${API_BASE}/api/network/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ssid, password }),
  });
}

export async function setupDevice(body: SetupRequest): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/setup`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export interface SetupStatus {
  phase: "idle" | "connecting" | "connected" | "failed";
  lan_ip: string;
  error: string;
  // Hardware-derived "Lamp-XXXX". Used by the web client to compute the
  // canonical mDNS hostname (`lamp-xxxx.local`) for the AP→STA auto-redirect.
  // Exposed on this open endpoint because /api/device/config is admin-gated
  // and fresh devices have no admin yet.
  mac: string;
  // Setup runs since the device booted, bumped when a run starts. Lets the
  // poller recognise its own run's verdict without having to catch the
  // "connecting" phase live — see useSetupStatusPolling. Optional: a device on
  // an older os-server build simply omits it.
  run?: number;
  // Whether the device has ever completed setup — what decides the initial
  // wizard vs the continue wizard (`SetupGate`). Optional for the same
  // older-build reason; absent falls back to the internet heuristic.
  set_up_completed?: boolean;
}

/** Polled by Setup.tsx during the AP→STA transition. Returns the device's
 *  current setup phase plus the LAN IP once Wi-Fi is associated, so the web
 *  client can redirect the user to the new URL. */
export async function getSetupStatus(): Promise<SetupStatus> {
  return apiRequest<SetupStatus>(`${API_BASE}/api/device/setup/status`);
}

export async function checkInternet(): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/network/check-internet`);
}


export async function getSetup(): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/setup`);
}

/** Sanitized device config — Has* booleans replace raw secrets so they
 *  never reach the DOM / sessionStorage / HAR captures. PUT
 *  /api/device/config still accepts plaintext writes through SecretUpdateField. */
export interface DeviceConfig {
  channel: string;
  telegram_user_id: string;
  slack_user_id: string;
  discord_guild_id: string;
  discord_user_id: string;
  llm_model: string;
  llm_base_url: string;
  llm_disable_thinking: boolean;
  stt_base_url: string;
  tts_base_url: string;
  stt_language: string;
  stt_model: string;
  tts_provider: string;
  tts_voice: string;
  wakeword: boolean;
  agent_name: string;
  wake_phrase?: string;
  wake_phrases: string[];
  realtime?: {
    enabled?: boolean;
    provider?: string;
    model?: string;
    voice?: string;
    reasoning?: string;
    base_url?: string;
    has_api_key?: boolean;
  };
  device_id: string;
  mac: string;
  network_ssid: string;
  mqtt_endpoint: string;
  mqtt_username: string;
  mqtt_port: number;
  fa_channel: string;
  fd_channel: string;

  has_telegram_bot_token: boolean;
  has_slack_bot_token: boolean;
  has_slack_app_token: boolean;
  has_discord_bot_token: boolean;
  has_llm_api_key: boolean;
  has_deepgram_api_key: boolean;
  has_stt_api_key: boolean;
  has_tts_api_key: boolean;
  has_network_password: boolean;
  has_mqtt_password: boolean;
  has_admin_password: boolean;
}

export async function getTTSVoices(provider?: string, lang?: string): Promise<string[]> {
  const qs = new URLSearchParams();
  if (provider) qs.set("provider", provider);
  if (lang) qs.set("lang", lang);
  const params = qs.toString() ? `?${qs.toString()}` : "";
  return apiRequest<string[]>(`${API_BASE}/api/device/voices${params}`);
}

export async function getTTSProviders(): Promise<string[]> {
  return apiRequest<string[]>(`${API_BASE}/api/device/tts-providers`);
}

export type LLMAuthKind = "api_key" | "device_code" | "byo";

export interface LLMProviderField {
  key: string;
  label: string;
  placeholder?: string;
  secret?: boolean;
}

export interface LLMProvider {
  key: string;
  name: string;
  auth: LLMAuthKind;
  base_url?: string;
  default_model?: string;
  docs_url?: string;
  hint?: string;
  fields?: LLMProviderField[];
  openclaw_api?: string;
}

export async function getLLMProviders(): Promise<LLMProvider[]> {
  return apiRequest<LLMProvider[]>(`${API_BASE}/api/device/llm-providers`);
}

export interface LLMOAuthStart {
  provider: string;
  user_code: string;
  device_code: string;
  verification_uri: string;
  verification_uri_complete?: string;
  expires_in: number;
  interval: number;
  base_url: string;
  default_model: string;
}

export interface LLMOAuthPoll {
  pending: boolean;
  interval?: number;
  provider?: string;
  access_token?: string;
  base_url?: string;
  default_model?: string;
}

export async function startLLMOAuth(provider: string): Promise<LLMOAuthStart> {
  return apiRequest<LLMOAuthStart>(`${API_BASE}/api/device/llm-oauth/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ provider }),
  });
}

export interface CompanionApp {
  id: string;
  name: string;
  platform: string;
  version?: string;
  summary: string;
  hint?: string;
  download_url: string;
  direct_url?: string;
  source_url: string;
  kind?: string;
  install_url?: string;
  subdir?: string;
}

export async function getCompanionApps(): Promise<CompanionApp[]> {
  return apiRequest<CompanionApp[]>(`${API_BASE}/api/device/companion-apps`);
}

export async function installPlugin(url: string, subdir?: string, id?: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/plugin/install`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url: url || undefined, subdir: subdir || undefined, id: id || undefined }),
  });
}

export async function installTrustedPlugin(id: string): Promise<boolean> {
  return installPlugin("", undefined, id);
}

export async function pollLLMOAuth(provider: string, deviceCode: string): Promise<LLMOAuthPoll> {
  return apiRequest<LLMOAuthPoll>(`${API_BASE}/api/device/llm-oauth/poll`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ provider, device_code: deviceCode }),
  });
}

export interface RealtimeOptions {
  providers: string[];
  voices: Record<string, string[]>;
  reasoning: Record<string, string[]>;
}

export async function getRealtimeOptions(): Promise<RealtimeOptions> {
  return apiRequest<RealtimeOptions>(`${API_BASE}/api/device/realtime-options`);
}

export interface AgentRuntimeStatus {
  current: string;
  options: string[];
}

export async function getAgentRuntime(): Promise<AgentRuntimeStatus> {
  return apiRequest<AgentRuntimeStatus>(`${API_BASE}/api/device/agent-runtime`);
}

/** POST /api/device/agent-runtime — swap the agentic backend (openclaw ⇄ hermes).
 *  The device restarts os-server right after, so the connection drops; callers
 *  should treat success as "accepted, reconnecting" and re-poll once it's back. */
export async function setAgentRuntime(runtime: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/agent-runtime`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ runtime }),
  });
}

export interface TimezoneStatus {
  current: string;
  zones: string[];
}

/** GET /api/device/timezone — current IANA zone + selectable list (from system tzdata). */
export async function getTimezone(): Promise<TimezoneStatus> {
  return apiRequest<TimezoneStatus>(`${API_BASE}/api/device/timezone`);
}

/** POST /api/device/timezone — apply an IANA zone (e.g. "Asia/Ho_Chi_Minh").
 *  Writes /etc/localtime + /etc/timezone and persists to config; takes effect
 *  without a HAL restart (clock helpers read /etc/timezone live). */
export async function setTimezone(timezone: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/timezone`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ timezone }),
  });
}

export interface SleepSchedule {
  enabled: boolean;
  sleep_at: string;
  wake_at: string;
  days?: number[];
}

export interface SleepStatus {
  sleeping: boolean;
  scheduled: boolean;
  emotion?: string;
  schedule: SleepSchedule;
  next_transition?: string;
  next_transition_kind?: "sleep" | "wake";
}

export async function getSleep(): Promise<SleepStatus> {
  return apiRequest<SleepStatus>(`${API_BASE}/api/device/sleep`);
}

export async function setSleepSchedule(sched: SleepSchedule): Promise<SleepStatus> {
  return apiRequest<SleepStatus>(`${API_BASE}/api/device/sleep`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sched),
  });
}

export async function sleepNow(): Promise<SleepStatus> {
  return apiRequest<SleepStatus>(`${API_BASE}/api/device/sleep/now`, { method: "POST" });
}

export async function wakeNow(): Promise<SleepStatus> {
  return apiRequest<SleepStatus>(`${API_BASE}/api/device/sleep/wake`, { method: "POST" });
}

export interface FeatureFlag {
  enabled: boolean;
}

export interface BehaviorsConfig {
  onboarded?: boolean;
  /** Household member who is the operator — People card "Me". */
  me?: string;
  morning_brief: {
    enabled: boolean;
    at: string;
    days?: number[];
    speak: boolean;
    telegram: boolean;
    weather: boolean;
    calendar: boolean;
    email: boolean;
    habits: boolean;
    max_seconds: number;
  };
  remember: { enabled: boolean; max_items: number };
  dance: { enabled: boolean; default_query: string };
  privacy: { camera_on_demand: boolean; face_follow_after_wake: boolean };
  connectors: { draft_not_send: boolean; ask?: string };
  presence: { idle_motion: boolean };
  doa: FeatureFlag;
  layered_motion: FeatureFlag;
  focus: { enabled: boolean; phone_nag: boolean; cooldown_min: number };
  kids: { enabled: boolean; session_min: number };
  greeter: { enabled: boolean; named_only: boolean };
  look: { enabled: boolean };
  kitchen: {
    enabled: boolean;
    lunch_start: string;
    lunch_end: string;
    dinner_start: string;
    dinner_end: string;
  };
  home_assistant: { enabled: boolean; url: string; token?: string };
  marionette: FeatureFlag;
  tools: { weather: boolean; time: boolean; search: boolean };
  hand_track: FeatureFlag;
  radio: FeatureFlag;
  telepresence: FeatureFlag;
  stories: { enabled: boolean; max_min: number };
  pomodoro: { enabled: boolean; work_min: number; break_min: number };
  wearables: { enabled: boolean; provider?: string };
}

export interface PomodoroStatus {
  running: boolean;
  phase?: string;
  ends_at?: string;
  remain_sec?: number;
}

export interface BehaviorsStatus {
  config: BehaviorsConfig;
  ha_token_set: boolean;
  meeting: boolean;
  last_brief?: string;
  next_brief?: string;
  memory_count: number;
  pomodoro: PomodoroStatus;
}

export interface MemoryItem {
  id: string;
  text: string;
  created_at: string;
}

export function defaultBehaviors(): BehaviorsConfig {
  return {
    morning_brief: {
      enabled: false, at: "07:30", speak: true, telegram: true,
      weather: true, calendar: true, email: true, habits: true, max_seconds: 40,
    },
    remember: { enabled: true, max_items: 200 },
    dance: { enabled: true, default_query: "upbeat dance pop" },
    privacy: { camera_on_demand: true, face_follow_after_wake: true },
    connectors: { draft_not_send: true, ask: "important_actions" },
    presence: { idle_motion: true },
    doa: { enabled: false },
    layered_motion: { enabled: false },
    focus: { enabled: false, phone_nag: true, cooldown_min: 15 },
    kids: { enabled: false, session_min: 30 },
    greeter: { enabled: true, named_only: false },
    look: { enabled: true },
    kitchen: {
      enabled: true,
      lunch_start: "11:30", lunch_end: "13:30",
      dinner_start: "18:30", dinner_end: "20:30",
    },
    home_assistant: { enabled: false, url: "" },
    marionette: { enabled: false },
    tools: { weather: true, time: true, search: true },
    hand_track: { enabled: false },
    radio: { enabled: false },
    telepresence: { enabled: false },
    stories: { enabled: false, max_min: 10 },
    pomodoro: { enabled: false, work_min: 25, break_min: 5 },
    wearables: { enabled: false, provider: "none" },
  };
}

export async function getBehaviors(): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/behaviors`);
}

export async function setMe(label: string): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/me`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ label }),
  });
}

export async function setBehaviors(cfg: BehaviorsConfig): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/behaviors`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cfg),
  });
}

export interface ServiceStatus {
  id: string;
  kind: string;
  connected: boolean;
  auth_type?: string;
  user_email?: string;
  label?: string;
  connect_how?: string;
}

export async function getServices(): Promise<ServiceStatus[]> {
  const raw = await apiRequest<ServiceStatus[]>(`${API_BASE}/api/device/services`);
  return Array.isArray(raw) ? raw : [];
}

export async function setCalendarICS(url: string): Promise<ServiceStatus[]> {
  const raw = await apiRequest<ServiceStatus[]>(`${API_BASE}/api/device/connectors/google_calendar`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ url }),
  });
  return Array.isArray(raw) ? raw : [];
}

export async function setGmailPAT(email: string, apiKey: string): Promise<ServiceStatus[]> {
  const raw = await apiRequest<ServiceStatus[]>(`${API_BASE}/api/device/connectors/gmail`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, api_key: apiKey }),
  });
  return Array.isArray(raw) ? raw : [];
}

export async function removeConnector(code: string): Promise<ServiceStatus[]> {
  const raw = await apiRequest<ServiceStatus[]>(`${API_BASE}/api/device/connectors/${encodeURIComponent(code)}`, {
    method: "DELETE",
  });
  return Array.isArray(raw) ? raw : [];
}

export interface HouseholdMember {
  label: string;
  role: string;
}

export interface HouseholdPublic {
  claimed: boolean;
  room?: string;
  owner_email?: string;
  setup_pin?: string;
  members?: HouseholdMember[];
  invite_code?: string;
  invite_role?: string;
  invite_ttl?: number;
}

export async function getHousehold(): Promise<HouseholdPublic> {
  return apiRequest<HouseholdPublic>(`${API_BASE}/api/device/household`);
}

export async function getClaimPublic(): Promise<HouseholdPublic> {
  return apiRequest<HouseholdPublic>(`${API_BASE}/api/device/claim`);
}

export async function confirmClaim(body: {
  pin?: string; code?: string; name: string; room?: string; role?: string; email?: string;
}): Promise<HouseholdPublic> {
  return apiRequest<HouseholdPublic>(`${API_BASE}/api/device/claim`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export async function startHouseholdInvite(role: string): Promise<HouseholdPublic> {
  return apiRequest<HouseholdPublic>(`${API_BASE}/api/device/household/invite`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ role }),
  });
}

export async function setMemberRole(label: string, role: string): Promise<HouseholdPublic> {
  return apiRequest<HouseholdPublic>(
    `${API_BASE}/api/device/household/members/${encodeURIComponent(label)}/role`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ role }),
    },
  );
}

export async function setHouseholdRoom(room: string): Promise<HouseholdPublic> {
  return apiRequest<HouseholdPublic>(`${API_BASE}/api/device/household/room`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ room }),
  });
}

export interface GoogleStatus {
  ready: boolean;
  connected: boolean;
  user_email?: string;
  has_client: boolean;
  has_secret: boolean;
  auth_type?: string;
}

export async function getGoogleStatus(): Promise<GoogleStatus> {
  return apiRequest<GoogleStatus>(`${API_BASE}/api/device/google`);
}

export async function setGoogleClient(clientId: string, clientSecret: string): Promise<GoogleStatus> {
  return apiRequest<GoogleStatus>(`${API_BASE}/api/device/google/client`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ client_id: clientId, client_secret: clientSecret }),
  });
}

export interface GoogleOAuthStart {
  user_code: string;
  device_code: string;
  verification_uri: string;
  verification_uri_complete?: string;
  expires_in: number;
  interval: number;
}

export async function startGoogleOAuth(): Promise<GoogleOAuthStart> {
  return apiRequest<GoogleOAuthStart>(`${API_BASE}/api/device/google/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
}

export async function pollGoogleOAuth(deviceCode: string): Promise<{
  pending: boolean; interval?: number; connected?: boolean; user_email?: string;
}> {
  return apiRequest(`${API_BASE}/api/device/google/poll`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ device_code: deviceCode }),
  });
}

export async function setTelegramChannel(token: string, userId: string): Promise<ServiceStatus[]> {
  const raw = await apiRequest<ServiceStatus[]>(`${API_BASE}/api/device/services/telegram`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ telegram_bot_token: token, telegram_user_id: userId }),
  });
  return Array.isArray(raw) ? raw : [];
}

export async function fireBriefNow(): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/behaviors/brief`, { method: "POST" });
}

export async function setMeeting(on: boolean): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/behaviors/meeting`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ on }),
  });
}

export async function startPomodoro(): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/behaviors/pomodoro/start`, { method: "POST" });
}

export async function stopPomodoro(): Promise<BehaviorsStatus> {
  return apiRequest<BehaviorsStatus>(`${API_BASE}/api/device/behaviors/pomodoro/stop`, { method: "POST" });
}

export async function listMemories(): Promise<MemoryItem[]> {
  return apiRequest<MemoryItem[]>(`${API_BASE}/api/device/behaviors/memory`);
}

export async function addMemory(text: string): Promise<MemoryItem> {
  return apiRequest<MemoryItem>(`${API_BASE}/api/device/behaviors/memory`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text }),
  });
}

export async function deleteMemory(id: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/behaviors/memory/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export interface TestTTSOptions {
  text?: string;
  /** BCP-47 stt_language code; picks a friendly demo phrase in that language. */
  lang?: string;
  provider?: string;
}

const TTS_DEMO_PHRASES: Record<string, string> = {
  en: "[laugh] Hey! How are you doing today?",
  vi: "[laugh] Chào bạn, hôm nay bạn thế nào?",
  "zh-CN": "[laugh] 嗨，你今天怎么样？",
  "zh-TW": "[laugh] 嗨，你今天怎麼樣？",
};

function demoPhraseFor(lang?: string): string {
  if (!lang) return TTS_DEMO_PHRASES.en;
  return TTS_DEMO_PHRASES[lang] || TTS_DEMO_PHRASES.en;
}

function ttsPreviewBody(voice: string, opts: TestTTSOptions) {
  return JSON.stringify({
    text: opts.text || demoPhraseFor(opts.lang),
    voice,
    provider: opts.provider || undefined,
  });
}

/** POST /api/voice/preview — server reads the TTS API key + base URL from
 *  cfg and forwards to the device. Browser never sees or ships the credential
 *  (audit web F13). Operator can still pick a non-default voice/provider for
 *  the test by passing `provider` in opts. Plays on the robot speaker. */
export async function testTTSVoice(voice: string, opts: TestTTSOptions = {}): Promise<void> {
  await apiRequest<boolean>(`${API_BASE}/api/voice/preview`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: ttsPreviewBody(voice, opts),
  });
}

let browserPreviewAudio: HTMLAudioElement | null = null;

/** POST /api/voice/preview-audio — same phrase and voice as testTTSVoice,
 *  but HAL returns a WAV and this computer plays it. Mute on the robot does
 *  not apply. Stops a previous in-browser preview first. */
export async function previewTTSInBrowser(voice: string, opts: TestTTSOptions = {}): Promise<void> {
  if (browserPreviewAudio) {
    browserPreviewAudio.pause();
    browserPreviewAudio.src = "";
    browserPreviewAudio = null;
  }
  const headers = new Headers({ "Content-Type": "application/json" });
  if (apiToken) headers.set("Authorization", `Bearer ${apiToken}`);
  const res = await fetch(`${API_BASE}/api/voice/preview-audio`, {
    method: "POST",
    credentials: "include",
    headers,
    body: ttsPreviewBody(voice, opts),
  });
  const ctype = res.headers.get("content-type") || "";
  if (!res.ok || ctype.includes("application/json")) {
    const json = (await res.json().catch(() => null)) as JSONResponse | null;
    throw new Error(json?.message || "Couldn't play a preview.");
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const audio = new Audio(url);
  browserPreviewAudio = audio;
  audio.onended = () => {
    URL.revokeObjectURL(url);
    if (browserPreviewAudio === audio) browserPreviewAudio = null;
  };
  audio.onerror = () => {
    URL.revokeObjectURL(url);
    if (browserPreviewAudio === audio) browserPreviewAudio = null;
  };
  await audio.play();
}

export async function getDeviceConfig(): Promise<DeviceConfig> {
  return apiRequest<DeviceConfig>(`${API_BASE}/api/device/config`);
}

export async function updateDeviceConfig(body: Partial<Record<string, unknown>>): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/config`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export interface IdentityPublic {
  name: string;
  wake_phrase: string;
  wake_phrases: string[];
  wakeword: boolean;
}

/** PUT /api/device/identity — name + optional exclusive wake phrase. */
export async function setIdentity(body: { name: string; wake_phrase?: string }): Promise<IdentityPublic> {
  const saved = await apiRequest<IdentityPublic>(`${API_BASE}/api/device/identity`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  publishRobotName(saved.name || body.name);
  return saved;
}

/** POST /api/login — server validates bcrypt(password) against
 *  config.AdminPasswordHash and sets the os_session cookie on success. */
export async function login(password: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
}

// MCP Tools — remote MCP tool endpoints (HF Spaces, public MCP servers).
// headers is optional; key-value pairs sent with every MCP request.
export interface MCPTool { name: string; url: string; headers?: Record<string, string> }

/** GET /api/device/mcp-tools */
export async function listMCPTools(): Promise<MCPTool[]> {
  return apiRequest<MCPTool[]>(`${API_BASE}/api/device/mcp-tools`);
}

/** POST /api/device/mcp-tools */
export async function addMCPTool(tool: MCPTool): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/mcp-tools`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(tool),
  });
}

/** DELETE /api/device/mcp-tools/:name */
export async function removeMCPTool(name: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/device/mcp-tools/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

// Plugins — standalone Python apps installed from git URLs.
export interface Plugin { name: string; version: string; description: string; status: string; url: string }

/** GET /api/plugin */
export async function listPlugins(): Promise<Plugin[]> {
  return apiRequest<Plugin[]>(`${API_BASE}/api/plugin`);
}

/** POST /api/plugin/:name/start */
export async function startPlugin(name: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/plugin/${encodeURIComponent(name)}/start`, {
    method: "POST",
  });
}

/** POST /api/plugin/:name/stop */
export async function stopPlugin(name: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/plugin/${encodeURIComponent(name)}/stop`, {
    method: "POST",
  });
}

/** DELETE /api/plugin/:name */
export async function uninstallPlugin(name: string): Promise<boolean> {
  return apiRequest<boolean>(`${API_BASE}/api/plugin/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

// HuggingFace plugin discovery — PARKED, not deleted (#213). Plugins move to
// our own catalog, beside skills. Restore this pair together with the Go
// handler and its route, and point the request at the catalog endpoint; the
// StoreSkill client just below is the shape to copy.
//
// export interface HFSpace {
//   id: string;
//   likes: number;
//   tags: string[];
//   cardData?: { title?: string; emoji?: string; description?: string };
// }
//
// /** GET /api/plugin/browse */
// export async function searchHFPlugins(): Promise<HFSpace[]> {
//   return apiRequest<HFSpace[]>(`${API_BASE}/api/plugin/browse`);
// }

// Agent Skills catalog — proxied through the backend (avoids CORS
// and keeps the catalog host server-side).
// Shapes mirror system/domain/skillstore.go.
export interface StoreSkill {
  id: string;
  name: string;
  slug?: string;
  description?: string;
  version?: string;
  category_id?: string;
  plan_required?: string;
  author?: string;
  license?: string;
  size?: string;
  icon_url?: string;
  compatibility?: string[];
  download_count?: number;
  creator_type?: string;
  source?: string;
}

export interface StoreSkillList {
  data: StoreSkill[];
  total: number;
}

/** One file unpacked from a downloaded `.skill` archive. `text` is inlined for
 *  UTF-8 files; binary or oversized entries carry metadata only. */
export interface SkillBundleFile {
  path: string;
  size: number;
  text?: string;
  binary?: boolean;
  truncated?: boolean;
}

export interface SkillBundle {
  id: string;
  files: SkillBundleFile[];
  skipped?: number;
}

/** GET /api/agent/skills/browse — catalog listing with optional filters. */
export async function browseStoreSkills(
  opts: { keyword?: string; page?: number; limit?: number } = {},
): Promise<StoreSkillList> {
  const q = new URLSearchParams();
  if (opts.keyword) q.set("keyword", opts.keyword);
  if (opts.page) q.set("page", String(opts.page));
  if (opts.limit) q.set("limit", String(opts.limit));
  const qs = q.toString();
  return apiRequest<StoreSkillList>(`${API_BASE}/api/agent/skills/browse${qs ? `?${qs}` : ""}`);
}

/** GET /api/agent/skills/bundle — downloads + unzips the skill server-side and
 *  returns its files. Preview only; nothing is installed. */
export async function fetchSkillBundle(id: string): Promise<SkillBundle> {
  return apiRequest<SkillBundle>(
    `${API_BASE}/api/agent/skills/bundle?id=${encodeURIComponent(id)}`);
}

/** A skill authored in the web UI's "Write skill" form. */
export interface SkillDraft {
  name: string;
  description: string;
  instructions: string;
}

/** POST /api/agent/skills — writes <name>/SKILL.md into the ACTIVE agent
 *  runtime's skills dir. Returns the path written. Rejects with the backend's
 *  message when the runtime can't store authored skills (HTTP 501) or the name
 *  is taken. */
export async function saveSkill(draft: SkillDraft): Promise<{ name: string; path: string }> {
  return apiRequest<{ name: string; path: string }>(`${API_BASE}/api/agent/skills`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(draft),
  });
}

/** One node in an installed skill's file tree. `children` is set only on dirs. */
export interface SkillNode {
  name: string;
  path: string;
  dir?: boolean;
  size?: number;
  children?: SkillNode[];
}

/** A skill present in the active runtime's skills dir. */
export interface InstalledSkill {
  name: string;
  description?: string;
  files: SkillNode[];
  /** Newest mtime anywhere in the skill's tree, Unix SECONDS. Omitted when
   *  nothing in the tree could be stat'd. */
  updated_at?: number;
}

/** GET /api/agent/skills — what the ACTIVE runtime currently has installed.
 *  Rejects with the backend's message when the runtime can't list skills
 *  (HTTP 501). An un-provisioned runtime returns an empty list, not an error. */
export async function listInstalledSkills(): Promise<InstalledSkill[]> {
  return apiRequest<InstalledSkill[]>(`${API_BASE}/api/agent/skills`);
}

/** GET /api/agent/skills/files — one installed skill's files with text inlined.
 *  Same `SkillBundle` shape the store preview returns, so both detail views
 *  render through the same component. 404 when the skill is gone (stale list). */
export async function readSkillFiles(name: string): Promise<SkillBundle> {
  return apiRequest<SkillBundle>(
    `${API_BASE}/api/agent/skills/files?name=${encodeURIComponent(name)}`);
}

/** POST /api/agent/skills/upload — installs a `.skill`/`.zip` the operator picked
 *  from their machine. Multipart (not base64) so a multi-MB archive isn't
 *  inflated a third on the wire. */
export async function uploadSkill(file: File): Promise<{ name: string; path: string }> {
  const body = new FormData();
  body.append("file", file);
  // No Content-Type header: the browser must set the multipart boundary itself.
  return apiRequest<{ name: string; path: string }>(
    `${API_BASE}/api/agent/skills/upload`, { method: "POST", body });
}

/** DELETE /api/agent/skills — removes the skill from the ACTIVE runtime's skills
 *  dir. Rejects with the backend's message when it isn't installed (HTTP 404) or
 *  the runtime can't uninstall (HTTP 501). */
export async function deleteSkill(name: string): Promise<{ name: string; path: string }> {
  return apiRequest<{ name: string; path: string }>(
    `${API_BASE}/api/agent/skills?name=${encodeURIComponent(name)}`, { method: "DELETE" });
}

/** POST /api/agent/skills/install — device downloads the catalog's `.skill`
 *  archive and extracts it into the ACTIVE runtime's skills dir. Rejects with
 *  the backend's message when the runtime can't install skills (HTTP 501). */
export async function installStoreSkill(
  id: string, name?: string,
): Promise<{ name: string; path: string }> {
  return apiRequest<{ name: string; path: string }>(`${API_BASE}/api/agent/skills/install`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ id, name }),
  });
}

export async function logout(): Promise<boolean> {
  setApiToken("");
  return apiRequest<boolean>(`${API_BASE}/api/logout`, { method: "POST" });
}
