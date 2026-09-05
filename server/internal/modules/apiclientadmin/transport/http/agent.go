package http

import (
	"net"
	"strings"

	clientapp "github.com/appkernia/appkernia/server/internal/modules/apiclientadmin/application"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

// AgentCallable marks the exact HTTP operations that may exchange an ak-api
// token for a delegated user context. Routes without this wrapper fail closed.
func AgentCallable(operationID string, next func(*ghttp.Request)) func(*ghttp.Request) {
	if !clientapp.IsAgentCallable(operationID) {
		panic("unregistered agent-callable operation: " + operationID)
	}
	return func(request *ghttp.Request) {
		var appID *uuid.UUID
		if raw := strings.TrimSpace(request.GetRouter("app_id").String()); raw != "" {
			parsed, err := uuid.Parse(raw)
			if err != nil {
				parsed = uuid.Nil
			}
			appID = &parsed
		}
		request.SetCtx(clientapp.WithAgentCall(request.Context(), clientapp.AgentCall{
			OperationID: operationID,
			Method:      request.Method,
			Path:        request.URL.Path,
			RequestID:   httpx.RequestID(request),
			IPAddress:   requestRemoteIP(request),
			UserAgent:   strings.TrimSpace(request.UserAgent()),
			AppID:       appID,
		}))
		next(request)
	}
}

func requestRemoteIP(request *ghttp.Request) string {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	return strings.TrimSpace(host)
}
