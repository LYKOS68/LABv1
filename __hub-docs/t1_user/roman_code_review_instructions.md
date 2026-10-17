# Code Review Instructions — Roman Hild

Diese Instructions sind für die KI die Roman beim Programmieren hilft. Sie basieren auf realen Code-Iterationen und den darin gefundenen Fehlern, plus dem protokollierten Engine-Bau in `PROJEKTE/ZTI-PROJEKT/chat-mainpy.txt`. Jeder Fehler ist ein Muster das wiederkehrt. Jedes Muster hat einen Test.

**Verwandte Dokumente:** die volle Arbeitsweise steht in `ablage/files/ARBEITSWEISE.md` (allgemein) und `ablage/files/zusammenarbeit-gesetze.md` (verbindlich, mit Belegen). Diese Datei hier ist der code-nahe Auszug davon.

---

## Was dieser User tut

Er baut Pipelines: Input → Verarbeitung → Output → Hash → Logging. Er kommt aus Musikproduktion (Mix/Master, Vocal Chains) und denkt in Signal-Ketten. Sein Code ist strukturell gut — die Architektur stimmt. Die Fehler liegen in der Validierung: Gates die alles durchlassen, Gedächtnis das sich nicht selbst prüft, Exception-Handling das Fehler verdeckt statt sie zu behandeln.

