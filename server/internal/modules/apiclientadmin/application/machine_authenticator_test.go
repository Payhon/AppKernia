package application

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	clients "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/domain"
	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

type machineRepositoryStub struct {
	client   clients.Client
	audits   []clients.AgentAudit
	auditErr error
}

func (stub *machineRepositoryStub) Get(context.Context, uuid.UUID, uuid.UUID) (clients.Client, error) {
	return stub.client, nil
}

func (stub *machineRepositoryStub) AuditAgentAuthentication(_ context.Context, audit clients.AgentAudit) error {
	stub.audits = append(stub.audits, audit)
	return stub.auditErr
}

type verifierStub struct{ claims iamapp.AccessClaims }

func (stub verifierStub) Verify(string, string) (iamapp.AccessClaims, error) { return stub.claims, nil }

type delegatedResolverStub struct {
	value iamdomain.AuthenticatedContext
}

func (stub delegatedResolverStub) ResolveDelegatedContext(context.Context, uuid.UUID, uuid.UUID) (iamdomain.AuthenticatedContext, error) {
	return stub.value, nil
}

type userAuthenticatorStub struct {
	value iamdomain.AuthenticatedContext
	err   error
}

func (stub userAuthenticatorStub) Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error) {
	return stub.value, stub.err
}

func TestAllowsIP(t *testing.T) {
	tests := []struct {
		name  string
		cidrs []string
		ip    string
		want  bool
	}{
		{name: "empty allowlist permits", ip: "203.0.113.4", want: true},
		{name: "ipv4 included", cidrs: []string{"10.20.0.0/16"}, ip: "10.20.4.9", want: true},
		{name: "ipv4 excluded", cidrs: []string{"10.20.0.0/16"}, ip: "10.21.4.9"},
		{name: "ipv6 included", cidrs: []string{"2001:db8::/32"}, ip: "2001:db8::8", want: true},
		{name: "malformed remote rejected", cidrs: []string{"10.20.0.0/16"}, ip: "not-an-ip"},
		{name: "malformed stored prefix ignored", cidrs: []string{"broken"}, ip: "10.20.4.9"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allowsIP(tt.cidrs, tt.ip); got != tt.want {
				t.Fatalf("allowsIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDelegatedAuthenticationIntersectsCurrentPermissionsAndAuditsReads(t *testing.T) {
	tenantID, clientID, userID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := &machineRepositoryStub{client: clients.Client{
		ID: clientID, TenantID: tenantID, Status: "active", BoundUserID: &userID,
		Permissions: []string{"app.content.read", "app.content.publish"}, AppIDs: []uuid.UUID{appID},
	}}
	claims := iamapp.AccessClaims{SessionID: clientID, TenantID: tenantID, RegisteredClaims: jwt.RegisteredClaims{Subject: clientID.String()}}
	authenticator := NewMachineAuthenticator(repository, verifierStub{claims: claims}, delegatedResolverStub{value: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{
		User: iamdomain.User{ID: userID}, Tenant: iamdomain.Tenant{ID: tenantID},
		Permissions: []string{"app.content.read", "iam.user.read"},
	}}})

	got, err := authenticator.AuthenticateDelegated(context.Background(), "machine-token", AgentCall{
		OperationID: "listAdminAppContentArticles", Method: "GET", Path: "/admin-api/v1/apps/" + appID.String() + "/content/articles",
		RequestID: "agent-read", IPAddress: "203.0.113.9", UserAgent: "akone/test", AppID: &appID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Permissions) != 1 || got.Permissions[0] != "app.content.read" {
		t.Fatalf("effective permissions=%v", got.Permissions)
	}
	if got.APIClientID == nil || *got.APIClientID != clientID || got.User.ID != userID || got.SessionID != uuid.Nil {
		t.Fatalf("delegated context=%#v", got)
	}
	if len(repository.audits) != 1 || repository.audits[0].ClientID != clientID || repository.audits[0].UserID != userID {
		t.Fatalf("audits=%#v", repository.audits)
	}

	otherApp := uuid.New()
	_, err = authenticator.AuthenticateDelegated(context.Background(), "machine-token", AgentCall{
		OperationID: "listAdminAppContentArticles", Method: "GET", RequestID: "denied", AppID: &otherApp,
	})
	if !errors.Is(err, clients.ErrForbidden) {
		t.Fatalf("cross-app error=%v", err)
	}
	if len(repository.audits) != 1 {
		t.Fatalf("denied call must not be audited as success: %#v", repository.audits)
	}
}

func TestDelegatedAuthenticationFailsClosedWithoutBindingOrOperationMarker(t *testing.T) {
	tenantID, clientID := uuid.New(), uuid.New()
	repository := &machineRepositoryStub{client: clients.Client{ID: clientID, TenantID: tenantID, Status: "active"}}
	claims := iamapp.AccessClaims{SessionID: clientID, TenantID: tenantID, RegisteredClaims: jwt.RegisteredClaims{Subject: clientID.String()}}
	authenticator := NewMachineAuthenticator(repository, verifierStub{claims: claims}, delegatedResolverStub{})

	if _, err := authenticator.AuthenticateDelegated(context.Background(), "machine-token", AgentCall{OperationID: "getAdminDashboardSummary"}); !errors.Is(err, clients.ErrCredential) {
		t.Fatalf("unbound client error=%v", err)
	}
	if _, err := authenticator.AuthenticateDelegated(context.Background(), "machine-token", AgentCall{OperationID: "listAdminUsers"}); !errors.Is(err, clients.ErrCredential) {
		t.Fatalf("unlisted operation error=%v", err)
	}
	if len(repository.audits) != 0 {
		t.Fatalf("failed delegation must not be audited as authorized: %#v", repository.audits)
	}
}

func TestDelegatedAuthenticationFailsClosedWhenAuthenticationAuditFails(t *testing.T) {
	tenantID, clientID, userID := uuid.New(), uuid.New(), uuid.New()
	auditErr := errors.New("audit unavailable")
	repository := &machineRepositoryStub{client: clients.Client{
		ID: clientID, TenantID: tenantID, Status: "active", BoundUserID: &userID,
	}, auditErr: auditErr}
	claims := iamapp.AccessClaims{SessionID: clientID, TenantID: tenantID, RegisteredClaims: jwt.RegisteredClaims{Subject: clientID.String()}}
	authenticator := NewMachineAuthenticator(repository, verifierStub{claims: claims}, delegatedResolverStub{value: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{
		User: iamdomain.User{ID: userID}, Tenant: iamdomain.Tenant{ID: tenantID},
	}}})

	_, err := authenticator.AuthenticateDelegated(context.Background(), "machine-token", AgentCall{
		OperationID: "getAdminDashboardSummary", Method: "GET", RequestID: "audit-failure",
	})
	if !errors.Is(err, auditErr) {
		t.Fatalf("audit failure=%v", err)
	}
}

func TestDelegatedWriteRecordsAuthenticationWithoutClaimingBusinessOutcome(t *testing.T) {
	tenantID, clientID, userID, appID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := &machineRepositoryStub{client: clients.Client{
		ID: clientID, TenantID: tenantID, Status: "active", BoundUserID: &userID, AppIDs: []uuid.UUID{appID},
	}}
	claims := iamapp.AccessClaims{SessionID: clientID, TenantID: tenantID, RegisteredClaims: jwt.RegisteredClaims{Subject: clientID.String()}}
	authenticator := NewMachineAuthenticator(repository, verifierStub{claims: claims}, delegatedResolverStub{value: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{
		User: iamdomain.User{ID: userID}, Tenant: iamdomain.Tenant{ID: tenantID},
	}}})

	if _, err := authenticator.AuthenticateDelegated(context.Background(), "machine-token", AgentCall{
		OperationID: "createAdminAppContentArticle", Method: "POST", RequestID: "write-authentication", AppID: &appID,
	}); err != nil {
		t.Fatal(err)
	}
	if len(repository.audits) != 1 || repository.audits[0].Method != "POST" || repository.audits[0].Operation != "createAdminAppContentArticle" {
		t.Fatalf("authentication audits=%#v", repository.audits)
	}
}

func TestDelegatedAuthenticatorRequiresRouteMarkerAndAdminAudience(t *testing.T) {
	tenantID, clientID, userID := uuid.New(), uuid.New(), uuid.New()
	repository := &machineRepositoryStub{client: clients.Client{
		ID: clientID, TenantID: tenantID, Status: "active", BoundUserID: &userID,
	}}
	claims := iamapp.AccessClaims{SessionID: clientID, TenantID: tenantID, RegisteredClaims: jwt.RegisteredClaims{Subject: clientID.String()}}
	machine := NewMachineAuthenticator(repository, verifierStub{claims: claims}, delegatedResolverStub{value: iamdomain.AuthenticatedContext{AuthContext: iamdomain.AuthContext{
		User: iamdomain.User{ID: userID}, Tenant: iamdomain.Tenant{ID: tenantID},
	}}})
	authenticator := NewDelegatedAuthenticator(userAuthenticatorStub{err: iamapp.ErrInvalidAccessToken}, machine)

	if _, err := authenticator.Authenticate(context.Background(), "machine-token", "ak-admin"); !errors.Is(err, iamapp.ErrInvalidAccessToken) {
		t.Fatalf("unmarked admin route error=%v", err)
	}
	marked := WithAgentCall(context.Background(), AgentCall{OperationID: "getAdminDashboardSummary", Method: "GET", RequestID: "audience-test"})
	if _, err := authenticator.Authenticate(marked, "machine-token", "ak-mobile"); !errors.Is(err, iamapp.ErrInvalidAccessToken) {
		t.Fatalf("mobile audience error=%v", err)
	}
	if len(repository.audits) != 0 {
		t.Fatalf("unmarked or non-admin routes must not delegate: %#v", repository.audits)
	}
}

func TestAgentCallableAllowlistMatchesOpenAPI(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err = yaml.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	marked := map[string]bool{}
	for path, methods := range document.Paths {
		for method, node := range methods {
			switch method {
			case "get", "post", "put", "patch", "delete", "head", "options":
			default:
				continue
			}
			var operation struct {
				OperationID   string                `yaml:"operationId"`
				AgentCallable bool                  `yaml:"x-appkernia-agent-callable"`
				Security      []map[string][]string `yaml:"security"`
			}
			if err := node.Decode(&operation); err != nil {
				t.Fatalf("decode OpenAPI operation %s %s: %v", method, path, err)
			}
			if !operation.AgentCallable {
				continue
			}
			marked[operation.OperationID] = true
			if !IsAgentCallable(operation.OperationID) {
				t.Errorf("OpenAPI marks unregistered operation %s %s (%s)", method, path, operation.OperationID)
			}
			hasMachineSecurity := false
			for _, alternative := range operation.Security {
				if _, ok := alternative["apiClientBearer"]; ok {
					hasMachineSecurity = true
				}
			}
			if !hasMachineSecurity {
				t.Errorf("agent operation %s lacks apiClientBearer", operation.OperationID)
			}
		}
	}
	for operationID := range agentCallableOperations {
		if !marked[operationID] {
			t.Errorf("runtime allowlist operation %s is not marked in OpenAPI", operationID)
		}
	}
}
