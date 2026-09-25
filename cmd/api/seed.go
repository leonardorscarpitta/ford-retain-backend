package main

import (
	"fmt"
	"time"

	"github.com/ford-retain/backend/internal/appointments"
	"github.com/ford-retain/backend/internal/dealerships"
	"github.com/ford-retain/backend/internal/notifications"
	"github.com/ford-retain/backend/internal/telemetry"
)

// demoVIN is the single vehicle seeded for local development.
const demoVIN = "1FTFW1E50NFA00001"

const day = 24 * time.Hour

func seedNotifications(now time.Time) []notifications.Notification {
	const (
		info     = notifications.SeverityInfo
		warning  = notifications.SeverityWarning
		critical = notifications.SeverityCritical
	)
	type n struct {
		ago      time.Duration
		severity notifications.Severity
		read     bool
		title    string
		message  string
	}
	items := []n{
		{25 * time.Minute, critical, false, "Front brake pads worn", "Front brake pads are at 15%. Braking distance may increase. Book a replacement as soon as possible."},
		{3 * time.Hour, critical, false, "Tyre pressure low — front left", "Front-left tyre is at 24 PSI (recommended 35). Check for punctures before long trips."},
		{9 * time.Hour, warning, false, "Engine oil life at 40%", "Your next oil change is due in about 1,200 km."},
		{26 * time.Hour, warning, false, "Coolant level below minimum", "Top up coolant or visit a dealership to check for leaks."},
		{2 * day, warning, false, "Air filter restriction rising", "Cabin air filter is 60% saturated. Replacement recommended at next service."},
		{3 * day, info, false, "Software update available", "SYNC 4 update 4.2.1 is ready. Install over Wi-Fi when parked."},
		{4 * day, warning, true, "12V battery charge low overnight", "Battery dropped to 11.9 V while parked. Avoid short trips for a few days."},
		{6 * day, info, true, "Recall check: no open recalls", "Your vehicle has no open safety recalls."},
		{9 * day, info, true, "Trip summary: 412 km this week", "Average consumption 11.8 L/100 km — 6% better than last week."},
		{12 * day, warning, true, "Rear brake pads at 45%", "Rear pads are wearing normally. Plan to replace within 8,000 km."},
		{15 * day, info, true, "Tyre rotation completed", "Thanks for visiting Ford Paulista. Next rotation in 10,000 km."},
		{21 * day, critical, true, "Check engine light cleared", "Fault P0171 (system too lean) was resolved after the O2 sensor replacement."},
		{28 * day, info, true, "Service reminder", "Your 50,000 km revision is coming up. Book early to secure your preferred date."},
		{35 * day, warning, true, "Wiper blades streaking", "Reported by rain sensor. Blades are inexpensive to replace at any dealership."},
		{44 * day, info, true, "Warranty extended", "Your powertrain warranty is valid until March 2028."},
		{58 * day, info, true, "Welcome to Ford Retain", "You'll receive maintenance alerts here and on WhatsApp."},
	}
	out := make([]notifications.Notification, len(items))
	for i, it := range items {
		out[i] = notifications.Notification{
			ID:        fmt.Sprintf("n%d", i+1),
			VIN:       demoVIN,
			Title:     it.title,
			Message:   it.message,
			Severity:  it.severity,
			Channel:   "whatsapp",
			Read:      it.read,
			CreatedAt: now.Add(-it.ago),
		}
	}
	return out
}

func seedTelemetry(now time.Time) []telemetry.Snapshot {
	const (
		good      = telemetry.StatusGood
		attention = telemetry.StatusAttention
		critical  = telemetry.StatusCritical
	)
	return []telemetry.Snapshot{{
		VIN:       demoVIN,
		Model:     "2022 Ford F-150 Lariat",
		MileageKm: 48210,
		Battery: telemetry.Battery{
			HealthPercent:       78,
			EstimatedMonthsLeft: 14,
			History:             []int{94, 93, 91, 90, 89, 87, 86, 84, 83, 81, 80, 78},
		},
		Components: []telemetry.Component{
			{Name: "Engine", Zone: telemetry.ZoneEngineBay, Health: 92, Status: good},
			{Name: "Engine oil", Zone: telemetry.ZoneEngineBay, Health: 40, Status: attention},
			{Name: "Coolant", Zone: telemetry.ZoneEngineBay, Health: 35, Status: attention},
			{Name: "Spark plugs", Zone: telemetry.ZoneEngineBay, Health: 81, Status: good},
			{Name: "Front brakes", Zone: telemetry.ZoneFrontWheel, Health: 15, Status: critical},
			{Name: "Front tyres", Zone: telemetry.ZoneFrontWheel, Health: 22, Status: critical},
			{Name: "Rear brakes", Zone: telemetry.ZoneRearWheel, Health: 45, Status: attention},
			{Name: "Rear tyres", Zone: telemetry.ZoneRearWheel, Health: 70, Status: good},
			{Name: "Transmission", Zone: telemetry.ZoneUnderbody, Health: 88, Status: good},
			{Name: "Suspension", Zone: telemetry.ZoneUnderbody, Health: 76, Status: good},
			{Name: "Exhaust", Zone: telemetry.ZoneUnderbody, Health: 90, Status: good},
			{Name: "Cabin air filter", Zone: telemetry.ZoneCabin, Health: 40, Status: attention},
			{Name: "Wiper blades", Zone: telemetry.ZoneCabin, Health: 30, Status: attention},
		},
		UpdatedAt: now.Add(-10 * time.Minute),
	}}
}

