# Stargate — Architektur- und Umsetzungsplan

> TUI, die SSH-Verbindungen aus `~/.ssh/config` (inkl. `Include`) ausliest, zur
> Auswahl stellt (Tastatur + Maus, mit Filter) und per echtem `ssh`-Client
> verbindet. Nach Beenden der SSH-Session kehrt die TUI zurück.

> **Hinweis:** Dies ist das ursprüngliche Design-Dokument aus der Entwurfsphase,
> auf Deutsch verfasst und als Kontext für Contributors erhalten. Es beschreibt
> die getroffenen Architekturentscheidungen und deren Begründung, nicht den
> tagesaktuellen Stand. Verbindlich für das Verhalten sind der Code und die
> [README](../README.md).

---

## 0. Entscheidungs-Log (aufgelöste Rückfragen)

| # | Frage | Entscheidung |
|---|---|---|
| Stack | Technologie | **Go + bubbletea** (Charm) |
| 1 | Nach SSH-Ende | **Zurück zur TUI** → Kindprozess-Variante (nicht `syscall.Exec`) |
| 2 | ProxyJump-Anzeige | **Ja** — pro Host „via Jump-Host" anzeigen |
| 3 | Gruppierung/Tags | **Nein** — bewusst out of scope |
| 4 | `Match exec …` parsen | **Nein** — out of scope |
| 5 | Config-Pfad-Override | **Ja** — `--config`-Flag + `STARGATE_CONFIG`-Env |
| 6 | Windows | **Nein** — nur Unix (vereinfacht TTY-Handling) |
| 7 | Verteilung | **`go install`** (braucht öffentlichen Module-Pfad + VCS-Tag) |
| 8 | Plan persistieren | **Ja** — diese Datei |
| 9 | Startbefehl | **`gate`** (Binary/Command-Paket) |
| 10 | Filterstart | **Direktes Tippen**; druckbare Zeichen sind Filtereingabe |

---

## 1. Technologie-Stack (final)

| Komponente | Wahl | Zweck |
|---|---|---|
| Sprache | **Go** | 1 statisches Binary, schneller Start, `go install` |
| TUI-Framework | **bubbletea** (Charm) | Elm-Architektur (Model/Update/View) |
| UI-Komponenten | **bubbles** | `list` (mit Fuzzy-Filter), `textinput`, Keys/Help |
| Styling | **lipgloss** | Layout/Farben/Footer |
| SSH-Config-Parser | **`github.com/kevinburke/ssh_config`** | OpenSSH-nahe Host-/Include-Semantik und first-match-wins; Symlinks löst Stargate auf |
| Logging (optional) | **`log/slog`** (Stdlib) | Diagnose ohne UI-Blocker |

**Begründung (Kurzform):** Ein Launcher profitiert von schnellem Start und einem
einzigen Binary; die relevante Config-Semantik (besonders `Include` und
first-match-wins) liefert `kevinburke/ssh_config`, statt eine fehleranfällige
Eigenimplementierung zu bauen. Da die Library enthaltene Aliase nicht öffentlich
exponiert, ergänzt Stargate sie um einen kleinen AST-Walker für die Alias-Ermittlung.
Die Unterstützung von `ProxyJump`/`ProxyCommand` erzwingt die Delegation an den
nativen `ssh`-Client — die TUI ist ein Launcher, keine SSH-Reimplementierung.

**Kanonischer Module-Pfad:** `github.com/nibra180/stargate-tui`
(abgeleitet aus dem konfigurierten Git-Remote).

---

## 2. Modul- / Dateistruktur

```
stargate/
├── cmd/gate/
│   ├── main.go                     # Entry: Flags/Env → Config laden → TUI → Auswahl → ssh exec
│   └── main_test.go
├── internal/
│   ├── config/
│   │   ├── loader.go               # Pfad-Auflösung (~/.ssh/config, Symlink, Override), Includes
│   │   ├── host.go                 # Host-Modell + Sichtbarkeits-/ProxyJump-Regeln
│   │   ├── loader_test.go          # Unit-Tests: Parsing, Include, Wildcard-Filter, Override
│   │   └── host_test.go
│   ├── tui/
│   │   ├── model.go                # Zustand: Hosts, Filter, Cursor, Phase
│   │   ├── model_test.go
│   │   ├── filter.go               # Direkter Type-to-filter-Einstieg
│   │   ├── mouse.go                # Klick-Hit-Testing + Mausrad-Navigation
│   │   ├── update.go               # Messages: Key/Mouse/Window-Resize
│   │   ├── view.go                 # Rendering: Liste + Filterzeile + Help/Footer
│   │   ├── keys.go                 # Keymap + Help-Bindings
│   │   └── styles.go               # lipgloss-Stile
│   └── ssh/
│       ├── executor.go             # Native ssh-Command-Erzeugung mit geladener Config
│       └── executor_test.go
├── .github/workflows/ci.yml        # test -race + vet auf Linux und macOS
├── docs/PLAN.md                    # dieses Dokument
├── .gitignore                      # Build-/Coverage-Artefakte
├── go.mod / go.sum
├── README.md                       # Installation, Bedienung, Entwicklung
├── CONTRIBUTING.md / SECURITY.md / RELEASING.md
└── Makefile                        # build / test / vet / check / coverage / install / clean
```

