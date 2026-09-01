package richtext

import (
	"encoding/json"
	"github.com/google/uuid"
	"testing"
)

func TestOnlyRenderedImagesAuthorizeFiles(t *testing.T) {
	app, file := uuid.New(), uuid.New()
	path := "/api/v1/public/content/assets/" + file.String()
	for _, tc := range []struct {
		markdown string
		want     bool
	}{{"![image](" + path + ")", true}, {"plain " + file.String(), false}, {"`![image](" + path + ")`", false}, {"<img src=\"" + path + "\">", false}, {"![other](/h5/apps/" + uuid.NewString() + "/assets/" + file.String() + ")", false}} {
		raw, _ := json.Marshal(tc.markdown)
		if got := ReferencesImage(raw, "markdown", app, file); got != tc.want {
			t.Errorf("%s got %v", tc.markdown, got)
		}
	}
}
