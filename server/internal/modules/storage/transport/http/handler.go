package http

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	stdhttp "net/http"
	"net/netip"
	"strings"

	iamapp "github.com/appkernia/appkernia/server/internal/modules/iam/application"
	storageapp "github.com/appkernia/appkernia/server/internal/modules/storage/application"
	"github.com/appkernia/appkernia/server/internal/modules/storage/domain"
	"github.com/appkernia/appkernia/server/internal/shared/httpx"
	"github.com/appkernia/appkernia/server/internal/shared/i18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

const adminAudience = "ak-admin"
const mobileAudience = "ak-mobile"

type Handler struct {
	auth    *iamapp.AuthService
	service *storageapp.Service
	catalog *i18n.Catalog
}

type createAvatarUploadRequest struct {
	FileName  string `json:"file_name"`
	MediaType string `json:"media_type"`
	SizeBytes int64  `json:"size_bytes"`
}

func NewHandler(auth *iamapp.AuthService, service *storageapp.Service, catalog *i18n.Catalog) *Handler {
	return &Handler{auth: auth, service: service, catalog: catalog}
}

func (handler *Handler) CreateAvatarUpload(request *ghttp.Request) {
	handler.createAvatarUpload(request, adminAudience, "PUT")
}

func (handler *Handler) CreateMobileAvatarUpload(request *ghttp.Request) {
	handler.createAvatarUpload(request, mobileAudience, "POST")
}

func (handler *Handler) createAvatarUpload(request *ghttp.Request, audience, method string) {
	principal, ok := handler.principal(request, audience)
	if !ok {
		return
	}
	var body createAvatarUploadRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || decoder.Decode(&struct{}{}) != io.EOF {
		handler.writeError(request, stdhttp.StatusUnprocessableEntity, "VALIDATION.FAILED", "errors.validation.failed")
		return
	}
	target, err := handler.service.CreateAvatarUpload(request.Context(), principal, storageapp.CreateAvatarUploadInput{
		OriginalName: body.FileName, MediaType: body.MediaType, SizeBytes: body.SizeBytes,
	})
	switch {
	case errors.Is(err, domain.ErrFeatureDisabled):
		handler.writeError(request, stdhttp.StatusNotFound, "COMMON.NOT_FOUND", "errors.common.not_found")
	case errors.Is(err, domain.ErrUploadInvalid):
		handler.writeError(request, stdhttp.StatusUnprocessableEntity, "STORAGE.AVATAR.INVALID", "errors.storage.avatar.invalid")
	case err != nil:
		handler.writeError(request, stdhttp.StatusInternalServerError, "STORAGE.AVATAR.UNAVAILABLE", "errors.storage.avatar.unavailable")
	default:
		request.Response.Header().Set("Cache-Control", "no-store")
		request.Response.WriteHeader(stdhttp.StatusCreated)
		uploadURL := target.UploadURL
		request.Response.WriteJsonExit(httpx.Success[map[string]any]{
			Code: "OK", Message: "OK", RequestID: httpx.RequestID(request),
			Data: map[string]any{
				"id": target.ID, "upload_url": uploadURL, "method": method,
				"expires_at": target.ExpiresAt,
			},
		})
	}
}

func (handler *Handler) UploadAvatarContent(request *ghttp.Request) {
	handler.uploadAvatarContent(request, adminAudience, false)
}

func (handler *Handler) UploadMobileAvatarContent(request *ghttp.Request) {
	handler.uploadAvatarContent(request, mobileAudience, true)
}

func (handler *Handler) uploadAvatarContent(request *ghttp.Request, audience string, multipartUpload bool) {
	principal, ok := handler.principal(request, audience)
	if !ok {
		return
	}
	// Read the path parameter directly. Request.Get also parses multipart form data,
	// which consumes uni.uploadFile's body before readAvatarContent can validate it.
	uploadID, err := avatarUploadID(request)
	if err != nil {
		handler.writeError(request, stdhttp.StatusUnprocessableEntity, "STORAGE.AVATAR.INVALID", "errors.storage.avatar.invalid")
		return
	}
	content, err := readAvatarContent(request.Header.Get("Content-Type"), request.Body, multipartUpload)
	if err != nil || int64(len(content)) > domain.MaxAvatarBytes {
		handler.writeError(request, stdhttp.StatusUnprocessableEntity, "STORAGE.AVATAR.INVALID", "errors.storage.avatar.invalid")
		return
	}
	fileID, err := handler.service.UploadAvatar(request.Context(), principal, uploadID, content, domain.ClientMetadata{
		RequestID: httpx.RequestID(request), IPAddress: clientIP(request), UserAgent: request.Header.Get("User-Agent"),
		HTTPMethod: request.Method, RequestPath: request.URL.Path,
	})
	switch {
	case errors.Is(err, domain.ErrFeatureDisabled):
		handler.writeError(request, stdhttp.StatusNotFound, "COMMON.NOT_FOUND", "errors.common.not_found")
	case errors.Is(err, domain.ErrUploadNotFound):
		handler.writeError(request, stdhttp.StatusNotFound, "STORAGE.AVATAR.UPLOAD_NOT_FOUND", "errors.storage.avatar.upload_not_found")
	case errors.Is(err, domain.ErrUploadInvalid):
		handler.writeError(request, stdhttp.StatusUnprocessableEntity, "STORAGE.AVATAR.INVALID", "errors.storage.avatar.invalid")
	case err != nil:
		handler.writeError(request, stdhttp.StatusInternalServerError, "STORAGE.AVATAR.UNAVAILABLE", "errors.storage.avatar.unavailable")
	default:
		request.Response.Header().Set("Cache-Control", "no-store")
		request.Response.WriteJsonExit(httpx.Success[map[string]any]{
			Code: "OK", Message: handler.catalog.Translate(httpx.Locale(request), "messages.profile.avatar_updated", nil),
			Data:      map[string]any{"file_id": fileID, "avatar_url": "/me/avatar/content?v=" + fileID.String()},
			RequestID: httpx.RequestID(request),
		})
	}
}

