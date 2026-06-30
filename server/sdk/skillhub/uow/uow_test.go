package uow_test

import (
	"context"
	"errors"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/uow"
)

func TestNoopTransactor_Success(t *testing.T) {
	tx := uow.NoopTransactor{}
	called := false
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !called {
		t.Errorf("expected fn to be called")
	}
}

func TestNoopTransactor_Error(t *testing.T) {
	tx := uow.NoopTransactor{}
	want := errors.New("boom")
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Errorf("expected %v, got %v", want, err)
	}
}
