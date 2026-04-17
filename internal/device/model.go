package device

import (
	"time"

	devicedb "device-api/internal/device/device_repository"
)

type State string

const (
	StateAvailable State = "available"
	StateInUse     State = "in-use"
	StateInactive  State = "inactive"
)

func (s State) Valid() bool {
	switch s {
	case StateAvailable, StateInUse, StateInactive:
		return true
	}
	return false
}

// Device represents a device in the system
type Device struct {
	ID           string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         string    `json:"name" example:"iPhone 15 Pro"`
	Brand        string    `json:"brand" example:"Apple"`
	State        State     `json:"state" enums:"available,in-use,inactive" example:"available"`
	CreationTime time.Time `json:"creation_time" example:"2026-01-15T10:30:00Z"`
}

func fromDB(row devicedb.Device) Device {
	return Device{
		ID:           row.ID,
		Name:         row.Name,
		Brand:        row.Brand,
		State:        State(row.State),
		CreationTime: row.Creationtime,
	}
}
