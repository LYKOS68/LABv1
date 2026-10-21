# push-main.ps1
# Ersetzt den main-Branch auf GitHub vollstaendig durch den lokalen Stand.
# Vorher wird der alte GitHub-Stand als Tag "backup-main-<Datum>" gesichert.
# Starten: Doppelklick auf push-main.cmd

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
git remote get-url origin *> $null
if ($LASTEXITCODE -ne 0) { Stop-Script "Kein Remote 'origin' eingerichtet." }

# 2. Secrets-Schutz: .env darf nie hochgeladen werden
git check-ignore -q .env
if ((Test-Path .env) -and $LASTEXITCODE -ne 0) { Stop-Script ".env wird nicht von .gitignore ignoriert." }

# 3. Anzeigen, was sich aendert
Write-Host "Repo:   $root"
Write-Host "Ziel:   $(git remote get-url origin) -> main"
Write-Host ""
Write-Host "Lokale Aenderungen (A=neu, M=geaendert, D=geloescht, ??=neu):" -ForegroundColor Cyan
git status --short
Write-Host ""
Write-Host "ACHTUNG: GitHub-main wird danach exakt wie dein lokaler Ordner sein." -ForegroundColor Yellow
Write-Host "Was nur auf GitHub liegt, verschwindet dort (bleibt im Backup-Tag)." -ForegroundColor Yellow
$ok = Read-Host "Zum Fortfahren JA eintippen"
if ($ok -ne "JA") { Stop-Script "Nicht bestaetigt, nichts geaendert." }

# 4. Commit-Nachricht
$msg = Read-Host "Kurze Beschreibung der Aenderung (Enter = 'update')"
if ([string]::IsNullOrWhiteSpace($msg)) { $msg = "update" }

# 5. Alles committen
git add -A
git diff --cached --quiet
if ($LASTEXITCODE -ne 0) {
    git commit -m $msg
    if ($LASTEXITCODE -ne 0) { Stop-Script "Commit fehlgeschlagen." }
} else {
    Write-Host "Nichts Neues zu committen, lade vorhandene Commits hoch."
}

# 6. Backup des alten GitHub-Stands als Tag
git fetch origin main
if ($LASTEXITCODE -eq 0) {
    $tag = "backup-main-" + (Get-Date -Format "yyyy-MM-dd_HH-mm-ss")
    git push origin "refs/remotes/origin/main:refs/tags/$tag"
    if ($LASTEXITCODE -ne 0) { Stop-Script "Backup-Tag konnte nicht angelegt werden." }
    Write-Host "Backup angelegt: Tag $tag" -ForegroundColor Green
} else {
    Write-Host "Kein main auf GitHub gefunden, Backup uebersprungen."
}

# 7. GitHub-main ersetzen
git push --force-with-lease origin main
if ($LASTEXITCODE -ne 0) { Stop-Script "Push fehlgeschlagen." }

Write-Host ""
Write-Host "FERTIG: GitHub-main entspricht jetzt deinem lokalen Stand." -ForegroundColor Green
Read-Host "Enter zum Schliessen"
