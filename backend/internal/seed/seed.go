package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"backend/internal/model"
	"backend/internal/rating"
	"backend/internal/repository"
)

type taskSeed struct {
	title              string
	initialDescription string
	context            string
	need               string
	users              string
	data               string
	constraints        string
	expectedResult     string
	successCriteria    string
	contact            string
	interactionFormat  string
	topic              string
}

type teamSeed struct {
	name         string
	interests    []string
	skills       []string
	technologies []string
}

type proposalSeed struct {
	taskTitle    string
	teamName     string
	idea         string
	plan         string
	deadline     string
	prototypeURL string
	status       string
}

var demoTasks = []taskSeed{
	{
		title:              "[DEMO] Last-mile delivery visibility",
		initialDescription: "Help dispatchers spot delayed deliveries before customers call.",
		context:            "Regional delivery teams coordinate routes manually.",
		topic:              "logistics",
	},
	{
		title:              "[DEMO] Skills practice planner",
		initialDescription: "Create a simple way for learners to plan weekly practice.",
		context:            "Adult learners need a lightweight study routine.",
		need:               "Show the next useful practice activity.",
		data:               "Course modules and learner-selected goals.",
		expectedResult:     "A weekly plan with clear next steps.",
		topic:              "education",
	},
	{
		title:              "[DEMO] Store shelf availability",
		initialDescription: "Help store staff identify products that need replenishment.",
		context:            "Retail teams receive stock updates from several systems.",
		need:               "Prioritize shelves that are likely to be empty.",
		data:               "Stock counts and recent sales by store.",
		expectedResult:     "A replenishment queue for each store.",
		successCriteria:    "Staff can find urgent items in under two minutes.",
		topic:              "retail",
	},
	{
		title:              "[DEMO] Operations insight board",
		initialDescription: "Give operations leads a shared view of weekly performance.",
		context:            "Performance data is reviewed across separate spreadsheets.",
		need:               "Bring the most important trends into one place.",
		data:               "Weekly volume, processing time, and incident counts.",
		constraints:        "The first version must use existing exports.",
		expectedResult:     "A compact board with trend and exception views.",
		successCriteria:    "Leads can identify the largest variance at a glance.",
		topic:              "analytics",
	},
	{
		title:              "[DEMO] Customer support triage",
		initialDescription: "Route incoming support requests to the right response queue.",
		context:            "A customer-service team handles requests from email and chat.",
		need:               "Reduce time spent sorting common requests.",
		users:              "Support agents and service managers.",
		data:               "Anonymized request text and queue categories.",
		constraints:        "Agents must be able to review every suggested route.",
		expectedResult:     "A triage view with suggested queue and reason.",
		successCriteria:    "Most routine requests reach the right queue quickly.",
		contact:            "demo-support@example.com",
		interactionFormat:  "Web dashboard with agent review.",
		topic:              "customer-service",
	},
}

var demoTeams = []teamSeed{
	{
		name:         "[DEMO] ByteBuilders",
		interests:    []string{"AI", "Logistics"},
		skills:       []string{"Backend", "Data Analysis"},
		technologies: []string{"Go", "SQLite"},
	},
	{
		name:         "[DEMO] Learning Loop",
		interests:    []string{"Education", "Product Design"},
		skills:       []string{"Frontend", "Research"},
		technologies: []string{"React", "TypeScript"},
	},
	{
		name:         "[DEMO] ShelfSignals",
		interests:    []string{"Retail", "Automation"},
		skills:       []string{"Data Engineering", "UX"},
		technologies: []string{"Python", "PostgreSQL"},
	},
	{
		name:         "[DEMO] Insight Forge",
		interests:    []string{"Analytics", "Operations"},
		skills:       []string{"Data Visualization", "Backend"},
		technologies: []string{"Go", "React"},
	},
	{
		name:         "[DEMO] Service Studio",
		interests:    []string{"Customer Service", "AI"},
		skills:       []string{"Product Design", "Frontend"},
		technologies: []string{"Python", "SQLite"},
	},
}

var demoProposals = []proposalSeed{
	{
		taskTitle:    "[DEMO] Last-mile delivery visibility",
		teamName:     "[DEMO] ByteBuilders",
		idea:         "A delay board that highlights route exceptions first.",
		plan:         "Start with daily imports, then add dispatcher filters and alerts.",
		deadline:     "2026-10-15",
		prototypeURL: "https://example.com/demo-delivery",
		status:       "pending",
	},
	{
		taskTitle:    "[DEMO] Skills practice planner",
		teamName:     "[DEMO] Learning Loop",
		idea:         "A weekly practice path based on goals and available time.",
		plan:         "Model goals, generate a small plan, and test it with learners.",
		deadline:     "2026-10-20",
		prototypeURL: "https://example.com/demo-learning",
		status:       "accepted",
	},
	{
		taskTitle:    "[DEMO] Store shelf availability",
		teamName:     "[DEMO] ShelfSignals",
		idea:         "A store heatmap for products likely to need restocking.",
		plan:         "Combine stock and sales exports, then validate priorities with staff.",
		deadline:     "2026-10-25",
		prototypeURL: "https://example.com/demo-shelf",
		status:       "rejected",
	},
	{
		taskTitle:    "[DEMO] Operations insight board",
		teamName:     "[DEMO] Insight Forge",
		idea:         "A variance board that keeps weekly exceptions visible.",
		plan:         "Normalize exports, add trend cards, and run a lead review session.",
		deadline:     "2026-11-01",
		prototypeURL: "https://example.com/demo-insight",
		status:       "pending",
	},
	{
		taskTitle:    "[DEMO] Customer support triage",
		teamName:     "[DEMO] Service Studio",
		idea:         "A review-first queue suggestion panel for support agents.",
		plan:         "Define categories, build the review flow, and measure routing time.",
		deadline:     "2026-11-05",
		prototypeURL: "https://example.com/demo-support",
		status:       "accepted",
	},
}

