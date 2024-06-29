package main

import (
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		for i := 0; i < 1000; i++ {
			log.Println("validating request, checking auth, checking stock, ... etc concurrently")
			go leakyGoroutine()
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong\n"))
	})

	// create endpoint to handle checkout
	// simulate multiple user > 1000 user using k8s
	// simulate parallel process in 1 checkout flow using locust
	// checkout:
	// 1. stock validation
	// 2. call payment request - bottleneck on payment
	// 3. created shipping request - bottleneck on shipping request

	log.Println("starting http server")
	http.ListenAndServe(":5000", nil)
}

func leakyGoroutine() {
	// Create a slice that will keep growing
	var data []int

	for {
		// Append to the slice to simulate memory usage
		data = append(data, 1)
		// Sleep for a short duration to slow down the loop
		time.Sleep(1000 * time.Millisecond)
	}
}