**Trennung** (`config` / `tui` / `ssh`) hält Parsing-Logik aus der UI und der CLI
heraus: Geschäftslogik gehört nicht in Controller- oder UI-Schichten.

---

## 3. Datenmodell: Host (Felder, keine Implementierung)

Pro sichtbarem Auswahlziel:

| Feld | Quelle | Hinweis |
|---|---|---|
| `Alias` | `Host <name>` | Anzeige- und Verbindungsname |
| `HostName` | effektiv (first-match) | Aufgelöster Zielhost |
| `User` | effektiv | — |
| `Port` | effektiv (default 22) | — |
| `IdentityFile` | effektiv | für Detailanzeige |
| `ViaJump` | `ProxyJump`/`ProxyCommand` vorhanden? | **bool** → „via Jump-Host"-Badge |
| `SourceFile` | Include-Herkunft | Diagnose/optionale Anzeige |

Verbindung erfolgt immer über `Alias` (nicht über zusammengebaute
`user@host:port`), damit ssh **alle** effekti­ven Einstellungen (Jump, Forwards,
IdentityFile) selbst korrekt anwendet.

---

## 4. Parsing-Logik

Aufbauend auf `kevinburke/ssh_config` (keine Eigenimplementierung):

1. **Pfad ermitteln** (Präzedenz): `--config`-Flag → `STARGATE_CONFIG`-Env →
   `~/.ssh/config` (HOME auflösen, **Symlink folgen** wie ssh).
2. **Parsen** über die Library; `Include` mit Globs wird aufgelöst
   (fehlertolerant bei nicht-matchenden Globs; Zyklen werden von der Lib begrenzt).
3. **Aliase ermitteln:** Ein AST-Walker traversiert Root- und Include-Dateien,
   weil die Library enthaltene Host-Blöcke nicht öffentlich exponiert. Dabei wird
   nur die bereits geparste AST-Struktur ausgewertet, keine eigene Grammatik gebaut.
4. **Effektive Werte pro Alias** berechnen (first-match-wins über alle
   passenden `Host`-Blöcke inkl. `Host *`).
