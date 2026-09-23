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
		title:              "[ДЕМО] Контроль доставки на последней миле",
		initialDescription: "Помочь диспетчерам замечать задержки доставки до звонка клиента.",
		context:            "Региональные команды доставки координируют маршруты вручную.",
		topic:              "logistics",
	},
	{
		title:              "[ДЕМО] Планировщик учебной практики",
		initialDescription: "Создать простой способ планировать еженедельную практику.",
		context:            "Взрослым учащимся нужен лёгкий учебный распорядок.",
		need:               "Показывать следующее полезное упражнение.",
		data:               "Модули курса и цели, выбранные учащимися.",
		expectedResult:     "План на неделю с понятными следующими шагами.",
		topic:              "education",
	},
	{
		title:              "[ДЕМО] Наличие товаров на полках",
		initialDescription: "Помочь сотрудникам магазина находить товары, которые нужно пополнить.",
		context:            "Розничные команды получают обновления об остатках из нескольких систем.",
		need:               "Определять полки, которые, вероятно, скоро опустеют.",
		data:               "Остатки и последние продажи по магазинам.",
		expectedResult:     "Очередь пополнения для каждого магазина.",
		successCriteria:    "Сотрудники находят срочные позиции менее чем за две минуты.",
		topic:              "retail",
	},
	{
		title:              "[ДЕМО] Доска операционных показателей",
		initialDescription: "Дать руководителям операций общий обзор результатов за неделю.",
		context:            "Данные о результатах проверяются в отдельных таблицах.",
		need:               "Собрать самые важные тенденции в одном месте.",
		data:               "Недельный объём, время обработки и количество инцидентов.",
		constraints:        "Первая версия должна использовать существующие выгрузки.",
		expectedResult:     "Компактная доска с графиками и исключениями.",
		successCriteria:    "Руководитель сразу видит самое большое отклонение.",
		topic:              "analytics",
	},
	{
		title:              "[ДЕМО] Распределение обращений клиентов",
		initialDescription: "Направлять новые обращения клиентов в подходящую очередь.",
		context:            "Команда поддержки обрабатывает обращения из почты и чата.",
		need:               "Сократить время сортировки типовых обращений.",
		users:              "Специалисты поддержки и руководители сервиса.",
		data:               "Обезличенный текст обращений и категории очередей.",
		constraints:        "Специалисты должны проверять каждое предложенное направление.",
		expectedResult:     "Экран распределения с предложенной очередью и объяснением.",
		successCriteria:    "Большинство типовых обращений быстро попадает в нужную очередь.",
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
		taskTitle:    "[ДЕМО] Контроль доставки на последней миле",
		teamName:     "[DEMO] ByteBuilders",
		idea:         "Доска задержек, которая сначала показывает отклонения маршрутов.",
		plan:         "Начать с ежедневных импортов, затем добавить фильтры и уведомления для диспетчера.",
		deadline:     "2026-10-15",
		prototypeURL: "https://example.com/demo-delivery",
		status:       "pending",
	},
	{
		taskTitle:    "[ДЕМО] Планировщик учебной практики",
		teamName:     "[DEMO] Learning Loop",
		idea:         "Еженедельный маршрут практики с учётом целей и доступного времени.",
		plan:         "Описать цели, составить небольшой план и проверить его с учащимися.",
		deadline:     "2026-10-20",
		prototypeURL: "https://example.com/demo-learning",
		status:       "accepted",
	},
	{
		taskTitle:    "[ДЕМО] Наличие товаров на полках",
		teamName:     "[DEMO] ShelfSignals",
		idea:         "Тепловая карта товаров, которым, вероятно, потребуется пополнение.",
		plan:         "Объединить выгрузки остатков и продаж, затем проверить приоритеты с сотрудниками.",
		deadline:     "2026-10-25",
		prototypeURL: "https://example.com/demo-shelf",
		status:       "rejected",
	},
	{
		taskTitle:    "[ДЕМО] Доска операционных показателей",
		teamName:     "[DEMO] Insight Forge",
		idea:         "Доска отклонений, которая сохраняет недельные исключения на виду.",
		plan:         "Нормализовать выгрузки, добавить карточки тенденций и провести обзор с руководителем.",
		deadline:     "2026-11-01",
		prototypeURL: "https://example.com/demo-insight",
		status:       "pending",
	},
	{
		taskTitle:    "[ДЕМО] Распределение обращений клиентов",
		teamName:     "[DEMO] Service Studio",
		idea:         "Панель предложений очереди с обязательной проверкой специалистом.",
		plan:         "Определить категории, собрать процесс проверки и измерить время распределения.",
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
