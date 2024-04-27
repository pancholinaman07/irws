package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	utils "irws/utils"
)

type SearchResult struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

func main() {

	var dumpPath string
	flag.StringVar(&dumpPath, "p", "https://firebasestorage.googleapis.com/v0/b/gamedevs-278bd.appspot.com/o/enwiki-latest-abstract1.xml.gz?alt=media&token=147c0770-99e3-4776-b17b-8c2d78dcfc73", "Firebase URL of the .gz file")
	flag.Parse()

	log.Println("Running Full Text Search")

	// Download the file from the Firebase URL
	resp, err := http.Get(dumpPath)
	if err != nil {
		log.Fatal("Failed to download file:", err)
	}
	defer resp.Body.Close()

	// Create a temporary file to store the downloaded content
	tempFile, err := os.CreateTemp("", "dump-*.gz")
	if err != nil {
		log.Fatal("Failed to create temporary file:", err)
	}
	defer tempFile.Close()

	// Copy the downloaded content to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		log.Fatal("Failed to write to temporary file:", err)
	}

	start := time.Now()
	docs, err := utils.LoadDocuments(tempFile.Name())
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Loaded %d documents in %v", len(docs), time.Since(start))

	start = time.Now()
	idx := make(utils.Index)
	idx.Add(docs)
	log.Printf("Indexed %d documents in %v", len(docs), time.Since(start))

	http.HandleFunc("/search/", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimPrefix(r.URL.Path, "/search/")
		query = strings.ReplaceAll(query, "%20", " ") // Handle space encoding

		start := time.Now()
		matchedIDs := idx.Search(query)
		log.Printf("Search found %d documents in %v", len(matchedIDs), time.Since(start))

		var searchResults []SearchResult

		for _, id := range matchedIDs {
			doc := docs[id]
			result := SearchResult{ID: id, Text: doc.Text}
			searchResults = append(searchResults, result)
		}

		jsonData, err := json.Marshal(searchResults)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Write JSON response
		_, err = w.Write(jsonData)
		if err != nil {
			return
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))

}
