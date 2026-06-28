package telemetry

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

type Observe struct {
	Tracer       trace.Tracer
	Meter        metric.Meter
	Logger       *slog.Logger
	SCounter     metric.Int64Counter
	SDuration    metric.Float64Histogram
	ErrCounter   metric.Int64Counter
	EventCounter metric.Int64Counter
	BytesMoved   metric.Int64Counter
	FextCounter  metric.Int64Counter
}

func newExporter(ctx context.Context) (sdktrace.SpanExporter, error) {
	return otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("127.0.0.1:4318"),
		otlptracehttp.WithInsecure())
}

func newResource() (*resource.Resource, error) {
	r, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			semconv.ServiceName("Noxie-Sort"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func newTracerProvider(exp sdktrace.SpanExporter, r *resource.Resource) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(r),
	)
}

func NewTelemetry() (func(context.Context) error, *Observe, error) {
	ctx := context.Background()

	exp, err := newExporter(ctx)
	if err != nil {
		return nil, nil, err
	}

	res, err := newResource()
	if err != nil {
		return nil, nil, err
	}
	tp := newTracerProvider(exp, res)

	meterProvider, err := newMeterProvider(ctx, res)
	if err != nil {
		return nil, nil, err
	}

	loggerProvider, err := newLoggerProvider(res)
	if err != nil {
		panic(err)
	}

	shutdown := func(ctx context.Context) error {
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		if err := tp.Shutdown(ctx); err != nil {
			return err
		}
		if err := loggerProvider.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	}

	global.SetLoggerProvider(loggerProvider)
	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	observer := getObserver()

	return shutdown, observer, nil
}

func newMeterProvider(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint("127.0.0.1:8428"),
		otlpmetrichttp.WithURLPath("/opentelemetry/v1/metrics"),
		otlpmetrichttp.WithInsecure())
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter,
			sdkmetric.WithInterval(10*time.Second))),
	)
	return meterProvider, nil
}

func newLoggerProvider(res *resource.Resource) (*log.LoggerProvider, error) {
	exporter, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}
	processor := log.NewBatchProcessor(exporter)
	provider := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(processor),
	)
	return provider, nil
}

func getObserver() *Observe {
	observer := &Observe{
		Meter:  otel.Meter("Noxie-Sort-Meter"),
		Tracer: otel.Tracer("Noxie-Sort-Tracer"),
		Logger: otelslog.NewLogger("Noxie-Sort-Logger"),
	}

	sCounter, _ := observer.Meter.Int64Counter(
		"sort.operations",
		metric.WithDescription("Number of sort operations"),
		metric.WithUnit("{call}"),
	)
	eventCounter, _ := observer.Meter.Int64Counter(
		"events.count",
		metric.WithDescription("Number of events"))
	sDuration, _ := observer.Meter.Float64Histogram(
		"sort.operation.duration",
		metric.WithDescription("Duration of sort operations"),
		metric.WithUnit("ms"),
	)
	errCounter, _ := observer.Meter.Int64Counter(
		"sort.errors",
		metric.WithDescription("Number of sorting errors"),
		metric.WithUnit("{error}"),
	)
	bytesMoved, _ := observer.Meter.Int64Counter(
		"bytes.moved",
		metric.WithDescription("Number of bytes moved"),
		metric.WithUnit("bytes"),
	)
	fextCounter, _ := observer.Meter.Int64Counter(
		"files.extension",
		metric.WithDescription("Number of files extensions"),
		metric.WithUnit("{operation}"))

	observer.SCounter = sCounter
	observer.SDuration = sDuration
	observer.ErrCounter = errCounter
	observer.EventCounter = eventCounter
	observer.BytesMoved = bytesMoved
	observer.FextCounter = fextCounter

	_, _ = observer.Meter.Float64ObservableGauge(
		"process.memory.alloc",
		metric.WithDescription("Currently heap usage (Heap Alloc) in Mb"),
		metric.WithUnit("MiB"),
		metric.WithFloat64Callback(func(_ context.Context, obs metric.Float64Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			memoryInMb := float64(m.Alloc) / (1024 * 1024)
			obs.Observe(memoryInMb)
			return nil
		}),
	)

	return observer
}
