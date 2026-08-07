package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	stdhtml "html"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	iamdomain "github.com/appkernia/appkernia/server/internal/modules/iam/domain"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	settings "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	"github.com/google/uuid"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Authenticator interface {
	Authenticate(context.Context, string, string) (iamdomain.AuthenticatedContext, error)
}

type Service struct {
	auth         Authenticator
	repo         notify.Repository
	dictionaries interface {
		ResolveDictionary(context.Context, *uuid.UUID, string, string, bool) (settings.ResolvedDictionary, error)
	}
	sealer notify.TargetSealer
	clock  func() time.Time
}

type Option func(*Service)

func WithDictionaryResolver(resolver interface {
	ResolveDictionary(context.Context, *uuid.UUID, string, string, bool) (settings.ResolvedDictionary, error)
}) Option {
	return func(service *Service) { service.dictionaries = resolver }
}

func WithTargetSealer(sealer notify.TargetSealer) Option {
	return func(service *Service) { service.sealer = sealer }
}

func NewService(auth Authenticator, repo notify.Repository, options ...Option) *Service {
	service := &Service{auth: auth, repo: repo, clock: time.Now}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) authorize(ctx context.Context, token, permission string) (iamdomain.AuthenticatedContext, error) {
	auth, err := s.auth.Authenticate(ctx, token, "ak-admin")
	if err != nil {
		return iamdomain.AuthenticatedContext{}, err
	}
	if !slices.Contains(auth.Permissions, permission) {
		return iamdomain.AuthenticatedContext{}, notify.ErrForbidden
	}
	return auth, nil
}

func principal(auth iamdomain.AuthenticatedContext, appID uuid.UUID, p notify.Principal) notify.Principal {
	p.TenantID, p.AppID, p.UserID, p.SessionID = auth.Tenant.ID, appID, auth.User.ID, auth.SessionID
	return p
}

func normalizePage(f notify.PageFilter) (notify.PageFilter, error) {
	f.Query, f.Status, f.Type, f.Channel, f.Locale = strings.TrimSpace(f.Query), strings.TrimSpace(f.Status), strings.TrimSpace(f.Type), strings.TrimSpace(f.Channel), strings.TrimSpace(f.Locale)
	if f.Page == 0 {
		f.Page = 1
	}
	if f.PageSize == 0 {
		f.PageSize = 20
	}
	if f.Page < 1 || f.PageSize < 1 || f.PageSize > 100 || len([]rune(f.Query)) > 160 {
		return f, notify.ErrInvalid
	}
	return f, nil
}

func normalizeMessage(in notify.MessageInput, notice bool, now time.Time) (notify.MessageInput, error) {
	in.Title, in.Body, in.BodyFormat, in.MessageType, in.AudienceScope = strings.TrimSpace(in.Title), strings.TrimSpace(in.Body), strings.TrimSpace(in.BodyFormat), strings.TrimSpace(in.MessageType), strings.TrimSpace(in.AudienceScope)
	if notice {
		in.MessageType = "notice"
	} else if in.MessageType == "" {
		in.MessageType = "system"
	}
	if in.BodyFormat == "" {
		in.BodyFormat = "plain"
	}
	if in.AudienceScope == "" {
		in.AudienceScope = "all"
	}
	if len([]rune(in.Title)) < 1 || len([]rune(in.Title)) > 300 || len([]rune(in.Body)) < 1 || len([]rune(in.Body)) > 100_000 || !oneOf(in.BodyFormat, "plain", "markdown", "html") || !oneOf(in.AudienceScope, "all", "selected") || (!notice && !oneOf(in.MessageType, "system", "private", "marketing", "security")) {
		return in, notify.ErrInvalid
	}
	if in.ScheduledAt != nil && in.ScheduledAt.Before(now.Add(-time.Minute)) || in.ExpiresAt != nil && !in.ExpiresAt.After(now) || in.ScheduledAt != nil && in.ExpiresAt != nil && !in.ExpiresAt.After(*in.ScheduledAt) {
		return in, notify.ErrInvalid
	}
	unique := make(map[uuid.UUID]struct{}, len(in.AudienceUserIDs))
	out := make([]uuid.UUID, 0, len(in.AudienceUserIDs))
	for _, id := range in.AudienceUserIDs {
		if id == uuid.Nil {
			return in, notify.ErrInvalid
		}
		if _, ok := unique[id]; !ok {
			unique[id] = struct{}{}
			out = append(out, id)
		}
	}
	if in.AudienceScope == "all" {
		out = []uuid.UUID{}
	} else if len(out) < 1 || len(out) > 500 {
		return in, notify.ErrInvalid
	}
	in.AudienceUserIDs = out
	if in.BodyFormat == "html" {
		in.Body = SanitizeHTML(in.Body)
		if strings.TrimSpace(stripHTML(in.Body)) == "" {
			return in, notify.ErrInvalid
		}
	}
	return in, nil
}

