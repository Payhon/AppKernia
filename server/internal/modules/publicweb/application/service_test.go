package application

import (
	"context"
	"encoding/json"
	"errors"
	apps "github.com/appkernia/appkernia/server/internal/modules/appmanagement/application"
	content "github.com/appkernia/appkernia/server/internal/modules/content/domain"
	releases "github.com/appkernia/appkernia/server/internal/modules/mobileprofile/domain"
	"github.com/google/uuid"
	"io"
	"strings"
	"testing"
)

type fakeApps struct {
	value apps.PublicWebApp
	err   error
}

func (a *fakeApps) PublicWeb(context.Context, uuid.UUID, string) (apps.PublicWebApp, error) {
	return a.value, a.err
}
func (a *fakeApps) PublicPage(context.Context, uuid.UUID, string, string) (apps.PublicPage, error) {
	return apps.PublicPage{Title: "Policy", BodyFormat: "markdown", Body: json.RawMessage(`"Published policy"`), Version: 3}, a.err
}
func (a *fakeApps) OpenPublicWebAsset(context.Context, uuid.UUID, uuid.UUID) (string, int64, io.ReadCloser, error) {
	return "", 0, nil, apps.ErrAppNotFound
}

type fakeContent struct {
	token string
	value content.PublicArticle
}

func (c *fakeContent) GetPublic(_ context.Context, token string, _ uuid.UUID, _, _ string) (content.PublicArticle, error) {
	c.token = token
	if c.value.Title != "" {
		return c.value, nil
	}
	return content.PublicArticle{Title: "Article", BodyFormat: "markdown", Body: json.RawMessage(`"Read me"`)}, nil
}
func (c *fakeContent) OpenPublicAsset(_ context.Context, token string, _, _ uuid.UUID) (content.ArticleAsset, io.ReadCloser, error) {
	c.token = token
	return content.ArticleAsset{}, nil, content.ErrNotFound
}

func TestArticleIncludesGalleryAndVideoMedia(t *testing.T) {
	appID, first, second := uuid.New(), uuid.New(), uuid.New()
	a := &fakeApps{value: apps.PublicWebApp{Name: "App", Enabled: true, PromotionEnabled: true, PromotionTitle: "Get App", PromotionDescription: "More content", PromotionButtonLabel: "Install"}}
	c := &fakeContent{value: content.PublicArticle{
		Title: "Gallery", Summary: "Summary", ContentType: "gallery", BodyFormat: "markdown", Body: json.RawMessage(`"Caption"`),
		Media: []content.PublicMedia{{URL: "/api/v1/public/content/assets/" + first.String(), AltText: "First"}, {URL: "/api/v1/public/content/assets/" + second.String(), AltText: "Second"}},
	}}
	s := NewService(a, c, &fakeReleases{}, "https://public.example")
	view, err := s.Article(context.Background(), appID, "gallery", "zh-CN")
	if err != nil || view.ContentType != "gallery" || len(view.Media) != 2 || !strings.Contains(view.Media[0].URL, first.String()) || !strings.Contains(string(view.Body), "Caption") || !view.PromotionEnabled || view.PromotionTitle != "Get App" || view.PromotionButtonLabel != "Install" {
		t.Fatalf("gallery view=%+v err=%v", view, err)
	}
	videoURL := "https://media.example.test/video.mp4"
	c.value = content.PublicArticle{Title: "Video", ContentType: "video", BodyFormat: "markdown", Body: json.RawMessage(`"Details"`), VideoURL: &videoURL}
	view, err = s.Article(context.Background(), appID, "video", "en-US")
	if err != nil || view.VideoURL != videoURL || view.MediaOrigin != "https://media.example.test" || !strings.Contains(string(view.Body), "Details") {
		t.Fatalf("video view=%+v err=%v", view, err)
	}
}

