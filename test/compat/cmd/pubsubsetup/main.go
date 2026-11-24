// Command pubsubsetup provisions the application-owned Pub/Sub resources
// before the reference and candidate start their streaming pull clients.
package main

import (
	"context"
	"log"
	"time"

	"github.com/xantos008/centrifunahuy-na-go/test/compat/internal/pubsubfixture"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for _, side := range []string{"ref", "sut"} {
		if err := pubsubfixture.Provision(ctx, side); err != nil {
			log.Fatalf("%s Pub/Sub setup: %v", side, err)
		}
	}
}
