package application

import (
	"context"
	"errors"
	"html/template"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	apps "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	releases "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/appkernia/appkernia/server/internal/shared/publicurl"
	"github.com/google/uuid"
)

var ErrNotFound = errors.New("public page not found")
var ErrUnavailable = errors.New("public web origin not configured")

type Apps interface {
	PublicWeb(context.Context, uuid.UUID, string) (apps.PublicWebApp, error)
	PublicPage(context.Context, uuid.UUID, string, string) (apps.PublicPage, error)
	OpenPublicWebAsset(context.Context, uuid.UUID, uuid.UUID) (string, int64, io.ReadCloser, error)
}
type Content interface {
	GetPublic(context.Context, string, uuid.UUID, string, string) (content.PublicArticle, error)
	OpenPublicAsset(context.Context, string, uuid.UUID, uuid.UUID) (content.ArticleAsset, io.ReadCloser, error)
}
type Releases interface {
	PublicRelease(context.Context, uuid.UUID, string) (releases.Release, error)
	SignedPublicWebPackageURL(releases.Release) *string
	PublicPackageFile(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (releases.PackageFile, error)
	OpenPublicWebPackageDownload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) (releases.PackageFile, io.ReadCloser, error)
}
type Link struct {
	Title string
	URL   string
}
type Download struct {
	Name     string
	Platform string
	URL      string
	APK      bool
}
type Media struct {
	URL, AltText string
}
type View struct {
	Kind, ContentType, Locale, Title, Summary, Canonical, OtherLanguageURL, DownloadURL, AppName, IconURL, CoverURL, ShareImageURL, PublishedAt, Version string
	PromotionTitle, PromotionDescription, PromotionButtonLabel                                                                                           string
	PromotionEnabled                                                                                                                                     bool
	Body                                                                                                                                                 template.HTML
	Media                                                                                                                                                []Media
	VideoURL, MediaOrigin                                                                                                                                string
	Screenshots                                                                                                                                          []string
	Downloads                                                                                                                                            []Download
	Pages                                                                                                                                                []Link
	Labels                                                                                                                                               map[string]string
	CSS, JS, MediaJS                                                                                                                                     string
	PreviewJS, PreviewOrigin                                                                                                                             string
}

func (v View) T(key string) string { return v.Labels[key] }

type Service struct {
	apps     Apps
	content  Content
	releases Releases
	base     string
}