func TestPromotionVisibilityFollowsAdminConfiguration(t *testing.T) {
	a := &fakeApps{value: apps.PublicWebApp{Name: "App", Enabled: true, PromotionEnabled: false, PromotionTitle: "Hidden"}}
	s := NewService(a, &fakeContent{}, &fakeReleases{}, "https://public.example")
	view, err := s.Page(context.Background(), uuid.New(), "privacy-policy", "zh-CN")
	if err != nil || view.PromotionEnabled || view.DownloadURL == "" {
		t.Fatalf("view=%+v err=%v", view, err)
	}
}

type fakeReleases struct {
	value  releases.Release
	opened bool
	err    error
	signed string
}

func (r *fakeReleases) PublicRelease(context.Context, uuid.UUID, string) (releases.Release, error) {
	return r.value, r.err
}
func (r *fakeReleases) SignedPublicWebPackageURL(releases.Release) *string { return &r.signed }
func (r *fakeReleases) OpenPublicWebPackageDownload(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, int64, string) (releases.PackageFile, io.ReadCloser, error) {
	r.opened = true
	return releases.PackageFile{}, io.NopCloser(strings.NewReader("apk")), nil
}
func TestDownloadVisibilityAndRecheckWithdrawal(t *testing.T) {
	id, file, release := uuid.New(), uuid.New(), uuid.New()
	a := &fakeApps{value: apps.PublicWebApp{Name: "App", Enabled: true, APKEnabled: true, Stores: []apps.WebStore{{Name: "iOS", Enabled: true, Platform: "ios", WebURL: "https://apps.apple.com/app/test"}, {Name: "Android", Enabled: true, Platform: "android", WebURL: "https://play.google.com/store/apps/test"}, {Name: "Disabled", Platform: "ios", WebURL: "https://example.test"}, {Name: "Unsafe", Enabled: true, Platform: "ios", WebURL: "javascript:bad"}}}}
	r := &fakeReleases{value: releases.Release{ID: release, PackageFileID: &file, PackageType: "native_app"}, signed: "/api/v1/public/app-version/download/a/b?expires=123&signature=signature"}
	s := NewService(a, &fakeContent{}, r, "https://public.example")
	view, err := s.Download(context.Background(), id, "en-US")
	if err != nil || len(view.Downloads) != 3 {
		t.Fatalf("view=%+v err=%v", view, err)
	}
	target, err := s.PackageURL(context.Background(), id)
	if err != nil || !strings.Contains(target, "/h5/apps/"+id.String()+"/packages/"+release.String()+"/"+file.String()) {
		t.Fatalf("target=%s err=%v", target, err)
	}
	if _, _, err = s.Package(context.Background(), id, uuid.New(), file, 123, "signature"); !errors.Is(err, ErrNotFound) || r.opened {
		t.Fatal("stale release opened")
	}
	a.value.APKEnabled = false
	if _, err = s.PackageURL(context.Background(), id); !errors.Is(err, ErrNotFound) {
		t.Fatal("disabled APK signed")
	}
	if _, _, err = s.Package(context.Background(), id, release, file, 123, "signature"); !errors.Is(err, ErrNotFound) || r.opened {
		t.Fatal("disabled APK opened")
	}
	a.value.Enabled = false
	if _, err = s.Download(context.Background(), id, "zh-CN"); !errors.Is(err, ErrNotFound) {
		t.Fatal("disabled landing visible")
	}
	c := &fakeContent{token: "sentinel"}
	s.content = c
	if _, err = s.Article(context.Background(), id, "read", "zh-CN"); err != nil || c.token != "" {
		t.Fatal("legacy article must work anonymously with landing disabled")
	}
	a.err = apps.ErrAppDisabled
	if _, err = s.Article(context.Background(), id, "read", "zh-CN"); !errors.Is(err, apps.ErrAppDisabled) {
		t.Fatal("disabled app exposed article")
	}
}

func (r *fakeReleases) PublicPackageFile(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (releases.PackageFile, error) {
	return releases.PackageFile{}, r.err
}
