package tools

import (
	"errors"
	"testing"
)

type mockBatchInteractor struct {
	mockInteractor
	batchCalled bool
	batchResult []bool
}

func (m *mockBatchInteractor) RequestBatchApproval(descs []string) ([]bool, error) {
	m.batchCalled = true
	if m.batchResult != nil {
		return m.batchResult, nil
	}
	result := make([]bool, len(descs))
	for i := range result {
		result[i] = m.approveAll
	}
	return result, nil
}

func TestRequestBatchOrSequential_BatchPath(t *testing.T) {
	inter := &mockBatchInteractor{
		mockInteractor: mockInteractor{approveAll: true},
		batchResult:    []bool{true, false, true},
	}
	results, err := RequestBatchOrSequential(inter, []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inter.batchCalled {
		t.Error("expected batch path to be used")
	}
	if len(results) != 3 || !results[0] || results[1] || !results[2] {
		t.Errorf("results = %v, want [true, false, true]", results)
	}
}

func TestRequestBatchOrSequential_SequentialFallback(t *testing.T) {
	inter := &mockInteractor{approveAll: true}
	results, err := RequestBatchOrSequential(inter, []string{"a", "b"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 || !results[0] || !results[1] {
		t.Errorf("results = %v, want [true, true]", results)
	}
	if len(inter.approvals) != 2 {
		t.Errorf("expected 2 sequential approvals, got %d", len(inter.approvals))
	}
}

func TestRequestBatchOrSequential_SequentialError(t *testing.T) {
	inter := &mockInteractor{approveErr: errors.New("disconnected")}
	_, err := RequestBatchOrSequential(inter, []string{"a"})
	if err == nil {
		t.Fatal("expected error from sequential fallback")
	}
}

func TestRequestBatchOrSequential_Empty(t *testing.T) {
	inter := &mockInteractor{}
	results, err := RequestBatchOrSequential(inter, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil for empty input, got %v", results)
	}
}
