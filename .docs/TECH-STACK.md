# TECH-STACK und Code-Richtlinie RAP-LAB

Verbindlich für jedes CLI-Modell. Änderungen nur mit Romans {Freigabe}.
Lokale Umgebungsdaten (Rechner, Netzwerk, Datenbankdienst) stehen nicht hier, sondern ausschließlich in `__hub-docs/` und gehören nicht ins Git.

## §1 Stack

(1) Sprache: Go 1.27 laut `go.mod`.
(2) HTTP: `net/http` der Standardbibliothek.
(3) HTML: `html/template` mit `embed`.
(4) Interaktion: htmx als lokale Datei, eigenes JavaScript nur wo nötig. Jede Seite funktioniert ohne JavaScript.
(5) Datenbank: PostgreSQL über `pgx/v5`. SQL wird von Hand geschrieben.
(6) Migrationen: eigener Runner mit der Tabelle `schema_migrations`.
(7) Passwörter: `bcrypt` aus `golang.org/x/crypto`.
(7a) Bildskalierung: `golang.org/x/image/draw`. Freigegeben von Roman am 2026-10-21.
(8) CSRF-Schutz: `http.CrossOriginProtection`.
(9) Logging: `log/slog`.
(10) E-Mail: `net/smtp`. Lokal werden Mails in die Konsole ausgegeben. Anbieter für den Online-Betrieb {LÜCKE}.
(11) Umkreissuche: Tabelle mit PLZ und Koordinaten, Entfernung wird in SQL berechnet. Datenquelle und Lizenz {LÜCKE}.
(12) Live-Nachrichten: Server-Sent Events, eine Goroutine je Verbindung.
(13) Hintergrundjobs: Goroutine-Worker für Mailversand und Sitzungsbereinigung.
(14) Schrift: Manrope als lokale `.woff2`-Datei.
(15) Design: nur `.docs/ui-design-tokens.css`, eigenes CSS, nur Dark Mode. Ein heller Modus ist Zukunftsidee.
(16) Versionen von htmx und Schrift werden beim Einbau aus der offiziellen Quelle geprüft und festgeschrieben.

## §2 Abhängigkeiten

(1) Erlaubt sind nur die Standardbibliothek und die Pakete aus §1.
(2) Jedes weitere Paket braucht Romans {Freigabe}.
(3) Kein CDN. Alle CSS-, JS-, Schrift- und Bilddateien werden in die Binärdatei eingebettet.

## §3 Struktur

(1) `cmd/lab-studio/` enthält nur Start, Konfiguration und Verdrahtung.
(2) `cmd/lab-studio/migrations/` enthält nummerierte SQL-Dateien `NNN_name.sql`.
(3) SQL steht ausschließlich in `internal/store/`.
(4) HTTP-Logik steht ausschließlich in `internal/handlers/`.
(5) Templates und statische Dateien stehen ausschließlich in `internal/web/`.
(6) Keine Schicht übernimmt die Aufgabe einer anderen.

## §4 Datenbank

(1) Jede Schemaänderung ist eine neue Migrationsdatei. Bestehende Migrationen werden nie geändert.
(2) Mehrteilige Schreibvorgänge laufen in einer Transaktion.
(3) Preise werden als Ganzzahl in Cent gespeichert.

## §5 Fehler

(1) Kein Fehler wird ignoriert. `_ =` auf einen Fehler ist verboten.
(2) Fehler werden mit `fmt.Errorf("kontext: %w", err)` weitergereicht.
(3) Fehler werden genau einmal im Handler geloggt.
(4) Der Nutzer sieht deutsche Fehlermeldungen ohne interne Details.

## §6 Sicherheit

(1) Jede Eingabe wird auf dem Server geprüft.
(2) CSRF-Schutz ist auf allen schreibenden Routen aktiv.
(3) Cookies sind `HttpOnly` und `SameSite=Lax`, online zusätzlich `Secure`.
(4) Nutzertext wird nie als `template.HTML` ausgegeben.
(5) Secrets nur aus Umgebungsvariablen, nie in Code, Logs oder Dateien unter Git.

## §7 Goroutinen

(1) Jede Goroutine erhält einen `context` und hat ein definiertes Ende.
(2) Goroutinen ohne Abbruchweg sind verboten.
(3) Beim Herunterfahren werden alle Worker sauber beendet.

## §8 Design

(1) Farben, Abstände, Rundungen und Schrift nur aus den Tokens. Keine frei gesetzten Farbwerte.
(2) Mobile-first ab 360px Breite.
(3) Neon-Grün nur für Hauptaktionen. Gold für Labels, Preise und aktive Zustände.
(4) Bilder nur aus `C:/Users/RPC/dev/rap-app/my-space/__source-docs/bilder`, kopiert, nicht verlinkt. Das Quellprojekt wird nicht verändert.

## §9 Frontend

