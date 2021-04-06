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

	pusher, err := cortex.InstallNewPipeline(config, controller.WithCollectPeriod(2*time.Second), controller.WithResource(resource.NewWithAttributes(attribute.String("R", "V"))))
	if err != nil {
		return err
	}

	meter := pusher.MeterProvider().Meter("meter")
	for _, m := range metrics {
		recorder := metric.Must(meter).NewFloat64ValueRecorder(
			m.Name,
		)
		attributes := make([]attribute.KeyValue, 0)
		for k, v := range m.Labels {
			attributes = append(attributes, attribute.String(k, v))
		}
		recorder.Record(ctx, m.Value, attributes...)
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
