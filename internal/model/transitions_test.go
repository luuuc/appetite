package model_test

import (
	"errors"
	"testing"

	"github.com/luuuc/appetite/internal/model"
)

func TestValidatePitchTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    model.PitchStatus
		to      model.PitchStatus
		wantErr bool
	}{
		{"shaping_to_shaped", model.PitchStatusShaping, model.PitchStatusShaped, false},
		{"shaped_to_bet", model.PitchStatusShaped, model.PitchStatusBet, false},
		{"shaped_to_passed", model.PitchStatusShaped, model.PitchStatusPassed, false},
		{"bet_to_building", model.PitchStatusBet, model.PitchStatusBuilding, false},
		{"building_to_shipped", model.PitchStatusBuilding, model.PitchStatusShipped, false},

		// Illegal — skipping states
		{"shaping_to_bet", model.PitchStatusShaping, model.PitchStatusBet, true},
		{"shaped_to_building", model.PitchStatusShaped, model.PitchStatusBuilding, true},
		{"shaped_to_shipped", model.PitchStatusShaped, model.PitchStatusShipped, true},
		{"bet_to_shipped", model.PitchStatusBet, model.PitchStatusShipped, true},

		// Illegal — backwards
		{"shaped_to_shaping", model.PitchStatusShaped, model.PitchStatusShaping, true},
		{"building_to_bet", model.PitchStatusBuilding, model.PitchStatusBet, true},

		// Illegal — same state
		{"shaped_to_shaped", model.PitchStatusShaped, model.PitchStatusShaped, true},

		// Illegal — from terminal
		{"shipped_to_anything", model.PitchStatusShipped, model.PitchStatusBuilding, true},
		{"passed_to_anything", model.PitchStatusPassed, model.PitchStatusBet, true},

		// Illegal — unknown
		{"unknown_from", model.PitchStatus("garbage"), model.PitchStatusBet, true},
		{"unknown_to", model.PitchStatusShaped, model.PitchStatus("garbage"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := model.ValidatePitchTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePitchTransition(%q, %q) err = %v, wantErr = %v", tt.from, tt.to, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, model.ErrInvalidTransition) {
				t.Errorf("error does not wrap ErrInvalidTransition: %v", err)
			}
		})
	}
}

func TestValidateCycleTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    model.CycleStatus
		to      model.CycleStatus
		wantErr bool
	}{
		{"building_to_shipping", model.CycleStatusBuilding, model.CycleStatusShipping, false},
		{"shipping_to_cooldown", model.CycleStatusShipping, model.CycleStatusCooldown, false},
		{"cooldown_to_closed", model.CycleStatusCooldown, model.CycleStatusClosed, false},

		// Illegal — skipping
		{"building_to_cooldown", model.CycleStatusBuilding, model.CycleStatusCooldown, true},
		{"building_to_closed", model.CycleStatusBuilding, model.CycleStatusClosed, true},

		// Illegal — backwards
		{"cooldown_to_shipping", model.CycleStatusCooldown, model.CycleStatusShipping, true},

		// Terminal
		{"closed_to_anything", model.CycleStatusClosed, model.CycleStatusBuilding, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := model.ValidateCycleTransition(tt.from, tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCycleTransition(%q, %q) err = %v, wantErr = %v", tt.from, tt.to, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, model.ErrInvalidTransition) {
				t.Errorf("error does not wrap ErrInvalidTransition: %v", err)
			}
		})
	}
}

func TestValidateCooldownTransition(t *testing.T) {
	if err := model.ValidateCooldownTransition(model.CooldownActive, model.CooldownClosed); err != nil {
		t.Errorf("active → closed: unexpected error %v", err)
	}
	if err := model.ValidateCooldownTransition(model.CooldownClosed, model.CooldownActive); err == nil {
		t.Error("closed → active: expected ErrInvalidTransition, got nil")
	}
	if err := model.ValidateCooldownTransition(model.CooldownActive, model.CooldownActive); err == nil {
		t.Error("active → active (same state): expected ErrInvalidTransition, got nil")
	}
}
