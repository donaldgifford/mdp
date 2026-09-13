# Use case diagram (new in Mermaid v12, keyword is `usecase-beta`)

Grammar gotchas: blocks close with `end` (not `}`), standalone use cases
need explicit IDs (`Browse(Browse Catalog)`, never bare `(Browse Catalog)`),
no space in `systemBoundary id[Title]`, one statement per line, and every
actor needs an `actor` declaration (actors are never inferred).

```mermaid
usecase-beta
direction LR
actor Customer
actor Admin
systemBoundary Shop[Online Shop]
Browse(Browse Catalog)
Checkout(Checkout)
Manage[Manage Inventory]
end
Customer --> Browse
Customer --> Checkout
Admin --> Manage
```
