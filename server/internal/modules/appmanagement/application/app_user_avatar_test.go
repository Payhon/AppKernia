package application

import (
	"testing"

	"github.com/google/uuid"
)

func TestAdminAppUserAvatarURL(t *testing.T) {
	appID := uuid.MustParse("123e4567-e89b-42d3-a456-426614174000")
	userID := uuid.MustParse("223e4567-e89b-42d3-a456-426614174000")
	fileID := uuid.MustParse("323e4567-e89b-42d3-a456-426614174000")
	if got := adminAppUserAvatarURL(appID, userID, nil); got != nil {
		t.Fatalf("adminAppUserAvatarURL(nil)=%q", *got)
	}
	want := "/apps/123e4567-e89b-42d3-a456-426614174000/users/223e4567-e89b-42d3-a456-426614174000/avatar/content?v=323e4567-e89b-42d3-a456-426614174000"
	got := adminAppUserAvatarURL(appID, userID, &fileID)
	if got == nil || *got != want {
		t.Fatalf("adminAppUserAvatarURL()=%v want=%q", got, want)
	}
}
