package trace

import (
	"bytes"
	"testing"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	var tracer Tracer
	if tracer.out != nil {
		t.Error("Return from New should be nil")
	} else {
		tracer.Trace("Hello, trace package.")
		if buf.String() != "" {
			t.Error("Trace should not write blank.")
		}
	}

	tracer = New(&buf)
	if tracer.out == nil {
		t.Error("Return from New should not be nil")
	} else {
		tracer.Trace("Hello, trace package.")
		if buf.String() != "Hello, trace package.\n" {
			t.Errorf("Trace should not write '%s'.", buf.String())
		}
	}
}
