# LogLens — Go Log Dashboard

A local web app that reads a CSV file of IT tickets/events and displays a simple dashboard with charts and stats.

## Quick Start

```bash
cd LogLens
go run .
```

Open **http://localhost:8080** in your browser.

## Requirements

- Go 1.16+

## Project Structure

```
LogLens/
├── main.go              # Go backend (HTTP server + CSV parsing)
├── static/
│   └── index.html       # Dashboard UI (Chart.js via CDN)
├── data/
│   └── tickets.csv      # Your ticket data
└── README.md
```

## API Endpoints

| Method | Endpoint      | Description                          |
|--------|---------------|--------------------------------------|
| GET    | `/`           | Serves the dashboard                 |
| GET    | `/api/summary`| Returns JSON of all computed stats   |
| POST   | `/api/reload` | Reloads the CSV and returns summary  |

## CSV Format

Place your ticket data in `./data/tickets.csv` with this structure:

```csv
id,created_at,closed_at,category,priority,status
1,2026-01-05,2026-01-05,Password Reset,Low,Closed
2,2026-01-05,2026-01-06,Printer,Medium,Closed
3,2026-01-06,,Network,High,Open
```

- **id** — Ticket ID (integer)
- **created_at** — Date opened (`YYYY-MM-DD`)
- **closed_at** — Date closed (`YYYY-MM-DD`), leave empty for open tickets
- **category** — Ticket category
- **priority** — Low / Medium / High
- **status** — Open / Closed (or similar)

## Using Your Own Data

1. Replace `./data/tickets.csv` with your file.
2. Ensure the header row matches: `id,created_at,closed_at,category,priority,status`
3. Use `YYYY-MM-DD` for dates. Leave `closed_at` empty for open tickets.
4. Restart the app or click **Reload CSV** in the dashboard.

## License

MIT
