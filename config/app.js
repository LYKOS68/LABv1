const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..");
const envFile = path.join(root, ".env");

const loadLocalEnvironment = () => {
  try {
    for (const line of fs.readFileSync(envFile, "utf8").split(/\r?\n/)) {
      const match = line.match(/^\s*([A-Z][A-Z0-9_]*)\s*=\s*(.*?)\s*$/);
      if (match && process.env[match[1]] === undefined) process.env[match[1]] = match[2];
    }
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
};

const requiredString = (name) => {
  const value = process.env[name];
  if (!value || !value.trim()) throw new Error(`${name} must be set.`);
  return value.trim();
};

const toPort = (value) => {
  const port = Number(value);
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error("PORT must be an integer from 1 to 65535.");
  return port;
};

loadLocalEnvironment();

module.exports = Object.freeze({
  environment: requiredString("NODE_ENV"),
  host: requiredString("HOST"),
  port: toPort(requiredString("PORT")),
  root,
  dataDirectory: path.join(root, "data"),
  sessionLifetimeMinutes: 60,
  maxUploadBytes: 10 * 1024 * 1024
});
