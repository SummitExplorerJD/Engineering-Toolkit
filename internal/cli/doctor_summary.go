package cli

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/SummitExplorerJD/Engineering-Toolkit/internal/doctor"
)

const doctorSummaryFieldLimit = 512

const (
	doctorSummaryTitle            = "Engineering Toolkit doctor"
	doctorSummaryNext             = "Next:"
	doctorSummaryOverallOK        = "Overall: OK"
	doctorSummaryOverallAttention = "Overall: NEEDS ATTENTION"
	doctorSummaryCountsFormat     = "Passed: %d  Warnings: %d  Errors: %d  Unavailable: %d"
)

func writeDoctorSummary(writer io.Writer, result doctor.Result) error {
	var output strings.Builder
	output.WriteString(doctorSummaryTitle)
	output.WriteString("\n\n")

	for _, finding := range result.Findings {
		label, err := doctorSummaryStatusLabel(finding.Status)
		if err != nil {
			return err
		}
		code := doctorSummaryField(finding.Code)
		summary := doctorSummaryField(finding.Summary)
		fmt.Fprintf(&output, "%-7s %s - %s\n", label, code, summary)
		if finding.Status != doctor.StatusPass && strings.TrimSpace(finding.Remediation) != "" {
			fmt.Fprintf(&output, "        %s %s\n", doctorSummaryNext, doctorSummaryField(finding.Remediation))
		}
	}

	output.WriteString("\n")
	if result.Summary.Healthy {
		output.WriteString(doctorSummaryOverallOK)
	} else {
		output.WriteString(doctorSummaryOverallAttention)
	}
	output.WriteString("\n")
	fmt.Fprintf(
		&output,
		doctorSummaryCountsFormat+"\n",
		result.Summary.Pass,
		result.Summary.Warning,
		result.Summary.Error,
		result.Summary.Unavailable,
	)

	rendered := output.String()
	written, err := io.WriteString(writer, rendered)
	if err != nil {
		return fmt.Errorf("write doctor summary: %w", err)
	}
	if written != len(rendered) {
		return fmt.Errorf("write doctor summary: %w", io.ErrShortWrite)
	}
	return nil
}

func doctorSummaryStatusLabel(status string) (string, error) {
	switch status {
	case doctor.StatusPass:
		return "[OK]", nil
	case doctor.StatusWarning:
		return "[WARN]", nil
	case doctor.StatusError:
		return "[ERROR]", nil
	case doctor.StatusUnavailable:
		return "[N/A]", nil
	default:
		return "", fmt.Errorf("render doctor summary: unsupported finding status %q", status)
	}
}

func doctorSummaryField(value string) string {
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if len(value) <= doctorSummaryFieldLimit {
		return value
	}

	prefixLimit := doctorSummaryFieldLimit - len("...")
	for prefixLimit > 0 && !utf8.ValidString(value[:prefixLimit]) {
		prefixLimit--
	}
	return value[:prefixLimit] + "..."
}
