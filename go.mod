module github.com/mtanda/prometheus-remote-write-exporter

go 1.16

require (
	github.com/aws/aws-lambda-go v1.23.0
	go.opentelemetry.io/contrib/exporters/metric/cortex v0.19.0
	go.opentelemetry.io/otel v0.19.0
	go.opentelemetry.io/otel/metric v0.19.0
	go.opentelemetry.io/otel/sdk v0.19.0
	go.opentelemetry.io/otel/sdk/metric v0.19.0
)
