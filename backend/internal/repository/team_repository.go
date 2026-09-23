package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"backend/internal/model"
)

var ErrTeamNotFound = errors.New("team not found")

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) CreateTeam(ctx context.Context, team *model.Team) error {
	if team == nil {
		return errors.New("team is nil")
	}

	interests, err := marshalStringSlice(team.Interests)
	if err != nil {
		return fmt.Errorf("encode team interests: %w", err)
	}
	skills, err := marshalStringSlice(team.Skills)
	if err != nil {
		return fmt.Errorf("encode team skills: %w", err)
	}
	technologies, err := marshalStringSlice(team.Technologies)
	if err != nil {
		return fmt.Errorf("encode team technologies: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO teams (name, interests, skills, technologies)
		VALUES (?, ?, ?, ?)
	`, team.Name, interests, skills, technologies)
	if err != nil {
		return fmt.Errorf("create team: %w", err)
	}

	team.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get created team id: %w", err)
	}

	return nil
}

func (r *TeamRepository) GetTeamByID(ctx context.Context, id int64) (*model.Team, error) {
	var team model.Team
	var interests string
	var skills string
	var technologies string

	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, interests, skills, technologies
		FROM teams
		WHERE id = ?
	`, id).Scan(&team.ID, &team.Name, &interests, &skills, &technologies)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: %d", ErrTeamNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}

	team.Interests, err = unmarshalStringSlice(interests)
	if err != nil {
		return nil, fmt.Errorf("decode team interests: %w", err)
	}
	team.Skills, err = unmarshalStringSlice(skills)
	if err != nil {
		return nil, fmt.Errorf("decode team skills: %w", err)
	}
	team.Technologies, err = unmarshalStringSlice(technologies)
	if err != nil {
		return nil, fmt.Errorf("decode team technologies: %w", err)
	}

	return &team, nil
}

func marshalStringSlice(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}

	encoded, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func unmarshalStringSlice(value string) ([]string, error) {
	var values []string
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return nil, err
	}
	if values == nil {
		values = []string{}
	}
	return values, nil
}
