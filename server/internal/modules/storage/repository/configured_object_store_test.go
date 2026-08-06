package repository

import (
	"encoding/json"
	"testing"
)

func TestNormalizeEndpointAndObjectKeySafety(t *testing.T) {
	host, secure, err := normalizeEndpoint("https://s3.example.test", false)
	if err != nil || host != "s3.example.test" || !secure {
		t.Fatalf("normalized endpoint = %q secure=%v err=%v", host, secure, err)
	}
	if _, _, err = normalizeEndpoint("https://user:secret@s3.example.test/path", true); err == nil {
		t.Fatal("endpoint with credentials and path must be rejected")
	}
	for _, key := range []string{"../escape", "/absolute", `bad\key`} {
		if validObjectKey(key) {
			t.Fatalf("unsafe object key accepted: %q", key)
		}
	}
	if !validObjectKey("appkernia/files/tenant/object") {
		t.Fatal("safe object key rejected")
	}
}

func TestPolicyValueParsingIsBoundedAndDeduplicated(t *testing.T) {
	values := map[string]json.RawMessage{
		"limit": []byte(`209715200`),
		"types": []byte(`["image/png","image/png"," text/plain ",""]`),
	}
	if got := boundedInt64(values, "limit", 12, 100); got != 12 {
		t.Fatalf("bounded value = %d", got)
	}
	types := stringSlice(values, "types", []string{"fallback"})
	if len(types) != 2 || types[0] != "image/png" || types[1] != "text/plain" {
		t.Fatalf("types = %#v", types)
	}
}

func TestVendorS3ProfilesRequireExpectedEndpointAndRegion(t *testing.T) {
	valid := []struct{ provider, endpoint, region string }{
		{"cos", "cos.ap-guangzhou.myqcloud.com", "ap-guangzhou"},
		{"oss", "oss-cn-hangzhou.aliyuncs.com", "cn-hangzhou"},
		{"qiniu", "s3-cn-east-1.qiniucs.com", "cn-east-1"},
	}
	for _, test := range valid {
		if err := validateProviderProfile(test.provider, test.endpoint, test.region); err != nil {
			t.Fatalf("valid %s profile rejected: %v", test.provider, err)
		}
	}
	invalid := []struct{ provider, endpoint, region string }{
		{"cos", "cos.ap-shanghai.myqcloud.com", "ap-guangzhou"},
		{"oss", "example.com", "cn-hangzhou"},
		{"qiniu", "s3-cn-east-1.qiniucs.com", ""},
	}
	for _, test := range invalid {
		if err := validateProviderProfile(test.provider, test.endpoint, test.region); err == nil {
			t.Fatalf("invalid %s profile accepted: %#v", test.provider, test)
		}
	}
}
