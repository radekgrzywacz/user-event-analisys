package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func getData(w http.ResponseWriter, r *http.Request) {
	log.Println("------------ Got data request --------------")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(body, &prettyJSON); err != nil {
		log.Println("Error parsing JSON:", err)
		log.Println("Raw body:", string(body))
	} else {
		prettyBody, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Println("Parsed JSON Body:\n", string(prettyBody))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, `{"message": "Data received successfully"}`)

	log.Println("--------------------------------------------")
}

func main() {
	http.HandleFunc("/ingestor", getData)

	log.Print("Ingestor server starting...")
	err := http.ListenAndServe(":8081", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}
