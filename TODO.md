# TODO

1. Support both grpc and http protocols. It seems that gRPC works fine
   1. https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver/otlpreceiver
   2. https://github.com/open-telemetry/opentelemetry-collector/blob/main/receiver/otlpreceiver/otlp.go
2. Write simple integration test to send logs via grpc and http protocols
3. Receive traces and spans to play with Grafana Beyla. In general better understand traces Jeger backend

