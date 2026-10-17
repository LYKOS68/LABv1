const http = require("node:http");
const fs = require("node:fs/promises");
const path = require("node:path");
const crypto = require("node:crypto");
const config = require("./config/app");

const { port: PORT, host: HOST, root: ROOT, dataDirectory: DATA } = config;
const PUBLIC = path.join(ROOT, "public");
const USERS_FILE = path.join(DATA, "users.json");

const json = (res, status, body) => {
  res.writeHead(status, { "Content-Type": "application/json; charset=utf-8" });
  res.end(JSON.stringify(body));
};

const initializeStorage = async () => {
  await fs.mkdir(DATA, { recursive: true, mode: 0o700 });
  await fs.chmod(DATA, 0o700);
};

const readBody = async (req) => {
  let data = "";
  for await (const chunk of req) {
    data += chunk;
    if (data.length > 16_384) throw new Error("Request too large");
  }
  return data ? JSON.parse(data) : {};
};

const loadUsers = async () => {
  await initializeStorage();
  try { return JSON.parse(await fs.readFile(USERS_FILE, "utf8")); }
  catch (error) { if (error.code === "ENOENT") return []; throw error; }
};

const saveUsers = async (users) => {
  await initializeStorage();
  const temp = `${USERS_FILE}.tmp`;
  await fs.writeFile(temp, `${JSON.stringify(users, null, 2)}\n`, { mode: 0o600 });
  await fs.rename(temp, USERS_FILE);
  await fs.chmod(USERS_FILE, 0o600);
};

const hashPassword = (password, salt = crypto.randomBytes(16).toString("hex")) => ({
  salt, hash: crypto.scryptSync(password, salt, 64).toString("hex")
});
const validPassword = (password, user) => {
  const candidate = Buffer.from(hashPassword(password, user.passwordSalt).hash, "hex");
  const known = Buffer.from(user.passwordHash, "hex");
  return candidate.length === known.length && crypto.timingSafeEqual(candidate, known);
};
const issueToken = () => crypto.randomBytes(32).toString("base64url");
const tokenHash = (token) => crypto.createHash("sha256").update(token).digest("hex");
const publicUser = (user) => ({ id: user.id, username: user.username, createdAt: user.createdAt });

const authenticate = async (req) => {
  const header = req.headers.authorization || "";
  const token = header.startsWith("Bearer ") ? header.slice(7) : "";
  if (!token) return null;
  const users = await loadUsers();
  const user = users.find((entry) => entry.tokenHash === tokenHash(token));
  return user ? { user, users } : null;
};

const api = async (req, res) => {
  if (req.method === "POST" && req.url === "/api/register") {
    const { username = "", password = "" } = await readBody(req);
    const normalized = username.trim().toLowerCase();
    if (!/^[a-z0-9_-]{3,24}$/.test(normalized)) return json(res, 400, { error: "Username: 3–24 chars; a-z, 0-9, _ or -." });
    if (typeof password !== "string" || password.length < 10) return json(res, 400, { error: "Password must contain at least 10 characters." });
    const users = await loadUsers();
    if (users.some((user) => user.username === normalized)) return json(res, 409, { error: "Username already exists." });
    const passwordData = hashPassword(password);
    const token = issueToken();
    const user = { id: crypto.randomUUID(), username: normalized, passwordSalt: passwordData.salt, passwordHash: passwordData.hash, tokenHash: tokenHash(token), createdAt: new Date().toISOString(), lastLoginAt: new Date().toISOString() };
    users.push(user);
    await saveUsers(users);
    return json(res, 201, { user: publicUser(user), token });
  }
  if (req.method === "POST" && req.url === "/api/login") {
    const { username = "", password = "" } = await readBody(req);
    const users = await loadUsers();
    const user = users.find((entry) => entry.username === String(username).trim().toLowerCase());
    if (!user || !validPassword(String(password), user)) return json(res, 401, { error: "Invalid username or password." });
    const token = issueToken();
    user.tokenHash = tokenHash(token); user.lastLoginAt = new Date().toISOString();
    await saveUsers(users);
    return json(res, 200, { user: publicUser(user), token });
  }
  if (req.method === "GET" && req.url === "/api/studio") {
    const session = await authenticate(req);
    if (!session) return json(res, 401, { error: "Studio access requires a valid token." });
    return json(res, 200, { user: publicUser(session.user), message: "Studio unlocked." });
  }
  return false;
};

const serveStatic = async (req, res) => {
  const requested = req.url === "/" ? "/index.html" : req.url;
  const safePath = path.normalize(requested).replace(/^([.][.][/\\])+/, "");
  const file = path.join(PUBLIC, safePath);
  if (!file.startsWith(PUBLIC)) return json(res, 403, { error: "Forbidden" });
  try {
    const content = await fs.readFile(file);
    const contentType = file.endsWith(".css") ? "text/css" : file.endsWith(".js") ? "text/javascript" : "text/html";
    res.writeHead(200, { "Content-Type": `${contentType}; charset=utf-8` }); res.end(content);
  } catch { json(res, 404, { error: "Not found" }); }
};

const server = http.createServer(async (req, res) => {
  try { const handled = await api(req, res); if (handled === false) await serveStatic(req, res); }
  catch (error) { console.error(error); json(res, 500, { error: "Server error" }); }
});

initializeStorage()
  .then(() => server.listen(PORT, HOST, () => console.log(`Rap-Lab: http://${HOST}:${PORT} (${config.environment})`)))
  .catch((error) => { console.error("Storage initialization failed:", error); process.exitCode = 1; });
