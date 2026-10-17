const assert = require("node:assert/strict");
const { spawn } = require("node:child_process");
const fs = require("node:fs/promises");
const path = require("node:path");
const http = require("node:http");

const root = path.resolve(__dirname, "..");
const usersFile = path.join(root, "data", "users.json");
const port = 31848;
const username = `p1-test-${Date.now()}`;
const password = "p1-test-password-2026";

const request = (method, route, body, token) => new Promise((resolve, reject) => {
  const payload = body ? JSON.stringify(body) : "";
  const req = http.request({ hostname: "127.0.0.1", port, path: route, method, headers: { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(payload), ...(token ? { Authorization: `Bearer ${token}` } : {}) } }, (res) => {
    let content = "";
    res.on("data", (chunk) => { content += chunk; });
    res.on("end", () => {
      let parsed = {};
      try { parsed = content ? JSON.parse(content) : {}; } catch { parsed = { raw: content }; }
      resolve({ status: res.statusCode, body: parsed });
    });
  });
  req.on("error", reject);
  req.end(payload);
});

const waitForServer = async () => {
  let lastError;
  for (let attempt = 0; attempt < 30; attempt += 1) {
    try { await request("GET", "/"); return; }
    catch (error) { lastError = error; await new Promise((resolve) => setTimeout(resolve, 100)); }
  }
  throw lastError;
};

(async () => {
  let usersBefore = null;
  try { usersBefore = await fs.readFile(usersFile, "utf8"); }
  catch (error) { if (error.code !== "ENOENT") throw error; }
  const child = spawn(process.execPath, ["server.js"], { cwd: root, env: { ...process.env, HOST: "0.0.0.0", PORT: String(port), NODE_ENV: "test" }, stdio: "pipe" });
  try {
    await waitForServer();
    const registration = await request("POST", "/api/register", { username, password });
    assert.equal(registration.status, 201, "Registration must succeed.");
    assert.ok(registration.body.token, "Registration must issue a token.");
    const unauthenticated = await request("GET", "/api/studio");
    assert.equal(unauthenticated.status, 401, "Missing token must be rejected.");
    const login = await request("POST", "/api/login", { username, password });
    assert.equal(login.status, 200, "Login must succeed.");
    assert.ok(login.body.token, "Login must issue a token.");
    assert.notEqual(login.body.token, registration.body.token, "Login must rotate the token.");
    const studio = await request("GET", "/api/studio", null, login.body.token);
    assert.equal(studio.status, 200, "Valid token must unlock the protected route.");
    console.log("P1 passed: registration, login, token rotation, authorization and 401 rejection work.");
  } finally {
    child.kill("SIGTERM");
    if (usersBefore === null) await fs.unlink(usersFile).catch(() => {});
    else await fs.writeFile(usersFile, usersBefore, { mode: 0o600 });
  }
})().catch((error) => { console.error(`P1 failed: ${error.message}`); process.exitCode = 1; });
