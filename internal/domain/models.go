package domain

// Статусы Pull Request
const (
	PRStatusOpen   = "OPEN"
	PRStatusMerged = "MERGED"
)

// Team - команда пользователей
type Team struct {
	ID   int    `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique"`
}

// User - участник команды
type User struct {
	ID       int    `json:"id" gorm:"primaryKey"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
	// Внешний ключ для связи с командой
	TeamID int   `json:"team_id"`
	Team   *Team `json:"team,omitempty" gorm:"foreignKey:TeamID"`
}

// PullRequest - основная сущность задачи
type PullRequest struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	Title     string `json:"title"`
	Status    string `json:"status"` // OPEN | MERGED
	AuthorID  int    `json:"author_id"`
	Author    *User  `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
	Reviewers []User `json:"reviewers" gorm:"many2many:pr_reviewers;"`
}
