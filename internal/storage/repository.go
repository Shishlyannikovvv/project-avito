package storage

import (
	"context"

	"github.com/Shishlyannikovvv/project-avito/internal/domain"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// --- Team ---

func (r *Repository) CreateTeam(ctx context.Context, team *domain.Team) error {
	return r.db.WithContext(ctx).Create(team).Error
}

func (r *Repository) GetTeamByName(ctx context.Context, name string) (*domain.Team, error) {
	var team domain.Team
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&team).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrTeamNotFound
		}
		return nil, err
	}
	return &team, nil
}

// --- User ---

func (r *Repository) CreateUser(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Preload("Team").First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) DeactivateUser(ctx context.Context, id int) error {
	// Обновляем поле IsActive на false
	result := r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *Repository) GetUsersByTeam(ctx context.Context, teamID int) ([]domain.User, error) {
	var users []domain.User
	// Нам нужны только активные пользователи для назначения ревью
	err := r.db.WithContext(ctx).Where("team_id = ? AND is_active = ?", teamID, true).Find(&users).Error
	return users, err
}

// --- Pull Request ---

func (r *Repository) CreatePR(ctx context.Context, pr *domain.PullRequest) error {
	return r.db.WithContext(ctx).Create(pr).Error
}

func (r *Repository) GetPRByID(ctx context.Context, id int) (*domain.PullRequest, error) {
	var pr domain.PullRequest
	// Preload загружает связанные сущности (Автора и Список ревьюеров)
	err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Reviewers").
		First(&pr, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrPRNotFound
		}
		return nil, err
	}
	return &pr, nil
}

func (r *Repository) UpdatePR(ctx context.Context, pr *domain.PullRequest) error {
	// Save сохраняет все поля, включая обновленные ассоциации (Reviewers)
	return r.db.WithContext(ctx).Save(pr).Error
}

func (r *Repository) GetPRsByReviewer(ctx context.Context, reviewerID int) ([]domain.PullRequest, error) {
	var prs []domain.PullRequest
	// Сложный запрос: найти PR, где в списке ревьюеров есть наш юзер
	// Используем JOIN таблицу pr_reviewers, которую GORM создал автоматически
	err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Reviewers").
		Joins("JOIN pr_reviewers ON pr_reviewers.pull_request_id = pull_requests.id").
		Where("pr_reviewers.user_id = ?", reviewerID).
		Find(&prs).Error

	return prs, err
}

// GetReviewerStats подсчитывает, сколько PR назначено каждому пользователю
func (r *Repository) GetReviewerStats(ctx context.Context) (map[int]int, error) {
	var results []struct {
		UserID int
		Count  int64
	}

	// Запрос к join-таблице many2many pr_reviewers
	err := r.db.WithContext(ctx).
		Model(&domain.PullRequest{}).
		Select("pr_reviewers.user_id, count(pull_request_id) as count").
		Joins("JOIN pr_reviewers ON pr_reviewers.pull_request_id = pull_requests.id").
		Group("pr_reviewers.user_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	stats := make(map[int]int)
	for _, res := range results {
		stats[res.UserID] = int(res.Count)
	}

	return stats, nil
}
