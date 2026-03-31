package cli

import (
	"testing"

	"github.com/2bit-software/zombiekit/internal/config"
)

func TestGUIService_Name(t *testing.T) {
	svc := NewGUIService(config.GUIConfig{})
	if svc.Name() != "gui" {
		t.Errorf("expected name 'gui', got '%s'", svc.Name())
	}
}

func TestRecallService_Name(t *testing.T) {
	svc := NewRecallService(config.RecallConfig{})
	if svc.Name() != "recall" {
		t.Errorf("expected name 'recall', got '%s'", svc.Name())
	}
}

func TestGUIService_ImplementsInterface(t *testing.T) {
	var _ Service = (*GUIService)(nil)
}

func TestRecallService_ImplementsInterface(t *testing.T) {
	var _ Service = (*RecallService)(nil)
}
