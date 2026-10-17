// HINWEIS: Use server-side sessions for Rap-Lab P1. Persist only SHA-256 token hashes; rotate on login and revoke by replacing or removing the stored hash.
module.exports = Object.freeze({
  model: "server-side-session",
  tokenStorage: "sha256-hash-only",
  rotation: "every-successful-login",
  revocation: "replace-or-remove-token-hash"
});
