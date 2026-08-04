package application

import (
	"context"
	"encoding/csv"
	"errors"
	"io"
	"net/mail"
	"sort"
	"strings"
	"time"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	userdomain "github.com/appkernia/appkernia/server/internal/modules/useradmin/domain"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	authenticator Authenticator
	repository    userdomain.Repository
}

func NewService(authenticator Authenticator, repository userdomain.Repository) *Service {
	return &Service{authenticator: authenticator, repository: repository}
}

func (service *Service) RoleOptions(ctx context.Context, token string) ([]userdomain.Reference, error) {
	auth, err := service.authorize(ctx, token, "iam.user.assign_role")
	if err != nil {
		return nil, err
	}
	return service.repository.ListRoleOptions(ctx, auth.Tenant.ID)
}

func (service *Service) List(ctx context.Context, token string, filters userdomain.Filters) (userdomain.Page, error) {
	auth, err := service.authorize(ctx, token, "iam.user.read")
	if err != nil {
		return userdomain.Page{}, err
	}
	if !validFilters(filters) {
		return userdomain.Page{}, userdomain.ErrInvalid
	}
	filters.Query = strings.TrimSpace(filters.Query)
	if filters.Page == 0 {
		filters.Page = 1
	}
	if filters.PageSize == 0 {
		filters.PageSize = 20
	}
	return service.repository.ListUsers(ctx, auth.Tenant.ID, filters)
}

