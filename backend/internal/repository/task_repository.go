package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"
)

var ErrTaskNotFound = errors.New("task not found")
var ErrTaskNotConfirmed = errors.New("task must be confirmed before publishing")

type TaskListFilter struct {
	Topic string
	Level string
	Sort  string
}

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	if task == nil {
		return errors.New("task is nil")
	}

	now := time.Now().UTC()
	task.ID = 0
	task.Rating = 0
	task.ReadinessLevel = "draft"
	task.Confirmed = false
	task.Published = false
	task.CreatedAt = now
	task.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (
			title,
			initial_description,
			context,
			need,
			users,
			data,
			constraints,
			expected_result,
			success_criteria,
			contact,
			interaction_format,
			topic,
			rating,
			readiness_level,
			confirmed,
			published,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		task.Title,
		task.InitialDescription,
		task.Context,
		task.Need,
		task.Users,
		task.Data,
		task.Constraints,
		task.ExpectedResult,
		task.SuccessCriteria,
		task.Contact,
		task.InteractionFormat,
		task.Topic,
		task.Rating,
		task.ReadinessLevel,
		task.Confirmed,
		task.Published,
		task.CreatedAt.Format(time.RFC3339Nano),
		task.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	task.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get created task id: %w", err)
	}

	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	var task model.Task
	var createdAt string
	var updatedAt string
	var confirmed int
	var published int

	err := r.db.QueryRowContext(ctx, `
		SELECT
			id,
			title,
			initial_description,
			context,
			need,
			users,
			data,
			constraints,
			expected_result,
			success_criteria,
			contact,
			interaction_format,
			topic,
			rating,
			readiness_level,
			confirmed,
			published,
			created_at,
			updated_at
		FROM tasks
		WHERE id = ?
	`, id).Scan(
		&task.ID,
		&task.Title,
		&task.InitialDescription,
		&task.Context,
		&task.Need,
		&task.Users,
		&task.Data,
		&task.Constraints,
		&task.ExpectedResult,
		&task.SuccessCriteria,
		&task.Contact,
		&task.InteractionFormat,
		&task.Topic,
		&task.Rating,
		&task.ReadinessLevel,
		&confirmed,
		&published,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", ErrTaskNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}

	task.Confirmed = confirmed != 0
	task.Published = published != 0
	task.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse task created_at: %w", err)
	}
	task.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse task updated_at: %w", err)
	}

	return &task, nil
}

func (r *TaskRepository) ListPublished(ctx context.Context, filter TaskListFilter) ([]model.Task, error) {
	query := `
		SELECT
			id,
			title,
			initial_description,
			context,
			need,
			users,
			data,
			constraints,
			expected_result,
			success_criteria,
			contact,
			interaction_format,
			topic,
			rating,
			readiness_level,
			confirmed,
			published,
			created_at,
			updated_at
		FROM tasks
		WHERE published = 1`
	args := make([]any, 0, 2)

	if filter.Topic != "" {
		query += " AND topic = ?"
		args = append(args, filter.Topic)
	}
	if filter.Level != "" {
		query += " AND readiness_level = ?"
		args = append(args, filter.Level)
	}
	if filter.Sort == "rating" {
		query += " ORDER BY rating DESC, id DESC"
	} else {
		query += " ORDER BY id DESC"
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list published tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan published task: %w", err)
		}
		tasks = append(tasks, *task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate published tasks: %w", err)
	}

	return tasks, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner rowScanner) (*model.Task, error) {
	var task model.Task
	var createdAt string
	var updatedAt string
	var confirmed int
	var published int

	err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.InitialDescription,
		&task.Context,
		&task.Need,
		&task.Users,
		&task.Data,
		&task.Constraints,
		&task.ExpectedResult,
		&task.SuccessCriteria,
		&task.Contact,
		&task.InteractionFormat,
		&task.Topic,
		&task.Rating,
		&task.ReadinessLevel,
		&confirmed,
		&published,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	task.Confirmed = confirmed != 0
	task.Published = published != 0
	task.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse task created_at: %w", err)
	}
	task.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse task updated_at: %w", err)
	}

	return &task, nil
}

func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	if task == nil {
		return errors.New("task is nil")
	}

	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET
			title = ?,
			context = ?,
			need = ?,
			users = ?,
			data = ?,
			constraints = ?,
			expected_result = ?,
			success_criteria = ?,
			contact = ?,
			interaction_format = ?,
			topic = ?,
			rating = ?,
			readiness_level = ?,
			updated_at = ?
		WHERE id = ?
	`,
		task.Title,
		task.Context,
		task.Need,
		task.Users,
		task.Data,
		task.Constraints,
		task.ExpectedResult,
		task.SuccessCriteria,
		task.Contact,
		task.InteractionFormat,
		task.Topic,
		task.Rating,
		task.ReadinessLevel,
		now.Format(time.RFC3339Nano),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated task count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: %d", ErrTaskNotFound, task.ID)
	}

	task.UpdatedAt = now
	return nil
}

func (r *TaskRepository) Confirm(ctx context.Context, task *model.Task) error {
	if task == nil {
		return errors.New("task is nil")
	}

	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET
			rating = ?,
			readiness_level = ?,
			confirmed = 1,
			updated_at = ?
		WHERE id = ?
	`,
		task.Rating,
		task.ReadinessLevel,
		now.Format(time.RFC3339Nano),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("confirm task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get confirmed task count: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: %d", ErrTaskNotFound, task.ID)
	}

	task.Confirmed = true
	task.UpdatedAt = now
	return nil
}

func (r *TaskRepository) Publish(ctx context.Context, task *model.Task) error {
	if task == nil {
		return errors.New("task is nil")
	}

	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET
			published = 1,
			updated_at = ?
		WHERE id = ? AND confirmed = 1
	`,
		now.Format(time.RFC3339Nano),
		task.ID,
	)
	if err != nil {
		return fmt.Errorf("publish task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get published task count: %w", err)
	}
	if rowsAffected == 0 {
		current, getErr := r.GetByID(ctx, task.ID)
		if errors.Is(getErr, ErrTaskNotFound) {
			return fmt.Errorf("%w: %d", ErrTaskNotFound, task.ID)
		}
		if getErr != nil {
			return fmt.Errorf("check task confirmation: %w", getErr)
		}
		if !current.Confirmed {
			return ErrTaskNotConfirmed
		}
		return fmt.Errorf("publish task affected no rows for task %d", task.ID)
	}

	task.Published = true
	task.UpdatedAt = now
	return nil
}
