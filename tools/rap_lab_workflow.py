#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Erzeugt ein überprüfbares RAP-LAB-Auditpaket samt Prompt-Entwürfen.

Standardmäßig read-only: Ausgabe auf stdout, keine Prompts ausführen, keine
Git-Änderungen. --output schreibt nur in einen neuen Pfad innerhalb der Repo-Wurzel.
Secret-Dateien wie .env werden nie eingelesen.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path

REQUIRED = (
    "VERTRAG.md",
    ".docs/TECH-STACK.md",
    "go.mod",
    ".docs/project_state.jsonl",
    ".docs/to-dos.jsonl",
)
SECRET_NAMES = {".env", ".env.local", ".env.production"}
SKIP_DIRS = {".git", ".idea", ".vscode", "node_modules", "vendor"}
SOURCE_SUFFIXES = {".go", ".html", ".tmpl", ".css", ".js", ".sql"}


def git(root: Path, *args: str) -> dict:
    """Liest Git-Zustand, ohne ihn zu verändern."""
    try:
        result = subprocess.run(
            ["git", "-C", str(root), *args], capture_output=True, text=True,
            encoding="utf-8", errors="replace", check=False,
        )
        return {"args": list(args), "exit_code": result.returncode,
                "stdout": result.stdout.strip(), "stderr": result.stderr.strip()}
    except OSError as exc:
        return {"args": list(args), "exit_code": None, "error": str(exc)}


def read_text(path: Path) -> str | None:
    try:
        return path.read_text(encoding="utf-8")
    except (OSError, UnicodeError):
        return None


def source_record(root: Path, relative: str) -> dict:
    path = root / relative
    if not path.is_file():
        return {"path": relative, "status": "MISSING"}
    text = read_text(path)
    if text is None:
        return {"path": relative, "status": "UNREADABLE"}
    raw = text.encode("utf-8")
    return {"path": relative, "status": "READ", "bytes": len(raw),
            "lines": len(text.splitlines()), "sha256": hashlib.sha256(raw).hexdigest()}


def discover_files(root: Path) -> list[Path]:
    found = []
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        rel = path.relative_to(root)
        if any(part in SKIP_DIRS for part in rel.parts):
            continue
        if any(part in SECRET_NAMES for part in rel.parts):
            continue
        if path.suffix in SOURCE_SUFFIXES or path.name.endswith("_test.go"):
            found.append(path)
    return sorted(found, key=lambda p: p.relative_to(root).as_posix().lower())


def inventory(root: Path) -> dict:
    routes, migrations, templates, static_files, tests, go_files = [], [], [], [], [], []
    pattern = re.compile(
        r'mux\.HandleFunc\(\s*"([^"]+)"|mux\.Handle\(\s*"([^"]+)"|'
        r'http\.HandleFunc\(\s*"([^"]+)"'
    )
    for path in discover_files(root):
        rel = path.relative_to(root).as_posix()
        if path.name.endswith("_test.go"):
            tests.append(rel)
        if path.suffix == ".go":
            go_files.append(rel)
            for match in pattern.finditer(read_text(path) or ""):
                routes.append({"file": rel, "route": next(x for x in match.groups() if x)})
        if path.suffix == ".sql" and "migration" in rel.lower():
            migrations.append(rel)
        if path.suffix in {".html", ".tmpl"}:
            templates.append(rel)
        if "/static/" in f"/{rel}/" or "/assets/" in f"/{rel}/":
            static_files.append(rel)
    return {
        "go_files": go_files, "http_route_declarations": routes,
        "migration_files": migrations, "template_files": templates,
        "static_files": static_files, "test_files": tests,
        "counts": {"go": len(go_files), "routes": len(routes),
                   "migrations": len(migrations), "templates": len(templates),
                   "static": len(static_files), "tests": len(tests)},
    }


def prompt_chain() -> list[dict]:
    """Prompt-Kette als Entwurf; keiner der Schritte wird automatisch ausgeführt."""
    stages = [
        ("P00", "Ist-Zustand und verbindliche Quellen sichern",
         ["Pflichtquellen mit Hash", "Branch/Status", "Datei- und Routeninventur"],
         ["Pflichtquelle fehlt", "Git-Zustand unbekannt"]),
        ("P01", "Ist/Soll vergleichen und LÜCKE-Punkte sammeln",
         ["Fakten/Annahmen getrennt", "belegte Abweichungen", "Entscheidungsfragen"],
         ["widersprüchliche Vorgabe", "ungeklärte Produktentscheidung"]),
        ("P02", "Funktionsinventur und Akzeptanzkriterien entwerfen",
         ["Prioritäten", "messbare Kriterien", "Abhängigkeiten"],
         ["Kriterium ohne Beleg", "V1-Ausschluss verletzt"]),
        ("P03", "Architektur, Datenmodell und HTTP-Verträge entwerfen",
         ["Komponenten- und Datenflussdiagramm", "Datenmodell", "Beispiel-Payloads"],
         ["neuer Stack nötig", "Berechtigungs- oder Schemafrage offen"]),
        ("P04", "Screenshots einholen und kleinen Designentwurf vorbereiten",
         ["begrenzter v0", "Abweichungsliste", "Freigabepunkt"],
         ["Referenz fehlt", "Design nicht freigegeben"]),
        ("P05", "Genau einen freigegebenen Nutzerablauf implementieren",
         ["kleiner Diff", "Tests", "Änderungsprotokoll"],
         ["Test schlägt fehl", "Änderung außerhalb freigegebener Pfade"]),
        ("P06", "Ablauf integrieren und Akzeptanzkriterien prüfen",
         ["Prüfbericht", "bekannte Grenzen", "Handoff"],
         ["Runtime/DB nicht prüfbar", "Sicherheitsfehler"]),
    ]
    return [{
        "id": ident, "purpose": purpose, "inputs": "nur belegte Ergebnisse des vorherigen Schritts",
        "allowed_changes": "nur der ausdrücklich freigegebene Umfang; sonst keine Änderung",
        "deliverables": deliverables,
        "checks": ["gofmt", "go build ./...", "go vet ./...", "go test ./...",
                   "Template-Parsing; relevante Route-/DB-Smokes soweit verfügbar"],
        "stop_conditions": stops, "auto_execute": False,
        "next": stages[i + 1][0] if i + 1 < len(stages) else None,
    } for i, (ident, purpose, deliverables, stops) in enumerate(stages)]


