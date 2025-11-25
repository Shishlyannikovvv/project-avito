package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL   = "http://localhost:8080/api/v1"
	rpsGoal   = 5                // RPS - 5
	duration  = 10 * time.Second // Тестируем 10 секунд
	userCount = 10               // Количество пользователей для теста
)

type user struct {
	ID     int `json:"id"`
	TeamID int `json:"team_id"`
}
type pr struct {
	ID int `json:"id"`
}

func main() {
	log.Println("Starting stress test setup...")

	// 1. Подготовка данных: Создание команды и пользователей
	teamID, users := setupTestData()

	// Проверка, что есть хотя бы 3 активных пользователя для теста
	if len(users) < 3 {
		log.Fatal("Not enough users for stress test. Need at least 3 active users in the team.")
	}

	log.Printf("Setup complete. Team ID: %d, Users: %d. Starting test for %s at %d RPS.", teamID, len(users), duration, rpsGoal)

	var wg sync.WaitGroup
	ticker := time.NewTicker(time.Second / time.Duration(rpsGoal))
	defer ticker.Stop()

	// Таймер для ограничения продолжительности теста
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var requestCounter int64
	var successCounter int64

	start := time.Now()

	// 2. Запуск горутин для RPS
	for i := 0; i < int(duration.Seconds()*float64(rpsGoal)); i++ {
		select {
		case <-ctx.Done():
			goto endLoop
		case <-ticker.C:
			wg.Add(1)
			requestCounter++

			// Выбираем случайного автора
			author := users[i%len(users)]

			go func(author user) {
				defer wg.Done()

				// 3. Создание PR
				prID, err := createPR(author.ID)
				if err != nil {
					log.Printf("Error creating PR for user %d: %v", author.ID, err)
					return
				}

				// 4. Мердж PR (для полной нагрузки)
				if err := mergePR(prID); err != nil {
					log.Printf("Error merging PR %d: %v", prID, err)
					return
				}

				successCounter++
			}(author)
		}
	}

endLoop:
	wg.Wait()

	elapsed := time.Since(start)

	// 5. Вывод результатов
	log.Println("--- Stress Test Results ---")
	log.Printf("Duration: %s", elapsed.Round(time.Millisecond))
	log.Printf("Total Requests Sent (Create PR + Merge): %d", requestCounter*2)
	log.Printf("Successful PR Cycles (Create + Merge): %d", successCounter)
	log.Printf("Measured RPS: %.2f (Goal: %d)", float64(requestCounter)/elapsed.Seconds(), rpsGoal)
	log.Printf("Success SLI: %.2f%% (Goal: 99.9%%)", float64(successCounter*2)/float64(requestCounter*2)*100)
}

// --- Helper Functions ---

func setupTestData() (int, []user) {
	// Создание команды
	var teamID int
	resp, _ := http.Post(baseURL+"/teams", "application/json", bytes.NewBufferString(`{"name": "StresserTeam"}`))
	if resp != nil && resp.StatusCode == http.StatusCreated {
		var t struct {
			ID int `json:"id"`
		}
		json.NewDecoder(resp.Body).Decode(&t)
		teamID = t.ID
		resp.Body.Close()
	} else {
		log.Fatal("Failed to create test team. Is the server running?")
	}

	// Создание пользователей
	users := make([]user, userCount)
	for i := 0; i < userCount; i++ {
		userData := fmt.Sprintf(`{"name": "StressUser_%d", "team_id": %d}`, i, teamID)
		resp, _ := http.Post(baseURL+"/users", "application/json", bytes.NewBufferString(userData))
		if resp != nil && resp.StatusCode == http.StatusCreated {
			var u user
			json.NewDecoder(resp.Body).Decode(&u)
			users[i] = u
			resp.Body.Close()
		}
	}
	return teamID, users
}

func createPR(authorID int) (int, error) {
	prData := fmt.Sprintf(`{"title": "Test PR by %d", "author_id": %d}`, authorID, authorID)
	resp, err := http.Post(baseURL+"/prs", "application/json", bytes.NewBufferString(prData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create PR, status: %d, body: %s", resp.StatusCode, body)
	}

	var p pr
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return 0, err
	}
	return p.ID, nil
}

func mergePR(prID int) error {
	url := fmt.Sprintf("%s/prs/%d/merge", baseURL, prID)
	req, _ := http.NewRequest(http.MethodPost, url, nil)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to merge PR %d, status: %d", prID, resp.StatusCode)
	}
	return nil
}
