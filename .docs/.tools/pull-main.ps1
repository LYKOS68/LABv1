# pull-main.ps1
# Macht den lokalen Ordner exakt gleich zum main-Branch auf GitHub.
# Vorher wird der lokale Stand gesichert:
#   - lokale Commits als Branch "backup-local-<Datum>"
#   - nicht committete und neue Dateien als Stash "backup-local-<Datum>"
# Ignorierte Dateien (.env, *.exe, ...) bleiben unberuehrt.
# Starten: siehe README.md

$ErrorActionPreference = "Continue"  # Git-Fehler werden ueber $LASTEXITCODE geprueft
$root = Resolve-Path (Join-Path $PSScriptRoot "..\..")
Set-Location $root

function Stop-Script($msg) {
    Write-Host ""
    Write-Host "ABBRUCH: $msg" -ForegroundColor Red
    Read-Host "Enter zum Schliessen"
    exit 1
}

# 1. Pruefen: richtiger Ordner, Branch main, Remote origin
git rev-parse --is-inside-work-tree *> $null
if ($LASTEXITCODE -ne 0) { Stop-Script "$root ist kein Git-Repo." }
$branch = git branch --show-current
if ($branch -ne "main") { Stop-Script "Du bist auf Branch '$branch', nicht auf 'main'." }

# 2. GitHub-Stand holen
git fetch origin main
if ($LASTEXITCODE -ne 0) { Stop-Script "GitHub-main konnte nicht geladen werden." }

# 3. Anzeigen, was sich aendert
Write-Host ""
Write-Host "Neue Commits von GitHub:" -ForegroundColor Cyan
git log --oneline HEAD..origin/main
Write-Host ""
Write-Host "Lokale Commits, die NICHT auf GitHub sind:" -ForegroundColor Cyan
git log --oneline origin/main..HEAD
Write-Host ""
Write-Host "Lokale Aenderungen (M=geaendert, D=geloescht, ??=neu):" -ForegroundColor Cyan
git status --short
Write-Host ""
Write-Host "ACHTUNG: Dein Ordner wird danach exakt wie GitHub-main sein." -ForegroundColor Yellow
Write-Host "Alles oben Gelistete verschwindet aus dem Ordner (bleibt im Backup)." -ForegroundColor Yellow
$ok = Read-Host "Zum Fortfahren JA eintippen"
if ($ok -ne "JA") { Stop-Script "Nicht bestaetigt, nichts geaendert." }

# 4. Backup des lokalen Stands
$tag = "backup-local-" + (Get-Date -Format "yyyy-MM-dd_HH-mm-ss")
git branch $tag
if ($LASTEXITCODE -ne 0) { Stop-Script "Backup-Branch konnte nicht angelegt werden." }
git stash push --include-untracked -m $tag
if ($LASTEXITCODE -ne 0) { Stop-Script "Backup der lokalen Aenderungen fehlgeschlagen." }
Write-Host "Backup angelegt: Branch und Stash $tag" -ForegroundColor Green

# 5. Ordner auf GitHub-main setzen
git reset --hard origin/main
if ($LASTEXITCODE -ne 0) { Stop-Script "Aktualisieren fehlgeschlagen. Backup: $tag" }

Write-Host ""
Write-Host "FERTIG: Dein Ordner entspricht jetzt GitHub-main." -ForegroundColor Green
Write-Host "Backup zurueckholen: siehe README.md, Name: $tag"
Read-Host "Enter zum Schliessen"
