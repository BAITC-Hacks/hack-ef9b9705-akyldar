package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"backend/internal/model"
)

var ErrProposalNotFound = errors.New("proposal not found")

type ProposalRepository struct {
	db *sql.DB
}

func NewProposalRepository(db *sql.DB) *ProposalRepository {
	return &ProposalRepository{db: db}
}

func (r *ProposalRepository) Create(ctx context.Context, proposal *model.Proposal) error {
	if proposal == nil {
		return errors.New("proposal is nil")
	}

	now := time.Now().UTC()
	proposal.ID = 0
	proposal.Status = "pending"
	proposal.CreatedAt = now
	proposal.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO proposals (
			task_id,
			team_id,
			idea,
			plan,
			deadline,
			prototype_url,
			status,
			created_at,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		proposal.TaskID,
		proposal.TeamID,
		proposal.Idea,
		proposal.Plan,
		proposal.Deadline,
		proposal.PrototypeURL,
		proposal.Status,
		proposal.CreatedAt.Format(time.RFC3339Nano),
		proposal.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create proposal: %w", err)
	}

	proposal.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get created proposal id: %w", err)
	}

	return nil
}

func (r *ProposalRepository) ListByTaskID(ctx context.Context, taskID int64) ([]model.Proposal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id,
			task_id,
			team_id,
			idea,
			plan,
			deadline,
			prototype_url,
			status,
			created_at,
			updated_at
		FROM proposals
		WHERE task_id = ?
		ORDER BY id DESC
	`, taskID)
	if err != nil {
		return nil, fmt.Errorf("list proposals: %w", err)
	}
	defer rows.Close()

	proposals := make([]model.Proposal, 0)
	for rows.Next() {
		proposal, err := scanProposal(rows)
		if err != nil {
			return nil, fmt.Errorf("scan proposal: %w", err)
		}
		proposals = append(proposals, *proposal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proposals: %w", err)
	}

	return proposals, nil
}

func (r *ProposalRepository) UpdateStatus(ctx context.Context, id int64, status string) (*model.Proposal, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `
		UPDATE proposals
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, status, now.Format(time.RFC3339Nano), id)
	if err != nil {
		return nil, fmt.Errorf("update proposal status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("get updated proposal count: %w", err)
	}
	if rowsAffected == 0 {
		return nil, fmt.Errorf("%w: %d", ErrProposalNotFound, id)
	}

	proposal, err := r.getByID(ctx, id)
	if errors.Is(err, ErrProposalNotFound) {
		return nil, fmt.Errorf("%w: %d", ErrProposalNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get updated proposal: %w", err)
	}

	return proposal, nil
}

func (r *ProposalRepository) getByID(ctx context.Context, id int64) (*model.Proposal, error) {
	proposal, err := scanProposal(r.db.QueryRowContext(ctx, `
		SELECT
			id,
			task_id,
			team_id,
			idea,
			plan,
			deadline,
			prototype_url,
			status,
			created_at,
			updated_at
		FROM proposals
		WHERE id = ?
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", ErrProposalNotFound, id)
	}
	if err != nil {
		return nil, err
	}

	return proposal, nil
}

type proposalScanner interface {
	Scan(dest ...any) error
}

func scanProposal(scanner proposalScanner) (*model.Proposal, error) {
	var proposal model.Proposal
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(
		&proposal.ID,
		&proposal.TaskID,
		&proposal.TeamID,
		&proposal.Idea,
		&proposal.Plan,
		&proposal.Deadline,
		&proposal.PrototypeURL,
		&proposal.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	var err error
	proposal.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return nil, fmt.Errorf("parse proposal created_at: %w", err)
	}
	proposal.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse proposal updated_at: %w", err)
	}

	return &proposal, nil
}
