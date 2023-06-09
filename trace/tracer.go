package trace

import (
	"fmt"
	"io"
)

type Tracer struct {
	out io.Writer
}

func (t Tracer) Trace(a ...any) {
	if t.out == nil {
		return
	}
	fmt.Fprintln(t.out, a...)
}
func New(w io.Writer) Tracer {
	return Tracer{out: w}
}
