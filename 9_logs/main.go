package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sloglogstash "github.com/samber/slog-logstash/v2"
)

const serviceName = "golang_app"

var (
	metricCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: serviceName,
			Name:      "http_request_count",
		},
		[]string{"method", "path", "code"}, // label
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

	metricsHistogram = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: serviceName,
			Name:      "transcation_processing_time",
			Buckets:   []float64{0.5, 1, 2},
		},
	)
)

func init() {
	prometheus.MustRegister(metricCounter)
	prometheus.MustRegister(metricGauge)
	prometheus.MustRegister(metricSummary)
	prometheus.MustRegister(metricsHistogram)
}

func main() {
	logstashAddr := "logstash:5000"
	conn, err := net.Dial("tcp", logstashAddr)
	if err != nil {
		log.Fatalf("could not connect to Logstash: %v", err)
	} else {
		println("connected to logstash successfully")
	}
	defer conn.Close()

	// Create a new logger with the Logstash handler
	_ = sloglogstash.Option{Level: slog.LevelDebug, Conn: conn}.NewLogstashHandler()

	logHandler := slog.NewJSONHandler(conn, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger = logger.With("environment", "dev")

	logger.Info("starting http server ...")

	ctx, cancelSimulationJob := context.WithCancel(context.Background())
	defer cancelSimulationJob()

	sigCH := make(chan os.Signal, 1)
	signal.Notify(sigCH, syscall.SIGINT, syscall.SIGTERM)

	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				log.Println("stopping job simulation")
				return
			default:
				// log.Printf("sending metrics at %s\n", time.Now().Format(time.RFC3339Nano))
				metricGauge.WithLabelValues("ID", "JAK").Add(float64(rand.Intn(100)))
				time.Sleep(1 * time.Second)
			}
		}
	}(ctx)

	server := http.Server{
		Addr: ":1000",
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		metricCounter.WithLabelValues("GET", "/ping", "200").Inc()
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
			time.Sleep(100 * time.Millisecond)
			metricCounter.WithLabelValues("POST", "/orders", "200").Inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(responseData)
		} else {
			time.Sleep(1300 * time.Millisecond)
			// calling external dependency, and having network failure
			err := errors.New("connection to inventory service disconnected")
			// logger.Error("failed to connect to inventory service with error: ", err)

			customerID := "123" // taken from http request payload
			logger.With("customer_id", customerID, "error", err, "product_id", "product-a", "product_category", "electronic").Error("failed to connect to inventory service")
			metricCounter.WithLabelValues("POST", "/orders", "500").Inc()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			responseData.Error = Error{
				Message: "failure, this is simulated by devs",
			}
			_ = json.NewEncoder(w).Encode(responseData)
		}

		duration := time.Since(startTime)
		metricSummary.Observe(duration.Seconds())
		metricsHistogram.Observe(duration.Seconds())
	})

	http.HandleFunc("/internal/orders", func(w http.ResponseWriter, r *http.Request) {
		// simulating httpcall failure to xendit api
		err := errors.New("payment gateway not responding, with http status code 502")
		logger.With("tracer_id", "trace-id-a", "request_id", "request-id-sample", "customer_id", "customer-1", "error", err, "product_id", "product-a", "order_id", "order-id-sample").Error("failing validating payment")
		metricCounter.WithLabelValues("GET", "/internal/orders", "200").Inc()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{}}`))
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

type LogstashHandler struct {
	conn net.Conn
}

func NewLogstashHandler(address string) (*LogstashHandler, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &LogstashHandler{conn: conn}, nil
}

func (h *LogstashHandler) Handle(ctx context.Context, record slog.Record) error {
	timestamp := time.Now().Format(time.RFC3339)
	message := record.Message
	logLine := timestamp + " " + record.Level.String() + " " + message + "\n"
	_, err := h.conn.Write([]byte(logLine))
	return err
}

func (h *LogstashHandler) Enabled(ctx context.Context, level slog.Level) {
	// return
}

func (h *LogstashHandler) Close() error {
	return h.conn.Close()
}
