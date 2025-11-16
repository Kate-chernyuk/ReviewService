package handlers

import (
	"PR/models"
	"PR/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.ReviewService
}

func NewHandler(service *service.ReviewService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateTeam(c *gin.Context) {
	var team models.Team
	if err := c.ShouldBindJSON(&team); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	if err := h.service.CreateTeam(team); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("TEAM_EXISTS", team.TeamName+" already exists"))
	}

	c.JSON(http.StatusCreated, gin.H{"team": team})
}

func (h *Handler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")

	team, err := h.service.GetTeam(teamName)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "team not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"team": team})
}

func (h *Handler) SetUserActive(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	user, err := h.service.SetUserActive(req.UserID, req.IsActive)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "user not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *Handler) CreatePR(c *gin.Context) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	pr, err := h.service.CreatePR(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		switch err.Error() {
		case "PR уже существует":
			c.JSON(http.StatusConflict, errorResponse("PR_EXISTS", "PR id already exists"))
		case "Автор не найден", "Команда не найдена":
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "author/team not found"))
		default:
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"pr": pr})
}

func (h *Handler) MergePR(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	pr, err := h.service.MergePR(req.PullRequestID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "PR not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"pr": pr})
}

func (h *Handler) ReassignReviewer(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	pr, newUserID, err := h.service.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		switch err.Error() {
		case "Нельзя переназначать ревьюера на замердженном PR":
			c.JSON(http.StatusConflict, errorResponse("PR_MERGED", "cannot reassign on merged PR"))
		case "Данный ревьюер и не был назначен на данный PR":
			c.JSON(http.StatusConflict, errorResponse("NOT_ASSIGNED", "reviewer is not assigned to this PR"))
		case "Нет доступных кандидатов для замены":
			c.JSON(http.StatusConflict, errorResponse("NO_CANDIDATE", "no active replacement candidate in team"))
		case "PR не найден", "Пользователь не найден":
			c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "PR/user not found"))
		default:
			c.JSON(http.StatusInternalServerError, errorResponse("INTERNAL_ERROR", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pr":          pr,
		"replaced_by": newUserID,
	})
}

func (h *Handler) GetUserReviews(c *gin.Context) {
	userID := c.Query("user_id")
	prs, err := h.service.GetUserReviews(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "user not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":       userID,
		"pull_requests": prs,
	})
}

func (h *Handler) GetUserStats(c *gin.Context) {
	userID := c.Query("user_id")
	stats, err := h.service.GetUserReviewStats(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "user not found"))
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *Handler) BulkDeactivateUsers(c *gin.Context) {
	var req struct {
		TeamName     string   `json:"team_name"`
		ExcludeUsers []string `json:"exclude_users,omitempty"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse("INVALID_INPUT", err.Error()))
		return
	}

	affected, err := h.service.BulkDeactivateUsers(req.TeamName, req.ExcludeUsers)
	if err != nil {
		c.JSON(http.StatusNotFound, errorResponse("NOT_FOUND", "team not found"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"team_name":         req.TeamName,
		"deactivated_users": affected,
	})

}

func errorResponse(code, message string) gin.H {
	return gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	}
}
