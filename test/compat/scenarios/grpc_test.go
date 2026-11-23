package scenarios

import (
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
	"google.golang.org/grpc/codes"
)

func TestCS_API_03(t *testing.T) {
	h.Profiles(t, "m")
	h.Run(t, "CS-API-03", func(s *h.Session) {
		call := s.GRPC()
		call("info", "{}", false, codes.Unauthenticated)
		for _, c := range apiCases(s) {
			call(c.method, c.valid, true, codes.OK)
			call(c.method, c.invalid, true, codes.OK)
		}
		call("info", "{}", true, codes.OK)
		for _, method := range []string{
			"connections", "update_user_status", "get_user_status", "delete_user_status",
			"block_user", "unblock_user", "revoke_token", "invalidate_user_tokens",
			"device_register", "device_update", "device_remove", "device_list",
			"device_topic_list", "device_topic_update", "user_topic_list", "user_topic_update",
			"send_push_notification", "update_push_status", "cancel_push",
		} {
			call(method, "{}", true, codes.Unimplemented)
			s.API(h.APIRequest{Path: "/api/" + method, Body: "{}"})
		}
	})
}
