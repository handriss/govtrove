package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type MockOpportunity struct {
	NoticeID           string              `json:"noticeId"`
	Title              string              `json:"title"`
	SolicitationNumber string              `json:"solicitationNumber"`
	FullParentPathName string              `json:"fullParentPathName"`
	FullParentPathCode string              `json:"fullParentPathCode"`
	PostedDate         string              `json:"postedDate"`
	Type               string              `json:"type"`
	BaseType           string              `json:"baseType"`
	ArchiveType        string              `json:"archiveType"`
	ArchiveDate        string              `json:"archiveDate"`
	SetAsideCode       string              `json:"setAsideCode"`
	SetAside           string              `json:"setAside"`
	ResponseDeadLine   string              `json:"responseDeadLine"`
	NAICSCode          string              `json:"naicsCode"`
	NAICSCodes         []string            `json:"naicsCodes"`
	ClassificationCode string              `json:"classificationCode"`
	Active             string              `json:"active"`
	OrganizationType   string              `json:"organizationType"`
	UILink             string              `json:"uiLink"`
	OfficeAddress      *OfficeAddress      `json:"officeAddress"`
	PointOfContact     []PointOfContact    `json:"pointOfContact"`
	PlaceOfPerformance *PlaceOfPerformance `json:"placeOfPerformance"`
}

type OfficeAddress struct {
	City        string `json:"city"`
	State       string `json:"state"`
	ZipCode     string `json:"zipcode"`
	CountryCode string `json:"countryCode"`
}

