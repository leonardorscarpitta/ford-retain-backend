// Package fordapi connects this service to Ford's existing platform.
//
// Ford's real endpoints aren't available to this project, so this package
// defines the contract we expect and implements it twice: Client (used by the
// API when FORD_API_URL is set) and NewMock (a stand-in server for local
// development and tests). Change both together when the real contract is known.
//
// Contract (JSON; every request carries "Authorization: Bearer <FORD_API_KEY>"):
//
//	GET  /v1/owners/{ownerId}/vehicles         → [{vin, model, paintName, paintHex}]
//	GET  /v1/vehicles/{vin}/telemetry          → telemetry snapshot, 404 if none
//	GET  /v1/vehicles/{vin}/alerts             → [alert] (the WhatsApp alert history)
//	POST /v1/vehicles/{vin}/alerts/{id}/read   → alert, 404 if unknown
//	GET  /v1/dealers                           → [dealer] including opening hours
package fordapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/telemetry"
	"github.com/ford-retain/backend/internal/vehicles"
)

// Client calls Ford's platform APIs.
type Client struct {
	base string
	key  string
	http *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		key:  apiKey,
		http: &http.Client{Timeout: 5 * time.Second},
	}
}

var errNotFound = errors.New("not found")

// do sends a request and decodes a JSON response into out. A 404 returns
// errNotFound so callers can tell "missing" from "Ford is down".
func (c *Client) do(ctx context.Context, method, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ford api %s %s: %w", method, path, err)
	}
	defer res.Body.Close()
	switch {
	case res.StatusCode == http.StatusNotFound:
		return errNotFound
	case res.StatusCode >= 300:
		return fmt.Errorf("ford api %s %s: status %d", method, path, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return fmt.Errorf("ford api %s %s: decode: %w", method, path, err)
	}
	return nil
}

// Vehicles adapts the client to vehicles.Source.
func (c *Client) Vehicles() vehicles.Source { return vehicleSource{c} }

// Telemetry adapts the client to telemetry.Source.
func (c *Client) Telemetry() telemetry.Source { return telemetrySource{c} }

// Alerts adapts the client to notifications.Source.
func (c *Client) Alerts() notifications.Source { return alertSource{c} }

type vehicleSource struct{ c *Client }

func (s vehicleSource) ForOwner(ctx context.Context, ownerID string) ([]vehicles.Vehicle, error) {
	var list []vehicles.Vehicle
	err := s.c.do(ctx, http.MethodGet, "/v1/owners/"+url.PathEscape(ownerID)+"/vehicles", &list)
	if errors.Is(err, errNotFound) {
		return []vehicles.Vehicle{}, nil // unknown owner: no vehicles
	}
	for i := range list {
		list[i].OwnerID = ownerID
	}
	return list, err
}

type telemetrySource struct{ c *Client }

func (s telemetrySource) Get(ctx context.Context, vin string) (telemetry.Snapshot, bool, error) {
	var snap telemetry.Snapshot
	err := s.c.do(ctx, http.MethodGet, "/v1/vehicles/"+url.PathEscape(vin)+"/telemetry", &snap)
	if errors.Is(err, errNotFound) {
		return telemetry.Snapshot{}, false, nil
	}
	return snap, err == nil, err
}

type alertSource struct{ c *Client }

func (s alertSource) ListByVIN(ctx context.Context, vin string) ([]notifications.Notification, error) {
	var list []notifications.Notification
	err := s.c.do(ctx, http.MethodGet, "/v1/vehicles/"+url.PathEscape(vin)+"/alerts", &list)
	if errors.Is(err, errNotFound) {
		return []notifications.Notification{}, nil
	}
	return list, err
}

func (s alertSource) MarkRead(ctx context.Context, vin, id string) (notifications.Notification, bool, error) {
	var n notifications.Notification
	err := s.c.do(ctx, http.MethodPost, "/v1/vehicles/"+url.PathEscape(vin)+"/alerts/"+url.PathEscape(id)+"/read", &n)
	if errors.Is(err, errNotFound) {
		return notifications.Notification{}, false, nil
	}
	return n, err == nil, err
}

// dealer is the wire format of Ford's dealer directory.
type dealer struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Phone    string   `json:"phone"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Rating   float64  `json:"rating"`
	Services []string `json:"services"`
	TimeZone string   `json:"timeZone"`
	Weekdays hours    `json:"weekdays"`
	Saturday *hours   `json:"saturday"` // null: closed
	Bays     int      `json:"bays"`
}

type hours struct {
	Open  int `json:"open"`
	Close int `json:"close"`
}

// Dealers fetches Ford's dealer directory.
func (c *Client) Dealers(ctx context.Context) ([]dealerships.Dealership, error) {
	var wire []dealer
	if err := c.do(ctx, http.MethodGet, "/v1/dealers", &wire); err != nil {
		return nil, err
	}
	out := make([]dealerships.Dealership, 0, len(wire))
	for _, d := range wire {
		loc, err := time.LoadLocation(d.TimeZone)
		if err != nil {
			return nil, fmt.Errorf("dealer %s: time zone %q: %w", d.ID, d.TimeZone, err)
		}
		sched := dealerships.Schedule{Location: loc, Weekdays: dealerships.OpenHours(d.Weekdays), Bays: d.Bays}
		if d.Saturday != nil {
			sat := dealerships.OpenHours(*d.Saturday)
			sched.Saturday = &sat
		}
		out = append(out, dealerships.Dealership{
			ID: d.ID, Name: d.Name, Address: d.Address, Phone: d.Phone,
			Lat: d.Lat, Lng: d.Lng, Rating: d.Rating, Services: d.Services, Schedule: sched,
		})
	}
	return out, nil
}
