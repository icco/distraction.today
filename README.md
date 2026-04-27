# distraction.today

A daily quote site, served at <https://distraction.today>.

## Routes

| Method | Path                       | Description                                     |
|--------|----------------------------|-------------------------------------------------|
| `GET`  | `/`                        | Redirects to today's quote (or latest).         |
| `GET`  | `/{YYYY-MM-DD}`            | Renders the quote for a given date, 404 if none.|
| `GET`  | `/about`                   | About page.                                     |
| `GET`  | `/feed.rss`, `/feed.atom`  | Quote feeds.                                    |
| `GET`  | `/healthz`                 | Liveness probe.                                 |
| `GET`  | `/metrics`                 | OTel HTTP metrics in Prometheus format.         |

## Running

```bash
go run .
```

```bash
docker build -t distraction.today .
docker run --rm -p 8080:8080 distraction.today
```
