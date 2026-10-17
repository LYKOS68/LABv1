const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const fs = require("node:fs/promises");
const path = require("node:path");
const http = require("node:http");

const root = path.resolve(__dirname, "..");
const config = require("../config/app");
const dataDirectory = path.join(root, "data");
const port = 31847;

const request = () => new Promise((resolve, reject) => {
  const req = http.get(`http://127.0.0.1:${port}/`, (res) => { res.resume(); res.on("end", () => resolve(res.statusCode)); });
  req.on("error", reject);
  req.setTimeout(500, () => req.destroy(new Error("Server did not respond.")));
});

const waitForServer = async () => {
  let lastError;
  for (let attempt = 0; attempt < 30; attempt += 1) {
    try { return await request(); } catch (error) { lastError = error; await new Promise((resolve) => setTimeout(resolve, 100)); }
  }
  throw lastError;
};

(async () => {
  assert.equal(config.host, "0.0.0.0", "Configuration must load HOST from .env.");
  assert.equal(config.port, 3000, "Configuration must load PORT from .env.");
  assert.equal(config.environment, "development", "Configuration must load NODE_ENV from .env.");
  const child = spawn(process.execPath, ["server.js"], { cwd: root, env: { ...process.env, HOST: "0.0.0.0", PORT: String(port), NODE_ENV: "test" }, stdio: "pipe" });
  try {
    assert.equal(await waitForServer(), 200, "Server must return HTTP 200.");
    const stat = await fs.stat(dataDirectory);
    assert.equal(stat.isDirectory(), true, "data must be a directory.");
    assert.equal(stat.mode & 0o777, 0o700, "data must grant access to its owner only.");
    assert.equal(child.exitCode, null, "Server must remain running after startup.");
    console.log("P0 passed: .env loads; server starts; data directory has mode 0700.");
  } finally { child.kill("SIGTERM"); }
})().catch((error) => { console.error(`P0 failed: ${error.message}`); process.exitCode = 1; });
