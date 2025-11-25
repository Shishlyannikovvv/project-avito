package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Shishlyannikovvv/project-avito/internal/domain"
	"github.com/gin-gonic/gin"
)

// Handler содержит ссылку на сервис, через который мы вызываем бизнес-логику
type Handler struct {
	service domain.Service
}

func NewHandler(s domain.Service) *Handler {
	return &Handler{service: s}
}

// --- Team Management ---

type createTeamRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var req createTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	team, err := h.service.CreateTeam(c.Request.Context(), req.Name)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, team)
}

// --- User Management ---

type createUserRequest struct {
	Name   string `json:"name" binding:"required"`
	TeamID int    `json:"team_id" binding:"required"`
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), req.Name, req.TeamID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *Handler) DeactivateUser(c *gin.Context) {
	idStr := c.Param("id")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.service.DeleteUser(c.Request.Context(), userID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- PR Management ---

type createPRRequest struct {
	Title    string `json:"title" binding:"required"`
	AuthorID int    `json:"author_id" binding:"required"`
}

func (h *Handler) CreatePR(c *gin.Context) {
	var req createPRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	pr, err := h.service.CreatePR(c.Request.Context(), req.Title, req.AuthorID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, pr)
}

func (h *Handler) MergePR(c *gin.Context) {
	idStr := c.Param("id")
	prID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PR ID"})
		return
	}

	pr, err := h.service.MergePR(c.Request.Context(), prID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr)
}

type rerollReviewerRequest struct {
	OldReviewerID int `json:"old_reviewer_id" binding:"required"`
}

func (h *Handler) RerollReviewer(c *gin.Context) {
	idStr := c.Param("id")
	prID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PR ID"})
		return
	}

	var req rerollReviewerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	pr, err := h.service.RerollReviewer(c.Request.Context(), prID, req.OldReviewerID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, pr)
}

func (h *Handler) GetPRsByReviewer(c *gin.Context) {
	idStr := c.Param("id")
	reviewerID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reviewer ID"})
		return
	}

	prs, err := h.service.GetReviewerPRs(c.Request.Context(), reviewerID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, prs)
}

// --- Error Handling Helper ---

func handleServiceError(c *gin.Context, err error) {
	log.Printf("Service error: %v", err)
	switch err {
	case domain.ErrUserNotFound, domain.ErrTeamNotFound, domain.ErrPRNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "Resource not found"})
	case domain.ErrPRAlreadyMerged:
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case domain.ErrNoReviewersFound:
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}
