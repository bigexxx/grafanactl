package format

import (
	"bytes"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestJSONCodecEncode_DoesNotEscapeAmpersandForUnstructured(t *testing.T) {
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"spec": map[string]any{
				"name": "Annotations & Alerts",
			},
		},
	}

	var buf bytes.Buffer
	c := NewJSONCodec()
	if err := c.Encode(&buf, u); err != nil {
		t.Fatalf("encode: %v", err)
	}

	out := buf.String()
	if bytes.Contains([]byte(out), []byte(`\u0026`)) {
		t.Fatalf("expected output to not contain \\\\u0026, got: %s", out)
	}
	if !bytes.Contains([]byte(out), []byte(`Annotations & Alerts`)) {
		t.Fatalf("expected output to contain unescaped '&', got: %s", out)
	}
}
