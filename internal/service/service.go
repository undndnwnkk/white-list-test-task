package service

import (
	"time"

	"github.com/undndnwnkk/white-list-test-task/internal/repository"
)

type Service struct {
	repo *repository.Repository
	now  func() time.Time
}

func New(repo *repository.Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, now: now}
}
