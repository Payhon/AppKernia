package http

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	notifyapp "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/application"
	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

type Handler struct {
	service *notifyapp.Service
	catalog *i18n.Catalog
}

func NewHandler(service *notifyapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{service: service, catalog: catalog}
}

func (h *Handler) Notices(r *ghttp.Request)  { h.listMessages(r, true) }
func (h *Handler) Messages(r *ghttp.Request) { h.listMessages(r, false) }
func (h *Handler) listMessages(r *ghttp.Request, notice bool) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	f.Status, f.Type = r.GetQuery("status").String(), r.GetQuery("message_type").String()
	out, err := h.service.ListMessages(r.Context(), token(r), appID(r), notice, f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) CreateNotice(r *ghttp.Request)  { h.createMessage(r, true) }
func (h *Handler) CreateMessage(r *ghttp.Request) { h.createMessage(r, false) }
func (h *Handler) createMessage(r *ghttp.Request, notice bool) {
	var body notify.MessageInput
	if !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.CreateMessage(r.Context(), token(r), appID(r), requestPrincipal(r), notice, body)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}

func (h *Handler) Notice(r *ghttp.Request)  { h.getMessage(r, true) }
func (h *Handler) Message(r *ghttp.Request) { h.getMessage(r, false) }
func (h *Handler) getMessage(r *ghttp.Request, notice bool) {
	id, ok := routerID(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.GetMessage(r.Context(), token(r), appID(r), id, notice)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) UpdateNotice(r *ghttp.Request)  { h.updateMessage(r, true) }
func (h *Handler) UpdateMessage(r *ghttp.Request) { h.updateMessage(r, false) }
func (h *Handler) updateMessage(r *ghttp.Request, notice bool) {
	id, ok := routerID(r)
	var body notify.MessageInput
	if !ok || !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.UpdateMessage(r.Context(), token(r), appID(r), requestPrincipal(r), id, notice, body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) PreviewNotice(r *ghttp.Request)  { h.preview(r, true) }
func (h *Handler) PreviewMessage(r *ghttp.Request) { h.preview(r, false) }
func (h *Handler) preview(r *ghttp.Request, notice bool) {
	id, ok := routerID(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.PreviewRecipients(r.Context(), token(r), appID(r), id, notice)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) PublishNotice(r *ghttp.Request)  { h.publish(r, true) }
func (h *Handler) PublishMessage(r *ghttp.Request) { h.publish(r, false) }
func (h *Handler) publish(r *ghttp.Request, notice bool) {
	id, ok := routerID(r)
	var body struct{}
	if !ok || !decodeOptional(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	message, recipients, err := h.service.PublishMessage(r.Context(), token(r), appID(r), requestPrincipal(r), id, notice)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"message": message, "recipients": recipients})
	}
}

func (h *Handler) CancelNotice(r *ghttp.Request)  { h.cancel(r, true) }
func (h *Handler) CancelMessage(r *ghttp.Request) { h.cancel(r, false) }
func (h *Handler) cancel(r *ghttp.Request, notice bool) {
	id, ok := routerID(r)
	var body struct{}
	if !ok || !decodeOptional(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.CancelMessage(r.Context(), token(r), appID(r), requestPrincipal(r), id, notice)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) NoticeRecipients(r *ghttp.Request)  { h.recipients(r, true) }
func (h *Handler) MessageRecipients(r *ghttp.Request) { h.recipients(r, false) }
func (h *Handler) recipients(r *ghttp.Request, notice bool) {
	id, ok := routerID(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.RecipientStats(r.Context(), token(r), appID(r), id, notice)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) Templates(r *ghttp.Request) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	f.Status, f.Channel, f.Locale = r.GetQuery("status").String(), r.GetQuery("channel").String(), r.GetQuery("locale").String()
	out, err := h.service.ListTemplates(r.Context(), token(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateTemplate(r *ghttp.Request) {
	var body notify.TemplateInput
	if !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.CreateTemplate(r.Context(), token(r), requestPrincipal(r), body)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateTemplate(r *ghttp.Request) {
	id, ok := routerID(r)
	var body notify.TemplateInput
	if !ok || !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.UpdateTemplate(r.Context(), token(r), requestPrincipal(r), id, body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) SMSTemplateBindings(r *ghttp.Request) {
	id, ok := routerID(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.ListSMSTemplateBindings(r.Context(), token(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) UpsertSMSTemplateBinding(r *ghttp.Request) {
	id, ok := routerID(r)
	var body notify.SMSTemplateBindingInput
	if !ok || !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	provider := strings.TrimSpace(r.GetRouter("provider").String())
	out, err := h.service.UpsertSMSTemplateBinding(r.Context(), token(r), requestPrincipal(r), id, provider, body)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) DeleteSMSTemplateBinding(r *ghttp.Request) {
	id, ok := routerID(r)
	provider := strings.TrimSpace(r.GetRouter("provider").String())
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	err := h.service.DeleteSMSTemplateBinding(r.Context(), token(r), requestPrincipal(r), id, provider)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}

func (h *Handler) TestTemplate(r *ghttp.Request) {
	id, ok := routerID(r)
	var body notify.TemplateTestInput
	if !ok || !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.TestTemplate(r.Context(), token(r), requestPrincipal(r), id, body)
	if !h.fail(r, err) {
		h.ok(r, 202, out)
	}
}

func (h *Handler) Deliveries(r *ghttp.Request) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	f.Status, f.Channel = r.GetQuery("status").String(), r.GetQuery("channel").String()
	out, err := h.service.ListDeliveries(r.Context(), token(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) Delivery(r *ghttp.Request) {
	id, ok := routerID(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.GetDelivery(r.Context(), token(r), id)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) RetryDelivery(r *ghttp.Request) {
	id, ok := routerID(r)
	var body struct {
		AcknowledgeDuplicateRisk bool `json:"acknowledge_duplicate_risk"`
	}
	if !ok || !decodeOptional(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.RetryDelivery(r.Context(), token(r), requestPrincipal(r), id, body.AcknowledgeDuplicateRisk)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func pageFilter(r *ghttp.Request) (notify.PageFilter, bool) {
	page, errPage := strconv.Atoi(r.GetQuery("page", 1).String())
	size, errSize := strconv.Atoi(r.GetQuery("page_size", 20).String())
	return notify.PageFilter{Query: r.GetQuery("q").String(), Page: int32(page), PageSize: int32(size)}, errPage == nil && errSize == nil
}
func appID(r *ghttp.Request) uuid.UUID {
	id, err := uuid.Parse(r.GetRouter("app_id").String())
	if err != nil {
		return uuid.Nil
	}
	return id
}

func requestPrincipal(r *ghttp.Request) notify.Principal {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return notify.Principal{RequestID: httpx.RequestID(r), IPAddress: strings.TrimSpace(host), UserAgent: strings.TrimSpace(r.Header.Get("User-Agent"))}
}

func (h *Handler) ok(r *ghttp.Request, status int, data any) {
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Success[any]{Code: "OK", Message: "OK", Data: data, RequestID: httpx.RequestID(r)})
}

func (h *Handler) fail(r *ghttp.Request, err error) bool {
	if err == nil {
		return false
	}
	status, code, key := 500, "COMMON.UNKNOWN", "errors.common.unknown"
	switch {
	case errors.Is(err, iamapp.ErrInvalidAccessToken):
		status, code, key = 401, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized"
	case errors.Is(err, notify.ErrForbidden):
		status, code, key = 403, "COMMON.FORBIDDEN", "errors.common.forbidden"
	case errors.Is(err, notify.ErrInvalid):
		status, code, key = 422, "VALIDATION.FAILED", "errors.validation.failed"
	case errors.Is(err, notify.ErrNotFound):
		status, code, key = 404, "COMMON.NOT_FOUND", "errors.common.not_found"
	case errors.Is(err, notify.ErrConflict):
		status, code, key = 409, "NOTIFY.LIFECYCLE.CONFLICT", "errors.common.conflict"
	case errors.Is(err, notify.ErrRecipientEmpty):
		status, code, key = 409, "NOTIFY.RECIPIENT.EMPTY", "errors.common.conflict"
	case errors.Is(err, notify.ErrRetryNotAllowed):
		status, code, key = 409, "NOTIFY.DELIVERY.RETRY_NOT_ALLOWED", "errors.common.conflict"
	case errors.Is(err, notify.ErrDeliveryUnavailable):
		status, code, key = 503, "NOTIFY.DELIVERY.UNAVAILABLE", "errors.common.unavailable"
	}
	r.Response.Header().Set("Cache-Control", "no-store")
	r.Response.WriteHeader(status)
	r.Response.WriteJsonExit(httpx.Error{Error: httpx.ErrorBody{Code: code, MessageKey: key, Message: h.catalog.Translate(httpx.Locale(r), key, nil)}, RequestID: httpx.RequestID(r)})
	return true
}

func decode(r *ghttp.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 256*1024))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil && decoder.Decode(&struct{}{}) == io.EOF
}
func decodeOptional(r *ghttp.Request, target any) bool {
	return r.ContentLength == 0 || decode(r, target)
}
func routerID(r *ghttp.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.GetRouter("id").String())
	return id, err == nil
}
func token(r *ghttp.Request) string {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}
