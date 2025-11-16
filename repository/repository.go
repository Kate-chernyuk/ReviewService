package repository

import (
	"PR/models"
	"errors"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateTeam(team models.Team) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&team).Error; err != nil {
			return errors.New("Команда с таким названием уже существует")
		}

		for i := range team.Members {
			team.Members[i].TeamName = team.TeamName
			if err := tx.Where(models.User{UserId: team.Members[i].UserId}).
				Assign(&team.Members[i]).
				FirstOrCreate(&team.Members[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) GetTeam(teamName string) (*models.Team, error) {
	var team models.Team
	if err := r.db.Preload("Members").Where("team_name = ?", teamName).First(&team).Error; err != nil {
		return nil, errors.New("Команда не найдена")
	}
	return &team, nil
}

func (r *Repository) GetUser(userId string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("user_id = ?", userId).First(&user).Error; err != nil {
		return nil, errors.New("Таких у нас нет")
	}
	return &user, nil
}

func (r *Repository) CreateUser(user models.User) error {
	var team models.Team
	if err := r.db.Where("team_name = ?", user.TeamName).First(&team).Error; err != nil {
		return errors.New("Нет такой команды")
	}

	var existingUser models.User
	if r.db.Where("user_id = ?", user.UserId).First(&existingUser).RowsAffected > 0 {
		return errors.New("Такой уже существует")
	}

	return r.db.Create(user).Error
}

func (r *Repository) UpdateUserActive(userId string, isActive bool) (*models.User, error) {
	user, err := r.GetUser(userId)
	if err != nil {
		return nil, errors.New("Таких у нас нет")
	}

	user.IsActive = isActive
	if err := r.db.Save(user).Error; err != nil {
		return nil, errors.New("Не удалось обновить статус активности")
	}
	return user, nil
}

func (r *Repository) DeleteUser(userId string) error {
	result := r.db.Where("user_id = ?", userId).Delete(&models.User{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("Таких и не было")
	}

	return nil
}

func (r *Repository) GetActiveTeamMembers(teamName string) ([]models.User, error) {
	var users []models.User
	if err := r.db.Where("team_name = ? AND is_active = ?", teamName, true).Find(&users).Error; err != nil {
		return nil, errors.New("Нет такой команды")
	}

	return users, nil
}

func (r *Repository) CreatePR(pr models.PullRequest) error {
	if _, err := r.GetUser(pr.AuthorID); err != nil {
		return errors.New("Автора не существует")
	}

	var existingPR models.PullRequest
	if err := r.db.Where("pull_request_id = ?", pr.PullRequestID).First(&existingPR).Error; err == nil {
		return errors.New("PR уже существует")
	}

	return r.db.Create(pr).Error
}

func (r *Repository) GetPR(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	if err := r.db.Where("pull_request_id = ?", prID).First(&pr).Error; err != nil {
		return nil, errors.New("PR с таким ID не существует")
	}

	return &pr, nil
}

func (r *Repository) UpdatePR(pr *models.PullRequest) error {
	return r.db.Save(pr).Error
}

func (r *Repository) GetPRStatus(PRId string) (models.PRStatus, error) {
	var status models.PRStatus
	if err := r.db.Where("pull_request_id = ?", PRId).First(&status).Error; err != nil {
		return models.StatusNotFound, errors.New("Нет PR с таким ID")
	}

	return status, nil
}

func (r *Repository) GetPRsByReviewer(userID string) ([]models.PullRequest, error) {
	var prs []models.PullRequest
	if err := r.db.Where("? = ANY(assigned_reviewers)", userID).Find(&prs).Error; err != nil {
		return nil, err
	}
	return prs, nil
}

func (r *Repository) BulkDeactivateUsers(teamName string, excludeUserIDs []string) (int64, error) {
	query := r.db.Model(&models.User{}).Where("team_name = ? AND is_active = ?", teamName, true)

	if len(excludeUserIDs) > 0 {
		query = query.Where("user_id NOT IN ?", excludeUserIDs)
	}

	result := query.Update("is_active", false)
	return result.RowsAffected, result.Error
}
