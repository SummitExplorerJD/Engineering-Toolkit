package localintelligence

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type contextStub struct {
	excerpts []Excerpt
	err      error
}

func (s contextStub) Excerpts(context.Context, string) ([]Excerpt, error) { return s.excerpts, s.err }

type adviserStub struct {
	response ModelResponse
	err      error
}

func (s adviserStub) Rank(context.Context, string, Request) (ModelResponse, ModelEvidence, error) {
	return s.response, ModelEvidence{OllamaVersion: "0.40.4", ModelTag: "granite4:3b", ModelDigest: "sha256:" + strings.Repeat("a", 64), ResponseDigest: "sha256:" + strings.Repeat("b", 64)}, s.err
}

func candidates() []Candidate {
	return []Candidate{{ID: "a@1.0.0", DeterministicRank: 1, Evidence: []string{"compatible"}}, {ID: "b@1.0.0", DeterministicRank: 2, Evidence: []string{"compatible"}}}
}

func TestAdviseUsesValidModelRanking(t *testing.T) {
	result, err := Advise(context.Background(), Options{
		Root: "/workspace", Goal: "choose", Enabled: true, Model: "granite4:3b", Candidates: candidates(),
		Context: contextStub{excerpts: []Excerpt{{Path: "README.md", SHA256: "sha256:" + strings.Repeat("c", 64), Text: "bounded"}}},
		Adviser: adviserStub{response: ModelResponse{Rankings: []Ranking{{CandidateID: "b@1.0.0", Score: 90, Reason: "best"}, {CandidateID: "a@1.0.0", Score: 70, Reason: "other"}}, Warnings: []string{}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != "ollama" || result.AdvisoryOrder[0] != "b@1.0.0" || result.Model == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestAdviseFallsBack(t *testing.T) {
	for _, options := range []Options{
		{Root: "/workspace", Goal: "choose", Candidates: candidates()},
		{Root: "/workspace", Goal: "choose", Enabled: true, Model: "qwen3.5:0.8b", Candidates: candidates(), Context: contextStub{err: errors.New("stale")}, Adviser: adviserStub{}},
		{Root: "/workspace", Goal: "choose", Enabled: true, Model: "qwen3.5:0.8b", Candidates: candidates(), Context: contextStub{}, Adviser: adviserStub{response: ModelResponse{Rankings: []Ranking{{CandidateID: "a@1.0.0"}}}}},
	} {
		result, err := Advise(context.Background(), options)
		if err != nil {
			t.Fatal(err)
		}
		if result.Mode != "deterministic" || strings.Join(result.AdvisoryOrder, ",") != "a@1.0.0,b@1.0.0" || len(result.Warnings) == 0 {
			t.Fatalf("unexpected fallback: %#v", result)
		}
	}
}

func TestFilterExcerpts(t *testing.T) {
	input := []Excerpt{
		{Path: ".agents/context/private.md", SHA256: "sha256:" + strings.Repeat("a", 64), Text: "excluded"},
		{Path: "docs/secret.md", SHA256: "sha256:" + strings.Repeat("b", 64), Text: "token: value"},
		{Path: "docs/key.md", SHA256: "sha256:" + strings.Repeat("d", 64), Text: "-----BEGIN EC PRIVATE KEY-----"},
		{Path: "README.md", SHA256: "sha256:" + strings.Repeat("c", 64), Text: strings.Repeat("x", MaxExcerptBytes+50)},
	}
	got := FilterExcerpts(input)
	if len(got) != 1 || got[0].Path != "README.md" || len([]byte(got[0].Text)) != MaxExcerptBytes {
		t.Fatalf("unexpected excerpts: %#v", got)
	}
}
