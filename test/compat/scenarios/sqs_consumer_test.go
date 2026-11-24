package scenarios

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/centrifugal/protocol"
	h "github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestCS_CONS_04(t *testing.T) {
	h.Profiles(t, "cloud")
	h.Run(t, "CS-CONS-04", func(s *h.Session) {
		aws := func(args ...string) string {
			s.T.Helper()
			return strings.TrimSpace(s.Exec("localstack", append([]string{"awslocal", "--region", "us-east-1"}, args...)...))
		}
		direct := aws("sqs", "create-queue", "--queue-name", "compat-direct", "--query", "QueueUrl", "--output", "text")
		envelopes := aws("sqs", "create-queue", "--queue-name", "compat-sns", "--query", "QueueUrl", "--output", "text")
		queueARN := aws("sqs", "get-queue-attributes", "--queue-url", envelopes, "--attribute-names", "QueueArn",
			"--query", "Attributes.QueueArn", "--output", "text")
		topicARN := aws("sns", "create-topic", "--name", s.Ch("compat-events"), "--query", "TopicArn", "--output", "text")
		defer aws("sns", "delete-topic", "--topic-arn", topicARN)
		policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"sns.amazonaws.com"},`+
			`"Action":"sqs:SendMessage","Resource":%q,"Condition":{"ArnEquals":{"aws:SourceArn":%q}}}]}`, queueARN, topicARN)
		attrs, err := json.Marshal(map[string]string{"Policy": policy})
		if err != nil {
			s.T.Fatal(err)
		}
		aws("sqs", "set-queue-attributes", "--queue-url", envelopes, "--attributes", string(attrs))
		aws("sns", "subscribe", "--topic-arn", topicARN, "--protocol", "sqs", "--notification-endpoint", queueARN)

		channel := s.Ch("hist:aws")
		ws := connect(s, h.HS256(h.Claims{Sub: "aws-reader"}))
		defer ws.Close()
		ws.Send(fmt.Sprintf(`{"id":2,"subscribe":{"channel":%q}}`, channel))
		ws.Recv(1)
		var subscribed protocol.Reply
		ws.Last(&subscribed)
		if subscribed.GetSubscribe() == nil {
			s.T.Fatalf("subscribe %s: %v", channel, &subscribed)
		}
		for i, queue := range []string{direct, envelopes} {
			data := fmt.Sprintf(`{"n":%d}`, i+1)
			payload := fmt.Sprintf(`{"channel":%q,"data":%s}`, channel, data)
			if i == 0 {
				aws("sqs", "send-message", "--queue-url", queue, "--message-body", payload,
					"--message-attributes", `{"centrifugo-method":{"DataType":"String","StringValue":"publish"}}`)
			} else {
				aws("sns", "publish", "--topic-arn", topicARN, "--message", `{"method":"publish","payload":`+payload+"}")
			}
			ws.Recv(1)
			var reply protocol.Reply
			ws.Last(&reply)
			push := reply.GetPush()
			pub := push.GetPub()
			if push.GetChannel() != channel || pub.GetOffset() != uint64(i+1) || string(pub.GetData()) != data {
				s.T.Fatalf("unexpected delivery from queue %s: %v", queue, &reply)
			}

			// A received message remains in flight until the consumer deletes it.
			deadline := time.Now().Add(15 * time.Second)
			for {
				raw := aws("sqs", "get-queue-attributes", "--queue-url", queue, "--attribute-names",
					"ApproximateNumberOfMessages", "ApproximateNumberOfMessagesNotVisible", "ApproximateNumberOfMessagesDelayed",
					"--query", "Attributes", "--output", "json")
				var counts map[string]string
				if err := json.Unmarshal([]byte(raw), &counts); err != nil {
					s.T.Fatalf("queue attributes: %v: %s", err, raw)
				}
				if counts["ApproximateNumberOfMessages"] == "0" && counts["ApproximateNumberOfMessagesNotVisible"] == "0" &&
					counts["ApproximateNumberOfMessagesDelayed"] == "0" {
					break
				}
				if time.Now().After(deadline) {
					s.T.Fatalf("queue %s still contains messages after delivery: %s", queue, raw)
				}
				time.Sleep(150 * time.Millisecond)
			}
			s.Note(fmt.Sprintf("publication %d delivered and its SQS queue drained", pub.GetOffset()))
		}
	})
}
