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
  const headers = new Headers(init.headers);
  headers.set("Content-Type", headers.get("Content-Type") || "application/json");
  const jwt = token();
  if (jwt) headers.set("Authorization", `Bearer ${jwt}`);
  const res = await fetch(`${API_BASE}${path}`, { ...init, headers });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body?.error?.message || `Request failed: ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

export async function upload<T>(path: string, file: File): Promise<T> {
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: token() ? { Authorization: `Bearer ${token()}` } : {},
    body: form
  });
  if (!res.ok) throw new Error("Upload failed");
  return res.json();
}
