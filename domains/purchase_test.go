package domains

import (
	"context"
	"testing"
)

type baseRegistrar struct{}

func (b *baseRegistrar) ListDomains(ctx context.Context) ([]ManagedDomain, error) {
	return nil, nil
}

func (b *baseRegistrar) GetDomain(ctx context.Context, name string) (*ManagedDomain, error) {
	return nil, nil
}

func (b *baseRegistrar) UpdateAutoRenewal(ctx context.Context, name string, enabled bool) error {
	return nil
}

func (b *baseRegistrar) GetRenewalInfo(ctx context.Context, name string) (*DomainCost, error) {
	return nil, nil
}

func (b *baseRegistrar) GetNameservers(ctx context.Context, name string) ([]string, error) {
	return nil, nil
}

func (b *baseRegistrar) UpdateNameservers(ctx context.Context, name string, nameservers []string) error {
	return nil
}

// purchaserRegistrar is a Registrar that also supports purchasing.
type purchaserRegistrar struct{ baseRegistrar }

func (p *purchaserRegistrar) Check(ctx context.Context, names []string) ([]Availability, error) {
	return []Availability{{Name: names[0], Registrable: true}}, nil
}

func (p *purchaserRegistrar) Register(ctx context.Context, name string, contact *RegistrantContact) (*RegistrationResult, error) {
	return &RegistrationResult{DomainName: name, State: RegistrationStateSucceeded, Completed: true}, nil
}

func (p *purchaserRegistrar) RegistrationStatus(ctx context.Context, name string) (*RegistrationResult, error) {
	return &RegistrationResult{DomainName: name, State: RegistrationStateSucceeded, Completed: true}, nil
}

func TestAsPurchaser(t *testing.T) {
	t.Run("registrar with purchaser capability", func(t *testing.T) {
		var r Registrar = &purchaserRegistrar{}

		p, ok := AsPurchaser(r)
		if !ok {
			t.Fatal("expected purchaser capability to be detected")
		}
		if p == nil {
			t.Fatal("expected non-nil purchaser")
		}

		avail, err := p.Check(context.Background(), []string{"example.dev"})
		if err != nil {
			t.Fatalf("Check returned error: %v", err)
		}
		if len(avail) != 1 || avail[0].Name != "example.dev" || !avail[0].Registrable {
			t.Errorf("unexpected availability result: %+v", avail)
		}
	})

	t.Run("registrar without purchaser capability", func(t *testing.T) {
		var r Registrar = &baseRegistrar{}

		p, ok := AsPurchaser(r)
		if ok {
			t.Error("did not expect purchaser capability")
		}
		if p != nil {
			t.Error("expected nil purchaser")
		}
	})
}

func TestRegistrationResultStates(t *testing.T) {
	r := &RegistrationResult{DomainName: "example.dev", State: RegistrationStateInProgress}
	if r.IsTerminal() {
		t.Error("in_progress should not be terminal")
	}

	r.State = RegistrationStateSucceeded
	if !r.IsTerminal() {
		t.Error("succeeded should be terminal")
	}

	r.State = RegistrationStateFailed
	if !r.IsTerminal() {
		t.Error("failed should be terminal")
	}

	r.State = RegistrationStateActionRequired
	if !r.IsTerminal() {
		t.Error("action_required should be terminal")
	}

	r.State = RegistrationStateBlocked
	if !r.IsTerminal() {
		t.Error("blocked should be terminal")
	}
}