(1) Architektur: Der Go-Server rendert fertiges HTML. Keine Single-Page-App, kein React, Vue oder Svelte, kein npm, kein Bundler, kein Build-Schritt für JavaScript.
(2) Templates: Basis-Layout `layout.html` mit Kopf, Navigation und Fuß. Jede Seite ist eine Datei unter `internal/web/templates/pages/`. Wiederverwendbare Bausteine liegen unter `internal/web/templates/partials/`. htmx-Antworten liefern nur Partials.
(3) htmx: nur für Teilaktualisierungen (Filter und Umkreissuche, Kommentar senden, Nachrichten nachladen, Formularprüfung, Favorit setzen). Jedes Formular und jeder Link funktioniert ohne htmx als normaler Seitenaufruf. Live-Nachrichten per SSE-Erweiterung von htmx oder eigenem `EventSource`.
(4) JavaScript: kleine eigene Module unter `internal/web/static/js/`, ein Zweck pro Datei, nur für Bedienkomfort und Effekte. Geschäftslogik und Prüfregeln immer auf dem Server. Kein Inline-JavaScript.
(5) CSS-Dateien in fester Reihenfolge: `tokens.css`, `base.css`, `layout.css`, `components.css`, `utilities.css`. Kein Inline-CSS.
(6) CSS-Regeln: Klassennamen nach Baustein (`.card`, `.card__price`, `.card--featured`). Grid und Flexbox. `@media` ab 360px aufwärts, Bausteine per `@container`. Schriftgrößen mit `clamp()`. Kein `!important`.
(7) Navigation: Handy mit fester unterer Leiste (Entdecken, Favoriten, Inserieren, Nachrichten, Profil). Ab 960px Kopfleiste mit denselben Zielen. Krone in der Navigation, Schriftzug im Hero.
(8) Bausteine Version 1: Button (primär, sekundär, Text), Eingabefeld, Textfeld, Auswahlfeld, Umschalter biete/suche, Kategorie-Kachel, Anzeigenkarte, Anzeigen-Detail, Kommentar, Nachrichtenliste, Nachrichtenverlauf, Profilkopf, Rollen-Badge, Hinweis-Meldung, Leerzustand, Ladezustand.
(9) Zustände: Jeder bedienbare Baustein hat Standard, Hover, Fokus, Gedrückt, Deaktiviert und Laden. Listen haben einen Leerzustand, Formulare einen Fehlerzustand.
(10) Bewegung: nur `transform` und `opacity`, 150 bis 300 ms. Neon-Glow nur bei Fokus, Hover und aktiver Aktion. `prefers-reduced-motion` schaltet Animationen ab.
(11) Barrierefreiheit: Kontrast mindestens 4,5:1 für Text und 3:1 für Bedienelemente. Sichtbarer Fokusring. Touch-Ziele mindestens 44 × 44 px. Semantisches HTML (`button`, `nav`, `main`, `label`). Alt-Text für jedes Bild. Volle Tastaturbedienung.
(12) Bilder: Markenbilder beim Einbau in passende Größen als WebP mit JPEG-Fallback. `loading="lazy"` unterhalb des sichtbaren Bereichs. Breite und Höhe immer angegeben.
(13) Performance: höchstens 300 KB pro Seite beim ersten Laden, Markenbilder ausgenommen. Statische Dateien mit Hash im Dateinamen und langen Cache-Zeiten.
(14) Texte: alle sichtbaren Texte in den Templates, nicht im Go-Code. Ansprache in der Du-Form. Leitfrage: „Was kann ich selbst, und wen brauche ich für den Rest?“

## §10 Sprache

(1) Alles, was Nutzer sehen, ist Deutsch.
(2) Code ist komplett englisch: Bezeichner, Dateinamen, Log-Meldungen, Testnamen.
(3) Kommentare sind deutsch und erklärend: Sie sagen, warum etwas so gebaut ist, nicht was die Zeile tut.

## §11 Umfang Version 1

(1) Enthalten: Registrierung und Login mit E-Mail und Bestätigungsmail, Profil mit beliebig vielen Rollen, Kleinanzeigen (biete/suche) mit einem Preisfeld und Umkreissuche, offene Kommentare unter Anzeigen, Nachrichten.
(2) Nicht enthalten: Bezahlung, Ranglisten, Audio- und Video-Snippets, Studio- und Label-Accounts, Login mit Google.
(3) Bild-Upload ist Teil von Version 1: ein Profilbild pro Nutzer und höchstens drei Bilder pro Anzeige, nur JPEG und PNG, höchstens 5 MB pro Datei.
(4) Hochgeladene Bilder werden auf dem Server mit der Standardbibliothek neu kodiert. Dabei fallen Metadaten wie GPS-Position weg. Die Skalierung auf feste Größen erfolgt mit `golang.org/x/image/draw` (§1(7a)).
(5) Uploads liegen lokal außerhalb von `internal/web/` und werden nur über einen Handler ausgeliefert. Der Speicherort ist per Umgebungsvariable einstellbar.

## §12 Prüfungen

(1) Nach jeder Änderung: `gofmt`, `go build ./...`, `go vet ./...`, `go test ./...`.
(2) Ein Test prüft, dass alle Templates fehlerfrei parsen.

## §13 Code-Stil

(1) Die Formatierung macht ausschließlich `gofmt` (Tabs). Keine Einrückung von Hand.
(2) Bezeichner werden klein geschrieben. Groß nur, wenn ein anderes Package sie braucht.
(3) `_` nur für bewusst ungenutzte Werte, die kein `error` sind. §5(1) bleibt bestehen.
(4) HTTP-Routen stehen ausschließlich als Tabelle im `ServeMux`, eine Zeile pro Route.
(5) Andere Verteiler (zum Beispiel Nachrichtentypen bei SSE) sind eine `map` mit `ok`-Prüfung. `switch` ist für wenige feste Fälle erlaubt.
(6) Funktionen stehen in Lesereihenfolge: zuerst öffentliche, dann die privaten Helfer darunter.
