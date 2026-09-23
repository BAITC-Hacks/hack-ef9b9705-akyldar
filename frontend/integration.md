# Frontend integration notes

This document records the backend contract expected by the frontend. It does not add or change backend routes.

## Configuration

Copy `.env.example` to `.env`:

```env
VITE_API_URL=http://localhost:8080
VITE_USE_MOCK=false
VITE_DEMO_TEAM_ID=1
```

`VITE_USE_MOCK=true` is an explicit local fallback only. It uses the existing seeded demo data and localStorage; it is not a connection to the real API.

## Task flow

The API layer in `src/api/tasks.js` expects these routes and payloads:

- `POST /api/tasks` with `{ "initial_description": "...", "topic": "..." }`.
- `GET /api/tasks` with optional `topic`, `level`, and `sort=rating` query parameters.
- `GET /api/tasks/{id}`.
- `PUT /api/tasks/{id}` with only the editable Task fields: `title`, `context`, `need`, `users`, `data`, `constraints`, `expected_result`, `success_criteria`, `contact`, `interaction_format`, and `topic`.
- `GET /api/tasks/{id}/rating`.
- `POST /api/tasks/{id}/confirm` without a body.
- `POST /api/tasks/{id}/publish` without a body.

The frontend does not send Task internal fields such as `id`, `rating`, `readiness_level`, `confirmed`, `published`, or timestamps in a PUT request.

## AI flow

`src/api/ai.js` is ready for the agreed AI contract:

- `POST /api/ai/questions` with `{ "description": "..." }` and a `{ "questions": ["..."] }` response.
- `POST /api/ai/card` with `{ "description": "...", "answers": [{ "question": "...", "answer": "..." }] }`.

If the AI routes are unavailable, the UI uses the explicit demo-question fallback and shows that fallback to the user.

## Proposal flow

`src/api/proposals.js` expects:

- `POST /api/tasks/{id}/proposals` with required `team_id`, `idea`, `plan`, `deadline`, and `prototype_url`.
- `GET /api/tasks/{id}/proposals`.
- `PATCH /api/proposals/{id}/status` with `{ "status": "accepted" }` or `{ "status": "rejected" }`.

The current backend contract does not provide a team-list route, so the proposal form uses the seeded team ID from `VITE_DEMO_TEAM_ID`.

## Error handling

The API client reads the backend `{ "error": "..." }` response shape and shows a short user-facing message. It does not render raw exception stacks.
