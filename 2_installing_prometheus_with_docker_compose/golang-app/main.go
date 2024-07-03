package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	println("starting http server ...")

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`pong`))
	})

	http.ListenAndServe(":1000", nil)
	println("stopped http server")
}
