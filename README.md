# RAP-LAB (LABv1)

RAP-LAB ist eine frühe Entwicklungsbasis für eine Vernetzungs- und Angebotsplattform für Musikschaffende. Ziel ist es, Artists, Songwriter, Producer, Beatmaker, Mixer, Videografen und Designer zusammenzubringen und gemeinsame Projekte sowie Angebote und Gesuche auffindbar zu machen.

> **Projektstatus:** Early Development. Funktionen und Architektur können sich ändern. Der aktuelle Implementierungsstand ist dem Quellcode zu entnehmen.

## Geplante Ausrichtung

- Künstlerprofile und Vernetzung
- Angebote und Gesuche für Beats, Mixing, Mastering, Musikvideos, Covergestaltung und gemeinsame Projekte
- Suche nach Mitstreitern und möglichen Kooperationen
- Ein Lyrics-Editor ist derzeit kein Schwerpunkt

## Technischer Stand

Die Angaben in diesem Abschnitt werden aus der aktuellen Moduldefinition abgeleitet:

- **Go:** Modulziel go 1.27.0
- **Datenbankzugriff:** PostgreSQL-Treiber über github.com/jackc/pgx/v5
- **Weitere Abhängigkeit:** golang.org/x/crypto
- **Repository und Zusammenarbeit:** Git / GitHub
- **Frontend:** HTML-Vorlagen sind vorgesehen; der konkrete Funktionsumfang ist im aktuellen Code zu prüfen

Die vollständigen Go-Abhängigkeiten und Versionen stehen in [go.mod](go.mod) und [go.sum](go.sum). Die Moduldefinition allein belegt nicht, dass alle geplanten Funktionen bereits implementiert oder lauffähig sind.

## Entwicklung

Das Projekt wird unter Anleitung des Projektverantwortlichen auch mithilfe lokaler KI-CLI-Modelle entwickelt. Änderungen sollen nachvollziehbar über Git erfolgen und vor der Übernahme geprüft werden.

## Lizenz

RAP-LAB wird unter der **PolyForm Noncommercial License 1.0.0** bereitgestellt. Sie erlaubt Nutzung, Änderungen und Weitergabe nur für nichtkommerzielle Zwecke gemäß dem vollständigen Lizenztext. Kommerzielle Nutzung ist durch diese Lizenz nicht gestattet; dafür ist eine separate schriftliche Erlaubnis erforderlich.

- Vollständiger Lizenztext: [LICENSE](LICENSE)
- Offizielle Lizenz: https://polyformproject.org/licenses/noncommercial/1.0.0
- Kontakt für Anfragen zur kommerziellen Nutzung: roman.hild4@gmail.com

## Projektinhaber

Roman Hild (LYKOS68) · https://github.com/LYKOS68