func (s *Service) ListMessages(ctx context.Context, token string, appID uuid.UUID, notice bool, f notify.PageFilter) (notify.MessagePage, error) {
	permission := "notify.message.read"
	if notice {
		permission = "notify.notice.read"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.MessagePage{}, err
	}
	f, err = normalizePage(f)
	if err != nil || !oneOf(f.Status, "", "draft", "scheduled", "published", "cancelled") || (!notice && !oneOf(f.Type, "", "system", "private", "marketing", "security")) {
		return notify.MessagePage{}, notify.ErrInvalid
	}
	return s.repo.ListMessages(ctx, auth.Tenant.ID, appID, notice, f)
}

func (s *Service) GetMessage(ctx context.Context, token string, appID, id uuid.UUID, notice bool) (notify.Message, error) {
	permission := "notify.message.read"
	if notice {
		permission = "notify.notice.read"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.Message{}, err
	}
	return s.repo.GetMessage(ctx, auth.Tenant.ID, appID, id, notice)
}

func (s *Service) CreateMessage(ctx context.Context, token string, appID uuid.UUID, p notify.Principal, notice bool, in notify.MessageInput) (notify.Message, error) {
	permission := "notify.message.create"
	if notice {
		permission = "notify.notice.create"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.Message{}, err
	}
	in, err = normalizeMessage(in, notice, s.clock().UTC())
	if err != nil || strings.TrimSpace(p.RequestID) == "" {
		return notify.Message{}, notify.ErrInvalid
	}
	return s.repo.CreateMessage(ctx, principal(auth, appID, p), notice, in)
}

func (s *Service) UpdateMessage(ctx context.Context, token string, appID uuid.UUID, p notify.Principal, id uuid.UUID, notice bool, in notify.MessageInput) (notify.Message, error) {
	permission := "notify.message.update"
	if notice {
		permission = "notify.notice.update"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.Message{}, err
	}
	in, err = normalizeMessage(in, notice, s.clock().UTC())
	if err != nil || id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return notify.Message{}, notify.ErrInvalid
	}
	return s.repo.UpdateMessage(ctx, principal(auth, appID, p), id, notice, in)
}

func (s *Service) PreviewRecipients(ctx context.Context, token string, appID, id uuid.UUID, notice bool) (notify.RecipientPreview, error) {
	permission := "notify.message.publish"
	if notice {
		permission = "notify.notice.publish"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.RecipientPreview{}, err
	}
	message, err := s.repo.GetMessage(ctx, auth.Tenant.ID, appID, id, notice)
	if err != nil {
		return notify.RecipientPreview{}, err
	}
	if message.Status != "draft" && message.Status != "scheduled" {
		return notify.RecipientPreview{}, notify.ErrConflict
	}
	return s.repo.PreviewRecipients(ctx, auth.Tenant.ID, appID, message)
}

func (s *Service) PublishMessage(ctx context.Context, token string, appID uuid.UUID, p notify.Principal, id uuid.UUID, notice bool) (notify.Message, notify.RecipientPreview, error) {
	permission := "notify.message.publish"
	if notice {
		permission = "notify.notice.publish"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.Message{}, notify.RecipientPreview{}, err
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return notify.Message{}, notify.RecipientPreview{}, notify.ErrInvalid
	}
	return s.repo.PublishMessage(ctx, principal(auth, appID, p), id, notice)
}

func (s *Service) CancelMessage(ctx context.Context, token string, appID uuid.UUID, p notify.Principal, id uuid.UUID, notice bool) (notify.Message, error) {
	permission := "notify.message.cancel"
	if notice {
		permission = "notify.notice.cancel"
	}
	auth, err := s.authorize(ctx, token, permission)
	if err != nil {
		return notify.Message{}, err
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return notify.Message{}, notify.ErrInvalid
	}
	return s.repo.CancelMessage(ctx, principal(auth, appID, p), id, notice)
}

func (s *Service) RecipientStats(ctx context.Context, token string, appID, id uuid.UUID, notice bool) (notify.RecipientStats, error) {
	auth, err := s.authorize(ctx, token, "notify.recipient.read")
	if err != nil {
		return notify.RecipientStats{}, err
	}
	return s.repo.RecipientStats(ctx, auth.Tenant.ID, appID, id, notice)
}

