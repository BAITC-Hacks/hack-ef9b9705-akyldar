# Alem frontend

React 19 + Vite 7 frontend for the Alem business-challenge marketplace.

## Requirements and setup

- Node.js 20.19+ (or 22.12+)
- npm, using the committed `package-lock.json`

```bash
cd frontend
npm ci
copy .env.example .env
npm run dev
```

Open the URL printed by Vite, normally `http://localhost:5173`.

## Commands

```bash
npm run dev       # local development server
npm run build     # production build
npm run preview   # preview the production build
```

There are currently no separate lint, TypeScript, or test scripts in `package.json`.

## Environment

```env
VITE_API_URL=http://localhost:8080
VITE_USE_MOCK=false
VITE_DEMO_TEAM_ID=1
```

Real API mode is the default. `VITE_USE_MOCK=true` is an explicit development fallback using seeded demo data and localStorage; it must not be treated as a live backend connection. `VITE_DEMO_TEAM_ID` selects the seeded team used by the proposal form because the current contract has no team-list endpoint.

## Implemented pages and flow

- Overview: entry points into the challenge catalog and builder.
- Explore challenges: published-task catalog with topic, readiness, and rating filters.
- Create challenge: create a draft, request AI questions, answer them, receive a structured card, edit fields, save, confirm, and publish.
- Task details: view the full challenge and open the proposal flow as a student or the proposal workspace as a business user.
- Proposal form: submit a proposal with `team_id`, idea, plan, deadline, and prototype URL.
- Proposal workspace: load proposals and accept or reject each proposal independently.

Network calls are centralized in `src/api/`. See [integration.md](integration.md) for the expected Task, AI, rating, and Proposal formats.

## Current integration status

The frontend is prepared for the agreed Go API at `VITE_API_URL`, but the backend is not part of this frontend directory and live end-to-end verification requires it to be running. AI routes may use the explicit demo-question fallback when unavailable. No backend or AI code is included or changed here.
