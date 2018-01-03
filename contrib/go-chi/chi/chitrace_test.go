package chi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	ddtrace "github.com/DataDog/dd-trace-go/opentracing"
	"github.com/DataDog/dd-trace-go/tracer/ext"
	"github.com/go-chi/chi"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/stretchr/testify/assert"
)

func TestTracingMiddleWare(t *testing.T) {
	assert := assert.New(t)

	// Tracing setup
	var tt testTracer
	opentracing.SetGlobalTracer(tt)

	// Router setup
	r := chi.NewRouter()
	r.Use(Middleware)

	r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, "ᕕ( ᐛ )ᕗ")
	})

	req := httptest.NewRequest("GET", "/test/24601", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	assert.Equal(res.Code, 200)
	assert.Equal(res.Body.String(), "ᕕ( ᐛ )ᕗ")

	assert.NotNil(latestSpan)
	assert.Equal(latestSpan.OperationName, "http.request")
	assert.Equal(latestSpan.Tags[ext.HTTPCode], "200")
	assert.Equal(latestSpan.Tags[ext.HTTPMethod], "GET")
	assert.Equal(latestSpan.Tags[ext.HTTPURL], "/test/24601")
	assert.Equal(latestSpan.Tags[ddtrace.ResourceName], "GET /test/{id}")
	assert.Nil(latestSpan.Tags[ddtrace.Error])
}

func TestTracingMiddleWareError(t *testing.T) {
	assert := assert.New(t)

	// Tracing setup
	var tt testTracer
	opentracing.SetGlobalTracer(tt)

	// Router setup
	r := chi.NewRouter()
	r.Use(Middleware)

	r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		fmt.Fprint(w, "¯\\_(ツ)_/¯")
	})

	req := httptest.NewRequest("GET", "/test/24601", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	assert.Equal(res.Code, 500)
	assert.Equal(res.Body.String(), "¯\\_(ツ)_/¯")

	assert.NotNil(latestSpan)
	assert.Equal(latestSpan.Tags[ext.HTTPCode], "500")
	assert.Equal(latestSpan.Tags[ddtrace.Error], "500")
}

// Fake Tracer, Span, and Context for test purposes
// Based on https://github.com/opentracing/opentracing-go/blob/master/testtracer_test.go
type testSpan struct {
	spanContext   testSpanContext
	OperationName string
	Tags          map[string]interface{}
}

func (s testSpan) SetTag(key string, value interface{}) opentracing.Span {
	s.Tags[key] = value
	return s
}

func (s testSpan) Equal(os opentracing.Span) bool                         { return true }
func (s testSpan) Context() opentracing.SpanContext                       { return s.spanContext }
func (s testSpan) Finish()                                                {}
func (s testSpan) FinishWithOptions(opts opentracing.FinishOptions)       {}
func (s testSpan) LogFields(fields ...log.Field)                          {}
func (s testSpan) LogKV(kvs ...interface{})                               {}
func (s testSpan) SetOperationName(operationName string) opentracing.Span { return s }
func (s testSpan) Tracer() opentracing.Tracer                             { return testTracer{} }
func (s testSpan) SetBaggageItem(key, val string) opentracing.Span        { return s }
func (s testSpan) BaggageItem(key string) string                          { return "" }
func (s testSpan) LogEvent(event string)                                  {}
func (s testSpan) LogEventWithPayload(event string, payload interface{})  {}
func (s testSpan) Log(data opentracing.LogData)                           {}

type testSpanContext struct {
	HasParent bool
	FakeID    int
}

func (n testSpanContext) ForeachBaggageItem(handler func(k, v string) bool) {}

type testTracer struct{}

var latestSpan testSpan

func (t testTracer) StartSpan(operationName string, opts ...opentracing.StartSpanOption) opentracing.Span {
	fakeID := nextFakeID()
	fmt.Printf("Starting Span %d", fakeID)

	s := testSpan{
		OperationName: operationName,
		Tags:          make(map[string]interface{}),
		spanContext: testSpanContext{
			HasParent: false,
			FakeID:    fakeID,
		},
	}
	latestSpan = s
	return s
}

var fakeIDSource = 0

func nextFakeID() int {
	fakeIDSource++
	return fakeIDSource
}

func (t testTracer) Inject(sp opentracing.SpanContext, format interface{}, carrier interface{}) error {
	return nil
}

func (t testTracer) Extract(format interface{}, carrier interface{}) (opentracing.SpanContext, error) {
	return nil, opentracing.ErrSpanContextNotFound
}