func seedDealerships() []dealerships.Dealership {
	const (
		weekday  = "Mon–Fri 8:00–18:00 · Sat 8:00–12:00"
		extended = "Mon–Sat 7:00–20:00"
	)
	var (
		full    = []string{"Maintenance", "Brakes", "Tyres", "Battery", "Diagnostics", "Body shop"}
		service = []string{"Maintenance", "Brakes", "Tyres", "Diagnostics"}
		quick   = []string{"Maintenance", "Tyres", "Battery"}
		ev      = []string{"Maintenance", "Brakes", "Diagnostics", "EV service", "Battery"}
	)
	return []dealerships.Dealership{
		// São Paulo city
		{ID: "d1", Name: "Ford Paulista", Address: "Av. Paulista, 1000 - Bela Vista, São Paulo, SP", Phone: "+55 11 3000-0001", Lat: -23.5631, Lng: -46.6544, Rating: 4.8, Hours: extended, Services: full},
		{ID: "d2", Name: "Ford Pinheiros", Address: "R. dos Pinheiros, 500 - Pinheiros, São Paulo, SP", Phone: "+55 11 3000-0002", Lat: -23.5667, Lng: -46.6920, Rating: 4.6, Hours: weekday, Services: service},
		{ID: "d3", Name: "Ford Santo Amaro", Address: "Av. Santo Amaro, 3000 - Brooklin, São Paulo, SP", Phone: "+55 11 3000-0003", Lat: -23.6150, Lng: -46.6660, Rating: 4.5, Hours: weekday, Services: full},
		{ID: "d4", Name: "Ford Moema", Address: "Av. Ibirapuera, 2200 - Moema, São Paulo, SP", Phone: "+55 11 3000-0004", Lat: -23.6010, Lng: -46.6650, Rating: 4.7, Hours: extended, Services: ev},
		{ID: "d5", Name: "Ford Tatuapé", Address: "R. Tuiuti, 1500 - Tatuapé, São Paulo, SP", Phone: "+55 11 3000-0005", Lat: -23.5405, Lng: -46.5760, Rating: 4.4, Hours: weekday, Services: service},
		{ID: "d6", Name: "Ford Santana", Address: "Av. Cruzeiro do Sul, 3100 - Santana, São Paulo, SP", Phone: "+55 11 3000-0006", Lat: -23.5020, Lng: -46.6250, Rating: 4.3, Hours: weekday, Services: quick},
		{ID: "d7", Name: "Ford Lapa", Address: "R. Guaicurus, 800 - Lapa, São Paulo, SP", Phone: "+55 11 3000-0007", Lat: -23.5260, Lng: -46.7040, Rating: 4.5, Hours: weekday, Services: service},
		{ID: "d8", Name: "Ford Morumbi", Address: "Av. Giovanni Gronchi, 5000 - Morumbi, São Paulo, SP", Phone: "+55 11 3000-0008", Lat: -23.6180, Lng: -46.7350, Rating: 4.9, Hours: extended, Services: full},
		{ID: "d9", Name: "Ford Ipiranga", Address: "Av. Nazaré, 1200 - Ipiranga, São Paulo, SP", Phone: "+55 11 3000-0009", Lat: -23.5870, Lng: -46.6080, Rating: 4.2, Hours: weekday, Services: quick},
		{ID: "d10", Name: "Ford Mooca", Address: "R. da Mooca, 2600 - Mooca, São Paulo, SP", Phone: "+55 11 3000-0010", Lat: -23.5570, Lng: -46.5990, Rating: 4.4, Hours: weekday, Services: service},
		// Greater São Paulo
		{ID: "d11", Name: "Ford Osasco", Address: "Av. dos Autonomistas, 1800 - Osasco, SP", Phone: "+55 11 3000-0011", Lat: -23.5320, Lng: -46.7760, Rating: 4.3, Hours: weekday, Services: full},
		{ID: "d12", Name: "Ford Guarulhos", Address: "Av. Tiradentes, 900 - Guarulhos, SP", Phone: "+55 11 3000-0012", Lat: -23.4630, Lng: -46.5330, Rating: 4.1, Hours: extended, Services: service},
		{ID: "d13", Name: "Ford São Bernardo", Address: "Av. Kennedy, 700 - São Bernardo do Campo, SP", Phone: "+55 11 3000-0013", Lat: -23.6940, Lng: -46.5650, Rating: 4.6, Hours: weekday, Services: full},
		{ID: "d14", Name: "Ford Santo André", Address: "Av. Industrial, 1500 - Santo André, SP", Phone: "+55 11 3000-0014", Lat: -23.6640, Lng: -46.5380, Rating: 4.5, Hours: weekday, Services: ev},
		{ID: "d15", Name: "Ford Barueri", Address: "Al. Rio Negro, 400 - Alphaville, Barueri, SP", Phone: "+55 11 3000-0015", Lat: -23.5020, Lng: -46.8490, Rating: 4.8, Hours: extended, Services: ev},
		// São Paulo state
		{ID: "d16", Name: "Ford Campinas", Address: "Av. Norte-Sul, 200 - Campinas, SP", Phone: "+55 19 3000-0016", Lat: -22.9056, Lng: -47.0608, Rating: 4.6, Hours: weekday, Services: full},
		{ID: "d17", Name: "Ford Jundiaí", Address: "Av. Jundiaí, 1100 - Jundiaí, SP", Phone: "+55 11 3000-0017", Lat: -23.1860, Lng: -46.8840, Rating: 4.4, Hours: weekday, Services: service},
		{ID: "d18", Name: "Ford Sorocaba", Address: "Av. Itavuvu, 2000 - Sorocaba, SP", Phone: "+55 15 3000-0018", Lat: -23.5010, Lng: -47.4580, Rating: 4.3, Hours: weekday, Services: service},
		{ID: "d19", Name: "Ford Santos", Address: "Av. Ana Costa, 300 - Santos, SP", Phone: "+55 13 3000-0019", Lat: -23.9600, Lng: -46.3330, Rating: 4.7, Hours: extended, Services: full},
		{ID: "d20", Name: "Ford São José dos Campos", Address: "Av. Andrômeda, 1500 - São José dos Campos, SP", Phone: "+55 12 3000-0020", Lat: -23.2240, Lng: -45.9000, Rating: 4.5, Hours: weekday, Services: ev},
		{ID: "d21", Name: "Ford Ribeirão Preto", Address: "Av. Presidente Vargas, 2500 - Ribeirão Preto, SP", Phone: "+55 16 3000-0021", Lat: -21.2000, Lng: -47.8100, Rating: 4.6, Hours: weekday, Services: full},
		// Other states
		{ID: "d22", Name: "Ford Barra da Tijuca", Address: "Av. das Américas, 5000 - Rio de Janeiro, RJ", Phone: "+55 21 3000-0022", Lat: -23.0000, Lng: -43.3650, Rating: 4.5, Hours: extended, Services: full},
		{ID: "d23", Name: "Ford Savassi", Address: "Av. do Contorno, 6000 - Belo Horizonte, MG", Phone: "+55 31 3000-0023", Lat: -19.9380, Lng: -43.9340, Rating: 4.4, Hours: weekday, Services: service},
		{ID: "d24", Name: "Ford Curitiba", Address: "Av. Sete de Setembro, 4000 - Curitiba, PR", Phone: "+55 41 3000-0024", Lat: -25.4400, Lng: -49.2700, Rating: 4.7, Hours: weekday, Services: ev},
	}
}

