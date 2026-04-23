package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/imryanparsa/kfqdn/internal/output"
	"github.com/imryanparsa/kfqdn/internal/resolver"
)

func makeStreams() (genericiooptions.IOStreams, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return genericiooptions.IOStreams{
		In:     strings.NewReader(""),
		Out:    buf,
		ErrOut: &bytes.Buffer{},
	}, buf
}

func TestPrintResults_Table(t *testing.T) {
	streams, buf := makeStreams()
	results := []resolver.Result{
		{Name: "my-svc.default.svc.cluster.local", Kind: "cluster-fqdn"},
	}
	output.PrintResults(streams, "my-svc", results, false, "table")

	got := buf.String()
	if !strings.Contains(got, "NAME") || !strings.Contains(got, "DNS NAME") || !strings.Contains(got, "KIND") {
		t.Errorf("header missing in output: %q", got)
	}
	if !strings.Contains(got, "my-svc") {
		t.Errorf("resource name missing in output: %q", got)
	}
	if !strings.Contains(got, "cluster-fqdn") {
		t.Errorf("kind missing in output: %q", got)
	}
	// --resolve not requested: no IP(S) column.
	if strings.Contains(got, "IP(S)") {
		t.Errorf("unexpected IP(S) column in non-resolve output: %q", got)
	}
}

func TestPrintResults_WithResolve(t *testing.T) {
	streams, buf := makeStreams()
	results := []resolver.Result{
		{Name: "my-svc.default.svc.cluster.local", Kind: "cluster-fqdn"},
	}
	output.PrintResults(streams, "my-svc", results, true, "table")

	got := buf.String()
	if !strings.Contains(got, "IP(S)") {
		t.Errorf("expected IP(S) column in resolve output: %q", got)
	}
}

func TestPrintNamedResults_WithNamespace(t *testing.T) {
	streams, buf := makeStreams()
	named := []resolver.NamedResults{
		{
			Namespace: "production",
			Name:      "my-svc",
			Results:   []resolver.Result{{Name: "my-svc.production.svc.cluster.local", Kind: "cluster-fqdn"}},
		},
	}
	output.PrintNamedResults(streams, named, true, false, "table")

	got := buf.String()
	if !strings.Contains(got, "NAMESPACE") {
		t.Errorf("expected NAMESPACE column: %q", got)
	}
	if !strings.Contains(got, "production") {
		t.Errorf("expected namespace value: %q", got)
	}
}

func TestPrintNamedResults_WithType(t *testing.T) {
	streams, buf := makeStreams()
	named := []resolver.NamedResults{
		{
			Namespace: "default",
			Name:      "my-svc",
			Type:      "svc",
			Results:   []resolver.Result{{Name: "my-svc.default.svc.cluster.local", Kind: "cluster-fqdn"}},
		},
	}
	output.PrintNamedResults(streams, named, true, false, "table")

	got := buf.String()
	if !strings.Contains(got, "TYPE") {
		t.Errorf("expected TYPE column: %q", got)
	}
	if !strings.Contains(got, "svc") {
		t.Errorf("expected type value: %q", got)
	}
}

func TestPrintNamedResults_WithType_ScanAll(t *testing.T) {
	// First entry has no Type; second does — TYPE column should still appear.
	streams, buf := makeStreams()
	named := []resolver.NamedResults{
		{
			Namespace: "default",
			Name:      "no-type",
			Type:      "",
			Results:   []resolver.Result{{Name: "no-type.default.svc.cluster.local", Kind: "cluster-fqdn"}},
		},
		{
			Namespace: "default",
			Name:      "has-type",
			Type:      "svc",
			Results:   []resolver.Result{{Name: "has-type.default.svc.cluster.local", Kind: "cluster-fqdn"}},
		},
	}
	output.PrintNamedResults(streams, named, true, false, "table")

	got := buf.String()
	if !strings.Contains(got, "TYPE") {
		t.Errorf("expected TYPE column when any entry has a type: %q", got)
	}
}

func TestPrintNamedResults_WideFormat(t *testing.T) {
	streams, buf := makeStreams()
	named := []resolver.NamedResults{
		{
			Namespace: "default",
			Name:      "my-svc",
			Results:   []resolver.Result{{Name: "my-svc.default.svc.cluster.local", Kind: "cluster-fqdn"}},
			Extra:     "80/TCP",
		},
	}
	output.PrintNamedResults(streams, named, false, false, "wide")

	got := buf.String()
	if !strings.Contains(got, "EXTRA") {
		t.Errorf("expected EXTRA column in wide output: %q", got)
	}
	if !strings.Contains(got, "80/TCP") {
		t.Errorf("expected port info in wide output: %q", got)
	}
}

func TestPrintNamedResults_JSONFormat(t *testing.T) {
	streams, buf := makeStreams()
	named := []resolver.NamedResults{
		{
			Namespace: "default",
			Name:      "my-svc",
			Results:   []resolver.Result{{Name: "my-svc.default.svc.cluster.local", Kind: "cluster-fqdn"}},
		},
	}
	output.PrintNamedResults(streams, named, true, false, "json")

	var rows []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 JSON row, got %d", len(rows))
	}
	if rows[0]["name"] != "my-svc" {
		t.Errorf("expected name=my-svc in JSON, got %v", rows[0]["name"])
	}
	if rows[0]["kind"] != "cluster-fqdn" {
		t.Errorf("expected kind=cluster-fqdn in JSON, got %v", rows[0]["kind"])
	}
}

func TestPrintNamedResults_Empty(t *testing.T) {
	streams, buf := makeStreams()
	output.PrintNamedResults(streams, nil, false, false, "table")

	got := buf.String()
	// Should still print a header.
	if !strings.Contains(got, "NAME") {
		t.Errorf("expected header even for empty list: %q", got)
	}
}

func TestPrintNamedResults_EmptyJSON(t *testing.T) {
	streams, buf := makeStreams()
	output.PrintNamedResults(streams, nil, false, false, "json")

	var rows []interface{}
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("empty JSON output is invalid: %v\nraw: %s", err, buf.String())
	}
	if len(rows) != 0 {
		t.Errorf("expected empty JSON array, got %d items", len(rows))
	}
}
