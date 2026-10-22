package providersync

import (
	"errors"
	"strings"
)

const adapterEvidenceDate = "2026-10-22"

func adapterContract(provider string) (AdapterReference, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "codex":
		return AdapterReference{Provider: provider, ContractVersion: "v1alpha1", EvidenceDate: adapterEvidenceDate, TargetRoot: ".agents/skills"}, nil
	case "claude":
		return AdapterReference{Provider: provider, ContractVersion: "v1alpha1", EvidenceDate: adapterEvidenceDate, TargetRoot: ".claude/skills"}, nil
	case "opencode":
		return AdapterReference{Provider: provider, ContractVersion: "v1alpha1", EvidenceDate: adapterEvidenceDate, TargetRoot: ".opencode/skills"}, nil
	default:
		return AdapterReference{}, errors.New("unsupported provider: " + provider)
	}
}
