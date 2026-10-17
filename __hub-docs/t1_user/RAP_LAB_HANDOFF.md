# Rap-Lab — Handoff

## Scope

- Project: `/sdcard/GPEA-DRAFTS/projekte/rap-lab`
- Product: classifieds-style Rap marketplace; no in-platform payment.
- Work in small vertical phases. Do not begin a phase without a lock and a smoke test.

## Delivered

- P0: `.env` loader + validation, `config/app.js`, `0.0.0.0:3000`, data directory initialization, P0 smoke.
- P1: registration, `scrypt` password hashing, server-side token-hash sessions, token rotation at login, protected `/api/studio` route, P1 smoke.
- Session policy: `config/session-policy.js` documents the server-side-session choice.

## Test commands

```bash
npm run test:p0
npm run test:p1
```

P0 and P1 passed before the project was moved from `/root` to `/sdcard`.

## Critical storage constraint

`/sdcard` is Android shared storage. Its `data/` directory cannot reliably enforce Unix mode `0700` (observed: `2770`). Do not store real user records, session data, or secrets there. Keep source code here; move private runtime data to Termux-internal storage before real users exist.

## Locks and smokes

- Lock: missing/invalid token -> `401`; Smoke: register -> login -> rotated token -> protected route -> `200`.
- Lock: data must be owner-only; Smoke: inspect actual mode. This lock currently fails on `/sdcard` and needs an internal data path.
- Review diffs before edits; preserve existing `t1_workflow`, `t2_source`, and `t3_output` content.

## Next: P2

Build profiles and authorization: public/private fields, owner-only edits, role model (`member`, `moderator`, `admin`), resource ownership checks, and a smoke proving user A cannot read or edit user B data.