var templateCode = regexp.MustCompile(`^[a-z][a-z0-9_.-]{1,95}$`)
var placeholder = regexp.MustCompile(`\{\{\s*([a-zA-Z][a-zA-Z0-9_.-]*)\s*\}\}`)

func normalizeTemplate(in notify.TemplateInput) (notify.TemplateInput, error) {
	in.Code, in.Name, in.Channel, in.BodyTemplate, in.BodyFormat, in.Status = strings.TrimSpace(in.Code), strings.TrimSpace(in.Name), strings.TrimSpace(in.Channel), strings.TrimSpace(in.BodyTemplate), strings.TrimSpace(in.BodyFormat), strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	if in.BodyFormat == "" {
		in.BodyFormat = "plain"
	}
	if in.Locale != nil {
		locale := strings.TrimSpace(*in.Locale)
		in.Locale = &locale
	}
	if in.SubjectTemplate != nil {
		subject := strings.TrimSpace(*in.SubjectTemplate)
		in.SubjectTemplate = &subject
	}
	if !templateCode.MatchString(in.Code) || len([]rune(in.Name)) < 1 || len([]rune(in.Name)) > 160 || len([]rune(in.BodyTemplate)) < 1 || len([]rune(in.BodyTemplate)) > 100_000 || !oneOf(in.Channel, "in_app", "email", "sms", "push", "webhook") || !oneOf(in.BodyFormat, "plain", "html") || in.BodyFormat == "html" && in.Channel != "email" || !oneOf(in.Status, "active", "disabled") || in.Locale != nil && !oneOf(*in.Locale, "zh-CN", "en-US") || in.SubjectTemplate != nil && len([]rune(*in.SubjectTemplate)) > 500 {
		return in, notify.ErrInvalid
	}
	if in.BodyFormat == "html" {
		in.BodyTemplate = SanitizeHTML(in.BodyTemplate)
		if strings.TrimSpace(stripHTML(in.BodyTemplate)) == "" {
			return in, notify.ErrInvalid
		}
	}
	if len(in.VariablesSchema) == 0 {
		in.VariablesSchema = json.RawMessage(`{"type":"object","properties":{}}`)
	}
	var schema struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if !json.Valid(in.VariablesSchema) || json.Unmarshal(in.VariablesSchema, &schema) != nil || schema.Type != "object" || len(schema.Properties) > 100 {
		return in, notify.ErrInvalid
	}
	for _, name := range schema.Required {
		if _, ok := schema.Properties[name]; !ok {
			return in, notify.ErrInvalid
		}
	}
	combined := in.BodyTemplate
	if in.SubjectTemplate != nil {
		combined += *in.SubjectTemplate
	}
	for _, match := range placeholder.FindAllStringSubmatch(combined, -1) {
		if _, ok := schema.Properties[match[1]]; !ok {
			return in, notify.ErrInvalid
		}
	}
	return in, nil
}

