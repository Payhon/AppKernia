package http

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

func headerAppID(r *ghttp.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(strings.TrimSpace(r.Header.Get("X-AppID")))
	return id, err == nil
}

func publicFilter(r *ghttp.Request) (content.PublicFilter, bool) {
	limit, err := strconv.Atoi(r.GetQuery("limit", 20).String())
	if err != nil {
		return content.PublicFilter{}, false
	}
	featured, ok := optionalBool(r, "featured")
	if !ok {
		return content.PublicFilter{}, false
	}
	return content.PublicFilter{Query: r.GetQuery("q").String(), CategorySlug: r.GetQuery("category").String(), TopicSlug: r.GetQuery("topic").String(), Tag: r.GetQuery("tag").String(), ContentType: r.GetQuery("type").String(), Cursor: r.GetQuery("cursor").String(), Featured: featured, Limit: int32(limit)}, true
}

func (h *Handler) PublicHome(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	limit, err := strconv.Atoi(r.GetQuery("limit", 10).String())
	if !ok || err != nil {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.PublicHome(r.Context(), token(r), appID, locale(r), int32(limit))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) PublicItems(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	filter, filterOK := publicFilter(r)
	if !ok || !filterOK {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.ListPublic(r.Context(), token(r), appID, locale(r), filter)
	if !h.fail(r, err) {
		h.okMeta(r, 200, map[string]any{"items": out.Items}, map[string]any{"next_cursor": out.NextCursor})
	}
}
func (h *Handler) PublicItem(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.GetPublic(r.Context(), token(r), appID, locale(r), r.GetRouter("slug").String())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) PublicCategories(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.PublicCategories(r.Context(), token(r), appID, locale(r))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) PublicTopics(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.PublicTopics(r.Context(), token(r), appID, locale(r))
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) PublicTopic(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.PublicTopic(r.Context(), token(r), appID, locale(r), r.GetRouter("slug").String())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) PublicAsset(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	fileID, fileOK := idAt(r, "file_id")
	if !ok || !fileOK {
		h.fail(r, content.ErrInvalid)
		return
	}
	asset, reader, err := h.service.OpenPublicAsset(r.Context(), token(r), appID, fileID)
	if h.fail(r, err) {
		return
	}
	defer reader.Close()
	r.Response.Header().Set("Content-Type", asset.MediaType)
	r.Response.Header().Set("Content-Length", fmt.Sprintf("%d", asset.SizeBytes))
	r.Response.Header().Set("Cache-Control", "public, max-age=300")
	r.Response.Header().Set("X-Content-Type-Options", "nosniff")
	r.Response.Header().Set("Vary", "X-AppID")
	_, _ = io.Copy(r.Response.BufferWriter, reader)
}
func (h *Handler) PublicComments(r *ghttp.Request) {
	appID, ok := headerAppID(r)
	articleID, articleOK := idAt(r, "id")
	f, filterOK := pageFilter(r)
	if !ok || !articleOK || !filterOK {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.PublicComments(r.Context(), token(r), appID, articleID, f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}

func (h *Handler) MyBookmarks(r *ghttp.Request) {
	f, ok := publicFilter(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.MyBookmarks(r.Context(), token(r), locale(r), f)
	if !h.fail(r, err) {
		h.okMeta(r, 200, map[string]any{"items": out.Items}, map[string]any{"next_cursor": out.NextCursor})
	}
}
func (h *Handler) CreateComment(r *ghttp.Request) {
	articleID, ok := idAt(r, "id")
	var input struct {
		ParentID *uuid.UUID `json:"parent_id"`
		Body     string     `json:"body"`
	}
	if !ok || !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.CreateComment(r.Context(), token(r), principal(r), articleID, input.ParentID, input.Body)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) DeleteOwnComment(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.DeleteOwnComment(r.Context(), token(r), principal(r), id)) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}
func (h *Handler) ReportComment(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var input struct {
		Reason  string `json:"reason"`
		Details string `json:"details"`
	}
	if !ok || !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.ReportComment(r.Context(), token(r), principal(r), id, input.Reason, input.Details)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) BlockUser(r *ghttp.Request) {
	id, ok := idAt(r, "user_id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.BlockUser(r.Context(), token(r), principal(r), id, true)) {
		h.ok(r, 200, map[string]bool{"blocked": true})
	}
}
func (h *Handler) UnblockUser(r *ghttp.Request) {
	id, ok := idAt(r, "user_id")
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.BlockUser(r.Context(), token(r), principal(r), id, false)) {
		h.ok(r, 200, map[string]bool{"blocked": false})
	}
}

func (h *Handler) Topics(r *ghttp.Request) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	f.Status = r.GetQuery("status").String()
	out, err := h.service.ListTopics(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateTopic(r *ghttp.Request) {
	var x content.Topic
	if !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.CreateTopic(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateTopic(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var x content.Topic
	if !ok || !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	x.ID = id
	out, err := h.service.UpdateTopic(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) DeleteTopic(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	version, err := strconv.Atoi(r.GetQuery("lock_version").String())
	if !ok || err != nil {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.DeleteTopic(r.Context(), token(r), appID(r), principal(r), id, int32(version))) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}
func (h *Handler) Tags(r *ghttp.Request) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	f.Status = r.GetQuery("status").String()
	out, err := h.service.ListTags(r.Context(), token(r), appID(r), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) CreateTag(r *ghttp.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.UpsertTag(r.Context(), token(r), appID(r), principal(r), input.Name)
	if !h.fail(r, err) {
		h.ok(r, 201, out)
	}
}
func (h *Handler) UpdateTag(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var x content.Tag
	if !ok || !decode(r, &x) {
		h.fail(r, content.ErrInvalid)
		return
	}
	x.ID = id
	out, err := h.service.UpdateTag(r.Context(), token(r), appID(r), principal(r), x)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) MergeTag(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var input struct {
		TargetID    uuid.UUID `json:"target_id"`
		LockVersion int32     `json:"lock_version"`
	}
	if !ok || !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.MergeTag(r.Context(), token(r), appID(r), principal(r), id, input.TargetID, input.LockVersion)) {
		h.ok(r, 200, map[string]bool{"merged": true})
	}
}
func (h *Handler) DeleteTag(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	version, err := strconv.Atoi(r.GetQuery("lock_version").String())
	if !ok || err != nil {
		h.fail(r, content.ErrInvalid)
		return
	}
	if !h.fail(r, h.service.DeleteTag(r.Context(), token(r), appID(r), principal(r), id, int32(version))) {
		h.ok(r, 200, map[string]bool{"deleted": true})
	}
}
func (h *Handler) AdminComments(r *ghttp.Request) {
	f, ok := pageFilter(r)
	articleID, articleOK := optionalID(r, "article_id")
	if !ok || !articleOK {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.AdminComments(r.Context(), token(r), appID(r), f, articleID, r.GetQuery("status").String())
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) ModerateComment(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var input struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if !ok || !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.ModerateComment(r.Context(), token(r), appID(r), principal(r), id, input.Status, input.Reason)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) BatchModerateComments(r *ghttp.Request) {
	var input struct {
		IDs    []uuid.UUID `json:"ids"`
		Status string      `json:"status"`
		Reason string      `json:"reason"`
	}
	if !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.BatchModerateComments(r.Context(), token(r), appID(r), principal(r), input.IDs, input.Status, input.Reason)
	if !h.fail(r, err) {
		h.ok(r, 200, map[string]any{"items": out})
	}
}
func (h *Handler) CommentReports(r *ghttp.Request) {
	f, ok := pageFilter(r)
	if !ok {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.AdminCommentReports(r.Context(), token(r), appID(r), r.GetQuery("status").String(), f)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
func (h *Handler) ResolveCommentReport(r *ghttp.Request) {
	id, ok := idAt(r, "id")
	var input struct {
		Status     string `json:"status"`
		Resolution string `json:"resolution"`
	}
	if !ok || !decode(r, &input) {
		h.fail(r, content.ErrInvalid)
		return
	}
	out, err := h.service.ResolveCommentReport(r.Context(), token(r), appID(r), principal(r), id, input.Status, input.Resolution)
	if !h.fail(r, err) {
		h.ok(r, 200, out)
	}
}
