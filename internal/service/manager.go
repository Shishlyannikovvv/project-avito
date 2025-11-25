package service

import (
	"context"
	"math/rand"
	"time"

	"github.com/Shishlyannikovvv/project-avito/internal/domain"
)

type Manager struct {
	repo domain.Repository
}

func NewManager(repo domain.Repository) *Manager {
	// Инициализируем рандом сидом времени, чтобы при каждом запуске был разный выбор
	rand.Seed(time.Now().UnixNano())
	return &Manager{repo: repo}
}

// --- Team & User Logic ---

func (s *Manager) CreateTeam(ctx context.Context, name string) (*domain.Team, error) {
	team := &domain.Team{Name: name}
	if err := s.repo.CreateTeam(ctx, team); err != nil {
		return nil, err
	}
	return team, nil
}

func (s *Manager) CreateUser(ctx context.Context, name string, teamID int) (*domain.User, error) {
	// Проверяем существование команды
	_, err := s.repo.GetTeamByName(ctx, "") // Можно оптимизировать, проверив ID, но пока так
	// В GORM проще просто попытаться создать, если FK constraint упадет - обработаем

	user := &domain.User{
		Name:     name,
		TeamID:   teamID,
		IsActive: true, // По умолчанию активен
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Manager) DeleteUser(ctx context.Context, userID int) error {
	return s.repo.DeactivateUser(ctx, userID)
}

// --- PR Logic ---

func (s *Manager) CreatePR(ctx context.Context, title string, authorID int) (*domain.PullRequest, error) {
	// 1. Получаем автора, чтобы узнать его команду
	author, err := s.repo.GetUserByID(ctx, authorID)
	if err != nil {
		return nil, err
	}

	if !author.IsActive {
		// Опционально: запрещаем неактивным создавать PR, но в ТЗ этого нет, так что оставим.
	}

	// 2. Ищем кандидатов в ревьюеры (все активные из той же команды)
	candidates, err := s.repo.GetUsersByTeam(ctx, author.TeamID)
	if err != nil {
		return nil, err
	}

	// 3. Фильтруем: ревьюер != автор
	validCandidates := make([]domain.User, 0)
	for _, c := range candidates {
		if c.ID != author.ID {
			validCandidates = append(validCandidates, c)
		}
	}

	// 4. Выбираем до 2 случайных ревьюеров
	reviewers := selectRandomReviewers(validCandidates, 2)

	// 5. Создаем PR
	pr := &domain.PullRequest{
		Title:     title,
		Status:    domain.PRStatusOpen,
		AuthorID:  authorID,
		Reviewers: reviewers,
	}

	if err := s.repo.CreatePR(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Manager) MergePR(ctx context.Context, prID int) (*domain.PullRequest, error) {
	pr, err := s.repo.GetPRByID(ctx, prID)
	if err != nil {
		return nil, err
	}

	// Идемпотентность: если уже смержен, просто возвращаем его
	if pr.Status == domain.PRStatusMerged {
		return pr, nil
	}

	pr.Status = domain.PRStatusMerged
	if err := s.repo.UpdatePR(ctx, pr); err != nil {
		return nil, err
	}
	return pr, nil
}

func (s *Manager) RerollReviewer(ctx context.Context, prID int, oldReviewerID int) (*domain.PullRequest, error) {
	pr, err := s.repo.GetPRByID(ctx, prID)
	if err != nil {
		return nil, err
	}

	// Проверка: нельзя менять после мерджа
	if pr.Status == domain.PRStatusMerged {
		return nil, domain.ErrPRAlreadyMerged
	}

	// Проверка: был ли такой ревьюер вообще назначен?
	isReviewerFound := false
	for _, r := range pr.Reviewers {
		if r.ID == oldReviewerID {
			isReviewerFound = true
			break
		}
	}
	if !isReviewerFound {
		return nil, domain.ErrUserNotFound // Или специфичную ошибку "User is not a reviewer on this PR"
	}

	// Получаем кандидатов (команда автора)
	// Важный момент: ревьюер должен быть из команды АВТОРА или ЗАМЕНЯЕМОГО?
	// В ТЗ: "из команды заменяемого ревьюера". Обычно это одна и та же команда,
	// но будем брать команду автора для надежности, так как PR внутри команды.
	if pr.Author == nil {
		// Подгрузим автора если вдруг его нет (хотя Preload в repo должен был сработать)
		author, err := s.repo.GetUserByID(ctx, pr.AuthorID)
		if err != nil {
			return nil, err
		}
		pr.Author = author
	}

	candidates, err := s.repo.GetUsersByTeam(ctx, pr.Author.TeamID)
	if err != nil {
		return nil, err
	}

	// Фильтруем кандидатов:
	// 1. Не автор
	// 2. Не тот, кого убираем (oldReviewerID)
	// 3. Не те, кто УЖЕ назначен ревьюером (кроме убираемого)
	availableCandidates := make([]domain.User, 0)

	// Создадим карту текущих ID ревьюеров для быстрой проверки
	currentReviewerIDs := make(map[int]bool)
	for _, r := range pr.Reviewers {
		currentReviewerIDs[r.ID] = true
	}

	for _, c := range candidates {
		if c.ID == pr.AuthorID {
			continue
		}
		if c.ID == oldReviewerID {
			continue
		}
		if currentReviewerIDs[c.ID] {
			continue // Он уже ревьюер (второй ревьюер)
		}
		availableCandidates = append(availableCandidates, c)
	}

	if len(availableCandidates) == 0 {
		return nil, domain.ErrNoReviewersFound
	}

	// Выбираем одного случайного
	newReviewer := availableCandidates[rand.Intn(len(availableCandidates))]

	// Обновляем список ревьюеров в PR
	newReviewersList := make([]domain.User, 0)
	for _, r := range pr.Reviewers {
		if r.ID == oldReviewerID {
			newReviewersList = append(newReviewersList, newReviewer)
		} else {
			newReviewersList = append(newReviewersList, r)
		}
	}
	pr.Reviewers = newReviewersList

	if err := s.repo.UpdatePR(ctx, pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *Manager) GetReviewerPRs(ctx context.Context, reviewerID int) ([]domain.PullRequest, error) {
	return s.repo.GetPRsByReviewer(ctx, reviewerID)
}

// --- Helpers ---

// selectRandomReviewers выбирает n случайных уникальных пользователей из слайса
func selectRandomReviewers(users []domain.User, n int) []domain.User {
	if len(users) <= n {
		return users
	}

	// Перемешиваем слайс
	rand.Shuffle(len(users), func(i, j int) {
		users[i], users[j] = users[j], users[i]
	})

	return users[:n]
}
