package http

import (
	"strconv"
	"time"

	notify "github.com/appkernia/appkernia/server/internal/modules/notificationadmin/domain"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

func operationsFilter(r *ghttp.Request) (notify.OperationsFilter, bool) {
	page, errPage := strconv.Atoi(r.GetQuery("page", 1).String())
	size, errSize := strconv.Atoi(r.GetQuery("page_size", 20).String())
	var from, to time.Time
	var errFrom, errTo error
	if value := r.GetQuery("from").String(); value != "" {
		from, errFrom = time.Parse(time.RFC3339, value)
	}
	if value := r.GetQuery("to").String(); value != "" {
		to, errTo = time.Parse(time.RFC3339, value)
	}
	return notify.OperationsFilter{
		Environment: r.GetQuery("environment").String(),
		Category:    r.GetQuery("category").String(),
		Channel:     r.GetQuery("channel").String(),
		Provider:    r.GetQuery("provider").String(),
		TaskKind:    r.GetQuery("task_kind").String(),
		Status:      r.GetQuery("status").String(),
		Query:       r.GetQuery("q").String(),
		From:        from,
		To:          to,
		Page:        int32(page),
		PageSize:    int32(size),
	}, errPage == nil && errSize == nil && errFrom == nil && errTo == nil
}

func (h *Handler) NotificationOperationsSummary(r *ghttp.Request) {
	f, ok := operationsFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.OperationsSummary(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) NotificationOperationsTrends(r *ghttp.Request) {
	f, ok := operationsFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.OperationsTrends(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out})
	}
}

func (h *Handler) NotificationRuns(r *ghttp.Request) {
	f, ok := operationsFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.ListMessageRuns(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) NotificationRun(r *ghttp.Request) {
	id, err := uuid.Parse(r.GetRouter("run_id").String())
	if err != nil {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, serviceErr := h.service.GetMessageRun(r.Context(), token(r), appID(r), id)
	if !h.fail(r, serviceErr) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) NotificationTasks(r *ghttp.Request) {
	f, ok := operationsFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.ListTaskRuns(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) NotificationTask(r *ghttp.Request) {
	id, err := uuid.Parse(r.GetRouter("task_id").String())
	if err != nil {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, serviceErr := h.service.GetTaskRun(r.Context(), token(r), appID(r), id)
	if !h.fail(r, serviceErr) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) NotificationFailures(r *ghttp.Request) {
	f, ok := operationsFilter(r)
	if !ok {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.ListFailures(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) RetryNotificationTasks(r *ghttp.Request) {
	var body notify.RetryInput
	if !decode(r, &body) {
		h.fail(r, notify.ErrInvalid)
		return
	}
	out, err := h.service.RetryTasks(r.Context(), token(r), appID(r), requestPrincipal(r), body)
	if !h.fail(r, err) {
		h.ok(r, 202, map[string]any{"items": out})
	}
}