type PointOfContact struct {
	Type     string `json:"type"`
	FullName string `json:"fullName"`
	Title    string `json:"title"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type PlaceOfPerformance struct {
	StreetAddress string       `json:"streetAddress"`
	City          *CityInfo    `json:"city"`
	State         *StateInfo   `json:"state"`
	Zip           string       `json:"zip"`
	Country       *CountryInfo `json:"country"`
}

type CityInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type StateInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CountryInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type SearchResponse struct {
	TotalRecords      int               `json:"totalRecords"`
	OpportunitiesData []MockOpportunity `json:"opportunitiesData"`
}

type DescriptionResponse struct {
	Description string `json:"description"`
}

var (
	opportunities []MockOpportunity
	totalRecords  int
)

func main() {
	port := flag.Int("port", 8080, "Server port")
	count := flag.Int("count", 500, "Number of mock opportunities to generate")
	flag.Parse()

	totalRecords = *count
	opportunities = generateOpportunities(*count)

	log.Printf("Generated %d mock opportunities", len(opportunities))

	http.HandleFunc("/prod/opportunities/v2/search", handleSearch)
	http.HandleFunc("/prod/opportunities/v1/noticedesc", handleDescription)
	http.HandleFunc("/health", handleHealth)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Mock SAM.gov server starting on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  GET /prod/opportunities/v2/search?api_key=X&limit=N&offset=N")
	log.Printf("  GET /prod/opportunities/v1/noticedesc?api_key=X&noticeid=X")
	log.Printf("  GET /health")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 1000
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	log.Printf("Search request: limit=%d offset=%d", limit, offset)

	end := offset + limit
	if end > len(opportunities) {
		end = len(opportunities)
	}

	var page []MockOpportunity
	if offset < len(opportunities) {
		page = opportunities[offset:end]
	}

	resp := SearchResponse{
		TotalRecords:      totalRecords,
		OpportunitiesData: page,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleDescription(w http.ResponseWriter, r *http.Request) {
	noticeID := r.URL.Query().Get("noticeid")
	if noticeID == "" {
		http.Error(w, "noticeid required", http.StatusBadRequest)
		return
	}

	log.Printf("Description request: noticeid=%s", noticeID)

	resp := DescriptionResponse{
		Description: generateDescription(noticeID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func generateOpportunities(count int) []MockOpportunity {
	opps := make([]MockOpportunity, count)

	types := []string{"o", "p", "k", "r", "g", "s", "i", "a"}
	baseTypes := []string{"solicitation", "presolicitation", "award", "special"}
	setAsideCodes := []string{"SBA", "8A", "HZC", "SDVOSBC", "WOSB", ""}
	naicsCodes := []string{"541330", "541511", "541512", "541519", "518210", "541690"}
	states := []string{"VA", "MD", "DC", "CA", "TX", "FL", "NY"}
	cities := []string{"Arlington", "Bethesda", "Washington", "San Diego", "Austin", "Miami", "New York"}

	departments := []string{
		"DEPT OF DEFENSE",
		"GENERAL SERVICES ADMINISTRATION",
		"DEPT OF VETERANS AFFAIRS",
		"DEPT OF HOMELAND SECURITY",
		"DEPT OF HEALTH AND HUMAN SERVICES",
	}

	for i := 0; i < count; i++ {
		noticeID := fmt.Sprintf("MOCK-%06d", i+1)
		postedDate := time.Now().AddDate(0, 0, -rand.Intn(30))
		responseDeadline := postedDate.AddDate(0, 0, 30+rand.Intn(60))

		opps[i] = MockOpportunity{
			NoticeID:           noticeID,
			Title:              fmt.Sprintf("Mock Opportunity %d - %s Services", i+1, randomWord()),
			SolicitationNumber: fmt.Sprintf("SOL-%06d", i+1),
			FullParentPathName: departments[rand.Intn(len(departments))],
			FullParentPathCode: fmt.Sprintf("%04d", rand.Intn(10000)),
			PostedDate:         postedDate.Format("2006-01-02"),
			Type:               types[rand.Intn(len(types))],
			BaseType:           baseTypes[rand.Intn(len(baseTypes))],
			ArchiveType:        "auto30",
			ArchiveDate:        responseDeadline.AddDate(0, 0, 30).Format("2006-01-02"),
			SetAsideCode:       setAsideCodes[rand.Intn(len(setAsideCodes))],
			SetAside:           "Small Business Set-Aside",
			ResponseDeadLine:   responseDeadline.Format("2006-01-02T15:04:05-07:00"),
			NAICSCode:          naicsCodes[rand.Intn(len(naicsCodes))],
			NAICSCodes:         randomNAICSCodes(naicsCodes),
			ClassificationCode: fmt.Sprintf("%c", 'A'+rand.Intn(26)),
			Active:             "Yes",
			OrganizationType:   "OFFICE",
			UILink:             fmt.Sprintf("https://sam.gov/opp/%s/view", noticeID),
			OfficeAddress: &OfficeAddress{
				City:        cities[rand.Intn(len(cities))],
				State:       states[rand.Intn(len(states))],
				ZipCode:     fmt.Sprintf("%05d", 10000+rand.Intn(90000)),
				CountryCode: "USA",
			},
			PointOfContact: []PointOfContact{
				{
					Type:     "primary",
					FullName: fmt.Sprintf("John %s", randomWord()),
					Title:    "Contract Specialist",
					Email:    fmt.Sprintf("contact%d@agency.gov", i),
					Phone:    fmt.Sprintf("555-%03d-%04d", rand.Intn(1000), rand.Intn(10000)),
				},
			},
			PlaceOfPerformance: &PlaceOfPerformance{
				StreetAddress: fmt.Sprintf("%d Main Street", rand.Intn(9999)+1),
				City:          &CityInfo{Code: "12345", Name: cities[rand.Intn(len(cities))]},
				State:         &StateInfo{Code: states[rand.Intn(len(states))], Name: "State"},
				Zip:           fmt.Sprintf("%05d", 10000+rand.Intn(90000)),
				Country:       &CountryInfo{Code: "USA", Name: "United States"},
			},
		}
	}

	return opps
}

func generateDescription(noticeID string) string {
	return fmt.Sprintf(`This is a mock description for notice %s.

The Government requires contractor support for various IT services including but not limited to:
- System administration and maintenance
- Software development and integration
- Network security and monitoring
- Help desk and user support

Period of Performance: 12 months base with 4 option years.

This notice is for mock/testing purposes only.`, noticeID)
}

func randomWord() string {
	words := []string{
		"Technology", "Engineering", "Support", "Management", "Analysis",
		"Development", "Integration", "Consulting", "Logistics", "Security",
		"Infrastructure", "Operations", "Maintenance", "Training", "Research",
	}
	return words[rand.Intn(len(words))]
}

func randomNAICSCodes(allCodes []string) []string {
	count := 1 + rand.Intn(3)
	codes := make([]string, 0, count)
	used := make(map[string]bool)

	for len(codes) < count {
		code := allCodes[rand.Intn(len(allCodes))]
		if !used[code] {
			codes = append(codes, code)
			used[code] = true
		}
	}
	return codes
}
