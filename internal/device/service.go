package device

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	devicedb "device-api/internal/device/device_repository"

	"github.com/google/uuid"
)

type Service struct {
	q devicedb.Querier
}

func NewService(q devicedb.Querier) *Service {
	return &Service{q: q}
}

type CreateInput struct {
	Name  string
	Brand string
	State State
}

type ReplaceInput struct {
	Name  string
	Brand string
	State State
}

type PatchInput struct {
	Name  *string
	Brand *string
	State *State
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*Device, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Brand = strings.TrimSpace(in.Brand)
	if in.Name == "" || in.Brand == "" {
		return nil, ErrInvalidInput
	}
	if in.State == "" {
		in.State = StateAvailable
	}
	if !in.State.Valid() {
		return nil, ErrInvalidState
	}

	row, err := s.q.CreateDevice(ctx, devicedb.CreateDeviceParams{
		Name:  in.Name,
		Brand: in.Brand,
		State: devicedb.DeviceState(in.State),
	})
	if err != nil {
		return nil, err
	}
	d := fromDB(row)
	return &d, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Device, error) {
	if id = strings.TrimSpace(id); id == "" {
		return nil, ErrInvalidInput
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidInput
	}

	row, err := s.q.GetDevice(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	d := fromDB(row)
	return &d, nil
}

func (s *Service) List(ctx context.Context, brand string, state State) ([]Device, error) {
	if state != "" && !state.Valid() {
		return nil, ErrInvalidState
	}

	params := devicedb.ListDevicesParams{}
	if b := strings.TrimSpace(brand); b != "" {
		params.Brand = sql.NullString{String: b, Valid: true}
	}
	if state != "" {
		params.State = devicedb.NullDeviceState{DeviceState: devicedb.DeviceState(state), Valid: true}
	}

	rows, err := s.q.ListDevices(ctx, params)
	if err != nil {
		return nil, err
	}

	out := make([]Device, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromDB(row))
	}
	return out, nil
}

func (s *Service) Patch(ctx context.Context, id string, in PatchInput) (*Device, error) {
	curDevice, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	name, brand, state := curDevice.Name, curDevice.Brand, curDevice.State
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, ErrInvalidInput
		}
		name = n
	}
	if in.Brand != nil {
		b := strings.TrimSpace(*in.Brand)
		if b == "" {
			return nil, ErrInvalidInput
		}
		brand = b
	}
	if in.State != nil {
		if !in.State.Valid() {
			return nil, ErrInvalidState
		}
		state = *in.State
	}

	row, err := s.q.UpdateDevice(ctx, devicedb.UpdateDeviceParams{
		ID:    id,
		Name:  name,
		Brand: brand,
		State: devicedb.DeviceState(state),
	})
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.q.GetDevice(ctx, id); errors.Is(getErr, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, ErrInUse
	}
	if err != nil {
		return nil, err
	}

	d := fromDB(row)
	return &d, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	affected, err := s.q.DeleteDevice(ctx, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		if _, getErr := s.q.GetDevice(ctx, id); errors.Is(getErr, sql.ErrNoRows) {
			return ErrNotFound
		}
		return ErrInUse
	}
	return nil
}
