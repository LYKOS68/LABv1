# HTTPS-Reverse-Proxy: Konfiguration und Prüfstatus

Ausgangsbasis: `main` bei `4803d1bd694b001d1b86816e4655005ef6a6436c`.
Automatisch erarbeiteter Änderungsvorschlag auf separatem Review-Branch.
Die Veröffentlichung des Branches wurde beauftragt; Merge und Deployment sind nicht freigegeben.

## Nachgewiesenes Verhalten

- Bisher wurde `Secure` an drei Cookie-Stellen nur aus `r.TLS` abgeleitet.
- Bisher verwendeten alle Auth-Rate-Limits ausschließlich die direkte Verbindungsadresse.
- Der Server startet mit `ListenAndServe`, ohne eigene TLS-Terminierung oder Proxy-Vertrauenskonfiguration.
- Ein konkreter Online-Proxy oder ein betroffener Produktionsbetrieb ist nicht nachgewiesen.

## Neue Konfiguration

Online: `BASE_URL=https://<öffentlicher-host>`. Sitzungs-Cookies erhalten standardmäßig immer
`Secure`, auch bei HTTP zwischen Proxy und Go. `HttpOnly`, `SameSite=Lax` und `Path=/`
gelten sowohl beim Erstellen als auch beim Löschen. Kein Protokoll-Header kann diese Regel abschwächen.
Eine fehlende bzw. nicht HTTPS-basierte Online-Basisadresse verhindert den Serverstart.

Lokale HTTP-Entwicklung: `ALLOW_INSECURE_LOCAL_HTTP=true`, dazu beispielsweise
`BASE_URL=http://localhost:8080`. Der Listener wird dann auf `127.0.0.1` beschränkt.
Die Ausnahme akzeptiert nur `localhost`, `127.0.0.1` oder `::1` als Basis-Host;
der Listener selbst ist IPv4-Loopback. Bei direktem TLS bleiben Cookies auch lokal `Secure`.
Die Umgebungsvariable ist standardmäßig deaktiviert. Keine `.env` wurde verändert.

`TRUSTED_PROXY_CIDRS` enthält kommaseparierte IPv4-/IPv6-CIDRs der tatsächlichen Proxy-Peers.
Leer bedeutet: keine Header-Vertrauensstellung. Einzelne Proxys mit `/32` bzw. `/128` angeben.
Keine automatischen privaten Netze oder Loopback-Ausnahmen. Globale `/0`-Netze und
IPv4-gemappte IPv6-Netze werden abgelehnt; native Netze verwenden.

Nur `X-Forwarded-For` wird unterstützt. Bei einem vertrauenswürdigen direkten Peer wird die
Kette von rechts bis zum ersten nicht vertrauenswürdigen Hop ausgewertet. Weitere linke
Einträge können vom Client stammen und werden nicht als Identität übernommen.
Mehrere Headerzeilen werden als eine geordnete Kette behandelt. Ungültige Einträge,
Zonen, IPs mit Ports, mehr als 32 Hops oder mehr als 4096 Zeichen führen zum Rückfall
auf die direkte Peer-IP. IPv4-gemappte Verbindungsadressen werden normalisiert.
`Forwarded`, `X-Real-IP` und `X-Forwarded-Proto` werden bewusst nicht ausgewertet.

## Vor dem Online-Betrieb noch offen: {LÜCKE}

- Proxy-Produkt, tatsächliche Peer-IP-Netze, Anzahl und Reihenfolge aller Proxy-Hops.
- Nachweis, dass der äußere Proxy eingehende Client-Header entfernt/überschreibt oder
  die tatsächlich beobachtete Gegenstelle zuverlässig rechts an `X-Forwarded-For` anhängt.
  Vertrauenswürdige Proxys dürfen keine ungeprüften Client-Angaben als eigene Identität liefern.
- Firewall/Listener-Netzwerk: Backend nur für vorgesehene Proxys erreichbar machen;
  freigegebene Netze so klein halten, dass fremde Workloads keine Header fälschen können.
- Öffentlichen Host unverändert weiterreichen; der bestehende CSRF-Schutz bleibt aktiv.
  Browser-Smoke-Test für Login, Logout, Passwortreset und Bestätigung hinter dem echten Proxy.
- HTTPS-Weiterleitung und TLS-Absicherung am öffentlichen Einstieg, ggf. auch auf dem Backend-Hop.
- Bei mehreren Go-Instanzen bleiben die bestehenden Zähler pro Prozess; ein globales
  Rate-Limit wäre eine gesonderte Infrastruktur-/Architekturentscheidung.

## Regressionen und Freigabe

Neue Tests decken direkte Verbindungen, TLS-Terminierung, untrusted Header,
manipulierte linke Hops, Proxy-Ketten, mehrere Headerzeilen, IPv6, gemappte IPv4,
fehlerhafte/lange Header, getrennte Client-Zähler, HTTP 429, Cookie-Attribute sowie
die lokale Listener-Ausnahme und ungültige Konfiguration ab.

Erforderlich laut TECH-STACK §12 und VERTRAG §5:

```sh
gofmt -w cmd/lab-studio/main.go cmd/lab-studio/main_test.go internal/handlers/auth.go internal/handlers/handlers.go internal/handlers/ratelimit.go internal/handlers/proxy.go internal/handlers/proxy_test.go
go build ./...
go vet ./...
go test ./...
```

{UNVERIFIED}: In der Bearbeitungsumgebung fehlen `go` und `gofmt`; alle vier Aufrufe
scheitern vor der Ausführung. Formatierung, Kompilierung, statische Analyse und Tests
sind daher ausstehend. Erforderlich ist die in `go.mod` festgelegte Go-Version 1.27.0.
Keine Dependencies, Migrationen oder verbindlichen Richtlinien wurden geändert.
Commit und Veröffentlichung dieses Review-Branches wurden im Folgeauftrag autorisiert.
Eine Merge- oder Deployment-Freigabe wurde damit nicht erteilt.
