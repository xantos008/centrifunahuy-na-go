package scenarios

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_PROXY_15(t *testing.T) {
	h.Profiles(t, "m", "r")
	h.Run(t, "CS-PROXY-15", func(s *h.Session) {
		channel := s.Ch("proxystate:document")
		reader := subscribeOn(s, fmt.Sprintf(`{"channel":%q,"delta":"fossil"}`, channel))
		writer := connect(s, h.HS256(h.Claims{Sub: "proxy-user"}))
		for i, change := range []struct {
			revision int
			epoch    string
			key      string
		}{{1, "first", ""}, {2, "first", ""}, {1, "first", ""},
			{1, "second", ""}, {0, "", "same"}, {0, "", "same"}} {
			data := fmt.Sprintf(`{"revision":%d,"epoch":%q,"key":%q,"sequence":%d,"text":"A sample document with a stable paragraph and a small revision."}`, change.revision, change.epoch, change.key, i+1)
			writer.Send(fmt.Sprintf(`{"id":%d,"publish":{"channel":%q,"data":%s}}`, i+2, channel, data))
			writer.Recv(1)
			var ack protocol.Reply
			writer.Last(&ack)
			if ack.Publish == nil || ack.Error != nil {
				s.T.Fatalf("publish %d: %v", i+1, &ack)
			}
		}
		for i := 0; i < 4; i++ {
			reader.Recv(1)
			var push protocol.Reply
			reader.Last(&push)
			pub := push.GetPush().GetPub()
			if pub.GetOffset() != uint64(i+1) || (i == 1 && !pub.GetDelta()) {
				s.T.Fatalf("unexpected offset or delta on publication %d: %v", i+1, &push)
			}
		}
		body := s.API(h.APIRequest{Path: "/api/history", Body: fmt.Sprintf(`{"channel":%q,"limit":8}`, channel)})
		var history struct {
			Result struct {
				Publications []struct {
					Data struct{ Sequence int }
				}
			}
		}
		if err := json.Unmarshal(body, &history); err != nil {
			s.T.Fatal(err)
		}
		want := []int{1, 2, 4, 5}
		stored := history.Result.Publications
		if len(stored) != len(want) {
			s.T.Fatalf("unexpected stored publications: %s", body)
		}
		for i, publication := range stored {
			if publication.Data.Sequence != want[i] {
				s.T.Fatalf("unexpected stored sequence at %d: %s", i, body)
			}
		}
	})
}
