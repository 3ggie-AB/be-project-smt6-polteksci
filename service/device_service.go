package service

import (
	"context"
	"errors"
	"strings"

	"project_smt6/domain"
	"project_smt6/repository"
)

type DeviceService struct {
	devices repository.DeviceRepository
}

func NewDeviceService(devices repository.DeviceRepository) *DeviceService {
	return &DeviceService{devices: devices}
}

func (s *DeviceService) List(ctx context.Context, workspaceID *uint) ([]domain.Device, error) {
	return s.devices.List(ctx, workspaceID)
}

func (s *DeviceService) Create(ctx context.Context, workspaceID *uint, device *domain.Device) error {
	if strings.TrimSpace(device.Name) == "" {
		return errors.New("device name is required")
	}
	if strings.TrimSpace(device.IPAddress) == "" {
		return errors.New("device ip_address is required")
	}
	if workspaceID != nil {
		device.WorkspaceID = *workspaceID
	}
	device.IPAddress = strings.TrimSpace(device.IPAddress)
	device.Name = strings.TrimSpace(device.Name)
	if device.DeviceType == "" {
		device.DeviceType = "network"
	}
	return s.devices.Create(ctx, device)
}

func (s *DeviceService) Delete(ctx context.Context, id uint) error {
	return s.devices.Delete(ctx, id)
}
