# Переменная для названия образа
SERVICE_NAME := reviewer-service

.PHONY: build run clean up

# Сборка Go-приложения
build:
	go build -o ./bin/app cmd/app/main.go

# Поднятие сервиса и базы через Docker Compose
up:
	docker-compose up --build -d

# Остановка сервиса и базы
down:
	docker-compose down

# Запуск тестов (пока пустой, но скоро пригодится)
test:
	go test -v ./...

# Очистка
clean:
	rm -rf ./bin
	docker rmi -f $(SERVICE_NAME)