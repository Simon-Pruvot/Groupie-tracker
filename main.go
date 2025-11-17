package main

import (
	"html/template"
	"log"
	"net/http"
)

func main() {
	// Servir les fichiers statiques (CSS, images, etc.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Route principale
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.ParseFiles("template/index.htm"))
		data := map[string]string{
			"Title": "Groupie Tracker - 日本",
		}
		tmpl.Execute(w, data)
	})

	log.Println("🚀 Serveur lancé sur http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
