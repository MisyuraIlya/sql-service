package sqlproxy

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Run(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if err := ValidateQueryReadOnly(req.Query); err != nil {
		return nil, err
	}

	timeout := 30 * time.Second
	if req.TimeoutMs > 0 && req.TimeoutMs < 120000 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return s.repo.Query(cctx, req)
}
