package repository

import (
	"testing"

	"github.com/google/uuid"
)

func TestMobileAvatarURL(t *testing.T) {
	if got := mobileAvatarURL(nil); got != nil {
		t.Fatalf("mobileAvatarURL(nil)=%q", *got)
	}
	id := uuid.MustParse("123e4567-e89b-42d3-a456-426614174000")
	got := mobileAvatarURL(&id)
	want := "/api/v1/public/content/assets/123e4567-e89b-42d3-a456-426614174000"
	if got == nil || *got != want {
		t.Fatalf("mobileAvatarURL()=%v want=%q", got, want)
	}
}
