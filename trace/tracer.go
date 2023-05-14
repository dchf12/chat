package trace

import (
	"fmt"
	"io"
)

type Tracer interface {
	Trace(...any)
}

type tracer struct {
	out io.Writer
}

func (t *tracer) Trace(a ...any) {
	t.out.Write([]byte(fmt.Sprint(a...)))
	t.out.Write([]byte("\n"))
}

type nilTracer struct{}

func (t *nilTracer) Trace(a ...any) {}

func New(w io.Writer) Tracer {
	return &tracer{out: w}
}

func Off() Tracer {
	return &nilTracer{}
}
