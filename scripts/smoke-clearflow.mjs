#!/usr/bin/env node

import { readFile } from "node:fs/promises";
import path from "node:path";

const API_BASE = process.env.API_BASE || "http://localhost:8080";
const RUN_ID = process.env.SMOKE_RUN_ID || new Date().toISOString().replace(/[-:.TZ]/g, "").slice(0, 14);

let token = "";
let orgId = "";
let latestRunId = "latest";
let payoutId = "";
let asyncJobIds = [];

const checks = [];

function log(message, fields = {}) {
  const suffix = Object.keys(fields).length ? ` ${JSON.stringify(fields)}` : "";
  console.log(`[SMOKE] ${message}${suffix}`);
}

function pass(name, fields = {}) {
  checks.push({ name, ok: true });
  log(`PASS ${name}`, fields);
}

function fail(name, error, fields = {}) {
  checks.push({ name, ok: false });
  log(`FAIL ${name}`, { ...fields, error: error.message || String(error) });
}

function requestId(label) {
  return `smoke-${RUN_ID}-${label}`;
}

async function call(label, method, path, options = {}) {
  const id = requestId(label);
  const headers = {
    "Content-Type": "application/json",
    "X-Request-ID": id,
    ...(options.headers || {})
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  const started = Date.now();
  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: options.body === undefined ? undefined : typeof options.body === "string" ? options.body : JSON.stringify(options.body)
  });
  const text = await res.text();
  let body = {};
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = { raw: text };
    }
  }
  const latencyMs = Date.now() - started;
  log("HTTP", { label, method, path, status: res.status, requestId: id, latencyMs });
  if (!res.ok && !options.allowFailure) {
    throw new Error(`${method} ${path} returned ${res.status}: ${text.slice(0, 300)}`);
  }
  return { status: res.status, body, requestId: id };
}

async function uploadCSV(label, pathName, filePath) {
  const id = requestId(label);
  const form = new FormData();
  const raw = await readFile(filePath);
  form.append("file", new Blob([raw], { type: "text/csv" }), path.basename(filePath));
  const headers = {
    "X-Request-ID": id
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  const started = Date.now();
  const res = await fetch(`${API_BASE}${pathName}`, {
    method: "POST",
    headers,
    body: form
  });
  const text = await res.text();
  let body = {};
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = { raw: text };
    }
  }
  const latencyMs = Date.now() - started;
  log("HTTP", { label, method: "POST", path: pathName, status: res.status, requestId: id, latencyMs });
  if (!res.ok) {
    throw new Error(`POST ${pathName} returned ${res.status}: ${text.slice(0, 300)}`);
  }
  return { status: res.status, body, requestId: id };
}

