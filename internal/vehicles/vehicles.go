// Package vehicles is the owner's garage: which vehicles they have and how to
// identify them (VIN, model, paint).
package vehicles

type Vehicle struct {
	VIN   string `json:"vin"`
	Model string `json:"model"`
	// Paint is the factory colour name and a hex value to render it with.
	PaintName string `json:"paintName"`
	PaintHex  string `json:"paintHex"`
}

// Store is a read-only in-memory garage. Order is preserved: the first
// vehicle is the owner's default.
type Store struct{ items []Vehicle }

func NewStore(seed []Vehicle) *Store { return &Store{items: seed} }

func (s *Store) All() []Vehicle {
	return append([]Vehicle(nil), s.items...)
}

func (s *Store) Get(vin string) (Vehicle, bool) {
	for _, v := range s.items {
		if v.VIN == vin {
			return v, true
		}
	}
	return Vehicle{}, false
}