func (s *Service) validateTemplateEvent(ctx context.Context, tenantID uuid.UUID, in notify.TemplateInput) error {
	if in.Channel != "email" && in.Channel != "sms" || s.dictionaries == nil {
		return nil
	}
	code := "notification." + in.Channel + ".template_event"
	dictionary, err := s.dictionaries.ResolveDictionary(ctx, &tenantID, code, valueOrDefault(in.Locale, "zh-CN"), false)
	if err != nil {
		return notify.ErrInvalid
	}
	for _, item := range dictionary.Items {
		if item.Value == in.Code {
			return nil
		}
	}
	return notify.ErrInvalid
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func (s *Service) ListTemplates(ctx context.Context, token string, f notify.PageFilter) (notify.TemplatePage, error) {
	auth, err := s.authorize(ctx, token, "notify.template.read")
	if err != nil {
		return notify.TemplatePage{}, err
	}
	f, err = normalizePage(f)
	if err != nil || !oneOf(f.Status, "", "active", "disabled") || !oneOf(f.Channel, "", "in_app", "email", "sms", "push", "webhook") || !oneOf(f.Locale, "", "zh-CN", "en-US", "global") {
		return notify.TemplatePage{}, notify.ErrInvalid
	}
	return s.repo.ListTemplates(ctx, auth.Tenant.ID, f)
}

func (s *Service) CreateTemplate(ctx context.Context, token string, p notify.Principal, in notify.TemplateInput) (notify.Template, error) {
	auth, err := s.authorize(ctx, token, "notify.template.create")
	if err != nil {
		return notify.Template{}, err
	}
	in, err = normalizeTemplate(in)
	if err != nil || strings.TrimSpace(p.RequestID) == "" || s.validateTemplateEvent(ctx, auth.Tenant.ID, in) != nil {
		return notify.Template{}, notify.ErrInvalid
	}
	return s.repo.CreateTemplate(ctx, principal(auth, uuid.Nil, p), in)
}

func (s *Service) UpdateTemplate(ctx context.Context, token string, p notify.Principal, id uuid.UUID, in notify.TemplateInput) (notify.Template, error) {
	auth, err := s.authorize(ctx, token, "notify.template.update")
	if err != nil {
		return notify.Template{}, err
	}
	in, err = normalizeTemplate(in)
	if err != nil || id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || s.validateTemplateEvent(ctx, auth.Tenant.ID, in) != nil {
		return notify.Template{}, notify.ErrInvalid
	}
	return s.repo.UpdateTemplate(ctx, principal(auth, uuid.Nil, p), id, in)
}

func (s *Service) ListDeliveries(ctx context.Context, token string, f notify.PageFilter) (notify.DeliveryPage, error) {
	auth, err := s.authorize(ctx, token, "notify.delivery.read")
	if err != nil {
		return notify.DeliveryPage{}, err
	}
	f, err = normalizePage(f)
	if err != nil || !oneOf(f.Status, "", "pending", "processing", "sent", "failed", "cancelled") || !oneOf(f.Channel, "", "email", "sms", "push", "webhook") {
		return notify.DeliveryPage{}, notify.ErrInvalid
	}
	return s.repo.ListDeliveries(ctx, auth.Tenant.ID, f)
}

func (s *Service) GetDelivery(ctx context.Context, token string, id uuid.UUID) (notify.Delivery, error) {
	auth, err := s.authorize(ctx, token, "notify.delivery.read")
	if err != nil {
		return notify.Delivery{}, err
	}
	return s.repo.GetDelivery(ctx, auth.Tenant.ID, id)
}

func (s *Service) RetryDelivery(ctx context.Context, token string, p notify.Principal, id uuid.UUID, acknowledgeDuplicateRisk bool) (notify.Delivery, error) {
	auth, err := s.authorize(ctx, token, "notify.delivery.retry")
	if err != nil {
		return notify.Delivery{}, err
	}
	if id == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return notify.Delivery{}, notify.ErrInvalid
	}
	return s.repo.RetryDelivery(ctx, principal(auth, uuid.Nil, p), id, acknowledgeDuplicateRisk)
}

func (s *Service) ListSMSTemplateBindings(ctx context.Context, token string, templateID uuid.UUID) ([]notify.SMSTemplateBinding, error) {
	auth, err := s.authorize(ctx, token, "notify.template.read")
	if err != nil {
		return nil, err
	}
	if templateID == uuid.Nil {
		return nil, notify.ErrInvalid
	}
	return s.repo.ListSMSTemplateBindings(ctx, auth.Tenant.ID, templateID)
}

