package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter настраивает все маршруты для приложения
func SetupRouter(handler *Handler) *gin.Engine {
	// Использование gin.ReleaseMode для продакшена, но пока оставим Default
	router := gin.Default()

	api := router.Group("/api/v1")
	{
		// Teams
		api.POST("/teams", handler.CreateTeam)

		// Users
		api.POST("/users", handler.CreateUser)
		api.DELETE("/users/:id", handler.DeactivateUser) // Деактивация пользователя

		// Pull Requests
		api.POST("/prs", handler.CreatePR)          // Создание PR с автоназначением
		api.POST("/prs/:id/merge", handler.MergePR) // Мердж PR (идемпотентный)

		// Переназначение ревьювера
		api.POST("/prs/:id/reroll", handler.RerollReviewer)

		// Получение PR для ревьювера
		api.GET("/users/:id/prs", handler.GetPRsByReviewer)
	}

	return router
}
