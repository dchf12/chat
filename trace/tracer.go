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

func New(w io.Writer) Tracer {
	return &tracer{out: w}
}
