# .tools

Starten: Doppelklick auf die `.cmd`-Datei.

| Werkzeug | Zweck |
| --- | --- |
| `push-main.cmd` | Ersetzt `main` auf GitHub vollständig durch den lokalen Stand. Fragt vorher nach `JA`, sichert den alten GitHub-Stand als Tag `backup-main-<Datum>` und bricht ab, wenn `.env` nicht ignoriert wird. |

Backup zurückholen: `git push --force origin backup-main-<Datum>:main`
