package query

import (
	"context"
	"strings"
	"testing"
)

type captureStep struct{ received string }

func (s *captureStep) Name() string   { return "capture" }
func (s *captureStep) Type() StepType { return StepRewrite }
func (s *captureStep) Enabled() bool  { return true }
func (s *captureStep) Process(_ context.Context, value string) (*Result, error) {
	s.received = value
	return &Result{Query: value}, nil
}

func TestLayerUsesSanitizedEphemeralModelInput(t *testing.T) {
	step := &captureStep{}
	layer := NewLayer(nil, step)
	const secret = "query-secret"
	_, err := layer.Process(context.Background(), "Authorization: Bearer "+secret)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(step.received, secret) {
		t.Fatalf("query step received secret: %q", step.received)
	}
}

func TestNormalizeIntentRejectsModelFreeText(t *testing.T) {
	if got := normalizeIntent(IntentType("user supplied pii")); got != IntentUnknown {
		t.Fatalf("normalizeIntent() = %q, want %q", got, IntentUnknown)
	}
	if got := normalizeIntent(IntentSearch); got != IntentSearch {
		t.Fatalf("normalizeIntent() changed valid intent to %q", got)
	}
}
