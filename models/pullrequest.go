package models

import (
	"time"

	"github.com/jackc/pgtype"
)

type PRStatus string

const (
	StatusOpen     PRStatus = "OPEN"
	StatusMerged   PRStatus = "MERGED"
	StatusNotFound PRStatus = "NOT_FOUND"
)

type PullRequest struct {
	PullRequestID     string           `gorm:"primaryKey;column:pull_request_id" json:"pull_request_id"`
	PullRequestName   string           `gorm:"column:pull_request_name" json:"pull_request_name"`
	AuthorID          string           `gorm:"column:author_id;not null" json:"author_id"`
	Status            PRStatus         `gorm:"type:varchar(20);default:'OPEN'" json:"status"`
	AssignedReviewers pgtype.TextArray `gorm:"type:text[]" json:"assigned_reviewers"`
	CreatedAt         time.Time        `gorm:"autoCreateTime" json:"createdAt"`
	MergedAt          *time.Time       `json:"mergedAt,omitempty"`
}

func (PullRequest) TableName() string {
	return "pull_requests"
}

type PullRequestShort struct {
	PullRequestID   string   `json:"pull_request_id"`
	PullRequestName string   `json:"pull_request_name"`
	AuthorID        string   `json:"author_id"`
	Status          PRStatus `json:"status"`
}