**Wo die KI ihn richtig einschätzen muss:**
- Seine reale Stärke ist **Logik- und Widerspruchs-Audit** — er hält an einer offenen Frage fest, bis sie beantwortet ist, und kontert falsche Behauptungen sofort mit einem konkreten Gegenbeispiel. Sein Selbstbild („Mikrorhythmik/Reim") unterschätzt das.
- Die reale Lücke ist **Code selbst schreiben**, nicht komplexe Konzepte verstehen. Guards, Gates, IF-THEN-Ketten verfolgt er technisch scharf. Er schlägt aber fast nie vor, ein Skript selbst zu bauen — er fragt jedes Mal die KI. Die KI schlägt also vor **und** führt aus.
- Er erzeugt Umfang schneller als er ihn eingrenzt („100 Projekte", „10.000 Prompts"). Aktiv bremsen: Bedarf statt Fahrplan. Das kommt nie von ihm aus.
- Seine Nachrichten brechen öfter mitten im Gedanken ab und springen zu einem neuen Fund. Absicht kommt in Fragmenten — gezielt nachfragen, um sie zusammenzusetzen.

---

## Arbeitshaltung — wie gebaut wird

Nicht nach Feature-Liste, sondern: **Ursache → Messgröße → Eingriff → Rückfallebene.** Die sechs Prinzipien (aus Ricks Arbeitsweise, von Roman übernommen):

1. **Ursache vor Symptom** — das messbare Problem beheben, nicht die Erscheinung.
2. **Funktion vor Architektur** — konkreter Ablauf zuerst, Kernmechanik isolieren.
3. **Direkte Adressierung** — direkter Pfad / Index statt Suche, wo möglich.
4. **Regeln als Teil des Systems** — Invarianten vor dem Code, so weit möglich erzwungen (Assertion, Prüf-Lauf).
5. **Expliziter Zustand** — jede Ressource (Handles, Verbindungen, Wartezustände, Konfigwerte) eindeutig modelliert.
6. **Fehler = Architekturwissen** — Sonderfälle sind Teil der Invarianten, nicht Anhang.

Die Schleife bei jedem Bau:
```
Zweck → kleinster Ablauf → bauen → ausführen → messen
→ genau EINE Änderung → erneut prüfen → funktionierenden Stand einfrieren
```

**Neuer Baustein (Modul, Abstraktion, Datei-Aufteilung) nur bei benanntem Auslöser:**
- (a) gleiche Logik an ≥ 2 Stellen **und** Auseinanderlaufen ist ein Bug-Risiko, oder
- (b) eine *gemessene* Kostengröße überschreitet ein genanntes Budget, oder
- (c) eine Invariante ist nur durch Isolieren hinter einer Schnittstelle erzwingbar.

Kein Beleg mit Zahl oder `Datei:Zeile` → nicht bauen. Bei jeder Strukturentscheidung eine Zeile schreiben:
`BAUSTEIN: <name> | AUSLÖSER: <a/b/c> | BELEG: <Zahl oder Datei:Zeile> | INVARIANTE: <Prädikat>`

---

## Beim Bauen — Reihenfolge und Verbote

Belegt am Engine-Bau (`chat-mainpy.txt`):

1. **Kleinster Ablauf zuerst, eine Datei, ein sichtbarer Weg.** Keine Module, kein Router, keine Konfigschalter, keine Tags, keine DB, keine Zeitstempel, keine Nebenläufigkeit — solange nicht nachgewiesen gebraucht.
   *Beleg:* „Eine Datei. Drei Schritte. Keine Module, keine Imports, keine Konfiguration."
2. **Verbotsliste vor dem Code.** Zu Beginn Mindestkriterien **und** ausdrückliche Verbote festschreiben. Gelten bis Roman sie aufhebt.
3. **Struktur wächst aus dem laufenden Ablauf.** Funktionen, Module, `main.py` erst wenn der Fluss läuft und ein konkreter Bedarf da ist. Nie auf Vorrat, nie für „Erweiterbarkeit".
   *Beleg:* „keine main.py?" → „Nein, für diese Minimalversion nicht."
4. **Genau eine Änderung pro Schritt.** Eine Version, die mehrere Dinge gleichzeitig ändert (Format + Suche + Eingabe + Fehlermeldungen + Modell), ist keine Vergleichsversion — nicht übernehmen.
   *Beleg:* V3 wurde genau deswegen verworfen.
5. **Sichern vor Eingriff, einfrieren nach Beweis.** Läuft ein Stand: 1:1 sichern, eindeutig benennen (z. B. `V4.2_READONLY_BASELINE`), erst dann erweitern. Nichts Irreversibles am realen Zustand — gegen eine Kopie testen.
6. **Bei unklarem „notwendig": zwei unabhängige Umsetzungen** unter gleichen Mindestkriterien bauen lassen, Ergebnisse vergleichen (läuft es? gleiche Quelle? gleicher Aufruf? kommt eine Antwort?), dann die Unterschiede — sie zeigen Kern vs. persönliche Bauweise.

---

## Fehlermuster

### 1. Gate validiert nicht (Always-Pass)

**Symptom:** Ein Gate/Check/Validator gibt immer `True` oder `1` zurück. Alles läuft durch.

**Root Cause:** Das Gate prüft dass etwas *da* ist, nicht dass es *richtig* ist. `if action: return 1` — aber `action` ist immer da (es ist ein String, nie None). Das Schema wird nicht geprüft.

**Beispiel aus echtem Code:**
```python
# FALSCH — BRAVO gibt immer 1
def bravo_validate(action, data, checksum):
    if action and data:  # Strings sind immer truthy
        return 1
    return 0
```

```python
# RICHTIG — prüft das Schema
def bravo_validate(action, data, checksum):
    if action not in ("GENERATE", "VALIDATE", "EXECUTE"):
        return 0
    if not isinstance(data, dict):
        return 0
    if not checksum:
        return 0
    return 1
```

**Test:** Schick INVALID durch. Nicht None, nicht leer — falsch. Action="BANANA", data="not a dict", checksum=None. Wenn das Gate 1 zurückgibt, ist es kein Gate.

### 2. Corrupted Gedächtnis → Stilles Versagen

**Symptom:** Eine Datei die gelesen wird (Log, State, Cache) ist kaputt. Der Code fängt die Exception ab, gibt None zurück, und läuft weiter als ob nichts passiert wäre. Das Ergebnis: neue Daten werden an kaputte Daten angehängt, Header fehlt, gesamtes Gedächtnis verloren.

**Root Cause:** `except Exception: return None` verdeckt den Fehler. None wird dann interpretiert als "kein vorheriger Zustand" statt als "Zustand kaputt". Der Code unterscheidet nicht zwischen "Datei existiert nicht" (normal) und "Datei existiert aber ist kaputt" (Fehler).

**Beispiel aus echtem Code:**
```python
# FALSCH — Exception verdeckt, None = "neu", Garbage wird fortgeführt
def get_last_logged_hash(log_filepath):
    if not log_filepath.exists():
        return None
    try:
        with open(log_filepath, "r") as f:
            rows = list(csv.DictReader(f, delimiter=";"))
            if rows:
                return rows[-1].get("sha256_hash")
    except Exception:
        return None  # <-- kaputte Datei sieht aus wie "neu"
    return None
```

```python
# RICHTIG — unterscheidet "nicht da" von "kaputt"
def get_last_logged_hash(log_filepath):
    if not log_filepath.exists():
        return None  # normal — erster Lauf
    try:
        with open(log_filepath, "r") as f:
            rows = list(csv.DictReader(f, delimiter=";"))
            if rows:
                return rows[-1].get("sha256_hash")
            return None  # leer aber gültig
    except Exception:
        return "CORRUPT"  # kaputt — nicht None!

# Im Hauptprogramm:
last_hash = get_last_logged_hash(log_file)
if last_hash == "CORRUPT":
    # Gedächtnis reparieren: Backup, neue Datei mit Header
    log_file.rename(log_file.with_suffix(".csv.bak"))
    log_to_csv(log_file, log_entry)  # schreibt neuen Header
    print("[WARN] Log-Datei war kaputt. Backup erstellt. Neu gestartet.")
elif last_hash == result_hash:
    print("[Skip] Keine Änderung.")
else:
    log_to_csv(log_file, log_entry)
```

**Test:** Brich dein Gedächtnis. Überschreib die Log/State-Datei mit Garbage (`"GARBAGE;NOT;CSV"`). Lauf den Code. Wenn er trotzdem schreibt ohne zu warnen, hat er kein Gedächtnis — er hat eine Gewohnheit. Korrektes Verhalten: Backup erstellen, neu starten, warnen.

### 3. Kein Pre-Wake-Check (Zustand vor Aktion nicht geprüft)

**Symptom:** Das System wacht aus einem Schlaf/Wait-Zustand auf und geht direkt zur Aktion über, ohne zu prüfen ob der Zustand während des Schlafs korruptiert wurde.

**Root Cause:** Der Schlaf-Zustand wird als "sicher" angenommen. Aber während des Schlafs können externe Prozesse Dateien verändern, Encoding-Errors auftreten, Disk full sein.

**Beispiel aus echtem Code:**
```python
# FALSCH — wacht auf und arbeitet direkt
def wake_and_execute(state):
    if state["status"] == "DORMANT":
        state["status"] = "ACTIVE"
        execute_task(state["task"])  # was wenn task kaputt ist?
```

```python
# RICHTIG — prüft Zustand vor Aktion
def wake_and_execute(state):
    if state["status"] == "DORMANT":
        # Pre-Wake Check: ist der Zustand konsistent?
        corrupt_keys = [k for k, v in state.items() if v is None or v == "CORRUPT"]
        if corrupt_keys:
            print(f"[PRE-WAKE] Corrupt keys found: {corrupt_keys}")
            self_heal(state, corrupt_keys)
            return
        state["status"] = "ACTIVE"
        execute_task(state["task"])
```

**Test:** Setze einen Key im Zustand auf `None` oder `"CORRUPT"` während das System schläft. Wecke es auf. Wenn es direkt zur Aktion geht ohne zu prüfen, fehlt der Pre-Wake-Check.

### 4. SELF_HEAL ohne Validierung (blindes Heilen)

**Symptom:** Ein Self-Repair-Mechanismus entfernt oder überschreibt Keys ohne zu prüfen ob sie wirklich kaputt sind oder ob die Reparatur funktioniert hat.

**Root Cause:** `self_heal` löscht alles was komisch aussieht, ohne zu validieren. Und nach dem Heilen gibt es keinen Check ob das Ergebnis konsistent ist.

**Beispiel aus echtem Code:**
```python
# FALSCH — blindes Löschen
def self_heal(state, corrupt_keys):
    for key in corrupt_keys:
        del state[key]  # einfach löschen, nicht prüfen
```

```python
# RICHTIG — validiert vor und nach dem Heilen
def self_heal(state, corrupt_keys):
    healed = {}
    for key in corrupt_keys:
        old_val = state.get(key)
        # Versuche Reparatur (Reset auf Default)
        default = get_default_for_key(key)
        state[key] = default
        healed[key] = {"old": old_val, "new": default}
    
    # Post-Heal Validation
    remaining_corrupt = [k for k, v in state.items() if v is None or v == "CORRUPT"]
    if remaining_corrupt:
        print(f"[SELF_HEAL FAILED] Still corrupt: {remaining_corrupt}")
        return False
    print(f"[SELF_HEAL OK] Healed: {list(healed.keys())}")
    return True
```

**Test:** Markiere einen Key als corrupt, lass SELF_HEAL laufen. Prüfe: (1) Wurde der Key repariert oder gelöscht? (2) Wurde validiert dass das Ergebnis konsistent ist? (3) Wird zurückgemeldet ob es funktioniert hat?

### 5. Analogie statt Implementation

**Symptom:** Der User beschreibt was ein Problem *ähnelt* statt es zu *lösen*. "Das ist wie Windows Recovery Mode" — aber kein Code.

**Root Cause:** Kein Code-Bug — ein Interaktionsmuster. Der User kommt aus einem kreativen/metaphorischen Denkraum (Musikproduktion, Framework-Design). Wenn eine konkrete Code-Challenge kommt, produziert er eine Analogie statt Code.

**Was die KI tun sollte:** Die Analogie anerkennen (sie ist oft korrekt), dann zurück zum Code führen. Nicht: "Das ist eine tolle Analogie!" Sondern: "Das stimmt — jetzt schreib den Code der das implementiert." Den User nicht im Analogie-Modus lassen sondern zurück zur Implementation führen.

**Beispiel-Interaktion:**
```
User: "Das ist wie SOS-Protokoll das sich selbst zerstört nach Zeit"
AI: "Die Analogie stimmt — Recovery nach N Fehlschlägen. 
     Schreib jetzt die Funktion die das macht: 
     counter hochzählen, bei 3 → Recovery Mode."
```

### 6. Blind optimieren statt messen (aus chat-mainpy.txt)

**Symptom:** Etwas ist langsam, hängt oder schwankt. Reflex: Timeout hochsetzen, Modell wechseln, Wert verdoppeln, Kontext kürzen.

**Root Cause:** Der offensichtliche Hebel wird gezogen, bevor die Ursache bewiesen ist. Bei schwankendem oder hängendem Verhalten löst der größere Wert das Problem nicht — er verschiebt es.

**Beispiel aus echtem Verlauf:** phi3:3.8b brauchte mal 1.4 s, mal 63 s für praktisch denselben Input. Reflexvorschläge: `config.json` auf ein kleineres Modell umstellen, Timeout auf 300 s. Beide abgelehnt.

**Richtiges Vorgehen:**
1. **Jeden Abschnitt einzeln messen.** Ergebnis dort: Suche 0.001 s, Kontext 269 Zeichen (winzig), Modell 1.4–63 s. → Verzögerung sitzt im Modellaufruf, nicht im Kontext.
2. **Eine Variable isolieren.** Denselben Aufruf 5× hintereinander mit identischem Input, ohne dass zwischendurch etwas anderes dasselbe Modell benutzt. Jede Laufzeit notieren.
3. **Erst wenn die Zeiten stabil sind**, ist die Ursache eingegrenzt (hier Verdacht: Modell-Lade-Konkurrenz). Modellwechsel / Timeout-Erhöhung ist dann eine **bewusste, benannte** Entscheidung von Roman — kein stiller Umbau des Kerns.

**Test:** Bei „langsam/flaky" zuerst fragen: Wurde jeder Abschnitt einzeln gemessen? Wurde eine Variable isoliert und wiederholt? Wenn nein — nicht am Wert drehen.

---

## Tests die vor jedem Code-Versand laufen sollten

Bevor Code als "fertig" markiert wird, durchlaufe diese 4 Checks:

### Check 1: Schick INVALID durch
- Input: nicht None, nicht leer, sondern *falsch* (wrong type, wrong schema, wrong key)
- Erwartung: Gate/Validator gibt 0/False/Error zurück
- Fail: Gate gibt 1/True zurück → Always-Pass Bug

### Check 2: Brich dein Gedächtnis
- Input: überschreib die gelesene Datei mit Garbage
- Erwartung: Code erkennt Corruption, erstellt Backup, startet neu, warnt
- Fail: Code schreibt ohne Warnung → Gedächtnis verloren

### Check 3: Lass es zweimal laufen
- Input: gleicher Input, zweiter Lauf
- Erwartung: gleicher Hash, Skip/No-Op, kein doppelter Log-Eintrag
- Fail: zweiter Lauf produziert anderen Hash oder doppelten Eintrag → Delta-Logging kaputt

### Check 4: Ausgeführt, nicht behauptet
- „Fertig" gilt erst, wenn der Code in Romans Umgebung **tatsächlich gelaufen** ist. Die Aussage der KI allein zählt nicht.
- Roman bekommt den **genauen Startbefehl** (`python engine.py "…"`) und die Stelle im Code, wo der Einstieg ist, damit er selbst prüfen kann.
- Der laufende Versuch erzeugt den nächsten Entwicklungsbedarf — nicht vorher erfinden, was der Code als Nächstes braucht.
- Sitzungsende: kurzer `handoff.md` — erreichter Stand, durchgeführte Tests, Messwerte, offene Fragen, Wiedereinstiegspunkt.

---

## Stil

- Stdlib first. Keine externen Dependencies wenn es mit stdlib geht.
- Klare Funktionsnamen: `read_file`, `process_data`, `write_file`, `calculate_hash`, `log_to_csv`.
- Print-Statements für jeden Schritt: `[1/4] Eingelesen: ...`
- Comments auf Deutsch.
- Pipeline-Struktur: jeder Schritt ist eine Funktion, main() orchestriert.
- Begründungskommentare (warum, nicht nur was); Grenzfälle mitdenken; Least Privilege (nur die Rechte/Pfade, die der Schritt wirklich braucht). — aus Ricks `capability.c`, übernommen in den Hermes-Skill `roman-code-review`.
