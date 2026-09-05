package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	"github.com/google/uuid"
)

type tokenRepositoryStub struct {
	clients.Repository
	client   clients.Client
	authErr  error
	audits   []clients.TokenExchangeAudit
	auditErr error
}

func (r *tokenRepositoryStub) Authenticate(context.Context, clients.Credential) (clients.Client, error) {
	return r.client, r.authErr
}

func (r *tokenRepositoryStub) AuditTokenExchange(_ context.Context, audit clients.TokenExchangeAudit) error {
	r.audits = append(r.audits, audit)
	return r.auditErr
}

type tokenIssuerStub struct {
	token     string
	expiresAt time.Time
	calls     int
}

func (i *tokenIssuerStub) Issue(uuid.UUID, uuid.UUID, uuid.UUID, string, int32) (string, time.Time, error) {
	i.calls++
	return i.token, i.expiresAt, nil
}

func TestTokenExchangeAuditsWithoutChangingCredentialFailure(t *testing.T) {
	clientID, secret := "ak_test_client", "aks_test_01234567890123456789012345678901"
	tenantID, internalID := uuid.New(), uuid.New()
	metadata := clients.TokenMetadata{RequestID: "request-1", IPAddress: "203.0.113.7", UserAgent: "akone/test"}
	wantHash := sha256.Sum256([]byte(clientID))
	expiresAt := time.Now().UTC().Add(time.Minute)

	t.Run("success is audited before returning token", func(t *testing.T) {
		repository := &tokenRepositoryStub{client: clients.Client{ID: internalID, TenantID: tenantID}}
		issuer := &tokenIssuerStub{token: "signed-token", expiresAt: expiresAt}
		accessToken, gotExpiry, err := NewService(nil, repository, issuer).Token(t.Context(), clientID, secret, metadata)
		if err != nil || accessToken != "signed-token" || !gotExpiry.Equal(expiresAt) {
			t.Fatalf("token=%q expires=%v err=%v", accessToken, gotExpiry, err)
		}
		if len(repository.audits) != 1 {
			t.Fatalf("audits=%#v", repository.audits)
		}
		audit := repository.audits[0]
		if audit.TenantID == nil || *audit.TenantID != tenantID || audit.Result != "success" || audit.FailureReason != "" ||
			audit.RequestID != metadata.RequestID || !bytes.Equal(audit.IdentifierHash, wantHash[:]) {
			t.Fatalf("audit=%#v", audit)
		}
	})

	t.Run("failed audit stays best effort", func(t *testing.T) {
		auditErr := errors.New("audit unavailable")
		repository := &tokenRepositoryStub{authErr: clients.ErrCredential, auditErr: auditErr}
		issuer := &tokenIssuerStub{}
		accessToken, _, err := NewService(nil, repository, issuer).Token(t.Context(), clientID, secret, metadata)
		if !errors.Is(err, clients.ErrCredential) || accessToken != "" || issuer.calls != 0 {
			t.Fatalf("token=%q issuer_calls=%d err=%v", accessToken, issuer.calls, err)
		}
		if len(repository.audits) != 1 || repository.audits[0].Result != "failure" ||
			repository.audits[0].FailureReason != "invalid_credentials" || !bytes.Equal(repository.audits[0].IdentifierHash, wantHash[:]) {
			t.Fatalf("audits=%#v", repository.audits)
		}
	})

	t.Run("successful exchange fails closed when audit is unavailable", func(t *testing.T) {
		auditErr := errors.New("audit unavailable")
		repository := &tokenRepositoryStub{client: clients.Client{ID: internalID, TenantID: tenantID}, auditErr: auditErr}
		accessToken, _, err := NewService(nil, repository, &tokenIssuerStub{token: "signed-token"}).Token(t.Context(), clientID, secret, metadata)
		if !errors.Is(err, auditErr) || accessToken != "" {
			t.Fatalf("token=%q err=%v", accessToken, err)
		}
	})
}
