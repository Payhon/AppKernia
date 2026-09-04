package bootstrap

import (
	"errors"
	"testing"

	settingsdomain "github.com/appkernia/appkernia/server/internal/modules/systemsettings/domain"
	platformcaptcha "github.com/appkernia/appkernia/server/internal/platform/captcha"
)

func TestConfiguredLoginCaptchaTypeFallbackAndFailure(t *testing.T) {
	databaseErr := errors.New("database unavailable")
	for _, test := range []struct {
		name    string
		value   string
		err     error
		want    platformcaptcha.Type
		wantErr error
	}{
		{name: "configured", value: "rotate", want: platformcaptcha.TypeRotate},
		{name: "missing", err: settingsdomain.ErrNotFound, want: platformcaptcha.TypeSlide},
		{name: "invalid", value: "unknown", want: platformcaptcha.TypeSlide},
		{name: "database failure", err: databaseErr, wantErr: databaseErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := configuredLoginCaptchaType(test.value, test.err)
			if got != test.want || !errors.Is(err, test.wantErr) {
				t.Fatalf("type=%q want=%q err=%v", got, test.want, err)
			}
		})
	}
}
