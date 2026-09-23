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
