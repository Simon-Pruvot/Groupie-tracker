package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	baseAPI     = "https://groupietrackers.herokuapp.com/api"
	artistsAPI  = baseAPI + "/artists"
	relationAPI = baseAPI + "/relation"
)

type Artist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Genres       []string `json:"genres"`
}

type relationIndex struct {
	Index []relation `json:"index"`
}

type relation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

type Concert struct {
	ArtistID   int
	ArtistName string
	Genres     []string
	Location   string
	City       string
	Country    string
	Date       time.Time
	RawDate    string
	PriceCents int
}

type IndexPageData struct {
	Artists []ArtistView
}

type ArtistView struct {
	ID           int
	Name         string
	Image        string
	Members      []string
	CreationDate int
	FirstAlbum   string
	Genres       []string
	Locations    []string
	Dates        []string
	Rel          map[string][]string // location -> dates
}

func main() {
	tmplPath := filepath.Join("template", "index.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	artistsView, err := loadArtistsView()
	if err != nil {
		log.Printf("warning: failed to load artists view: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := IndexPageData{Artists: artistsView}
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("template execute error: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	})

	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func loadConcerts() ([]Concert, error) {
	var artists []Artist
	if err := fetchJSON(artistsAPI, &artists); err != nil {
		return nil, err
	}
	idToArtist := make(map[int]Artist, len(artists))
	for _, a := range artists {
		idToArtist[a.ID] = a
	}

	var relIdx relationIndex
	if err := fetchJSON(relationAPI, &relIdx); err != nil {
		return nil, err
	}

	var concerts []Concert
	for _, rel := range relIdx.Index {
		a := idToArtist[rel.ID]
		for loc, dates := range rel.DatesLocations {
			city, country := parseLocation(loc)
			for _, dateStr := range dates {
				dt := parseDate(dateStr)
				concerts = append(concerts, Concert{
					ArtistID:   a.ID,
					ArtistName: a.Name,
					Genres:     a.Genres,
					Location:   loc,
					City:       city,
					Country:    country,
					Date:       dt,
					RawDate:    dateStr,
					PriceCents: 0,
				})
			}
		}
	}
	return concerts, nil
}

type datesIndex struct {
	Index []struct {
		ID    int      `json:"id"`
		Dates []string `json:"dates"`
	} `json:"index"`
}

type locationsIndex struct {
	Index []struct {
		ID        int      `json:"id"`
		Locations []string `json:"locations"`
	} `json:"index"`
}

func loadArtistsView() ([]ArtistView, error) {
	var artists []Artist
	if err := fetchJSON(artistsAPI, &artists); err != nil {
		return nil, err
	}
	var dIdx datesIndex
	if err := fetchJSON(baseAPI+"/dates", &dIdx); err != nil {
		return nil, err
	}
	var lIdx locationsIndex
	if err := fetchJSON(baseAPI+"/locations", &lIdx); err != nil {
		return nil, err
	}
	var rIdx relationIndex
	if err := fetchJSON(relationAPI, &rIdx); err != nil {
		return nil, err
	}

	datesByID := make(map[int][]string, len(dIdx.Index))
	for _, it := range dIdx.Index {
		datesByID[it.ID] = it.Dates
	}
	locsByID := make(map[int][]string, len(lIdx.Index))
	for _, it := range lIdx.Index {
		locsByID[it.ID] = it.Locations
	}
	relByID := make(map[int]map[string][]string, len(rIdx.Index))
	for _, it := range rIdx.Index {
		relByID[it.ID] = it.DatesLocations
	}

	// Sort artists by name for stable display
	sort.Slice(artists, func(i, j int) bool { return artists[i].Name < artists[j].Name })

	out := make([]ArtistView, 0, len(artists))
	for _, a := range artists {
		av := ArtistView{
			ID:           a.ID,
			Name:         a.Name,
			Image:        a.Image,
			Members:      a.Members,
			CreationDate: a.CreationDate,
			FirstAlbum:   a.FirstAlbum,
			Genres:       a.Genres,
			Locations:    locsByID[a.ID],
			Dates:        datesByID[a.ID],
			Rel:          relByID[a.ID],
		}
		out = append(out, av)
	}
	return out, nil
}

func fetchJSON(url string, out any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return &httpError{Code: resp.StatusCode, Body: string(b)}
	}
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

type httpError struct {
	Code int
	Body string
}

func (e *httpError) Error() string { return "http error: " + http.StatusText(e.Code) }

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02", // ISO
		"02-01-2006", // dd-mm-yyyy
		"01-02-2006", // mm-dd-yyyy
		"2006/01/02",
		"02/01/2006",
		"01/02/2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseLocation(s string) (city, country string) {
	// API locations are typically like "city-country" (often lowercase)
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}
	parts := strings.Split(s, "-")
	if len(parts) >= 2 {
		country = parts[len(parts)-1]
		city = strings.Join(parts[:len(parts)-1], "-")
	} else {
		city = s
		country = ""
	}
	city = titleCase(city)
	country = titleCase(country)
	return
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	// Replace underscores with spaces then title-case words split by space or hyphen
	s = strings.ReplaceAll(s, "_", " ")
	// Handle spaces
	words := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '-' })
	// Rebuild preserving hyphens by splitting manually
	// Simpler: lower all, then title each word, and re-join with space
	for i := range words {
		if words[i] == "" {
			continue
		}
		w := strings.ToLower(words[i])
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
