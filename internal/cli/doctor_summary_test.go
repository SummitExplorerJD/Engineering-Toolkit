package cli

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/doctor"
)

func TestWriteDoctorSummaryRendersCanonicalResult(t *testing.T) {
	result := doctor.Result{
		Findings: []doctor.Finding{
			{Code: "core.runtime", Status: doctor.StatusPass, Summary: "Runtime is available.", Path: "secret/path", Evidence: []doctor.Evidence{{Name: "token", Value: "secret-value"}}, Remediation: "must not render"},
			{Code: "workspace.context", Status: doctor.StatusWarning, Summary: "Context is stale.", Remediation: "Run context rebuild."},
			{Code: "workspace.config", Status: doctor.StatusError, Summary: "Configuration is invalid.", Remediation: "Restore configuration."},
			{Code: "integration.ollama", Status: doctor.StatusUnavailable, Summary: "Ollama is unavailable.", Remediation: "Install it independently."},
		},
		Summary: doctor.Summary{Pass: 1, Warning: 1, Error: 1, Unavailable: 1, RequiredErrors: 1, Healthy: false},
	}

	var output bytes.Buffer
	if err := writeDoctorSummary(&output, result); err != nil {
		t.Fatal(err)
	}

	want := "Engineering Toolkit doctor\n\n" +
		"[OK]    core.runtime - Runtime is available.\n" +
		"[WARN]  workspace.context - Context is stale.\n" +
		"        Next: Run context rebuild.\n" +
		"[ERROR] workspace.config - Configuration is invalid.\n" +
		"        Next: Restore configuration.\n" +
		"[N/A]   integration.ollama - Ollama is unavailable.\n" +
		"        Next: Install it independently.\n\n" +
		"Overall: NEEDS ATTENTION\n" +
		"Passed: 1  Warnings: 1  Errors: 1  Unavailable: 1\n"
	if output.String() != want {
		t.Fatalf("summary output:\n%s\nwant:\n%s", output.String(), want)
	}
	for _, excluded := range []string{"secret/path", "token", "secret-value", "must not render"} {
		if strings.Contains(output.String(), excluded) {
			t.Fatalf("summary exposed %q: %s", excluded, output.String())
		}
	}
}

func TestWriteDoctorSummaryHealthyFooter(t *testing.T) {
	result := doctor.Result{
		Findings: []doctor.Finding{
			{Code: "core.runtime", Status: doctor.StatusPass, Summary: "Ready."},
			{Code: "integration.optional", Status: doctor.StatusUnavailable, Summary: "Optional integration is unavailable.", Remediation: "Configure it only when needed."},
		},
		Summary: doctor.Summary{Pass: 1, Unavailable: 1, Healthy: true},
	}
	var output bytes.Buffer
	if err := writeDoctorSummary(&output, result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "[N/A]   integration.optional - Optional integration is unavailable.\n") ||
		!strings.Contains(output.String(), "Overall: OK\nPassed: 1  Warnings: 0  Errors: 0  Unavailable: 1\n") {
		t.Fatalf("healthy summary = %q", output.String())
	}
}

func TestDoctorSummaryFieldNormalizesAndTruncatesUTF8(t *testing.T) {
	value := "line\r\ncolumn\t" + strings.Repeat("é", 300)
	got := doctorSummaryField(value)
	if strings.ContainsAny(got, "\r\n\t") {
		t.Fatalf("field contains a line control: %q", got)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("field is not valid UTF-8: %q", got)
	}
	if len(got) > doctorSummaryFieldLimit || !strings.HasSuffix(got, "...") {
		t.Fatalf("field length/suffix = %d, %q", len(got), got)
	}
}

func TestWriteDoctorSummaryRefusesUnknownStatusWithoutOutput(t *testing.T) {
	result := doctor.Result{Findings: []doctor.Finding{{Code: "future", Status: "future", Summary: "Unknown."}}}
	var output bytes.Buffer
	err := writeDoctorSummary(&output, result)
	if err == nil || !strings.Contains(err.Error(), "unsupported finding status") {
		t.Fatalf("error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("partial output = %q", output.String())
	}
}

func TestWriteDoctorSummaryPropagatesFinalWriteError(t *testing.T) {
	wantErr := errors.New("write failed")
	result := doctor.Result{Summary: doctor.Summary{Healthy: true}}
	err := writeDoctorSummary(errorWriter{err: wantErr}, result)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
}

type errorWriter struct {
	err error
}

func (writer errorWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

var _ io.Writer = errorWriter{}
