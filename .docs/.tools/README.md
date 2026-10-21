# .tools

Starten im Terminal (PowerShell) aus der Repo-Wurzel:

```powershell
powershell -ExecutionPolicy Bypass -File .\.docs\.tools\<SKRIPT>.ps1
```

| Skript | Zweck |
| --- | --- |
| `push-main.ps1` | Ersetzt `main` auf GitHub vollständig durch den lokalen Stand. Fragt vorher nach `JA`, sichert den alten GitHub-Stand als Tag `backup-main-<Datum>` und bricht ab, wenn `.env` nicht ignoriert wird. |
| `pull-main.ps1` | Macht den lokalen Ordner exakt gleich zu `main` auf GitHub. Fragt vorher nach `JA` und sichert lokale Commits und Änderungen als Branch und Stash `backup-local-<Datum>`. Ignorierte Dateien (`.env`, `*.exe`) bleiben unberührt. |

## push-main.ps1

```powershell
powershell -ExecutionPolicy Bypass -File .\.docs\.tools\push-main.ps1
```

Backup zurückholen: `git push --force origin backup-main-<Datum>:main`

## pull-main.ps1

```powershell
powershell -ExecutionPolicy Bypass -File .\.docs\.tools\pull-main.ps1
```

Backup zurückholen:

```powershell
git reset --hard backup-local-<Datum>
git stash apply "stash^{/backup-local-<Datum>}"
```
