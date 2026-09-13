# Mermaid examples — all in one

Every diagram from this directory on a single page. See the per-diagram
files for grammar notes. Preview with `make build && ./mdp serve
docs/examples/all.md`.

## Flowchart (ELK default — compare with `dagre = true`)

```mermaid
flowchart TB
    A[Start] --> B{Approved?}
    B -->|Yes| C[Merge]
    B -->|No| D[Request changes]
    C --> E[Deploy]
    D --> E
```

## Sequence

```mermaid
sequenceDiagram
    Alice ->> John: Hello John, how are you?
    John -->> Alice: Great!
    Alice ->> John: See you later!
```

## State

```mermaid
stateDiagram-v2
    [*] --> Still
    Still --> [*]
    Still --> Moving
    Moving --> Still
    Moving --> Crash
    Crash --> [*]
```

## Use case (v12 `usecase-beta`)

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

## Agentflow-beta (v12, beta syntax)

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
```

## Kanban

```mermaid
kanban
  Todo
    [Create Documentation]
    [Write Blog Post]
  In Progress
    [Update Styles]
  Done
    [Project Setup]
    [Release v1.0]
```

## Timeline

```mermaid
timeline
    title History of Social Media
    2002 : LinkedIn
    2004 : Facebook
         : Google
    2005 : Youtube
    2006 : Twitter
```

## Git graph

```mermaid
gitGraph
    commit
    commit
    branch develop
    checkout develop
    commit
    commit
    checkout main
    merge develop
    commit
```

## AWS web service (built-in icons)

```mermaid
architecture-beta
group edge(internet)[Edge]
group app(cloud)[Application]
group data(database)[Data]
service dns(internet)[Route 53] in edge
service cdn(cloud)[CloudFront] in edge
service alb(server)[ALB] in app
service api(server)[API Service] in app
service worker(server)[Worker] in app
service db(database)[RDS Postgres] in data
service cache(database)[ElastiCache] in data
dns:R --> L:cdn
cdn:R --> L:alb
alb:R --> L:api
api:B --> T:db
api:B --> T:cache
api:R --> L:worker
```

## AWS web service (Iconify `logos:` icons, needs network)

```mermaid
architecture-beta
group edge(internet)[Edge]
group app(cloud)[Application]
group data(database)[Data]
service dns(logos:aws-route53)[Route 53] in edge
service cdn(logos:aws-cloudfront)[CloudFront] in edge
service alb(logos:aws-elb)[ALB] in app
service api(logos:aws-ecs)[ECS Service] in app
service worker(logos:aws-lambda)[Worker] in app
service db(logos:aws-rds)[RDS Postgres] in data
service cache(logos:aws-elasticache)[ElastiCache] in data
dns:R --> L:cdn
cdn:R --> L:alb
alb:R --> L:api
api:B --> T:db
api:B --> T:cache
api:R --> L:worker
```

## AWS EKS (built-in icons)

```mermaid
architecture-beta
group vpc(cloud)[VPC]
group public(cloud)[Public Subnets]
group private(cloud)[Private Subnets]
group cluster(server)[EKS Cluster]
service alb(server)[ALB Controller] in public
service nodes(server)[Node Group] in private
service pods(server)[API Pods] in cluster
service registry(disk)[ECR] in vpc
service db(database)[RDS Postgres] in vpc
alb:B --> T:nodes
nodes:R --> L:pods
registry:R --> L:pods
pods:B --> T:db
```

## AWS EKS (Iconify icons, needs network)

```mermaid
architecture-beta
group vpc(logos:aws-vpc)[VPC]
group public(cloud)[Public Subnets]
group private(cloud)[Private Subnets]
group cluster(logos:aws-eks)[EKS Cluster]
service alb(logos:aws-elb)[ALB Controller] in public
service nodes(logos:aws-ec2)[Node Group] in private
service pods(logos:kubernetes)[API Pods] in cluster
service registry(devicon:docker)[Container Images] in vpc
service db(devicon:postgresql)[RDS Postgres] in vpc
alb:B --> T:nodes
nodes:R --> L:pods
registry:R --> L:pods
pods:B --> T:db
```

## ER — e-commerce Postgres

```mermaid
erDiagram
CUSTOMERS ||--o{ ORDERS : places
ORDERS ||--|{ ORDER_ITEMS : contains
PRODUCTS ||--o{ ORDER_ITEMS : listed_in
CUSTOMERS {
    uuid id PK
    varchar email UK
    text full_name
    timestamptz created_at
}
ORDERS {
    uuid id PK
    uuid customer_id FK
    numeric total
    varchar status
    timestamptz placed_at
}
PRODUCTS {
    uuid id PK
    varchar sku UK
    text title
    numeric price
    int stock
}
ORDER_ITEMS {
    uuid id PK
    uuid order_id FK
    uuid product_id FK
    int quantity
    numeric unit_price
}
```

## ER — docs CMS

```mermaid
erDiagram
SITES ||--o{ DOC_VERSIONS : publishes
DOC_VERSIONS ||--o{ PAGES : contains
PAGES ||--o{ ASSETS : embeds
SITES ||--o{ REDIRECTS : defines
SITES {
    uuid id PK
    varchar slug UK
    text title
}
DOC_VERSIONS {
    uuid id PK
    uuid site_id FK
    varchar version
    boolean is_current
    timestamptz released_at
}
PAGES {
    uuid id PK
    uuid version_id FK
    varchar slug
    text body_markdown
    jsonb frontmatter
    timestamptz updated_at
}
ASSETS {
    uuid id PK
    uuid page_id FK
    varchar path
    varchar content_type
}
REDIRECTS {
    uuid id PK
    uuid site_id FK
    varchar from_path
    varchar to_path
}
```

## Class diagram — mdp server core

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

## Requirements — mdp preview

```mermaid
requirementDiagram
requirement live_reload {
id: 1
text: Preview updates in browser on save.
risk: low
verifymethod: test
}
requirement scroll_sync {
id: 2
text: Preview scrolls to cursor line.
risk: medium
verifymethod: test
}
element browser_tab {
type: browser
}
live_reload - satisfies -> browser_tab
scroll_sync - satisfies -> browser_tab
```
