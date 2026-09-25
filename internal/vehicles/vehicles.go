// Package vehicles is each owner's garage: which vehicles they have and how
// to identify them (VIN, model, paint).
package vehicles

type Vehicle struct {
	VIN     string `json:"vin"`
	OwnerID string `json:"-"`
	Model   string `json:"model"`
	// Paint is the factory colour name and a hex value to render it with.
	PaintName string `json:"paintName"`
	PaintHex  string `json:"paintHex"`
}

// Store is a read-only in-memory registry of vehicles. Order is preserved:
// an owner's first vehicle is their default.
type Store struct{ items []Vehicle }

func NewStore(seed []Vehicle) *Store { return &Store{items: seed} }

// ForOwner returns the owner's vehicles, in order.
func (s *Store) ForOwner(ownerID string) []Vehicle {
	out := []Vehicle{}
	for _, v := range s.items {
		if v.OwnerID == ownerID {
			out = append(out, v)
		}
	}
	return out
}

// Owned returns the vehicle if it exists and belongs to the owner.
func (s *Store) Owned(ownerID, vin string) (Vehicle, bool) {
	for _, v := range s.items {
		if v.VIN == vin && v.OwnerID == ownerID && ownerID != "" {
			return v, true
		}
	}
	return Vehicle{}, false
}
