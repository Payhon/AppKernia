package http

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"
)

func TestReadAvatarContentAcceptsOneMultipartFile(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.jpg")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("jpeg-payload")
	if _, err = part.Write(want); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := readAvatarContent(writer.FormDataContentType(), &body, true)
	if err != nil {
		t.Fatalf("readAvatarContent() error=%v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("readAvatarContent()=%q want=%q", got, want)
	}
}

func TestReadAvatarContentRejectsUnexpectedMultipartField(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("other", "avatar.jpg")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("jpeg-payload"))
	_ = writer.Close()
	if _, err = readAvatarContent(writer.FormDataContentType(), &body, true); err == nil {
		t.Fatal("readAvatarContent() expected error")
	}
}

func TestAvatarUploadIDPreservesMultipartBody(t *testing.T) {
	wantID := uuid.MustParse("01956f02-55ee-7b4d-8f25-3c2347a89d01")
	wantContent := []byte("multipart-body-must-remain-readable")
	server := ghttp.GetServer("storage-avatar-router-body-test")
	server.BindHandler("POST:/uploads/{id}", func(request *ghttp.Request) {
		gotID, err := avatarUploadID(request)
		if err != nil || gotID != wantID {
			request.Response.WriteStatus(http.StatusBadRequest)
			return
		}
		content, err := readAvatarContent(request.Header.Get("Content-Type"), request.Body, true)
		if err != nil {
			request.Response.WriteStatus(http.StatusUnprocessableEntity)
			return
		}
		request.Response.Write(content)
	})
	server.SetPort(0)
	server.SetDumpRouterMap(false)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "avatar.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(wantContent); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/uploads/%s", server.GetListenedPort(), wantID), &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !bytes.Equal(content, wantContent) {
		t.Fatalf("status=%d content=%q want=%q", response.StatusCode, content, wantContent)
	}
}