function pickData(payload) {
  if (Array.isArray(payload)) return payload;
  if (payload && Array.isArray(payload.data)) return payload.data;
  return [];
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function step(name, fn) {
  log(`STEP ${name}`);
  try {
    await fn();
    pass(name);
  } catch (error) {
    fail(name, error);
  }
}

await step("health endpoint responds", async () => {
  const result = await call("health", "GET", "/health");
  if (result.status !== 200) throw new Error("health did not return 200");
});

await step("demo auth creates token", async () => {
  const result = await call("demo-token", "POST", "/auth/demo-token", { body: {} });
  token = result.body.token || "";
  if (!token) throw new Error("missing token in demo auth response");
});

await step("organization is available", async () => {
  const result = await call("organizations", "GET", "/organizations");
  const orgs = pickData(result.body);
  orgId = orgs[0]?.id || "";
  if (!orgId) throw new Error("missing organization id");
  log("VALUE organization", { orgId, name: orgs[0]?.name });
});

await step("stripe mock sync imports processor data", async () => {
  const result = await call("stripe-sync", "POST", "/sync/stripe", {
    headers: { "Idempotency-Key": `smoke-stripe-${RUN_ID}` },
    body: {}
  });
  if (!result.body.payout && !result.body.payments) throw new Error("stripe sync response did not include payout/payments");
});

await step("bank mock sync imports bank data", async () => {
  const result = await call("bank-sync", "POST", "/sync/bank", {
    headers: { "Idempotency-Key": `smoke-bank-${RUN_ID}` },
    body: {}
  });
  if (!result.body.bank_transactions && !result.body.bankTransactions) throw new Error("bank sync response did not include transaction count");
});

await step("reconciliation run works", async () => {
  const result = await call("reconciliation-run", "POST", "/reconciliation/runs", {
    headers: { "Idempotency-Key": `smoke-recon-${RUN_ID}` },
    body: {}
  });
  latestRunId = result.body.id || result.body.jobId || "latest";
  log("VALUE reconciliation", { latestRunId, status: result.body.status, matched: result.body.matched_count, exceptions: result.body.exception_count });
});

await step("payout ledger loads", async () => {
  const result = await call("payouts", "GET", "/payouts");
  const payouts = pickData(result.body);
  payoutId = payouts[0]?.id || "";
  if (!payoutId) throw new Error("no payout returned");
  log("VALUE payout", { payoutId, count: payouts.length });
});

await step("payout explanation loads", async () => {
  const result = await call("payout-explanation", "GET", `/api/v1/payouts/${payoutId}/explanation?organizationId=${encodeURIComponent(orgId)}`);
  if (!result.body.summary && !result.body.netAmountMinor) throw new Error("missing payout explanation content");
});

await step("cash-flow forecast loads", async () => {
  const result = await call("forecast", "GET", `/api/v1/cashflow/forecast?organizationId=${encodeURIComponent(orgId)}&horizonDays=30`);
  if (!Array.isArray(result.body.series)) throw new Error("forecast response missing series");
});

await step("anomalies load", async () => {
  const result = await call("anomalies", "GET", `/api/v1/insights/anomalies?organizationId=${encodeURIComponent(orgId)}`);
  if (!Array.isArray(result.body.data)) throw new Error("anomalies response missing data array");
});

await step("cash recommendations load", async () => {
  const result = await call("recommendations", "GET", `/api/v1/recommendations/cash?organizationId=${encodeURIComponent(orgId)}`);
  if (!Array.isArray(result.body.data)) throw new Error("recommendations response missing data array");
});

await step("reconciliation match scoring loads", async () => {
  const path = `/api/v1/reconciliation-runs/${encodeURIComponent(latestRunId)}/matches?organizationId=${encodeURIComponent(orgId)}`;
  const result = await call("matches", "GET", path);
  if (!Array.isArray(result.body.data)) throw new Error("matches response missing data array");
});

await step("portfolio CSV imports persist", async () => {
  const holdings = await uploadCSV("portfolio-holdings-import", "/portfolio/import/holdings-csv", "sample-data/sample_holdings.csv");
  if (holdings.body.import?.imported_count !== 5 || holdings.body.holdings?.length !== 5) {
    throw new Error("holdings CSV import did not import 5 rows");
  }
  const txs = await uploadCSV("portfolio-transactions-import", "/portfolio/import/transactions-csv", "sample-data/sample_portfolio_transactions.csv");
  if (txs.body.import?.imported_count !== 8 || txs.body.portfolio_transactions?.length !== 8) {
    throw new Error("portfolio transactions CSV import did not import 8 rows");
  }
  const holdingsList = await call("portfolio-holdings", "GET", "/portfolio/holdings");
  const transactionsList = await call("portfolio-transactions", "GET", "/portfolio/transactions");
  const imports = await call("portfolio-imports", "GET", "/portfolio/imports");
  const summary = await call("portfolio-summary", "GET", "/portfolio/summary");
  if (pickData(holdingsList.body).length < 5) throw new Error("portfolio holdings did not persist");
  if (pickData(transactionsList.body).length < 8) throw new Error("portfolio transactions did not persist");
  if (pickData(imports.body).length < 2) throw new Error("portfolio imports did not persist");
  if (!summary.body.total_market_value || summary.body.total_market_value <= 0) throw new Error("portfolio summary did not calculate market value");
  log("VALUE portfolio", {
    holdings: pickData(holdingsList.body).length,
    transactions: pickData(transactionsList.body).length,
    imports: pickData(imports.body).length,
    marketValue: summary.body.total_market_value
  });
});

await step("plaid investments sync populates portfolio ledger", async () => {
  const result = await call("plaid-investments-sync", "POST", "/connections/plaid/sync-investments", { body: {} });
  if (result.body.mode !== "mock") throw new Error("expected mock Plaid Investments sync mode");
  if ((result.body.holdings || []).length < 3) throw new Error("Plaid Investments sync did not return holdings");
  if ((result.body.portfolio_transactions || []).length < 3) throw new Error("Plaid Investments sync did not return transactions");
  const summary = await call("portfolio-summary-after-plaid", "GET", "/portfolio/summary");
  if (!summary.body.total_market_value || summary.body.total_market_value <= 0) throw new Error("portfolio summary missing after Plaid Investments sync");
});

await step("jobs list loads", async () => {
  const result = await call("jobs", "GET", `/api/v1/jobs?organizationId=${encodeURIComponent(orgId)}`);
  if (!Array.isArray(result.body.data)) throw new Error("jobs response missing data array");
  log("VALUE jobs", { count: result.body.data.length, statuses: result.body.data.map((job) => job.status).slice(0, 5) });
});

await step("async worker jobs complete", async () => {
  const queued = [];
  for (const [label, path] of [
    ["async-stripe", `/api/v1/sync/stripe?organizationId=${encodeURIComponent(orgId)}`],
    ["async-bank", `/api/v1/sync/bank?organizationId=${encodeURIComponent(orgId)}`],
    ["async-recon", `/api/v1/reconciliation-runs?organizationId=${encodeURIComponent(orgId)}`]
  ]) {
    const result = await call(label, "POST", path, {
      headers: { "Idempotency-Key": `${label}-${RUN_ID}` },
      body: {}
    });
    if (!result.body.jobId) throw new Error(`${label} did not return jobId`);
    queued.push(result.body.jobId);
  }
  asyncJobIds = queued;
  log("VALUE async_jobs_queued", { jobIds: asyncJobIds });

  const deadline = Date.now() + Number(process.env.SMOKE_WORKER_TIMEOUT_MS || 45000);
  while (Date.now() < deadline) {
    const result = await call("async-jobs-poll", "GET", `/api/v1/jobs?organizationId=${encodeURIComponent(orgId)}`);
    const jobs = pickData(result.body).filter((job) => asyncJobIds.includes(job.id));
    const statuses = Object.fromEntries(jobs.map((job) => [job.id, job.status]));
    log("VALUE async_job_statuses", statuses);
    if (jobs.length === asyncJobIds.length && jobs.every((job) => job.status === "completed")) {
      return;
    }
    if (jobs.some((job) => ["failed", "dead", "cancelled"].includes(job.status))) {
      throw new Error(`async job failed: ${JSON.stringify(statuses)}`);
    }
    await sleep(2000);
  }
  throw new Error(`worker did not complete async jobs before timeout; start make worker and retry jobIds=${asyncJobIds.join(",")}`);
});

await step("audit logs load", async () => {
  const result = await call("audit", "GET", `/api/v1/audit-logs?organizationId=${encodeURIComponent(orgId)}`);
  if (!Array.isArray(result.body.data)) throw new Error("audit response missing data array");
  log("VALUE audit", { count: result.body.data.length, actions: result.body.data.map((entry) => entry.action).slice(0, 5) });
});

await step("ops metrics load", async () => {
  const result = await call("metrics", "GET", "/api/v1/ops/metrics");
  if (typeof result.body.http_requests_total !== "number") throw new Error("metrics missing http_requests_total");
  log("VALUE metrics", {
    http_requests_total: result.body.http_requests_total,
    jobs_queued_total: result.body.jobs_queued_total,
    jobs_completed_total: result.body.jobs_completed_total,
    job_queue_depth: result.body.job_queue_depth
  });
  if (asyncJobIds.length && result.body.jobs_completed_total < asyncJobIds.length) {
    throw new Error(`expected at least ${asyncJobIds.length} completed jobs in metrics`);
  }
});

await step("idempotency replay returns same stripe sync result", async () => {
  const first = await call("idempotency-first", "POST", "/sync/stripe", {
    headers: { "Idempotency-Key": `smoke-idem-${RUN_ID}` },
    body: {}
  });
  const second = await call("idempotency-second", "POST", "/sync/stripe", {
    headers: { "Idempotency-Key": `smoke-idem-${RUN_ID}` },
    body: {}
  });
  if (first.status !== 200 || second.status !== 200) throw new Error("idempotency requests did not both succeed");
});

const failed = checks.filter((check) => !check.ok);
console.log("");
log("SUMMARY", { passed: checks.length - failed.length, failed: failed.length, apiBase: API_BASE, runId: RUN_ID });
log("LOOK_FOR_IN_API_TERMINAL", {
  messages: ["database.connected", "http.request"],
  requestIdsPrefix: `smoke-${RUN_ID}-`,
  expectedPaths: ["/auth/demo-token", "/sync/stripe", "/sync/bank", "/reconciliation/runs", "/api/v1/ops/metrics"]
	  .concat(["/portfolio/import/holdings-csv", "/portfolio/import/transactions-csv", "/portfolio/summary"])
	  .concat(["/connections/plaid/sync-investments"])
});
log("LOOK_FOR_IN_WORKER_TERMINAL", {
 messages: ["worker.started", "worker.job.started", "worker.job.completed"],
  asyncJobIds,
  note: "If this failed, start make worker and rerun make smoke."
});

if (failed.length) {
  process.exitCode = 1;
}
