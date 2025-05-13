package executor

import "go.opentelemetry.io/otel"

var Tracer = otel.Tracer("mid/provider/executor")