func avatarUploadID(request *ghttp.Request) (uuid.UUID, error) {
	value := request.GetRouter("id")
	if value == nil {
		return uuid.Nil, domain.ErrUploadInvalid
	}
	return uuid.Parse(value.String())
}

func (handler *Handler) AvatarContent(request *ghttp.Request) {
	handler.avatarContent(request, adminAudience)
}

func (handler *Handler) MobileAvatarContent(request *ghttp.Request) {
	handler.avatarContent(request, mobileAudience)
}

func (handler *Handler) avatarContent(request *ghttp.Request, audience string) {
	principal, ok := handler.principal(request, audience)
	if !ok {
		return
	}
	object, reader, err := handler.service.OpenAvatar(request.Context(), principal)
	switch {
	case errors.Is(err, domain.ErrFeatureDisabled), errors.Is(err, domain.ErrAvatarNotFound), errors.Is(err, domain.ErrObjectNotFound):
		handler.writeError(request, stdhttp.StatusNotFound, "STORAGE.AVATAR.NOT_FOUND", "errors.storage.avatar.not_found")
		return
	case err != nil:
		handler.writeError(request, stdhttp.StatusInternalServerError, "STORAGE.AVATAR.UNAVAILABLE", "errors.storage.avatar.unavailable")
		return
	}
	defer func() { _ = reader.Close() }()
	etag := object.SHA256
	if len(etag) == 0 {
		fallback := sha256.Sum256([]byte(object.FileID.String()))
		etag = fallback[:]
	}
	request.Response.Header().Set("Content-Type", object.MediaType)
	request.Response.Header().Set("Content-Length", stringInt64(object.SizeBytes))
	request.Response.Header().Set("Cache-Control", "private, max-age=300")
	request.Response.Header().Set("ETag", `"`+hex.EncodeToString(etag)+`"`)
	request.Response.Header().Set("X-Content-Type-Options", "nosniff")
	request.Response.Header().Set("Vary", "Authorization")
	if _, err = io.Copy(request.Response.BufferWriter, reader); err != nil {
		return
	}
}

func (handler *Handler) principal(request *ghttp.Request, audience string) (domain.Principal, bool) {
	authenticated, err := handler.auth.Authenticate(request.Context(), bearerToken(request.Header.Get("Authorization")), audience)
	if err != nil {
		handler.writeError(request, stdhttp.StatusUnauthorized, "AUTH.SESSION.UNAUTHORIZED", "errors.common.unauthorized")
		return domain.Principal{}, false
	}
	return domain.Principal{
		UserID: authenticated.User.ID, TenantID: authenticated.Tenant.ID, SessionID: authenticated.SessionID,
	}, true
}

func readAvatarContent(contentType string, body io.Reader, multipartUpload bool) ([]byte, error) {
	if !multipartUpload {
		return io.ReadAll(io.LimitReader(body, domain.MaxAvatarBytes+1))
	}
	mediaType, parameters, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" || strings.TrimSpace(parameters["boundary"]) == "" {
		return nil, domain.ErrUploadInvalid
	}
	reader := multipart.NewReader(body, parameters["boundary"])
	part, err := reader.NextPart()
	if err != nil || part.FormName() != "file" || strings.TrimSpace(part.FileName()) == "" {
		return nil, domain.ErrUploadInvalid
	}
	content, err := io.ReadAll(io.LimitReader(part, domain.MaxAvatarBytes+1))
	if err != nil {
		return nil, err
	}
	if next, nextErr := reader.NextPart(); nextErr != io.EOF || next != nil {
		return nil, domain.ErrUploadInvalid
	}
	return content, nil
}

func (handler *Handler) writeError(request *ghttp.Request, status int, code, messageKey string) {
	request.Response.Header().Set("Cache-Control", "no-store")
	request.Response.WriteHeader(status)
	request.Response.WriteJsonExit(httpx.Error{
		Error:     httpx.ErrorBody{Code: code, MessageKey: messageKey, Message: handler.catalog.Translate(httpx.Locale(request), messageKey, nil)},
		RequestID: httpx.RequestID(request),
	})
}

func bearerToken(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return parts[1]
	}
	return ""
}

func clientIP(request *ghttp.Request) *netip.Addr {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		host = request.RemoteAddr
	}
	address, err := netip.ParseAddr(strings.TrimSpace(host))
	if err != nil {
		return nil
	}
	return &address
}

func stringInt64(value int64) string {
	return fmt.Sprintf("%d", value)
}