// Run creates the deterministic synthetic demo dataset. Existing demo records
// are reused by their stable titles and names, making repeated runs safe.
func Run(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("seed database is nil")
	}

	taskRepository := repository.NewTaskRepository(db)
	teamRepository := repository.NewTeamRepository(db)
	proposalRepository := repository.NewProposalRepository(db)

	tasks := make(map[string]*model.Task, len(demoTasks))
	for _, seed := range demoTasks {
		task, err := ensureTask(ctx, db, taskRepository, seed)
		if err != nil {
			return fmt.Errorf("seed task %q: %w", seed.title, err)
		}
		tasks[seed.title] = task
	}

	teams := make(map[string]*model.Team, len(demoTeams))
	for _, seed := range demoTeams {
		team, err := ensureTeam(ctx, db, teamRepository, seed)
		if err != nil {
			return fmt.Errorf("seed team %q: %w", seed.name, err)
		}
		teams[seed.name] = team
	}

	for _, seed := range demoProposals {
		task := tasks[seed.taskTitle]
		team := teams[seed.teamName]
		if task == nil || team == nil {
			return fmt.Errorf("seed proposal %q references missing demo record", seed.idea)
		}
		if err := ensureProposal(ctx, db, proposalRepository, task.ID, team.ID, seed); err != nil {
			return fmt.Errorf("seed proposal %q: %w", seed.idea, err)
		}
	}

	return nil
}

func ensureTask(ctx context.Context, db *sql.DB, repo *repository.TaskRepository, seed taskSeed) (*model.Task, error) {
	id, err := findTaskID(ctx, db, seed.title)
	if errors.Is(err, sql.ErrNoRows) {
		task := &model.Task{
			Title:              seed.title,
			InitialDescription: seed.initialDescription,
			Context:            seed.context,
			Need:               seed.need,
			Users:              seed.users,
			Data:               seed.data,
			Constraints:        seed.constraints,
			ExpectedResult:     seed.expectedResult,
			SuccessCriteria:    seed.successCriteria,
			Contact:            seed.contact,
			InteractionFormat:  seed.interactionFormat,
			Topic:              seed.topic,
		}
		if err := repo.Create(ctx, task); err != nil {
			return nil, err
		}
		id = task.ID
	} else if err != nil {
		return nil, fmt.Errorf("find task: %w", err)
	}

	task, err := repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	calculation := rating.Calculate(*task)
	task.Rating = calculation.Score
	task.ReadinessLevel = calculation.Level
	if err := repo.Update(ctx, task); err != nil {
		return nil, err
	}
	if err := repo.Confirm(ctx, task); err != nil {
		return nil, err
	}
	if err := repo.Publish(ctx, task); err != nil {
		return nil, err
	}

	return repo.GetByID(ctx, id)
}

func ensureTeam(ctx context.Context, db *sql.DB, repo *repository.TeamRepository, seed teamSeed) (*model.Team, error) {
	id, err := findTeamID(ctx, db, seed.name)
	if errors.Is(err, sql.ErrNoRows) {
		team := &model.Team{
			Name:         seed.name,
			Interests:    seed.interests,
			Skills:       seed.skills,
			Technologies: seed.technologies,
		}
		if err := repo.CreateTeam(ctx, team); err != nil {
			return nil, err
		}
		return team, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find team: %w", err)
	}

	return repo.GetTeamByID(ctx, id)
}

func ensureProposal(ctx context.Context, db *sql.DB, repo *repository.ProposalRepository, taskID, teamID int64, seed proposalSeed) error {
	id, err := findProposalID(ctx, db, taskID, teamID, seed.idea)
	if errors.Is(err, sql.ErrNoRows) {
		proposal := &model.Proposal{
			TaskID:       taskID,
			TeamID:       teamID,
			Idea:         seed.idea,
			Plan:         seed.plan,
			Deadline:     seed.deadline,
			PrototypeURL: seed.prototypeURL,
		}
		if err := repo.Create(ctx, proposal); err != nil {
			return err
		}
		id = proposal.ID
	} else if err != nil {
		return fmt.Errorf("find proposal: %w", err)
	}

	if seed.status != "pending" {
		if _, err := repo.UpdateStatus(ctx, id, seed.status); err != nil {
			return err
		}
	}
	return nil
}

func findTaskID(ctx context.Context, db *sql.DB, title string) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM tasks WHERE title = ? LIMIT 1", title).Scan(&id)
	return id, err
}

func findTeamID(ctx context.Context, db *sql.DB, name string) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM teams WHERE name = ? LIMIT 1", name).Scan(&id)
	return id, err
}

func findProposalID(ctx context.Context, db *sql.DB, taskID, teamID int64, idea string) (int64, error) {
	var id int64
	err := db.QueryRowContext(ctx, `
		SELECT id
		FROM proposals
		WHERE task_id = ? AND team_id = ? AND idea = ?
		LIMIT 1
	`, taskID, teamID, idea).Scan(&id)
	return id, err
}
