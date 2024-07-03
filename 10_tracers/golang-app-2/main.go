package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics"
)

func InitJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := config.Configuration{
		ServiceName: "inventory_service",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:           true,
			LocalAgentHostPort: "jaeger:6831",
		},
	}

	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger), config.Metrics(metrics.NullFactory))
	if err != nil {
		log.Fatalf("cannot initialize Jaeger Tracer: %v", err)
	}

	return tracer, closer
}

func main() {
	tracer, closer := InitJaeger("inventory_service")
	defer closer.Close()
	opentracing.SetGlobalTracer(tracer)

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`pong`))
	})

	http.HandleFunc("/checkstock", func(w http.ResponseWriter, r *http.Request) {
		var serverSpan opentracing.Span
		wireContext, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		if err != nil {
			serverSpan = tracer.StartSpan("validating_stock")
		} else {
			serverSpan = tracer.StartSpan("validating_stock", ext.RPCServerOption(wireContext))
		}
		defer serverSpan.Finish()

		// simulating latency
		time.Sleep(850 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`success`))
	})

	log.Println("inventory service is running on http://localhost:2000")
	log.Fatal(http.ListenAndServe(":2000", nil))
}

// port: 2000
