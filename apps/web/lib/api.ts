"use client";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

export function token() {
  if (typeof window === "undefined") return "";
  return localStorage.getItem("fynora_token") || "";
}

export function setToken(value: string) {
  localStorage.setItem("fynora_token", value);
}

export function logout() {
  localStorage.removeItem("fynora_token");
  window.location.href = "/";
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const started = performance.now();
  const requestId = crypto.randomUUID();
  const headers = new Headers(init.headers);
  headers.set("Content-Type", headers.get("Content-Type") || "application/json");
  headers.set("X-Request-ID", requestId);
  const jwt = token();
  if (jwt) headers.set("Authorization", `Bearer ${jwt}`);
  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  const durationMs = Math.round(performance.now() - started);
  const responseRequestId = res.headers.get("X-Request-ID") || requestId;
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    console.error("[clearflow-api:error]", { path, status: res.status, durationMs, requestId: responseRequestId, message: body?.error?.message });
    throw new Error(body?.error?.message || `Request failed: ${res.status}`);
  }
  console.info("[clearflow-api]", { path, method: init.method || "GET", status: res.status, durationMs, requestId: responseRequestId });
  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function upload<T>(path: string, file: File): Promise<T> {
  const started = performance.now();
  const requestId = crypto.randomUUID();
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: token() ? { Authorization: `Bearer ${token()}`, "X-Request-ID": requestId } : { "X-Request-ID": requestId },
    body: form
  });
  const durationMs = Math.round(performance.now() - started);
  const responseRequestId = res.headers.get("X-Request-ID") || requestId;
  if (!res.ok) {
    console.error("[clearflow-api:error]", { path, status: res.status, durationMs, requestId: responseRequestId });
    throw new Error("Upload failed");
  }
  console.info("[clearflow-api]", { path, method: "POST", status: res.status, durationMs, requestId: responseRequestId });
  return res.json();
}