func normalizeBinding(provider string, in notify.SMSTemplateBindingInput) (string, notify.SMSTemplateBindingInput, error) {
	provider, in.ExternalTemplateID, in.SignName, in.Status = strings.TrimSpace(provider), strings.TrimSpace(in.ExternalTemplateID), strings.TrimSpace(in.SignName), strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = "active"
	}
	if len(in.ParameterOrder) == 0 {
		in.ParameterOrder = json.RawMessage(`[]`)
	}
	var parameters []string
	if !oneOf(provider, "aliyun", "tencent") || len(in.ExternalTemplateID) < 1 || len(in.ExternalTemplateID) > 255 || len(in.SignName) > 120 || !oneOf(in.Status, "active", "disabled") || json.Unmarshal(in.ParameterOrder, &parameters) != nil || len(parameters) > 50 {
		return provider, in, notify.ErrInvalid
	}
	seen := map[string]bool{}
	for _, parameter := range parameters {
		if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_.-]{0,63}$`).MatchString(parameter) || seen[parameter] {
			return provider, in, notify.ErrInvalid
		}
		seen[parameter] = true
	}
	return provider, in, nil
}

func (s *Service) UpsertSMSTemplateBinding(ctx context.Context, token string, p notify.Principal, templateID uuid.UUID, provider string, in notify.SMSTemplateBindingInput) (notify.SMSTemplateBinding, error) {
	auth, err := s.authorize(ctx, token, "notify.template.update")
	if err != nil {
		return notify.SMSTemplateBinding{}, err
	}
	provider, in, err = normalizeBinding(provider, in)
	if err != nil || templateID == uuid.Nil || strings.TrimSpace(p.RequestID) == "" {
		return notify.SMSTemplateBinding{}, notify.ErrInvalid
	}
	template, err := s.repo.GetTemplate(ctx, auth.Tenant.ID, templateID)
	if err != nil || template.Channel != "sms" {
		return notify.SMSTemplateBinding{}, notify.ErrInvalid
	}
	var variableSchema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	var parameterOrder []string
	if json.Unmarshal(template.VariablesSchema, &variableSchema) != nil || json.Unmarshal(in.ParameterOrder, &parameterOrder) != nil {
		return notify.SMSTemplateBinding{}, notify.ErrInvalid
	}
	for _, parameter := range parameterOrder {
		if _, declared := variableSchema.Properties[parameter]; !declared {
			return notify.SMSTemplateBinding{}, notify.ErrInvalid
		}
	}
	return s.repo.UpsertSMSTemplateBinding(ctx, principal(auth, uuid.Nil, p), templateID, provider, in)
}

func (s *Service) DeleteSMSTemplateBinding(ctx context.Context, token string, p notify.Principal, templateID uuid.UUID, provider string) error {
	auth, err := s.authorize(ctx, token, "notify.template.update")
	if err != nil {
		return err
	}
	if templateID == uuid.Nil || !oneOf(provider, "aliyun", "tencent") || strings.TrimSpace(p.RequestID) == "" {
		return notify.ErrInvalid
	}
	return s.repo.DeleteSMSTemplateBinding(ctx, principal(auth, uuid.Nil, p), templateID, provider)
}

func normalizeTarget(channel, target string) (string, string, error) {
	target = strings.TrimSpace(target)
	if channel == "email" {
		parsed, err := mail.ParseAddress(target)
		if err != nil || parsed.Address != target || len(target) > 320 {
			return "", "", notify.ErrInvalid
		}
		parts := strings.SplitN(target, "@", 2)
		return strings.ToLower(target), string([]rune(parts[0])[:1]) + "***@" + parts[1], nil
	}
	if channel == "sms" && regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`).MatchString(target) {
		runes := []rune(target)
		return target, string(runes[:min(4, len(runes))]) + "***" + string(runes[max(0, len(runes)-4):]), nil
	}
	return "", "", notify.ErrInvalid
}

func renderTemplate(template notify.Template, variables map[string]string) (string, string, error) {
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if json.Unmarshal(template.VariablesSchema, &schema) != nil {
		return "", "", notify.ErrInvalid
	}
	for variable := range variables {
		if _, declared := schema.Properties[variable]; !declared {
			return "", "", notify.ErrInvalid
		}
	}
	for _, required := range schema.Required {
		if strings.TrimSpace(variables[required]) == "" {
			return "", "", notify.ErrInvalid
		}
	}
	replace := func(input string, escape bool) (string, error) {
		missing := false
		result := placeholder.ReplaceAllStringFunc(input, func(match string) string {
			parts := placeholder.FindStringSubmatch(match)
			value, ok := variables[parts[1]]
			if !ok {
				missing = true
				return ""
			}
			if escape {
				return stdhtml.EscapeString(value)
			}
			return value
		})
		if missing {
			return "", notify.ErrInvalid
		}
		return result, nil
	}
	body, err := replace(template.BodyTemplate, template.BodyFormat == "html")
	if err != nil {
		return "", "", err
	}
	subject := ""
	if template.SubjectTemplate != nil {
		subject, err = replace(*template.SubjectTemplate, false)
	}
	return subject, body, err
}

