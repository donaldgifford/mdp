# Agentflow-beta (new in Mermaid v12, still beta — syntax may shift)

Three edge flavors: `-->` sequence, `-.-` reference, `--x` failure.
Shapes via `@{ shape: ... }`: `task`, `tool`, `input`, `decision`,
`refdoc`, `action`.

```mermaid
agentflow-beta TB
  flow reviewer["Review Agent"]
    changes["Gather changes"]@{ shape: input }
    analyse["Analyse"]@{ shape: task }
    lint["run_linter"]@{ shape: tool }
    spec["API spec"]@{ shape: refdoc }
    decide["Ship it?"]@{ shape: decision }
    changes --> analyse --> lint --> decide
    analyse -.- spec
    decide --x|needs work| analyse
  end
  linkStyle 4 stroke:danger
```
