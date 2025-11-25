package domain

import "context"

// Repository описывает методы работы с базой данных
type Repository interface {
	// Team methods
	CreateTeam(ctx context.Context, team *Team) error
	GetTeamByName(ctx context.Context, name string) (*Team, error)

	// Statistic methods
	GetReviewerStats(ctx context.Context) (map[int]int, error) // Возвращает map[UserID]Count

	// User methods
	CreateUser(ctx context.Context, user *User) error
	GetUserByID(ctx context.Context, id int) (*User, error)
	DeactivateUser(ctx context.Context, id int) error

	// Для алгоритма выбора случайного ревьюера нам нужно получать всех юзеров команды
	GetUsersByTeam(ctx context.Context, teamID int) ([]User, error)

	// PR methods
	CreatePR(ctx context.Context, pr *PullRequest) error
	GetPRByID(ctx context.Context, id int) (*PullRequest, error)
	UpdatePR(ctx context.Context, pr *PullRequest) error // Для смены статуса или ревьюеров
	GetPRsByReviewer(ctx context.Context, reviewerID int) ([]PullRequest, error)
}

// Service описывает бизнес-логику (то, что вызывается из HTTP хендлеров)
type Service interface {
	// Команды и пользователи
	CreateTeam(ctx context.Context, name string) (*Team, error)
	CreateUser(ctx context.Context, name string, teamID int) (*User, error)
	DeleteUser(ctx context.Context, userID int) error // Soft delete / деактивация

	// PR логика
	CreatePR(ctx context.Context, title string, authorID int) (*PullRequest, error)
	MergePR(ctx context.Context, prID int) (*PullRequest, error)

	// Доп функционал (переназначение)
	RerollReviewer(ctx context.Context, prID int, oldReviewerID int) (*PullRequest, error)
	GetReviewerPRs(ctx context.Context, reviewerID int) ([]PullRequest, error)
}
