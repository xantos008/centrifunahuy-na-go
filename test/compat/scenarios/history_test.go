package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"

	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

type position struct {
	Offset uint64 `json:"offset"`
	Epoch  string `json:"epoch"`
}

// CS-HIST-01: history API: position only, limit, since, reverse, oversized limit.
func TestCS_HIST_01(t *testing.T) {
	h.Run(t, "CS-HIST-01", func(s *h.Session) {
		ch := s.Ch("hist:h1")
		var last struct {
			Result position `json:"result"`
		}
		for i := 1; i <= 3; i++ {
			body := s.API(h.APIRequest{Path: "/api/publish", Body: fmt.Sprintf(`{"channel":%q,"data":{"n":%d}}`, ch, i)})
			if err := json.Unmarshal(body, &last); err != nil {
				s.T.Fatal(err)
			}
		}
		hist := func(extra string) {
			s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q%s}`, ch, extra)})
		}
		hist(``)            // position only
		hist(`,"limit":2`)  // first two
		hist(`,"limit":-1`) // all
		hist(fmt.Sprintf(`,"limit":10,"since":{"offset":1,"epoch":%q}`, last.Result.Epoch))
		hist(`,"limit":2,"reverse":true`)
		hist(`,"limit":1000`) // above any page limit
		hist(`,"limit":10,"since":{"offset":1,"epoch":"wrong-epoch"}`)
		s.API(h.APIRequest{Path: "/api/history_remove", Body: fmt.Sprintf(`{"channel":%q}`, ch)})
		hist(`,"limit":10`) // empty, top position kept
	})
}
