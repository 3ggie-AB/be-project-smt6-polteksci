package service

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"project_smt6/domain"
	"project_smt6/repository"
)

type TargetService struct {
	targets repository.MonitoringTargetRepository
}

func NewTargetService(targets repository.MonitoringTargetRepository) *TargetService {
	return &TargetService{targets: targets}
}

func (s *TargetService) List(ctx context.Context, workspaceID *uint) ([]domain.MonitoringTarget, error) {
	return s.targets.List(ctx, workspaceID)
}

func (s *TargetService) Create(ctx context.Context, workspaceID *uint, target *domain.MonitoringTarget) error {
	target.Name = strings.TrimSpace(target.Name)
	target.Host = strings.TrimSpace(target.Host)
	if target.Name == "" {
		return errors.New("target name is required")
	}
	if target.Host == "" {
		return errors.New("target host is required")
	}
	if workspaceID != nil {
		target.WorkspaceID = *workspaceID
	}
	if target.CheckType == "" {
		target.CheckType = domain.CheckTypePing
	}
	normalizeTargetHost(target)
	switch target.CheckType {
	case domain.CheckTypePing:
		target.Port = 0
		if target.IntervalSec == 0 {
			target.IntervalSec = 5
		}
		if target.TimeoutSec == 0 {
			target.TimeoutSec = 3
		}
	case domain.CheckTypeTCP:
		if target.Port <= 0 {
			return errors.New("target port is required for tcp check")
		}
		if target.IntervalSec == 0 {
			target.IntervalSec = 30
		}
		if target.TimeoutSec == 0 {
			target.TimeoutSec = 3
		}
	default:
		return errors.New("check_type must be ping or tcp")
	}
	target.IsActive = true
	return s.targets.Create(ctx, target)
}

func (s *TargetService) Delete(ctx context.Context, id uint) error {
	return s.targets.Delete(ctx, id)
}

func normalizeTargetHost(target *domain.MonitoringTarget) {
	raw := target.Host
	if !strings.Contains(raw, "://") {
		return
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return
	}
	if parsed.Hostname() == "" {
		return
	}
	target.Host = parsed.Hostname()
	if target.Port == 0 && parsed.Port() != "" {
		port, _ := strconv.Atoi(parsed.Port())
		target.Port = port
	}
	if target.Port == 0 {
		switch parsed.Scheme {
		case "http":
			target.Port = 80
		case "https":
			target.Port = 443
		}
	}
}
