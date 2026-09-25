package main

import (
	"time"

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/telemetry"
)

// demoVIN is the single vehicle seeded for local development.
const demoVIN = "1FTFW1E50NFA00001"

func seedNotifications(now time.Time) []notifications.Notification {
	return []notifications.Notification{
		{ID: "n1", VIN: demoVIN, Title: "Brake pads worn", Message: "Front brake pads are at 15%. Schedule a replacement soon.", Severity: notifications.SeverityCritical, Channel: "whatsapp", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "n2", VIN: demoVIN, Title: "Oil change due", Message: "Your next oil change is due in 500 km.", Severity: notifications.SeverityWarning, Channel: "whatsapp", CreatedAt: now.Add(-26 * time.Hour)},
		{ID: "n3", VIN: demoVIN, Title: "Tyre rotation completed", Message: "Thanks for visiting Ford Paulista.", Severity: notifications.SeverityInfo, Channel: "whatsapp", Read: true, CreatedAt: now.Add(-15 * 24 * time.Hour)},
	}
}

func seedTelemetry(now time.Time) []telemetry.Snapshot {
	return []telemetry.Snapshot{{
		VIN:       demoVIN,
		MileageKm: 48210,
		Battery:   telemetry.Battery{HealthPercent: 78, EstimatedMonthsLeft: 14},
		Components: []telemetry.Component{
			{Name: "Engine", Health: 92, Status: telemetry.StatusGood},
			{Name: "Brakes", Health: 15, Status: telemetry.StatusCritical},
			{Name: "Engine oil", Health: 40, Status: telemetry.StatusAttention},
			{Name: "Tyres", Health: 70, Status: telemetry.StatusGood},
			{Name: "Transmission", Health: 88, Status: telemetry.StatusGood},
		},
		UpdatedAt: now.Add(-10 * time.Minute),
	}}
}

func seedDealerships() []dealerships.Dealership {
	return []dealerships.Dealership{
		{ID: "d1", Name: "Ford Paulista", Address: "Av. Paulista, 1000 - São Paulo, SP", Phone: "+55 11 3000-0001", Lat: -23.5631, Lng: -46.6544},
		{ID: "d2", Name: "Ford Pinheiros", Address: "R. dos Pinheiros, 500 - São Paulo, SP", Phone: "+55 11 3000-0002", Lat: -23.5667, Lng: -46.6920},
		{ID: "d3", Name: "Ford Santo Amaro", Address: "Av. Santo Amaro, 3000 - São Paulo, SP", Phone: "+55 11 3000-0003", Lat: -23.6150, Lng: -46.6660},
		{ID: "d4", Name: "Ford Campinas", Address: "Av. Norte-Sul, 200 - Campinas, SP", Phone: "+55 19 3000-0004", Lat: -22.9056, Lng: -47.0608},
	}
}

func seedAppointments(now time.Time) []appointments.Appointment {
	day := 24 * time.Hour
	return []appointments.Appointment{
		{ID: "apt-seed-1", VIN: demoVIN, DealershipID: "d1", ScheduledAt: now.Add(-15 * day), Service: "Tyre rotation", CreatedAt: now.Add(-30 * day)},
		{ID: "apt-seed-2", VIN: demoVIN, DealershipID: "d2", ScheduledAt: now.Add(-120 * day), Service: "Oil change", Notes: "Synthetic oil", CreatedAt: now.Add(-130 * day)},
		{ID: "apt-seed-3", VIN: demoVIN, DealershipID: "d1", ScheduledAt: now.Add(-300 * day), Service: "10,000 km revision", CreatedAt: now.Add(-310 * day)},
	}
}