func seedAppointments(now time.Time) []appointments.Appointment {
	// Anchor at 09:00 so seeded times look like real bookings.
	base := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
	type a struct {
		days       int
		hour       int
		dealership string
		service    string
		notes      string
	}
	items := []a{
		// upcoming
		{2, 1, "d1", "Front brake pads replacement", "Critical alert from telemetry"},
		{9, 5, "d4", "Tyre replacement (front pair)", ""},
		{34, 0, "d1", "50,000 km revision", "Include coolant check"},
		// past
		{-15, 2, "d1", "Tyre rotation", ""},
		{-21, 0, "d8", "O2 sensor replacement", "Check engine light P0171"},
		{-60, 3, "d2", "Battery test", "Slow cranking in the morning"},
		{-120, 1, "d2", "Oil change", "Synthetic oil"},
		{-180, 4, "d15", "Wheel alignment", ""},
		{-240, 0, "d1", "40,000 km revision", ""},
		{-300, 2, "d16", "Air conditioning service", "Before summer road trip"},
		{-365, 1, "d3", "30,000 km revision", ""},
		{-430, 6, "d19", "Windshield repair", "Chip from gravel"},
		{-500, 0, "d1", "20,000 km revision", ""},
		{-610, 3, "d8", "Software update (SYNC 4)", ""},
		{-700, 0, "d1", "10,000 km revision", "First revision"},
	}
	out := make([]appointments.Appointment, len(items))
	for i, it := range items {
		at := base.Add(time.Duration(it.days)*day + time.Duration(it.hour)*time.Hour)
		out[i] = appointments.Appointment{
			ID:           fmt.Sprintf("apt-seed-%d", i+1),
			VIN:          demoVIN,
			DealershipID: it.dealership,
			ScheduledAt:  at,
			Service:      it.service,
			Notes:        it.notes,
			CreatedAt:    at.Add(-14 * day),
		}
	}
	return out
}
