package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	client := &http.Client{Timeout: time.Second * 5}

	url := "http://localhost:3000/api/latency/"
	data := make(map[string]interface{})

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(data); err != nil {
		log.Fatalf("could not encode body: %s\n", err)
	}

	// Construct the request
	req, err := http.NewRequest(http.MethodGet, url, body)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("X-Api-Key", "Sr0KK2WnVqRDXbFMbNkqmX1bPhktDTKBeI3gRChg8X8R")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s %s %s\n", req.Method, req.URL.String(), res.Status)

	// Check the status from the client
	if res.StatusCode != 200 {
		res.Body.Close()
		log.Fatalf("could not access Kahu service: %s\n", res.Status)
	}

	// Read the response from Kahu
	defer res.Body.Close()
	info := make(map[string]interface{})
	if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
		log.Fatalf("could not parse kahu response: %s\n", err)
		return
	}

	// Print the response
	if d, err := json.MarshalIndent(info, "", "  "); err != nil {
		log.Fatalf("could not parse kahu response: %s\n", err)
	} else {
		fmt.Println(string(d))
	}
}
