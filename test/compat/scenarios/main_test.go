// Package scenarios holds the compatibility scenarios (CS-* in
// docs/testing/compatibility-suite.md). Run them with `make compat`.
package scenarios

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/xantos008/centrifunahuy-na-go/test/compat/harness"
)

func TestMain(m *testing.M) {
	for _, err := range []error{harness.CheckProfile(), harness.CheckRecordList()} {
		if err != nil {
			fmt.Fprintln(os.Stderr, "compat:", err)
			os.Exit(1)
		}
	}
	var need []harness.Target
	switch harness.Mode() {
	case "diff":
		need = append(harness.RefNodes(), harness.SUTNodes()...)
	case "golden":
		need = harness.SUTNodes()
	case "record":
		need = harness.RefNodes()
	}
	for _, tg := range need {
		if err := harness.WaitReady(tg, 30*time.Second); err != nil {
			fmt.Fprintln(os.Stderr, "compat:", err, "- start the targets with `make compat-up`")
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}
