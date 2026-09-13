# Class diagram — mdp's own server core

Modeled on the real structs in `internal/server` (`Config`, `Server`) and
`pkg/livereload` (`Hub`), plus `pkg/theme` (`Theme`). Member types stay
bracket-free since `[`/`]` collide with mermaid syntax.

```mermaid
classDiagram
class Config {
    +string File
    +int Port
    +bool OpenBrowser
    +string Theme
    +bool ScrollSync
    +bool Dagre
}
class Server {
    -Config cfg
    +Broadcast(markdown) error
    +SendCursor(line) error
}
class Hub {
    +Broadcast(msg)
    +HandleWebSocket(w, r)
}
class Theme {
    +string CSS
    +IsAuto() bool
}
Server *-- Config : configured by
Server --> Hub : publishes to
Server --> Theme : renders with
```