func NewService(a Apps, c Content, r Releases, base string) *Service {
	return &Service{a, c, r, strings.TrimRight(base, "/")}
}
func (s *Service) view(ctx context.Context, id uuid.UUID, locale, suffix string) (View, apps.PublicWebApp, error) {
	if s.base == "" {
		return View{}, apps.PublicWebApp{}, ErrUnavailable
	}
	a, e := s.apps.PublicWeb(ctx, id, locale)
	if e != nil {
		return View{}, a, e
	}
	other := "en-US"
	if locale == "en-US" {
		other = "zh-CN"
	}
	v := View{Locale: locale, AppName: a.Name, Title: a.Name, Summary: a.Introduction, Canonical: publicurl.Link(s.base, id, suffix, locale), OtherLanguageURL: publicurl.Link(s.base, id, suffix, other), PromotionEnabled: a.PromotionEnabled, PromotionTitle: a.PromotionTitle, PromotionDescription: a.PromotionDescription, PromotionButtonLabel: a.PromotionButtonLabel}
	if a.Enabled {
		v.DownloadURL = publicurl.Link(s.base, id, "/download", locale)
	}
	if a.IconFileID != nil {
		v.IconURL = publicurl.Path(id, "/assets/"+a.IconFileID.String())
		v.ShareImageURL = s.base + v.IconURL
	}
	return v, a, nil
}
func (s *Service) Article(ctx context.Context, id uuid.UUID, slug, locale string) (View, error) {
	v, _, e := s.view(ctx, id, locale, "/articles/"+url.PathEscape(slug))
	if e != nil {
		return v, e
	}
	x, e := s.content.GetPublic(ctx, "", id, locale, slug)
	if e != nil {
		return v, e
	}
	v.Kind = "article"
	v.ContentType = x.ContentType
	v.Title = x.Title
	v.Summary = x.Summary
	if x.PublishedAt != nil {
		v.PublishedAt = x.PublishedAt.UTC().Format(time.DateOnly)
	}
	if x.CoverURL != nil {
		if image := bodyImage(*x.CoverURL, id); image != "" {
			v.CoverURL = image
			v.ShareImageURL = s.base + image
		}
	}
	for _, media := range x.Media {
		if image := bodyImage(media.URL, id); image != "" {
			v.Media = append(v.Media, Media{URL: image, AltText: media.AltText})
		}
	}
	if x.VideoURL != nil {
		if video := bodyImage(*x.VideoURL, id); video != "" {
			v.VideoURL = video
		} else {
			v.VideoURL, v.MediaOrigin = externalVideo(*x.VideoURL)
		}
	}
	v.Body, e = RenderBody(x.Body, x.BodyFormat, id)
	return v, e
}
func (s *Service) Page(ctx context.Context, id uuid.UUID, slug, locale string) (View, error) {
	v, _, e := s.view(ctx, id, locale, "/pages/"+url.PathEscape(slug))
	if e != nil {
		return v, e
	}
	x, e := s.apps.PublicPage(ctx, id, slug, locale)
	if e != nil {
		return v, e
	}
	v.Kind = "document"
	v.Title = x.Title
	v.Summary = x.Title
	v.Version = strconv.FormatInt(int64(x.Version), 10)
	v.Body, e = RenderBody(x.Body, x.BodyFormat, id)
	return v, e
}
func (s *Service) Download(ctx context.Context, id uuid.UUID, locale string) (View, error) {
	v, a, e := s.view(ctx, id, locale, "/download")
	if e != nil {
		return v, e
	}
	if !a.Enabled {
		return v, ErrNotFound
	}
	v.Kind = "download"
	for _, file := range a.ScreenshotIDs {
		v.Screenshots = append(v.Screenshots, publicurl.Path(id, "/assets/"+file.String()))
	}
	for _, store := range a.Stores {
		u, e := url.Parse(store.WebURL)
		if store.Enabled && (store.Platform == "ios" || store.Platform == "android" || store.Platform == "harmony") && e == nil && u.Scheme == "https" && u.Host != "" && u.User == nil {
			v.Downloads = append(v.Downloads, Download{Name: store.Name, Platform: store.Platform, URL: store.WebURL})
		}
	}
	if a.APKEnabled {
		r, e := s.releases.PublicRelease(ctx, id, "android")
		if e == nil && r.PackageFileID != nil && r.PackageType == "native_app" {
			if _, err := s.releases.PublicPackageFile(ctx, id, r.ID, *r.PackageFileID); err == nil {
				v.Downloads = append(v.Downloads, Download{Platform: "android", URL: publicurl.Path(id, "/apk"), APK: true})
			} else if !errors.Is(err, releases.ErrReleaseFileInvalid) && !errors.Is(err, releases.ErrReleaseNotFound) {
				return v, err
			}
		} else if e != nil && !errors.Is(e, releases.ErrReleaseNotFound) {
			return v, e
		}
	}
	for _, p := range a.Pages {
		v.Pages = append(v.Pages, Link{p.Title, publicurl.Link(s.base, id, "/pages/"+url.PathEscape(p.Slug), locale)})
	}
	return v, nil
}
func (s *Service) Asset(ctx context.Context, id, file uuid.UUID) (string, int64, io.ReadCloser, error) {
	// Resolve active App even for the compatibility asset route.
	if _, e := s.apps.PublicWeb(ctx, id, "zh-CN"); e != nil {
		return "", 0, nil, e
	}
	media, size, reader, e := s.apps.OpenPublicWebAsset(ctx, id, file)
	if e == nil {
		return media, size, reader, nil
	}
	if !errors.Is(e, apps.ErrAppNotFound) {
		return "", 0, nil, e
	}
	asset, reader, e := s.content.OpenPublicAsset(ctx, "", id, file)
	return asset.MediaType, asset.SizeBytes, reader, e
}
func (s *Service) PackageURL(ctx context.Context, id uuid.UUID) (string, error) {
	a, e := s.apps.PublicWeb(ctx, id, "zh-CN")
	if e != nil {
		return "", e
	}
	if !a.Enabled || !a.APKEnabled {
		return "", ErrNotFound
	}
	r, e := s.releases.PublicRelease(ctx, id, "android")
	if e != nil {
		return "", e
	}
	if r.PackageFileID == nil || r.PackageType != "native_app" {
		return "", ErrNotFound
	}
	if _, err := s.releases.PublicPackageFile(ctx, id, r.ID, *r.PackageFileID); err != nil {
		return "", err
	}
	raw := s.releases.SignedPublicWebPackageURL(r)
	if raw == nil {
		return "", ErrNotFound
	}
	u, e := url.Parse(*raw)
	if e != nil || u.IsAbs() || u.Host != "" {
		return "", ErrNotFound
	}
	return publicurl.Path(id, "/packages/"+r.ID.String()+"/"+r.PackageFileID.String()) + "?" + u.RawQuery, nil
}
func (s *Service) Package(ctx context.Context, id, release, file uuid.UUID, expires int64, signature string) (releases.PackageFile, io.ReadCloser, error) {
	a, e := s.apps.PublicWeb(ctx, id, "zh-CN")
	if e != nil {
		return releases.PackageFile{}, nil, e
	}
	if !a.Enabled || !a.APKEnabled {
		return releases.PackageFile{}, nil, ErrNotFound
	}
	r, e := s.releases.PublicRelease(ctx, id, "android")
	if e != nil {
		return releases.PackageFile{}, nil, e
	}
	if r.ID != release || r.PackageFileID == nil || *r.PackageFileID != file {
		return releases.PackageFile{}, nil, ErrNotFound
	}
	return s.releases.OpenPublicWebPackageDownload(ctx, id, release, file, expires, signature)
}