def gaps() -> list[dict]:
    return [
        {"id": "LÜCKE-01", "question": "Bitte Referenz-Screenshots für das Frontend bereitstellen.",
         "blocks": ["vollständige Frontend-Gestaltung"], "source": "Projektzustand 2026-10-21"},
        {"id": "LÜCKE-02", "question": "Soll die Navigation einen separaten Kategorien-Punkt erhalten?",
         "blocks": ["finale Navigation"], "source": "TECH-STACK §9(7) und Projektzustand"},
        {"id": "LÜCKE-03", "question": "Welche lizenzierte PLZ-/Koordinatenquelle ist für Umkreissuche freigegeben?",
         "blocks": ["produktive Umkreissuche"], "source": "TECH-STACK §1(11)"},
        {"id": "LÜCKE-04", "question": "Welcher E-Mail-Anbieter ist für den Onlinebetrieb freigegeben?",
         "blocks": ["produktive Bestätigungsmails"], "source": "TECH-STACK §1(10)"},
        {"id": "LÜCKE-05", "question": "Welche konkreten Berechtigungen gelten für Kommentare und Direktnachrichten?",
         "blocks": ["finale Nachrichten-/Kommentarverträge"], "source": "aus MVP-Dateien nicht vollständig belegbar"},
    ]


def build_package(root: Path) -> dict:
    required = [source_record(root, item) for item in REQUIRED]
    return {
        "schema_version": "1.0",
        "package_type": "rap-lab-audit-and-prompt-chain-draft",
        "generated_at_utc": datetime.now(timezone.utc).replace(microsecond=0).isoformat(),
        "repository": "LYKOS68/LABv1",
        "mode": "read-only by default",
        "execution_policy": {"prompts_executed": False, "secrets_read": False,
                             "main_modified": False, "commit_or_push_performed": False},
        "git": {"branch": git(root, "branch", "--show-current"),
                "status": git(root, "status", "--short", "--branch"),
                "head": git(root, "rev-parse", "HEAD"),
                "main_head": git(root, "rev-parse", "main")},
        "required_sources": required,
        "inventory": inventory(root),
        "gaps": gaps(),
        "prompt_chain_drafts": prompt_chain(),
        "verification": {"go_build": "NOT_RUN", "go_vet": "NOT_RUN",
                         "go_test": "NOT_RUN", "postgres_runtime": "NOT_CHECKED",
                         "http_runtime": "NOT_CHECKED",
                         "note": "Der Bericht kennzeichnet nicht ausgeführte Prüfungen ausdrücklich."},
        "next_gate": "Screenshots und LÜCKE-Entscheidungen einholen; vollständiges Frontend nicht vor Designfreigabe bauen.",
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path.cwd(), help="Repository-Wurzel")
    parser.add_argument("--output", type=Path, help="Optional: neue JSON-Datei innerhalb der Repo-Wurzel")
    args = parser.parse_args()
    root = args.root.resolve()
    if not root.is_dir():
        print(f"Fehler: keine Verzeichniswurzel: {root}", file=sys.stderr)
        return 2
    package = build_package(root)
    rendered = json.dumps(package, ensure_ascii=False, indent=2) + "\n"
    if args.output:
        target = args.output.resolve()
        if root not in target.parents:
            print("Fehler: --output muss innerhalb der Repo-Wurzel liegen.", file=sys.stderr)
            return 2
        if target.exists():
            print(f"Fehler: Überschreiben verweigert: {target}", file=sys.stderr)
            return 2
        try:
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(rendered, encoding="utf-8")
        except OSError as exc:
            print(f"Fehler beim Schreiben: {exc}", file=sys.stderr)
            return 1
        print(f"[1/1] Paket erstellt: {target.relative_to(root).as_posix()}")
    else:
        sys.stdout.write(rendered)
    return 1 if any(row["status"] != "READ" for row in package["required_sources"]) else 0


if __name__ == "__main__":
    raise SystemExit(main())
