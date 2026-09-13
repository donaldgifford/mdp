# Flowchart

ELK is the default layout in Mermaid v12. Compare with the dagre escape
hatch (`dagre = true` plugin opt / `mdp serve --dagre`).

```mermaid
flowchart TB
    A[Start] --> B{Approved?}
    B -->|Yes| C[Merge]
    B -->|No| D[Request changes]
    C --> E[Deploy]
    D --> E
```
