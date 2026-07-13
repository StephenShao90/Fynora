#!/usr/bin/env node

import { spawn } from "node:child_process";
import net from "node:net";

const root = new URL("..", import.meta.url).pathname;
const runSmoke = process.argv.includes("--smoke");

const managedPorts = [
  { name: "API", port: 8080, target: "make api" },
  { name: "Web", port: 3000, target: "make web" }
];

const children = [];
let shuttingDown = false;

function log(message, data = undefined) {
  if (data) {
    console.log(`[dev] ${message}`, data);
    return;
  }
  console.log(`[dev] ${message}`);
}

function run(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    log(`running ${[command, ...args].join(" ")}`);
    const child = spawn(command, args, {
      cwd: root,
      stdio: "inherit",
      env: process.env,
      ...options
    });
    child.on("error", reject);
    child.on("exit", (code, signal) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`${command} ${args.join(" ")} exited with ${signal || code}`));
    });
  });
}

function start(name, command, args, options = {}) {
  log(`starting ${name}: ${[command, ...args].join(" ")}`);
  const child = spawn(command, args, {
    cwd: root,
    stdio: "inherit",
    env: process.env,
    ...options
  });
  child.name = name;
  children.push(child);
  child.on("exit", (code, signal) => {
    if (shuttingDown) {
      return;
    }
    console.error(`[dev] ${name} exited unexpectedly with ${signal || code}`);
    shutdown(1);
  });
}

function isPortOpen(port) {
  return new Promise((resolve) => {
    const socket = net.createConnection({ host: "127.0.0.1", port });
    socket.setTimeout(500);
    socket.on("connect", () => {
      socket.destroy();
      resolve(true);
    });
    socket.on("timeout", () => {
      socket.destroy();
      resolve(false);
    });
    socket.on("error", () => resolve(false));
  });
}

async function assertPortsFree() {
  const occupied = [];
  for (const item of managedPorts) {
    if (await isPortOpen(item.port)) {
      occupied.push(item);
    }
  }
  if (occupied.length === 0) {
    return;
  }
  for (const item of occupied) {
    console.error(`[dev] port ${item.port} is already in use by another process; stop it before running make dev`);
  }
  process.exit(1);
}

async function waitForPostgres() {
  const attempts = 20;
  for (let i = 1; i <= attempts; i += 1) {
    const child = spawn("docker", ["compose", "exec", "-T", "postgres", "pg_isready", "-U", "postgres"], {
      cwd: root,
      stdio: "ignore",
      env: process.env
    });
    const ready = await new Promise((resolve) => {
      child.on("exit", (code) => resolve(code === 0));
      child.on("error", () => resolve(false));
    });
    if (ready) {
      log("postgres is ready");
      return;
    }
    log(`waiting for postgres (${i}/${attempts})`);
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error("postgres did not become ready");
}

async function waitForHttp(name, url) {
  const attempts = 60;
  for (let i = 1; i <= attempts; i += 1) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        log(`${name} is ready`);
        return;
      }
    } catch {
      // Keep polling until the child process finishes booting.
    }
    log(`waiting for ${name} (${i}/${attempts})`);
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  throw new Error(`${name} did not become ready at ${url}`);
}

function shutdown(code = 0) {
  if (shuttingDown) {
    return;
  }
  shuttingDown = true;
  log("stopping API, worker, and web");
  for (const child of children) {
    if (!child.killed) {
      child.kill("SIGTERM");
    }
  }
  setTimeout(() => process.exit(code), 750).unref();
}

process.on("SIGINT", () => shutdown(0));
process.on("SIGTERM", () => shutdown(0));

try {
  await assertPortsFree();
  await run("node", ["scripts/use-local-env.mjs"]);
  await run("docker", ["compose", "up", "-d", "postgres", "redis"]);
  await waitForPostgres();
  await run("make", ["migrate"]);

  start("api", "make", ["api"]);
  start("worker", "make", ["worker"]);
  start("web", "make", ["web"]);

  log("local stack is starting");
  log("open http://localhost:3000");
  log("press Ctrl+C here to stop API, worker, and web");

  if (runSmoke) {
    await waitForHttp("api", "http://127.0.0.1:8080/api/v1/health");
    await waitForHttp("web", "http://127.0.0.1:3000");
    await run("node", ["scripts/smoke-clearflow.mjs"]);
    log("smoke verification passed");
    shutdown(0);
  }
} catch (error) {
  console.error(`[dev] ${error.message}`);
  shutdown(1);
}
