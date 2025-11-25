package service_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/Shishlyannikovvv/project-avito/internal/domain"
	"github.com/Shishlyannikovvv/project-avito/internal/service"
	"github.com/Shishlyannikovvv/project-avito/internal/storage"
	"github.com/stretchr/testify/assert"
)

// Тестовые константы (должны совпадать с docker-compose)
const (
	TestDBHost     = "localhost" // При запуске вне docker-compose
	TestDBUser     = "postgres"
	TestDBPassword = "postgres"
	TestDBName     = "reviewer_db"
	TestDBPort     = "5432"
)

var (
	testRepo    domain.Repository
	testService domain.Service
)

func TestMain(m *testing.M) {
	// Инициализация тестовой БД
	db, err := storage.NewPostgresDB(TestDBHost, TestDBUser, TestDBPassword, TestDBName, TestDBPort)
	if err != nil {
		log.Fatalf("Could not connect to test DB: %v", err)
	}

	// Используем gorm.DB для очистки таблиц перед каждым тестом
	testRepo = storage.NewRepository(db)
	testService = service.NewManager(testRepo)

	code := m.Run()
	os.Exit(code)
}

// setupTest очищает таблицы перед каждым тестом
func setupTest(t *testing.T) {
	// GORM не предоставляет простой способ очистки Many-to-Many таблиц, поэтому используем raw SQL
	gdb := testRepo.(*storage.Repository).DB() // Получаем доступ к gorm.DB

	gdb.Exec("TRUNCATE pr_reviewers, pull_requests, users, teams RESTART IDENTITY;")
}

func TestPRAssignmentAndMerge(t *testing.T) {
	setupTest(t)
	ctx := context.Background()

	// 1. Создание команды
	team, err := testService.CreateTeam(ctx, "Avengers")
	assert.NoError(t, err)
	assert.NotNil(t, team)

	// 2. Создание пользователей
	userAuthor, _ := testService.CreateUser(ctx, "Tony Stark", team.ID)
	userReviewer1, _ := testService.CreateUser(ctx, "Steve Rogers", team.ID)
	userReviewer2, _ := testService.CreateUser(ctx, "Bruce Banner", team.ID)
	userInactive, _ := testService.CreateUser(ctx, "Thor (inactive)", team.ID)
	testService.DeleteUser(ctx, userInactive.ID) // Деактивируем

	// 3. Создание PR (должно назначить до 2 активных ревьюеров: userReviewer1, userReviewer2)
	pr, err := testService.CreatePR(ctx, "Fix the reactor", userAuthor.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.PRStatusOpen, pr.Status)
	assert.Len(t, pr.Reviewers, 2, "Should assign exactly 2 active reviewers")

	// Проверяем, что автор не назначен
	for _, r := range pr.Reviewers {
		assert.NotEqual(t, userAuthor.ID, r.ID, "Author should not be a reviewer")
		assert.True(t, r.IsActive, "Reviewer must be active")
	}

	// 4. Мердж PR
	mergedPR, err := testService.MergePR(ctx, pr.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.PRStatusMerged, mergedPR.Status)

	// 5. Попытка изменить ревьюера после мерджа (должно вернуть ошибку)
	err = pr.Reviewers[0].ID
	_, errReroll := testService.RerollReviewer(ctx, mergedPR.ID, err)
	assert.ErrorIs(t, errReroll, domain.ErrPRAlreadyMerged, "Cannot reroll merged PR")

	// 6. Проверка идемпотентности (повторный мердж не должен вызывать ошибку)
	mergedPR2, err := testService.MergePR(ctx, pr.ID)
	assert.NoError(t, err)
	assert.Equal(t, domain.PRStatusMerged, mergedPR2.Status)
}

