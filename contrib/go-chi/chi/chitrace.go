// Package chi provides tracing middleware for the Chi web framework.
package chi

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	ddtrace "github.com/DataDog/dd-trace-go/opentracing"
	"github.com/DataDog/dd-trace-go/tracer/ext"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	opentracing "github.com/opentracing/opentracing-go"
	otext "github.com/opentracing/opentracing-go/ext"
)

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract span from incoming request if possible, create a new span otherwise.
		var span opentracing.Span
		var ctx context.Context
		if clientSpanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header)); err == nil {
			span = opentracing.StartSpan("http.request", otext.RPCServerOption(clientSpanCtx))
			ctx = opentracing.ContextWithSpan(r.Context(), span)
		} else {
			span, ctx = opentracing.StartSpanFromContext(r.Context(), "http.request")
		}
		defer span.Finish()

		// Pass the span through the request's context
		rr := r.WithContext(ctx)

		// Wrap the ResponseWriter in order to get access to the status code.
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Serve Request
		next.ServeHTTP(ww, rr)

		var route string
		// Get the resource associated to this request
		// It's recommended by Chi to do this after having served the request
		// https://github.com/go-chi/chi/blob/f2aed534ddaae0ec331ea305ef9a804b67b672f6/context.go#L80
		rc := chi.RouteContext(rr.Context())
		if rc == nil {
			route = "unknown"
		} else {
			route = strings.Replace(strings.Join(rc.RoutePatterns, ""), "/*/", "/", -1)
			if route == "" {
				route = "unknown"
			}
		}

		resource := rr.Method + " " + route

		span.SetTag(ddtrace.SpanType, "http")
		span.SetTag(ddtrace.ResourceName, resource)
		span.SetTag(ext.HTTPMethod, rr.Method)
		span.SetTag(ext.HTTPURL, rr.URL.Path)
		span.SetTag(ext.HTTPCode, strconv.Itoa(ww.Status()))

		if ww.Status() >= 500 {
			span.SetTag(ddtrace.Error, strconv.Itoa(ww.Status()))
		}

	})
}
