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

type Device struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Brand        string    `json:"brand"`
	State        State     `json:"state"`
	CreationTime time.Time `json:"creation_time"`
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
