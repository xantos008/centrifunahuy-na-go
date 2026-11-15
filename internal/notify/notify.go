// Modified by the Cedrazeka authors, 2026. See NOTICE.

package notify

import (
	"github.com/centrifugal/centrifuge"
	"github.com/xantos008/centrifunahuy-na-go/internal/usage"
)

func RegisterHandlers(node *centrifuge.Node, statsSender *usage.Sender) {
	handlers := map[string]centrifuge.NotificationHandler{
		usage.LastSentUpdateNotificationOp: func(event centrifuge.NotificationEvent) {
			if statsSender == nil {
				// Different configurations on different nodes.
				return
			}
			statsSender.UpdateLastSentAt(event.Data)
		},
	}
	node.OnNotification(func(event centrifuge.NotificationEvent) {
		h, ok := handlers[event.Op]
		if !ok {
			return
		}
		go h(event)
	})
}
