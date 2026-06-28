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
	Tracer     trace.Tracer
	Meter      metric.Meter
	Logger     *slog.Logger
	OpCounter  metric.Int64Counter
	OpDuration metric.Float64Histogram
	ErrCounter metric.Int64Counter
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
			semconv.ServiceName("Noxie-Calc"),
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
		Meter:  otel.Meter("Noxie-Calc-Meter"),
		Tracer: otel.Tracer("Noxie-Calc-Tracer"),
		Logger: otelslog.NewLogger("Noxie-Calc-Logger"),
	}

	opCounter, _ := observer.Meter.Int64Counter(
		"api.counter",
		metric.WithDescription("Number of API calls."),
		metric.WithUnit("{call}"),
	)
	opDuration, _ := observer.Meter.Float64Histogram(
		"calc.operation.duration",
		metric.WithDescription("Duration of calc operations"),
		metric.WithUnit("ms"),
	)
	errCounter, _ := observer.Meter.Int64Counter(
		"calc.errors",
		metric.WithDescription("Number of calculation errors"),
		metric.WithUnit("{error}"),
	)

	observer.OpCounter = opCounter
	observer.OpDuration = opDuration
	observer.ErrCounter = errCounter

	_, _ = observer.Meter.Float64ObservableGauge(
		"process.memory.alloc",
		metric.WithDescription("Поточне споживання кучі (Heap Alloc) в мегабайтах"),
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
