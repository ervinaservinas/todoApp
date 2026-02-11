# Task App (Go + Web)

This is a simple to-do app with:
- Backend: Go (`net/http`)
- Frontend: Vanilla HTML/CSS/JavaScript
- Storage: `data/tasks.json` (file-based persistence)

## Why this frontend
For your first version, vanilla JS is the fastest way to get a working app.
After you are comfortable, the best upgrade is **React + TypeScript**.

## Run

```bash
go run .
```

Open:

```text
http://localhost:8080
```

## API

- `GET /api/tasks` -> list tasks
- `POST /api/tasks` -> create task (`{"title":"Buy milk"}`)
- `PATCH /api/tasks/:id` -> toggle done/undone
- `DELETE /api/tasks/:id` -> delete task

## Project structure

- `main.go` - API + static file server
- `web/index.html` - UI markup
- `web/styles.css` - UI styles
- `web/app.js` - frontend logic
- `data/tasks.json` - saved tasks (auto-created)

## Next improvements

1. Add user accounts (login/register + per-user tasks)
2. Replace file storage with PostgreSQL
3. Add due dates, priorities, and filters
4. Move frontend to React + TypeScript
# todoApp
