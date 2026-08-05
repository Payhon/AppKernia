package httpx

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
)

func TestRequestContextResolvesContentLanguage(t *testing.T) {
	server := ghttp.GetServer("httpx-request-context-test")
	server.BindMiddlewareDefault(RequestContext)
	server.BindHandler("/", func(request *ghttp.Request) {
		request.Response.Write(string(Locale(request)))
	})
	server.SetPort(0)
	server.SetDumpRouterMap(false)
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Shutdown()
	time.Sleep(100 * time.Millisecond)

	for _, test := range []struct {
		acceptLanguage string
		want           string
	}{
		{acceptLanguage: "en-US,en;q=0.9", want: "en-US"},
		{acceptLanguage: "fr-FR,fr;q=0.9", want: "zh-CN"},
	} {
		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/", server.GetListenedPort()), nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Accept-Language", test.acceptLanguage)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}

		if got := response.Header.Get("Content-Language"); got != test.want {
			t.Fatalf("Content-Language=%q want %q", got, test.want)
		}
		if got := string(body); got != test.want {
			t.Fatalf("resolved locale body=%q want %q", got, test.want)
		}
	}
}
