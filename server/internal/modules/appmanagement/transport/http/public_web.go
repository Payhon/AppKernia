package http

import (
	app "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
	"net/http"
)

func (h *Handler) AdminPublicWebConfig(r *ghttp.Request) {
	id, e := uuid.Parse(r.GetRouter("app_id").String())
	if e != nil {
		h.fail(r, 422, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	var out app.PublicWebConfig
	if r.Method == http.MethodGet {
		out, e = h.service.GetAdminPublicWebConfig(r.Context(), bearer(r), id)
	} else {
		var input app.PublicWebInput
		if !decode(r, &input) {
			h.fail(r, 422, "VALIDATION.FAILED", "errors.validation.failed")
			return
		}
		out, e = h.service.UpdateAdminPublicWebConfig(r.Context(), bearer(r), id, input, httpx.RequestID(r))
	}
	if !h.adminFailure(r, e) {
		r.Response.WriteJsonExit(httpx.Success[app.PublicWebConfig]{Code: "OK", Message: "OK", Data: out, RequestID: httpx.RequestID(r)})
	}
}
