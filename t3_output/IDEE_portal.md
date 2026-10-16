# RAP-LAB – Idee (festgehalten)

| Feld | Wert |
|---|---|
| Stand | 2026-10-16 |
| Status | IDEE – keine Spec, kein Bauauftrag |
| Quelle | Romans Aussagen im Chat dieser Session |

---

## Kern (von Roman entschieden)

Ein Portal wie eBay Kleinanzeigen – aber für Künstler und Artists aus der Musik.
Nutzer bieten an, was sie können; andere finden es.

- **App + Website**
- **Login** – Nutzer haben ein Konto (wie bei Kleinanzeigen)
- **Inserieren** – Anzeige erstellen und posten, mit **Foto- und Video-Upload**
- **Suchen und filtern** – Suche oben, auf den eigenen Standort gefiltert
- **Kategorien** als Navigationsleiste unten
- **Startseite** – Anzeigen als Kacheln mit Foto, anklickbar

Kleinanzeigen ist **Referenz für die Logik**, nicht für Farben, GUI oder Texte. Nichts wird übernommen.

**Später, getrennt:** Lyrics-Werkbank (Canvas + Song-Lyrics-Editor, vorhandene `main.go`). Entwickelt Roman später.

---

## Beispiele für Anzeigen (Romans Text)

- „Mix & Master – 50 € pro Song."
- „Beats für Rapper – individuelle Produktionen."
- „Ghostwriting – Hooks und Parts."
- „Musikvideo – Konzept, Dreh und Schnitt."
- Vocal Presets verkaufen
- Feature suchen

Leitsatz: *Du musst nur etwas können, das jemand anderes für sein Projekt braucht.*

---

## Offen (noch nicht entschieden)

| Nr. | Frage |
|---|---|
| O1 | Kategorien-Liste (Vorschlag: Beats · Mix & Master · Songwriting · Video & Cover · Mehr) |
| O2 | Biete **und** Suche als Anzeigenart? |
| O3 | Ort: Stadt/PLZ oder „online/deutschlandweit"; Umkreissuche später? |
| O4 | Felder einer Anzeige; Audio-Hörprobe zusätzlich zu Foto/Video? |
| O5 | Kontakt: Nachrichten im Portal oder externer Kontakt? |
| O6 | Bezahlung über das Portal: ja/nein? |
| O7 | Speicherung: Go-stdlib allein reicht für Login + Uploads nicht sicher – Datenbank-Treiber nötig? (Konflikt mit „nur stdlib") |
| O8 | „App": echte Android/iOS-App oder installierbare Website (PWA)? |
| O9 | Referenzbild und `main.go` – Pfad fehlt |