func (service *Service) Get(ctx context.Context, token string, id uuid.UUID) (userdomain.User, error) {
	auth, err := service.authorize(ctx, token, "iam.user.read")
	if err != nil {
		return userdomain.User{}, err
	}
	if id == uuid.Nil {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	return service.repository.GetUser(ctx, auth.Tenant.ID, id)
}

func (service *Service) Create(ctx context.Context, token string, principal userdomain.Principal, input userdomain.CreateInput) (userdomain.User, error) {
	auth, err := service.authorize(ctx, token, "iam.user.create")
	if err != nil {
		return userdomain.User{}, err
	}
	input = normalizeCreate(input)
	if !validCreate(input) {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	hash, err := iamapp.HashPassword(input.TemporaryPassword)
	if err != nil {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	input.TemporaryPassword = ""
	return service.repository.CreateUser(ctx, scoped(auth, principal), input, hash)
}

func (service *Service) Update(ctx context.Context, token string, principal userdomain.Principal, id uuid.UUID, input userdomain.UpdateInput) (userdomain.User, error) {
	auth, err := service.authorize(ctx, token, "iam.user.update")
	if err != nil {
		return userdomain.User{}, err
	}
	input.DisplayName, input.Locale, input.TimeZone = strings.TrimSpace(input.DisplayName), strings.TrimSpace(input.Locale), strings.TrimSpace(input.TimeZone)
	if id == uuid.Nil || len(input.DisplayName) < 1 || len(input.DisplayName) > 120 || !validLocale(input.Locale) || len(input.TimeZone) < 1 || len(input.TimeZone) > 64 {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	return service.repository.UpdateUser(ctx, scoped(auth, principal), id, input)
}

func (service *Service) SetStatus(ctx context.Context, token string, principal userdomain.Principal, id uuid.UUID, enabled bool) (userdomain.User, error) {
	permission, status := "iam.user.disable", "suspended"
	if enabled {
		permission, status = "iam.user.enable", "active"
	}
	auth, err := service.authorize(ctx, token, permission)
	if err != nil {
		return userdomain.User{}, err
	}
	if id == uuid.Nil || (!enabled && id == auth.User.ID) {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	return service.repository.SetMemberStatus(ctx, scoped(auth, principal), id, status)
}

func (service *Service) Unlock(ctx context.Context, token string, principal userdomain.Principal, id uuid.UUID) error {
	auth, err := service.authorize(ctx, token, "iam.user.unlock")
	if err != nil {
		return err
	}
	if id == uuid.Nil {
		return userdomain.ErrInvalid
	}
	return service.repository.UnlockUser(ctx, scoped(auth, principal), id)
}

func (service *Service) ResetPassword(ctx context.Context, token string, principal userdomain.Principal, id uuid.UUID, password string) (int64, error) {
	auth, err := service.authorize(ctx, token, "iam.user.reset_password")
	if err != nil {
		return 0, err
	}
	hash, err := iamapp.HashPassword(password)
	if id == uuid.Nil || err != nil {
		return 0, userdomain.ErrInvalid
	}
	return service.repository.ResetPassword(ctx, scoped(auth, principal), id, hash)
}

func (service *Service) ReplaceRoles(ctx context.Context, token string, principal userdomain.Principal, id uuid.UUID, roleIDs []uuid.UUID) (userdomain.User, error) {
	auth, err := service.authorize(ctx, token, "iam.user.assign_role")
	if err != nil {
		return userdomain.User{}, err
	}
	if id == uuid.Nil || !uniqueIDs(roleIDs) {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	return service.repository.ReplaceRoles(ctx, scoped(auth, principal), id, roleIDs)
}

func (service *Service) ReplaceAssignments(ctx context.Context, token string, principal userdomain.Principal, id uuid.UUID, input userdomain.AssignmentInput) (userdomain.User, error) {
	auth, err := service.authorize(ctx, token, "org.assignment.update")
	if err != nil {
		return userdomain.User{}, err
	}
	if id == uuid.Nil || !uniqueIDs(input.UnitIDs) || !uniqueIDs(input.PositionIDs) || !containsOptional(input.UnitIDs, input.PrimaryUnitID) || !containsOptional(input.PositionIDs, input.PrimaryPositionID) {
		return userdomain.User{}, userdomain.ErrInvalid
	}
	return service.repository.ReplaceAssignments(ctx, scoped(auth, principal), id, input)
}

func (service *Service) Sessions(ctx context.Context, token string, target uuid.UUID) ([]userdomain.Session, error) {
	auth, err := service.authorize(ctx, token, "iam.session.read")
	if err != nil {
		return nil, err
	}
	return service.repository.ListSessions(ctx, auth.Tenant.ID, target, auth.SessionID)
}

func (service *Service) RevokeSession(ctx context.Context, token string, principal userdomain.Principal, target, sessionID uuid.UUID) error {
	auth, err := service.authorize(ctx, token, "iam.session.revoke")
	if err != nil {
		return err
	}
	if target == uuid.Nil || sessionID == uuid.Nil || sessionID == auth.SessionID {
		return userdomain.ErrInvalid
	}
	return service.repository.RevokeSession(ctx, scoped(auth, principal), target, sessionID)
}

func (service *Service) Import(ctx context.Context, token string, principal userdomain.Principal, reader io.Reader) (userdomain.ImportResult, error) {
	auth, err := service.authorize(ctx, token, "iam.user.import")
	if err != nil {
		return userdomain.ImportResult{}, err
	}
	records, err := csv.NewReader(io.LimitReader(reader, 2<<20)).ReadAll()
	if err != nil || len(records) < 2 || len(records) > 501 {
		return userdomain.ImportResult{}, userdomain.ErrInvalid
	}
	if strings.Join(records[0], ",") != "email,display_name,locale,time_zone,temporary_password" {
		return userdomain.ImportResult{}, userdomain.ErrInvalid
	}
	result := userdomain.ImportResult{Errors: []userdomain.ImportError{}}
	for index, record := range records[1:] {
		row := index + 2
		if len(record) != 5 {
			result.Failed++
			result.Errors = append(result.Errors, userdomain.ImportError{Row: row, Code: "VALIDATION.FAILED"})
			continue
		}
		input := normalizeCreate(userdomain.CreateInput{Email: record[0], DisplayName: record[1], Locale: record[2], TimeZone: record[3], TemporaryPassword: record[4]})
		if !validCreate(input) {
			result.Failed++
			result.Errors = append(result.Errors, userdomain.ImportError{Row: row, Code: "VALIDATION.FAILED"})
			continue
		}
		hash, hashErr := iamapp.HashPassword(input.TemporaryPassword)
		if hashErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, userdomain.ImportError{Row: row, Code: "VALIDATION.FAILED"})
			continue
		}
		input.TemporaryPassword = ""
		_, createErr := service.repository.CreateUser(ctx, scoped(auth, principal), input, hash)
		if createErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, userdomain.ImportError{Row: row, Code: publicCode(createErr)})
			continue
		}
		result.Created++
	}
	return result, nil
}

func (service *Service) Export(ctx context.Context, token string, filters userdomain.Filters, writer io.Writer) error {
	auth, err := service.authorize(ctx, token, "iam.user.export")
	if err != nil {
		return err
	}
	if !validFilters(filters) {
		return userdomain.ErrInvalid
	}
	filters.Page, filters.PageSize = 1, 500
	page, err := service.repository.ListUsers(ctx, auth.Tenant.ID, filters)
	if err != nil {
		return err
	}
	csvWriter := csv.NewWriter(writer)
	if err = csvWriter.Write([]string{"email", "display_name", "status", "locale", "time_zone", "roles", "units", "positions", "created_at"}); err != nil {
		return err
	}
	for _, user := range page.Items {
		if err = csvWriter.Write([]string{user.Email, user.DisplayName, user.Status, user.Locale, user.TimeZone, referenceNames(user.Roles), referenceNames(user.Units), referenceNames(user.Positions), user.CreatedAt.UTC().Format(time.RFC3339)}); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func (service *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	auth, err := service.authenticator.Authenticate(ctx, token, adminAudience)
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	for _, candidate := range auth.Permissions {
		if candidate == permission {
			return auth, nil
		}
	}
	return iamdomain.AuthenticatedContext{}, userdomain.ErrForbidden
}

func scoped(auth iamdomain.AuthenticatedContext, principal userdomain.Principal) userdomain.Principal {
	principal.TenantID, principal.UserID, principal.SessionID = auth.Tenant.ID, auth.User.ID, auth.SessionID
	return principal
}
func normalizeCreate(input userdomain.CreateInput) userdomain.CreateInput {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Locale = strings.TrimSpace(input.Locale)
	input.TimeZone = strings.TrimSpace(input.TimeZone)
	return input
}
func validCreate(input userdomain.CreateInput) bool {
	address, err := mail.ParseAddress(input.Email)
	return err == nil && address.Address == input.Email && len(input.Email) <= 254 && len(input.DisplayName) >= 1 && len(input.DisplayName) <= 120 && validLocale(input.Locale) && len(input.TimeZone) >= 1 && len(input.TimeZone) <= 64 && len(input.TemporaryPassword) >= 12 && len(input.TemporaryPassword) <= 256
}
func validLocale(value string) bool { return value == "zh-CN" || value == "en-US" }
func validFilters(input userdomain.Filters) bool {
	return (input.Status == "" || input.Status == "active" || input.Status == "disabled" || input.Status == "pending" || input.Status == "locked") && input.Page >= 0 && input.Page <= 1_000_000 && input.PageSize >= 0 && input.PageSize <= 100 && (input.Sort == "" || input.Sort == "created_desc" || input.Sort == "created_asc" || input.Sort == "name_asc" || input.Sort == "last_login_desc")
}
func uniqueIDs(ids []uuid.UUID) bool {
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if id == uuid.Nil || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
func containsOptional(ids []uuid.UUID, value *uuid.UUID) bool {
	if value == nil {
		return true
	}
	for _, id := range ids {
		if id == *value {
			return true
		}
	}
	return false
}
func referenceNames(values []userdomain.Reference) string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Name)
	}
	sort.Strings(names)
	return strings.Join(names, " | ")
}
func publicCode(err error) string {
	if errors.Is(err, userdomain.ErrEmailConflict) {
		return "IAM.USER.EMAIL_EXISTS"
	}
	return "COMMON.UNKNOWN"
}
