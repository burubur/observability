package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "golang_app",
			Name:      "http_request_count",
		},
		[]string{"method", "path", "code"},
	)
)

func init() {
	prometheus.MustRegister(metricCounter)
}

func main() {
	println("starting http server ...")

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`pong`))
	})

	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		type Data struct {
			OrderID int `json:"order_id"`
		}

		type Error struct {
			Message     string `json:"message,omitempty"`
			Description string `json:"description,omitempty"`
		}

		type ResponseData struct {
			Data  Data  `json:"data"`
			Error Error `json:"error,omitempty"`
		}

		responseData := ResponseData{
			Data: Data{
				OrderID: rand.Intn(1000),
			},
		}

		// to simulate successful or failing http status codes
		if responseData.Data.OrderID%2 == 0 {
			time.Sleep(200 * time.Millisecond)
			metricCounter.WithLabelValues("POST", "/orders", "200")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(responseData)
		} else {
			time.Sleep(600 * time.Millisecond)
			metricCounter.WithLabelValues("POST", "/orders", "500")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			responseData.Error = Error{
				Message: "failure, this is simulated by devs",
			}
			_ = json.NewEncoder(w).Encode(responseData)
		}
	})

	http.ListenAndServe(":1000", nil)
	println("stopped http server")
}
