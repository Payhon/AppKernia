package application

import (
	"context"
	"errors"
	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
	iam "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/google/uuid"
	"strings"
	"testing"
)

func TestFeedbackInputBoundaries(t *testing.T) {
	valid := f.Input{Description: "An issue", Platform: "ios", AppVersion: "1.0.0", FileIDs: []uuid.UUID{}}
	cases := []struct {
		name   string
		change func(*f.Input)
		want   bool
	}{
		{"valid", func(*f.Input) {}, true}, {"empty", func(x *f.Input) { x.Description = "" }, false}, {"long unicode", func(x *f.Input) { x.Description = strings.Repeat("问", 2001) }, false}, {"contact limit", func(x *f.Input) { x.Contact = strings.Repeat("a", 201) }, false}, {"four files", func(x *f.Input) { x.FileIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()} }, false}, {"duplicate files", func(x *f.Input) { id := uuid.New(); x.FileIDs = []uuid.UUID{id, id} }, false}, {"nil file", func(x *f.Input) { x.FileIDs = []uuid.UUID{uuid.Nil} }, false}, {"unsupported platform", func(x *f.Input) { x.Platform = "web" }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := valid
			tc.change(&x)
			if (ValidateInput(x) == nil) != tc.want {
				t.Fatal("unexpected validation result")
			}
		})
	}
}

type scopeAuth struct {
	a        iam.AuthenticatedContext
	audience string
}

func (a *scopeAuth) Authenticate(_ context.Context, _ string, audience string) (iam.AuthenticatedContext, error) {
	a.audience = audience
	return a.a, nil
}

type scopeRepo struct {
	f.Repository
	called bool
}

func (r *scopeRepo) CheckScope(context.Context, f.Scope) error { r.called = true; return nil }
func TestAuthorizationDoesNotTrustAppOrPermissions(t *testing.T) {
	id := uuid.New()
	a := &scopeAuth{a: iam.AuthenticatedContext{AppID: &id}}
	r := &scopeRepo{}
	s := NewService(a, r, nil)
	if _, e := s.Scope(context.Background(), "token", uuid.New(), "", "request"); !errors.Is(e, f.ErrForbidden) || r.called {
		t.Fatal("cross App mobile scope accepted")
	}
	if _, e := s.Scope(context.Background(), "token", id, "app.feedback.reply", "request"); !errors.Is(e, f.ErrForbidden) || r.called || a.audience != "ak-admin" {
		t.Fatal("admin permission bypass")
	}
	if _, e := s.Scope(context.Background(), "token", id, "", "request"); e != nil || !r.called || a.audience != "ak-mobile" {
		t.Fatal("mobile authentication scope lost")
	}
}
