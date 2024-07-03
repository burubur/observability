package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const serviceName = "golang_app"

var (
	metricCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "http_request_count",
		},
		[]string{"method", "path", "code"},
	)

	metricGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: serviceName,
			Name:      "active_users",
		},
		[]string{"country_id", "city_id"},
	)

	// usecase: order processing duration summary
	metricSummary = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace:  serviceName,
			Name:       "order_processing_duration",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
	)
)

func init() {
	prometheus.MustRegister(metricCounter)
	prometheus.MustRegister(metricGauge)
	prometheus.MustRegister(metricSummary)
}

func main() {
	println("starting http server ...")

	ctx, cancelSimulationJob := context.WithCancel(context.Background())
	defer cancelSimulationJob()

	sigCH := make(chan os.Signal, 1)
	signal.Notify(sigCH, syscall.SIGINT, syscall.SIGTERM)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				println("stopping job simulation")
				return
			default:
				log.Printf("sending metrics at %s\n", time.Now().Format(time.RFC3339Nano))
				metricGauge.WithLabelValues("ID", "JAK").Add(float64(rand.Intn(1000)))
				time.Sleep(300 * time.Millisecond)
			}
		}
	}(ctx)

	server := http.Server{
		Addr: ":1000",
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`pong`))
	})

	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		log.Println("serving traffic ...")
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

		duration := time.Since(startTime)
		metricSummary.Observe(float64(duration))
	})

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("HTTP server ListenAndServe: %v\n", err)
		}
	}()

	// waiting termination signal to stop the program/app
	<-sigCH
	log.Print("termination signal received, shutting down...")
	cancelSimulationJob()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		log.Printf("failled to shutdown http server due to error: %s\n", shutdownErr)
	} else {
		log.Println("http server stopped gracefully")
	}

	// just to wait all logs from goroutine is also printed, for debugging purpose
	time.Sleep(2 * time.Second)
}
