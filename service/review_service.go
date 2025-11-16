package service

import (
	"PR/models"
	"PR/repository"
	"errors"
	"math/rand"
	"time"

	"github.com/jackc/pgtype"
)

type ReviewService struct {
	repo *repository.Repository
	rng  *rand.Rand
}

func NewReviewService(repo *repository.Repository) *ReviewService {
	scr := rand.NewSource(time.Now().UnixNano())
	return &ReviewService{
		repo: repo,
		rng:  rand.New(scr),
	}
}

func (rs *ReviewService) CreateTeam(team models.Team) error {
	return rs.repo.CreateTeam(team)
}

func (rs *ReviewService) GetTeam(teamName string) (*models.Team, error) {
	return rs.repo.GetTeam(teamName)
}

func (rs *ReviewService) SetUserActive(UserId string, IsActive bool) (*models.User, error) {
	return rs.repo.UpdateUserActive(UserId, IsActive)
}

func (rs *ReviewService) CreatePR(prID, prName, authorID string) (*models.PullRequest, error) {
	if existing, _ := rs.repo.GetPR(prID); existing != nil {
		return nil, errors.New("PR уже существует")
	}

	author, err := rs.repo.GetUser(authorID)
	if err != nil {
		return nil, errors.New("Автор не найден")
	}

	teamMembers, err := rs.repo.GetActiveTeamMembers(author.TeamName)
	if err != nil {
		return nil, errors.New("Команда не найдена")
	}

	candidates := rs.FilterCandidates(teamMembers, authorID)
	reviewers := rs.SelectReviewers(candidates, 2)

	reviewersArray := pgtype.TextArray{}
	if err := reviewersArray.Set(reviewers); err != nil {
		return nil, err
	}

	pr := models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            models.StatusOpen,
		AssignedReviewers: reviewersArray,
		CreatedAt:         time.Now(),
	}

	if err := rs.repo.CreatePR(pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (rs *ReviewService) MergePR(prID string) (*models.PullRequest, error) {
	pr, err := rs.repo.GetPR(prID)
	if err != nil {
		return nil, errors.New("PR не найден")
	}

	if pr.Status == models.StatusMerged {
		return pr, nil
	}

	pr.Status = models.StatusMerged
	now := time.Now()
	pr.MergedAt = &now

	if err := rs.repo.UpdatePR(pr); err != nil {
		return nil, errors.New("Не получилось обновить статус PR")
	}

	return pr, nil
}

func (rs *ReviewService) ReassignReviewer(prID, oldUserID string) (*models.PullRequest, string, error) {
	pr, err := rs.repo.GetPR(prID)
	if err != nil {
		return nil, "", errors.New("PR не найден")
	}

	if pr.Status == models.StatusMerged {
		return nil, "", errors.New("Нельзя переназначать ревьюера на замердженном PR")
	}

	var currentReviewers []string
	if err := pr.AssignedReviewers.AssignTo(&currentReviewers); err != nil {
		return nil, "", errors.New("Ошибка при чтении списка ревьюеров")
	}

	if !rs.Contains(currentReviewers, oldUserID) {
		return nil, "", errors.New("Данный ревьюер и не был назначен на данный PR")
	}

	oldUser, err := rs.repo.GetUser(oldUserID)
	if err != nil {
		return nil, "", errors.New("Пользователь не найден")
	}

	candidates, err := rs.repo.GetActiveTeamMembers(oldUser.TeamName)
	if err != nil || len(candidates) <= 2 {
		return nil, "", errors.New("Нет доступных кандидатов для замены")
	}

	availableCandidates := rs.FilterReassignmentCandidates(candidates, currentReviewers, pr.AuthorID, oldUserID)

	if len(availableCandidates) == 0 {
		return nil, "", errors.New("Нет доступных кандидатов для замены")
	}

	newReviewer := availableCandidates[rs.rng.Intn(len(availableCandidates))].UserId
	newReviewers := rs.ReplaceReviewer(currentReviewers, oldUserID, newReviewer)

	newReviewersArray := pgtype.TextArray{}
	if err := newReviewersArray.Set(newReviewers); err != nil {
		return nil, "", err
	}

	pr.AssignedReviewers = newReviewersArray

	if err := rs.repo.UpdatePR(pr); err != nil {
		return nil, "", err
	}

	return pr, newReviewer, nil
}

func (rs *ReviewService) GetUserReviews(userID string) ([]models.PullRequestShort, error) {
	prs, err := rs.repo.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	var result []models.PullRequestShort
	for _, pr := range prs {
		result = append(result, models.PullRequestShort{
			PullRequestID:   pr.PullRequestID,
			PullRequestName: pr.PullRequestName,
			AuthorID:        pr.AuthorID,
			Status:          pr.Status,
		})
	}

	return result, nil
}

func (rs *ReviewService) BulkDeactivateUsers(teamName string, excludeUserIDs []string) (int64, error) {
	return rs.repo.BulkDeactivateUsers(teamName, excludeUserIDs)
}

func (rs *ReviewService) GetUserReviewStats(userID string) (map[string]interface{}, error) {
	prs, err := rs.repo.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	total := len(prs)
	open := 0
	for _, pr := range prs {
		if pr.Status == models.StatusOpen {
			open++
		}
	}

	return map[string]interface{}{
		"user_id":           userID,
		"total_reviews":     total,
		"open_reviews":      open,
		"completed_reviews": total - open,
	}, nil
}

func (rs *ReviewService) FilterCandidates(candidates []models.User, authorID string) []models.User {
	var result []models.User
	for _, user := range candidates {
		if user.UserId != authorID {
			result = append(result, user)
		}
	}
	return result
}

func (rs *ReviewService) SelectReviewers(candidates []models.User, max int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	shuffled := make([]models.User, len(candidates))
	copy(shuffled, candidates)
	rs.rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	count := min(len(shuffled), max)
	reviewers := make([]string, count)
	for i := 0; i < count; i++ {
		reviewers[i] = shuffled[i].UserId
	}

	return reviewers
}

func (rs *ReviewService) FilterReassignmentCandidates(candidates []models.User, currentReviewers []string, authorID, oldUserID string) []models.User {
	var result []models.User

	for _, user := range candidates {
		if user.UserId != oldUserID && user.UserId != authorID && !rs.Contains(currentReviewers, user.UserId) {
			result = append(result, user)
		}
	}

	return result
}

func (rs *ReviewService) Contains(slice []string, item string) bool {
	for _, sliceItem := range slice {
		if sliceItem == item {
			return true
		}
	}
	return false
}

func (rs *ReviewService) ReplaceReviewer(reviewers []string, old, new string) []string {
	result := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		if reviewer != old {
			result[i] = reviewer
		} else {
			result[i] = new
		}
	}
	return result
}
