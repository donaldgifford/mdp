# AWS web service architecture (built-in icons)

`architecture-beta` syntax: `service id(icon)[Title] in group`,
edges pin sides (`a:R --> L:b`). Built-in icons only:
`cloud`, `database`, `disk`, `internet`, `server`. See
`architecture-aws-icons.md` for the Iconify-pack version.

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
