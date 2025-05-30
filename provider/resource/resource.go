package resource

import (
	"go.opentelemetry.io/otel"
)

var Tracer = otel.Tracer("mid/provider/resource")
