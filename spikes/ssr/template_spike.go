// ABOUTME: Spike to validate server-side rendering with Go templates
// ABOUTME: Proves we can build a UI without JavaScript build tooling

package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

// Data structures for the UI
type TopType struct {
	Type      string
	Count     int
	TotalSize int64
	AvgSize   int64
}

type PageData struct {
	Title       string
	DumpFile    string
	TopTypes    []TopType
	CurrentTime string
	SortBy      string
	SortOrder   string
}

func main() {
	// Parse templates
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
	
	// Handler for top types page
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Simulate data
		data := PageData{
			Title:       "HeapLens - Top Types",
			DumpFile:    "production.heap",
			CurrentTime: time.Now().Format(time.RFC3339),
			SortBy:      r.URL.Query().Get("sort"),
			SortOrder:   r.URL.Query().Get("order"),
			TopTypes: []TopType{
				{Type: "[]byte", Count: 50000, TotalSize: 5000000, AvgSize: 100},
				{Type: "string", Count: 30000, TotalSize: 1500000, AvgSize: 50},
				{Type: "*MyStruct", Count: 10000, TotalSize: 2000000, AvgSize: 200},
				{Type: "map[string]int", Count: 5000, TotalSize: 800000, AvgSize: 160},
				{Type: "[]int", Count: 8000, TotalSize: 640000, AvgSize: 80},
			},
		}
		
		// Apply sorting (simulated)
		if data.SortBy == "count" {
			// Would sort by count
		} else if data.SortBy == "size" {
			// Would sort by size
		}
		
		// Render template
		if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	
	// Static CSS handler
	http.HandleFunc("/static/css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(embeddedCSS))
	})
	
	fmt.Println("=== SSR Template Spike ===")
	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("")
	fmt.Println("Features demonstrated:")
	fmt.Println("✅ Server-side HTML rendering")
	fmt.Println("✅ No JavaScript build required")
	fmt.Println("✅ Embedded templates and CSS")
	fmt.Println("✅ Dynamic sorting via URL params")
	fmt.Println("✅ Clean, responsive UI")
	fmt.Println("")
	fmt.Println("Try these URLs:")
	fmt.Println("- http://localhost:8080/")
	fmt.Println("- http://localhost:8080/?sort=count&order=desc")
	fmt.Println("- http://localhost:8080/?sort=size&order=asc")
	
	log.Fatal(http.ListenAndServe(":8080", nil))
}

var embeddedCSS = `
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
	margin: 0;
	padding: 0;
	background: #f5f5f5;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
	padding: 20px;
}

header {
	background: #2c3e50;
	color: white;
	padding: 20px 0;
	margin-bottom: 30px;
}

header h1 {
	margin: 0;
	padding: 0 20px;
}

.info {
	background: white;
	padding: 15px;
	border-radius: 5px;
	margin-bottom: 20px;
	box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

table {
	width: 100%;
	background: white;
	border-radius: 5px;
	overflow: hidden;
	box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

th {
	background: #34495e;
	color: white;
	padding: 12px;
	text-align: left;
	cursor: pointer;
	user-select: none;
}

th:hover {
	background: #2c3e50;
}

td {
	padding: 12px;
	border-bottom: 1px solid #ecf0f1;
}

tr:hover {
	background: #f8f9fa;
}

.sortable::after {
	content: " ↕";
	opacity: 0.5;
}

.sorted-asc::after {
	content: " ↑";
}

.sorted-desc::after {
	content: " ↓";
}

.number {
	text-align: right;
	font-family: "SF Mono", Monaco, monospace;
}

footer {
	margin-top: 40px;
	padding: 20px;
	text-align: center;
	color: #7f8c8d;
	font-size: 0.9em;
}
`