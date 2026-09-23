# Alem frontend

Requires Node.js 20.19+ (or 22.12+).

```bash
cd frontend
npm install
npm run dev
```

Open the URL printed by Vite. The frontend uses the Go API at `VITE_API_URL` (default `http://localhost:8080`). Copy `.env.example` to `.env` for local development.

The real backend is the default mode. Set `VITE_USE_MOCK=true` only for the explicit local demo fallback; mock data persists in `localStorage` under `alem-demo-v1`.

The proposal screen uses `VITE_DEMO_TEAM_ID` because the current backend contract has no team-list endpoint. Set it to the seeded demo team's ID.
