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

const createTeamsTable = `
CREATE TABLE IF NOT EXISTS teams (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL DEFAULT '',
	interests TEXT NOT NULL DEFAULT '[]',
	skills TEXT NOT NULL DEFAULT '[]',
	technologies TEXT NOT NULL DEFAULT '[]'
);`

const createProposalsTable = `
CREATE TABLE IF NOT EXISTS proposals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id INTEGER NOT NULL,
	team_id INTEGER NOT NULL,
	idea TEXT NOT NULL DEFAULT '',
	plan TEXT NOT NULL DEFAULT '',
	deadline TEXT NOT NULL DEFAULT '',
	prototype_url TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
	FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE RESTRICT
);`

func InitSchema(db *sql.DB) error {
	statements := []struct {
		name string
		sql  string
	}{
		{name: "tasks", sql: createTasksTable},
		{name: "teams", sql: createTeamsTable},
		{name: "proposals", sql: createProposalsTable},
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement.sql); err != nil {
			return fmt.Errorf("initialize %s table: %w", statement.name, err)
		}
	}

	return nil
}