func (s *Service) TestTemplate(ctx context.Context, token string, p notify.Principal, templateID uuid.UUID, in notify.TemplateTestInput) (notify.Delivery, error) {
	auth, err := s.authorize(ctx, token, "notify.template.test")
	if err != nil {
		return notify.Delivery{}, err
	}
	if s.sealer == nil || templateID == uuid.Nil || strings.TrimSpace(p.RequestID) == "" || len(in.Variables) > 100 {
		return notify.Delivery{}, notify.ErrDeliveryUnavailable
	}
	template, err := s.repo.GetTemplate(ctx, auth.Tenant.ID, templateID)
	if err != nil || template.Status != "active" || !oneOf(template.Channel, "email", "sms") {
		return notify.Delivery{}, notify.ErrInvalid
	}
	provider := "smtp"
	if template.Channel == "sms" {
		provider = strings.TrimSpace(in.Provider)
		if !in.ConfirmBillable || !oneOf(provider, "aliyun", "tencent") {
			return notify.Delivery{}, notify.ErrInvalid
		}
	}
	target, hint, err := normalizeTarget(template.Channel, in.Target)
	if err != nil {
		return notify.Delivery{}, err
	}
	subject, body, err := renderTemplate(template, in.Variables)
	if err != nil {
		return notify.Delivery{}, err
	}
	payload, _ := json.Marshal(in.Variables)
	targetCiphertext, targetVersion, err := s.sealer.Seal([]byte(target), auth.Tenant.ID.String())
	if err != nil {
		return notify.Delivery{}, notify.ErrDeliveryUnavailable
	}
	payloadCiphertext, payloadVersion, err := s.sealer.Seal(payload, auth.Tenant.ID.String()+":notification-payload")
	if err != nil {
		return notify.Delivery{}, notify.ErrDeliveryUnavailable
	}
	hash := sha256.Sum256([]byte(target))
	return s.repo.CreateTestDelivery(ctx, principal(auth, uuid.Nil, p), notify.CreateDelivery{
		TemplateID: template.ID, Channel: template.Channel, Provider: provider,
		TargetCiphertext: targetCiphertext, TargetHash: hash[:], TargetHint: hint, TargetKeyVersion: targetVersion,
		PayloadCiphertext: payloadCiphertext, PayloadKeyVersion: payloadVersion,
		RenderedSubject: subject, RenderedBody: body, DedupeKey: "test:" + p.RequestID + ":" + template.ID.String(),
	})
}

func oneOf(value string, allowed ...string) bool { return slices.Contains(allowed, value) }

var allowedElements = map[string]map[string]bool{
	"p": {}, "br": {}, "strong": {}, "em": {}, "u": {}, "s": {}, "blockquote": {}, "code": {}, "pre": {},
	"ul": {}, "ol": {}, "li": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"table": {}, "thead": {}, "tbody": {}, "tr": {}, "th": {}, "td": {}, "hr": {}, "a": {"href": true, "title": true},
}
var droppedElements = map[string]bool{"script": true, "style": true, "iframe": true, "object": true, "embed": true, "form": true, "input": true, "button": true, "svg": true, "math": true}

// SanitizeHTML enforces the notification body allowlist used by the Admin preview contract.
func SanitizeHTML(raw string) string {
	container := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := html.ParseFragment(strings.NewReader(raw), container)
	if err != nil {
		return ""
	}
	var clean func(*html.Node) []*html.Node
	clean = func(node *html.Node) []*html.Node {
		if node.Type == html.CommentNode {
			return nil
		}
		if node.Type != html.ElementNode {
			copy := *node
			copy.Parent, copy.FirstChild, copy.LastChild, copy.PrevSibling, copy.NextSibling = nil, nil, nil, nil, nil
			return []*html.Node{&copy}
		}
		name := strings.ToLower(node.Data)
		if droppedElements[name] {
			return nil
		}
		children := make([]*html.Node, 0)
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			children = append(children, clean(child)...)
		}
		attrs, allowed := allowedElements[name]
		if !allowed {
			return children
		}
		copy := &html.Node{Type: html.ElementNode, Data: name}
		for _, attr := range node.Attr {
			key := strings.ToLower(attr.Key)
			if !attrs[key] {
				continue
			}
			value := strings.TrimSpace(attr.Val)
			if key == "href" && !safeURL(value) {
				continue
			}
			copy.Attr = append(copy.Attr, html.Attribute{Key: key, Val: value})
		}
		for _, child := range children {
			copy.AppendChild(child)
		}
		return []*html.Node{copy}
	}
	var out bytes.Buffer
	for _, node := range nodes {
		for _, cleaned := range clean(node) {
			_ = html.Render(&out, cleaned)
		}
	}
	return strings.TrimSpace(out.String())
}

func safeURL(value string) bool {
	if match := placeholder.FindStringIndex(value); match != nil && match[0] == 0 && match[1] == len(value) {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "" || parsed.Scheme == "https" || parsed.Scheme == "http" || parsed.Scheme == "mailto")
}

func stripHTML(value string) string {
	doc, err := html.Parse(strings.NewReader(value))
	if err != nil {
		return ""
	}
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			out.WriteString(node.Data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)
	return out.String()
}
