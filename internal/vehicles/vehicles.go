// Package vehicles is each owner's garage: which vehicles they have and how
// to identify them (VIN, model, paint).
package vehicles

import "context"

type Vehicle struct {
	VIN     string `json:"vin"`
	OwnerID string `json:"-"`
	Model   string `json:"model"`
	// Paint is the factory colour name and a hex value to render it with.
	PaintName string `json:"paintName"`
	PaintHex  string `json:"paintHex"`
}

// Source is where ownership comes from: Store (seed data) offline, Ford's
// vehicle registry in production.
type Source interface {
	// ForOwner returns the owner's vehicles, in order.
	ForOwner(ctx context.Context, ownerID string) ([]Vehicle, error)
}

// Owned returns the vehicle if it belongs to the owner.
func Owned(ctx context.Context, src Source, ownerID, vin string) (Vehicle, bool, error) {
	if ownerID == "" {
		return Vehicle{}, false, nil
	}
	list, err := src.ForOwner(ctx, ownerID)
	if err != nil {
		return Vehicle{}, false, err
	}
	for _, v := range list {
		if v.VIN == vin {
			return v, true, nil
		}
	}
	return Vehicle{}, false, nil
}

// Store is a read-only in-memory registry of vehicles. Order is preserved:
// an owner's first vehicle is their default.
type Store struct{ items []Vehicle }

func NewStore(seed []Vehicle) *Store { return &Store{items: seed} }

// ForOwner returns the owner's vehicles, in order.
func (s *Store) ForOwner(_ context.Context, ownerID string) ([]Vehicle, error) {
	out := []Vehicle{}
	for _, v := range s.items {
		if v.OwnerID == ownerID {
			out = append(out, v)
		}
	}
	return out, nil
}