5. **Sichtbarkeit filtern** („direkt verbinden" gefordert):
   - Block ausblenden, wenn **alle** Namen Wildcards/Negierung enthalten
     (`*`, `?`, `!`).
   - Block mit mehreren Namen (`Host prod prod.example`) → jeden konkreten
     Namen als eigene Auswahl anbieten.
6. **`ViaJump`-Flag** setzen, wenn `ProxyJump` oder `ProxyCommand` in der
   effektiven Konfiguration des Alias vorhanden und nicht `none` ist.
7. **`Match exec …`** wird **nicht** ausgewertet (out of scope); die Library
   lehnt es sicher ab. Andere `Match`-Blöcke werden bei der Alias-Sammlung übergangen.

---

## 5. Interaktionsmodell

**Zustände:** `Browsing` → (optional) `Filtering` → Aktion `Connect` →
`InSession` (ssh läuft, TUI pausiert) → zurück zu `Browsing`.

| Aktion | Tastatur | Maus |
|---|---|---|
| Navigieren | `↑`/`↓` | Scrollrad / Klick = auswählen |
| Filtern | Direkt tippen; `/` bleibt optional (Fuzzy über Alias+HostName+User@Host:Port) | — |
| Filter zurücksetzen | `Esc` | — |
| Verbinden | `Enter` | **Klick** auf Eintrag |
| Beenden | `Ctrl+C` oder `Esc` ohne aktiven Filter | — |

Druckbare Zeichen — einschließlich `j`, `k` und `q` — sind Filtereingabe.
Dadurch gibt es keine datenabhängigen Konflikte; Navigation erfolgt eindeutig
über Pfeiltasten.

Maus via `tea.WithMouseCellMotion`. Filter über Bubbles `list`-Filter
(Fuzzy-Matcher). Bei 10 Hosts kaum nötig, aber skalierbar.

---

## 6. Verbindungsmodell

**Strategie: Kindprozess mit TTY-Vererbung + Rückkehr zur TUI.**

Ablauf:
1. Auswahl trifft ein → `tea.ExecProcess` lässt Bubble Tea Raw-Modus und
   Alternate-Screen kontrolliert freigeben.
2. Native Ausführung als `ssh -F <geladene-config> -- <alias>`. Bubble Tea
   verbindet stdin/stdout/stderr direkt mit dem Terminal; ein eigener PTY- oder
   `Setctty`-Layer ist nicht nötig.
3. Auf Beendigung von ssh warten; Exit-Status im Callback erfassen.
4. Bubble Tea stellt Terminal und TUI automatisch wieder her; der Filter wird
   geleert, die vollständige Liste mit dem verbundenen Host als Auswahl gezeigt
   und eine Statusmeldung bei Erfolg bzw. Exit-Code/Fehler ausgegeben.

**Warum nicht prozessersetzt (`syscall.Exec`)?** Dann ist keine Rückkehr zur
TUI möglich — Entscheidung 1 verlangt aber die Rückkehr. Warum **nicht**
eingebettetes PTY? ProxyJump, ProxyCommand, Port-Forwarding und
ssh-Escape-Sequenzen funktionieren nur zuverlässig mit dem nativen Client.

---

## 7. CLI-Oberfläche

```
gate [--config PATH]
```

- `--config PATH`: alternative Config-Datei (Präzedenz über Env).
- `STARGATE_CONFIG`: Env-Override (unterhalb von `--config`).
- Ohne TTY an stdin oder stdout → Abbruch mit klarem Hinweis.
- `ssh` wird in `$PATH` erwartet (Start-Check, sonst klare Fehlermeldung).

---

## 8. Fehlerbehandlung

| Fall | Verhalten |
|---|---|
| Config fehlt / Symlink kaputt | Meldung + Hinweis; startbar mit leerer Liste oder Abbruch |
| Parse-Fehler (Syntax) | Fehler mit Datei-/Parser-Kontext; vollständig abbrechen, da Teil-Parsing first-match-/Include-Semantik verfälschen könnte |
| Include-Glob ohne Treffer | Warnung, fortfahren (wie ssh) |
| Keine sichtbaren Hosts | Empty-State („nur Wildcard-Hosts gefunden") |
| `ssh` nicht in `$PATH` | Start-Check → klare Fehlermeldung |
| Kein TTY | Abbruch mit Hinweis |
| Verbindungsfehler | ssh behandelt selbst; Exit-Status nach Rückkehr ggf. anzeigen |

---

## 9. Verteilung

- **`go install github.com/nibra180/stargate-tui/cmd/gate@latest`** als
  primärer Verteilweg.
- Kanonischer Module-Pfad stimmt mit dem Git-Remote überein.
- `Makefile`-Targets für Build, Test, Vet, Check, Coverage, Install und Clean.
- README mit Installation, Bedienung und Config-Semantik; Release-Schritte in
  [RELEASING.md](../RELEASING.md).

---

## 10. Umsetzung in Meilensteinen

Die ursprüngliche Roadmap, in der Reihenfolge, in der sie umgesetzt wurde:

1. **Scaffold & Config:** `go.mod`, `internal/config` (Loader + Host-Modell +
   Wildcard-Filter + ViaJump) inkl. Unit-Tests. Override via Flag/Env.
2. **TUI-Shell:** bubbletea `model/update/view`, Liste, Navigation,
   Fuzzy-Filter, Help/Footer, „via Jump-Host"-Anzeige und CLI-Entry.
3. **SSH-Executor:** nativer Kindprozess über `tea.ExecProcess`, geladene Config
   via `-F`, sichere Alias-Trennung via `--`, Rückkehr zur TUI und
   Exit-Status-Handling.
4. **Polish:** Mouse-Cell-Motion, Mausrad-Navigation, Direktverbindung per
   Klick, direkter Type-to-filter-Einstieg, präziser Empty-State, längere
   Statusmeldungen und TTY-Validierung.
5. **Release-Readiness:** README, Makefile, `.gitignore`, kanonischer
   Module-Pfad und `go install`-Smoke-Test.

---

## 11. Explizit out of scope

- `Match exec …` und bedingte Config-Blöcke
- Gruppierung/Tags (kein native SSH-Config-Konzept)
- Windows-Unterstützung
- SFTP / Dateitransfer, Port-Forwarding-UI (läuft über ssh selbst, nicht über die TUI)
- Eingebettetes PTY / Reimplementierung des SSH-Clients
- Favoriten/Zuletzt verwendet (nicht angefragt)
