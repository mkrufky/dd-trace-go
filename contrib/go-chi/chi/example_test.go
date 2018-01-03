package chi_test

import (
	"fmt"
	"log"
	"net/http"

	chitrace "github.com/DataDog/dd-trace-go/contrib/go-chi/chi"
	ddtrace "github.com/DataDog/dd-trace-go/opentracing"
	"github.com/go-chi/chi"
	opentracing "github.com/opentracing/opentracing-go"
)

// To start tracing requests, add the trace middleware to your Chi router.
func Example() {
	// Set up opentracing Tracer
	config := ddtrace.NewConfiguration()
	config.ServiceName = "my-web-app"
	config.AgentHostname = "datadog-client-hostname"

	tracer, closer, err := ddtrace.NewTracer(config)
	if err != nil {
		log.Fatal(err)
	}
	defer closer.Close()
	// set the Datadog tracer as a GlobalTracer
	opentracing.SetGlobalTracer(tracer)

	// Create your router and use the middleware.
	r := chi.NewRouter()
	r.Use(chitrace.Middleware)

	r.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprintln(w, "(っ^‿^)っ")
	})

	http.ListenAndServe(":1234", r)
}
