package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	baseAPI      = "https://groupietrackers.herokuapp.com/api"
	artistsAPI   = baseAPI + "/artists"
	relationAPI  = baseAPI + "/relation"
	datesAPI     = baseAPI + "/dates"
	locationsAPI = baseAPI + "/locations"
)

// --- Data Structures ---

type Artist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Genres       []string `json:"genres"`
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

type IndexPageData struct {
	Artists  []ArtistView
	Concerts []ConcertView
	Cities   []string
}

type ConcertView struct {
	ArtistName  string
	ArtistImage string
	ArtistID    int
	Location    string
	Date        string
	Genre       string
}

type Index2Data struct {
	ArtistName   string
	ArtistImage  string
	Members      []string
	CreationDate int
	FirstAlbum   string
	Genres       []string
	Locations    []string
	Dates        []string
	Rel          map[string][]string
}

type PageData struct {
	Music string
}

// --- API Response Structures (Internal) ---

type relationIndex struct {
	Index []struct {
		ID             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
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

// --- Main ---

func main() {
	// math add mod func
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"mod": func(a, b int) int { return a % b },
	}

	// load template
	pattern := filepath.Join("src", "template", "*.html")
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob(pattern)

	if err != nil {
		log.Fatalf("failed to parse templates: %v", err)
	}

	// Load data once at startup
	artistsView, err := loadArtistsView()
	if err != nil {
		log.Printf("warning: failed to load artists view: %v", err)
	}

	// Home / Search
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := IndexPageData{Artists: artistsView}

		// Build dynamic list of cities from API data (artistsView)
		citiesMap := make(map[string]struct{})
		for _, a := range artistsView {
			for _, loc := range a.Locations {
				if loc != "" {
					citiesMap[loc] = struct{}{}
				}
			}
			if a.Rel != nil {
				for loc := range a.Rel {
					if loc != "" {
						citiesMap[loc] = struct{}{}
					}
				}
			}
		}
		cities := make([]string, 0, len(citiesMap))
		for c := range citiesMap {
			cities = append(cities, c)
		}
		sort.Strings(cities)
		data.Cities = cities

		// Handle Search
		if r.Method == http.MethodPost {
			recherche := r.FormValue("recherche")
			if recherche != "" {
				data = searchGroup(recherche, data)
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Printf("template execute error: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})

	// Index2 (Artist Detail)
	http.HandleFunc("/index2", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index2" {
			http.NotFound(w, r)
			return
		}

		// Get ID from query param
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			id = 1 // Default to 1 if invalid or missing
		}

		// Try to load artist
		var data Index2Data

		// Try to find in preloaded data first (faster and has relations)
		found := false
		for _, item := range artistsView {
			if item.ID == id {
				data = Index2Data{
					ArtistName:   item.Name,
					ArtistImage:  item.Image,
					Members:      item.Members,
					CreationDate: item.CreationDate,
					FirstAlbum:   item.FirstAlbum,
					Genres:       item.Genres,
					Locations:    item.Locations,
					Dates:        item.Dates,
					Rel:          item.Rel,
				}
				found = true
				break
			}
		}

		if !found {
			// Fallback to direct fetch (might miss relations if we don't fetch them too, but for now let's just fetch artist info)
			var a Artist
			if err := fetchJSON(artistsAPI+"/"+strconv.Itoa(id), &a); err == nil {
				data = Index2Data{
					ArtistName:   a.Name,
					ArtistImage:  a.Image,
					Members:      a.Members,
					CreationDate: a.CreationDate,
					FirstAlbum:   a.FirstAlbum,
					Genres:       a.Genres,
				}
			}
		}

		// Reload templates to see changes
		t, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
		if err != nil {
			log.Printf("template parse error: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := t.ExecuteTemplate(w, "index2.html", data); err != nil {
			log.Printf("template execute error (index2): %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	})

	// Static Assets
	http.Handle("/CSS/", http.StripPrefix("/CSS/", http.FileServer(http.Dir(filepath.Join("src", "CSS")))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(filepath.Join("src", "images")))))
	http.Handle("/musique/", http.StripPrefix("/musique/", http.FileServer(http.Dir(filepath.Join("src", "musique")))))

	// Other Pages
	http.HandleFunc("/contact", makeHandler(tmpl, "contact.html", "/contact", IndexPageData{Artists: artistsView}))
	http.HandleFunc("/panier", makeHandler(tmpl, "panier.html", "/panier", IndexPageData{Artists: artistsView}))

	// Start Server
	log.Println("Server running on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// --- Helpers ---

func makeHandler(tmpl *template.Template, pageName string, path string, data interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, pageName, data); err != nil {
			log.Printf("template execute error (%s): %v", pageName, err)
			http.NotFound(w, r)
		}
	}
}

func loadArtistsView() ([]ArtistView, error) {
	var artists []Artist
	if err := fetchJSON(artistsAPI, &artists); err != nil {
		return nil, err
	}
	var dIdx datesIndex
	if err := fetchJSON(datesAPI, &dIdx); err != nil {
		return nil, err
	}
	var lIdx locationsIndex
	if err := fetchJSON(locationsAPI, &lIdx); err != nil {
		return nil, err
	}
	var rIdx relationIndex
	if err := fetchJSON(relationAPI, &rIdx); err != nil {
		return nil, err
	}

	// Create lookup maps
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

	// Sort artists by name
	sort.Slice(artists, func(i, j int) bool { return artists[i].Name < artists[j].Name })

	// Build view
	out := make([]ArtistView, 0, len(artists))
	for _, a := range artists {
		out = append(out, ArtistView{
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
		})
	}
	return out, nil
}

func searchGroup(query string, data IndexPageData) IndexPageData {
	var result []ArtistView
	for _, artist := range data.Artists {
		if strings.EqualFold(artist.Name, query) {
			result = append(result, artist)
		}
	}
	return IndexPageData{Artists: result}
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

// parseDate attempts to parse a date string using several common layouts.
// Returns the parsed time and true on success, otherwise zero time and false.
func parseDate(s string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"02/01/2006",
		"02 Jan 2006",
		"January 2, 2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// buildConcerts flattens ArtistView.Rel into a slice of ConcertView entries.
func buildConcerts(artists []ArtistView) []ConcertView {
	out := make([]ConcertView, 0)
	for _, a := range artists {
		if a.Rel != nil {
			for loc, dates := range a.Rel {
				for _, d := range dates {
					out = append(out, ConcertView{
						ArtistName:  a.Name,
						ArtistImage: a.Image,
						ArtistID:    a.ID,
						Location:    loc,
						Date:        d,
						Genre:       firstGenre(a.Genres),
					})
				}
			}
		}
	}
	return out
}

func firstGenre(genres []string) string {
	if len(genres) == 0 {
		return ""
	}
	return genres[0]
}
