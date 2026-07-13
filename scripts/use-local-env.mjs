#!/usr/bin/env node

import { existsSync, readFileSync, writeFileSync } from "node:fs";

const envPath = ".env";
const defaults = {
  PORT: "8080",
  DATABASE_URL: "postgres://postgres:postgres@localhost:5433/fynora?sslmode=disable",
  REDIS_ENABLED: "true",
  REDIS_URL: "redis://localhost:6379/0",
  REDIS_TLS: "false",
  APP_ENV: "development",
  NEXT_PUBLIC_API_BASE_URL: "http://localhost:8080"
};

function parseEnv(text) {
  const lines = text.split(/\r?\n/);
  const seen = new Set();
  const out = [];
  for (const line of lines) {
    const match = line.match(/^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/);
    if (!match) {
      out.push({ raw: line });
      continue;
    }
    seen.add(match[1]);
    out.push({ key: match[1], value: match[2] });
  }
  return { lines: out, seen };
}

const original = existsSync(envPath) ? readFileSync(envPath, "utf8") : "";
const parsed = parseEnv(original);
for (const line of parsed.lines) {
  if (line.key && Object.prototype.hasOwnProperty.call(defaults, line.key)) {
    line.value = defaults[line.key];
  }
}
for (const [key, value] of Object.entries(defaults)) {
  if (!parsed.seen.has(key)) {
    parsed.lines.push({ key, value });
  }
}

const rendered = parsed.lines
  .map((line) => line.key ? `${line.key}=${line.value}` : line.raw)
  .join("\n")
  .replace(/\n*$/, "\n");

writeFileSync(envPath, rendered);
console.log("[local-env] wrote local Docker Postgres/Redis settings to .env");
console.log("[local-env] preserved any existing provider secrets not listed in the local defaults");