func TestRerollReviewer(t *testing.T) {
	setupTest(t)
	ctx := context.Background()

	team, _ := testService.CreateTeam(ctx, "Justice League")
	userAuthor, _ := testService.CreateUser(ctx, "Clark Kent", team.ID)
	userOldReviewer, _ := testService.CreateUser(ctx, "Diana Prince", team.ID)
	userReviewer2, _ := testService.CreateUser(ctx, "Barry Allen", team.ID)
	userNewCandidate, _ := testService.CreateUser(ctx, "Victor Stone", team.ID)
	userCandidateInactive, _ := testService.CreateUser(ctx, "Arthur Curry", team.ID)
	testService.DeleteUser(ctx, userCandidateInactive.ID)

	// Создаем PR вручную (чтобы знать, кто назначен)
	pr := &domain.PullRequest{
		Title: "Test Reroll", Status: domain.PRStatusOpen, AuthorID: userAuthor.ID,
		Reviewers: []domain.User{*userOldReviewer, *userReviewer2},
	}
	testRepo.CreatePR(ctx, pr)

	// Переназначаем userOldReviewer на userNewCandidate
	rerolledPR, err := testService.RerollReviewer(ctx, pr.ID, userOldReviewer.ID)
	assert.NoError(t, err)

	// Проверяем, что старый ревьюер удален, а новый добавлен
	foundOld := false
	foundNew := false
	foundSecond := false

	for _, r := range rerolledPR.Reviewers {
		if r.ID == userOldReviewer.ID {
			foundOld = true
		}
		if r.ID == userNewCandidate.ID {
			foundNew = true
		}
		if r.ID == userReviewer2.ID {
			foundSecond = true
		}
	}

	assert.False(t, foundOld, "Old reviewer should be removed")
	assert.True(t, foundNew, "New candidate should be assigned")
	assert.True(t, foundSecond, "Second reviewer should remain")
	assert.Len(t, rerolledPR.Reviewers, 2)
}

// Тест для проверки массовой деактивации и переназначения
func TestMassDeactivate(t *testing.T) {
	setupTest(t)
	ctx := context.Background()

	// 1. Создание команд и пользователей
	teamA, _ := testService.CreateTeam(ctx, "Team Alpha")
	teamB, _ := testService.CreateTeam(ctx, "Team Beta")

	// Пользователи Team A
	userA1, _ := testService.CreateUser(ctx, "U A1 Author", teamA.ID)
	userA2, _ := testService.CreateUser(ctx, "U A2 Reviewer", teamA.ID)  // Будет деактивирован
	userA3, _ := testService.CreateUser(ctx, "U A3 Reviewer", teamA.ID)  // Будет деактивирован
	userA4_safe, _ := testService.CreateUser(ctx, "U A4 Safe", teamA.ID) // Кандидат на переназначение

	// Пользователь Team B (должен остаться нетронутым)
	userB1, _ := testService.CreateUser(ctx, "U B1", teamB.ID)

	// 2. Создание PR, где A2 и A3 - ревьюеры
	pr1, _ := testService.CreatePR(ctx, "PR by A1", userA1.ID)

	// Ручная проверка, что A2 и A3 были назначены
	pr1, _ = testRepo.GetPRByID(ctx, pr1.ID)

	// Изначально A2 и A3 назначены, A4 свободен.
	initialReviewers := make(map[int]bool)
	for _, r := range pr1.Reviewers {
		initialReviewers[r.ID] = true
	}

	// 3. Массовая деактивация Team A (A2 и A3 деактивируются и должны быть переназначены)
	err := testService.MassDeactivateTeamUsers(ctx, teamA.ID)
	assert.NoError(t, err)

	// 4. Проверка: A2 и A3 должны быть неактивны
	checkA2, _ := testRepo.GetUserByID(ctx, userA2.ID)
	checkA3, _ := testRepo.GetUserByID(ctx, userA3.ID)
	assert.False(t, checkA2.IsActive, "User A2 must be deactivated")
	assert.False(t, checkA3.IsActive, "User A3 must be deactivated")

	// 5. Проверка: PR1 должен иметь новых активных ревьюеров
	updatedPR1, _ := testRepo.GetPRByID(ctx, pr1.ID)

	newReviewerCount := 0
	newReviewerActiveCount := 0

	for _, r := range updatedPR1.Reviewers {
		newReviewerCount++
		if r.IsActive {
			newReviewerActiveCount++
		}
		// Проверяем, что ни один из старых (A2, A3) не остался
		assert.False(t, initialReviewers[r.ID], "Old reviewer (A2/A3) must be replaced")
		// Проверяем, что назначен только активный кандидат (A4)
		assert.True(t, r.ID == userA4_safe.ID || r.ID == userB1.ID, "Assigned reviewer should be the safe candidate")
	}

	// Должны быть назначены 2 активных ревьюера (A4 - единственный активный в команде A)
	// Так как A4 - единственный кандидат, то Reroll попытается заменить обоих на A4, что не получится,
	// но в итоге останется 1 ревьюер (A4).
	assert.Len(t, updatedPR1.Reviewers, 1, "Should result in 1 reviewer (A4 is the only active candidate)")
	assert.Equal(t, 1, newReviewerActiveCount, "The assigned reviewer must be active")
}
