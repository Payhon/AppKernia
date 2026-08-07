package worker

import (
	"bytes"
	"encoding/json"
	"testing"

	settingsrepo "github.com/appkernia/appkernia/server/internal/modules/systemsettings/repository"
	"github.com/google/uuid"
)

func TestOTPEmailLateRenderKeepsPlaintextOutOfStoredDelivery(t *testing.T) {
	tenantID := uuid.New()
	sealer, err := settingsrepo.NewAESGCMSealer(bytes.Repeat([]byte{7}, 32), 1)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{"code": "731942", "expires_minutes": 10, "app_id": tenantID.String()})
	if err != nil {
		t.Fatal(err)
	}
	ciphertext, _, err := sealer.Seal(payload, tenantID.String()+":notification-payload")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, []byte("731942")) {
		t.Fatal("OTP code must not be stored in plaintext ciphertext")
	}
	variables, err := decryptNotificationVariables(sealer, ciphertext, tenantID)
	if err != nil {
		t.Fatal(err)
	}
	subject, body, err := renderNotificationTemplate("Your code", "Code {{ code }} expires in {{expires_minutes}} minutes", variables)
	if err != nil || subject != "Your code" || body != "Code 731942 expires in 10 minutes" {
		t.Fatalf("late render subject=%q body=%q err=%v", subject, body, err)
	}
	if _, _, err = renderNotificationTemplate("{{code}}", "{{missing}}", map[string]any{"code": "731942"}); err == nil {
		t.Fatal("missing OTP template variable must permanently fail rendering")
	}
}
