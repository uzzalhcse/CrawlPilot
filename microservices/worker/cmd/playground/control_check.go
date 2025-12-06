package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	targetURL := "https://tls.peet.ws/api/all"
	fmt.Printf("Navigating to %s with Standard Go Client\n", targetURL)

	start := time.Now()
	resp, err := http.Get(targetURL)
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	fmt.Printf("Request took %v\n", time.Since(start))

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read body: %v", err)
	}

	fmt.Println("\n--- Response Body (Standard Go) ---")
	fmt.Println(string(content))
	fmt.Println("-----------------------------------")
}
