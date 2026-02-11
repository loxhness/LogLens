package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	csvPath   = "./data/tickets.csv"
	dateLayout = "2006-01-02"
)

// Ticket represents a single row from the CSV
type Ticket struct {
	ID        int
	CreatedAt time.Time
	ClosedAt  *time.Time // nil if still open
	Category  string
	Priority  string
	Status    string
}

// Summary holds all computed dashboard statistics
type Summary struct {
	TicketsPerDay              []DayCount          `json:"tickets_per_day"`
	TopCategories              []CategoryCount     `json:"top_categories"`
	AvgResolutionHoursByCat    []CategoryAvgHours  `json:"avg_resolution_hours_by_category"`
	OpenVsClosed               OpenClosedCounts    `json:"open_vs_closed"`
	TotalTickets               int                 `json:"total_tickets"`
	OpenTickets                int                 `json:"open_tickets"`
	ClosedTickets              int                 `json:"closed_tickets"`
}

type DayCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type CategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

type CategoryAvgHours struct {
	Category string  `json:"category"`
	AvgHours float64 `json:"avg_hours"`
}

type OpenClosedCounts struct {
	Open   int `json:"open"`
	Closed int `json:"closed"`
}

var (
	tickets []Ticket
	mu      sync.RWMutex
)

func main() {
	if err := loadTickets(); err != nil {
		log.Fatalf("Failed to load tickets at startup: %v", err)
	}

	// Static file server for dashboard
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/api/summary", handleSummary)
	http.HandleFunc("/api/reload", handleReload)

	log.Println("LogLens running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// loadTickets reads and parses the CSV file
func loadTickets() error {
	f, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return err
	}

	if len(rows) < 2 {
		return nil // header only, no tickets
	}

	var parsed []Ticket
	for i, row := range rows[1:] {
		if len(row) < 6 {
			continue
		}

		id, _ := strconv.Atoi(row[0])
		createdAt, err := time.Parse(dateLayout, row[1])
		if err != nil {
			log.Printf("Skipping row %d: invalid created_at: %s", i+2, row[1])
			continue
		}

		var closedAt *time.Time
		if row[2] != "" {
			t, err := time.Parse(dateLayout, row[2])
			if err == nil {
				closedAt = &t
			}
		}

		ticket := Ticket{
			ID:        id,
			CreatedAt: createdAt,
			ClosedAt:  closedAt,
			Category:  row[3],
			Priority:  row[4],
			Status:    row[5],
		}
		parsed = append(parsed, ticket)
	}

	mu.Lock()
	tickets = parsed
	mu.Unlock()
	return nil
}

// computeSummary builds the dashboard statistics from tickets
func computeSummary() Summary {
	mu.RLock()
	t := tickets
	mu.RUnlock()

	// tickets_per_day
	dayMap := make(map[string]int)
	for _, ticket := range t {
		day := ticket.CreatedAt.Format(dateLayout)
		dayMap[day]++
	}
	var ticketsPerDay []DayCount
	for d, c := range dayMap {
		ticketsPerDay = append(ticketsPerDay, DayCount{Date: d, Count: c})
	}
	sort.Slice(ticketsPerDay, func(i, j int) bool { return ticketsPerDay[i].Date < ticketsPerDay[j].Date })

	// top_categories
	catMap := make(map[string]int)
	for _, ticket := range t {
		catMap[ticket.Category]++
	}
	var topCategories []CategoryCount
	for c, n := range catMap {
		topCategories = append(topCategories, CategoryCount{Category: c, Count: n})
	}
	sort.Slice(topCategories, func(i, j int) bool { return topCategories[i].Count > topCategories[j].Count })

	// avg_resolution_hours_by_category (only closed tickets)
	catHours := make(map[string][]float64)
	for _, ticket := range t {
		if ticket.ClosedAt == nil {
			continue
		}
		hours := ticket.ClosedAt.Sub(ticket.CreatedAt).Hours()
		catHours[ticket.Category] = append(catHours[ticket.Category], hours)
	}
	var avgByCat []CategoryAvgHours
	for cat, hours := range catHours {
		var sum float64
		for _, h := range hours {
			sum += h
		}
		avgByCat = append(avgByCat, CategoryAvgHours{
			Category: cat,
			AvgHours: sum / float64(len(hours)),
		})
	}
	sort.Slice(avgByCat, func(i, j int) bool { return avgByCat[i].Category < avgByCat[j].Category })

	// open_vs_closed
	var open, closed int
	for _, ticket := range t {
		if ticket.ClosedAt != nil {
			closed++
		} else {
			open++
		}
	}

	return Summary{
		TicketsPerDay:           ticketsPerDay,
		TopCategories:          topCategories,
		AvgResolutionHoursByCat: avgByCat,
		OpenVsClosed:           OpenClosedCounts{Open: open, Closed: closed},
		TotalTickets:           len(t),
		OpenTickets:            open,
		ClosedTickets:          closed,
	}
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(computeSummary())
}

func handleReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := loadTickets(); err != nil {
		http.Error(w, "Failed to reload CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(computeSummary())
}
