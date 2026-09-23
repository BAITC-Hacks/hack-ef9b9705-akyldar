package database

import (
	"database/sql"
	"fmt"
)

const createTasksTable = `
CREATE TABLE IF NOT EXISTS tasks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL DEFAULT '',
	initial_description TEXT NOT NULL,
	context TEXT NOT NULL DEFAULT '',
	need TEXT NOT NULL DEFAULT '',
	users TEXT NOT NULL DEFAULT '',
	data TEXT NOT NULL DEFAULT '',
	constraints TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	success_criteria TEXT NOT NULL DEFAULT '',
	contact TEXT NOT NULL DEFAULT '',
	interaction_format TEXT NOT NULL DEFAULT '',
	topic TEXT NOT NULL DEFAULT '',
	rating INTEGER NOT NULL DEFAULT 0,
	readiness_level TEXT NOT NULL DEFAULT 'draft',
	confirmed INTEGER NOT NULL DEFAULT 0,
	published INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);`

func InitSchema(db *sql.DB) error {
	if _, err := db.Exec(createTasksTable); err != nil {
		return fmt.Errorf("initialize database schema: %w", err)
	}

	return nil
}
