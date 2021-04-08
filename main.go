package main

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/resource"
)

type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

type LambdaRequest struct {
	Metrics []Metric `json:"metrics"`
}

type LambdaResponse struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

func send(ctx context.Context, metrics []Metric) error {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:9090/api/v1/write"
	}
	config := cortex.Config{
		Endpoint:      endpoint,
		RemoteTimeout: 10 * time.Second,
		PushInterval:  2 * time.Second,
	}
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	if username != "" && password != "" {
		config.BasicAuth = map[string]string{
			"username": username,
			"password": password,
		}
	}

	pusher, err := cortex.InstallNewPipeline(config, controller.WithCollectPeriod(2*time.Second), controller.WithResource(resource.NewWithAttributes(attribute.String("job", "prometheus-remote-write-exporter"))))
	if err != nil {
		return err
	}

	meter := pusher.MeterProvider().Meter("meter")
	metricsMap := make(map[string][]Metric)
	for _, m := range metrics {
		if _, ok := metricsMap[m.Name]; !ok {
			metricsMap[m.Name] = make([]Metric, 0)
		}
		metricsMap[m.Name] = append(metricsMap[m.Name], m)
	}
	for name, m := range metricsMap {
		observerCallback := func(m []Metric) func(context.Context, metric.Float64ObserverResult) {
			return func(_ context.Context, result metric.Float64ObserverResult) {
				for _, mm := range m {
					attributes := make([]attribute.KeyValue, 0)
					for k, v := range mm.Labels {
						attributes = append(attributes, attribute.String(k, v))
					}
					result.Observe(mm.Value, attributes...)
				}
			}
		}(m)
		_ = metric.Must(meter).NewFloat64ValueObserver(name, observerCallback)
	}

	return pusher.Stop(ctx)
}

func Handler(lambdaReq LambdaRequest) (LambdaResponse, error) {
	ctx := context.Background()
	err := send(ctx, lambdaReq.Metrics)
	if err != nil {
		return LambdaResponse{
			StatusCode: 500,
			Body:       "error",
		}, err
	}
	return LambdaResponse{
		StatusCode: 200,
		Body:       "ok",
	}, nil
}

func main() {
	lambda.Start(Handler)
}
