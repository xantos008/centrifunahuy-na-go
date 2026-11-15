// Copyright (c) Centrifugal Labs LTD
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.
//
// Modified by the Cedrazeka authors, 2026. See NOTICE.
package main

import (
	"github.com/xantos008/centrifunahuy-na-go/internal/app"
	"github.com/xantos008/centrifunahuy-na-go/internal/cli"
)

//go:generate internal/gen/api/generate.sh
func main() {
	root := app.Centrifugo(app.Options{})
	// Register helper CLI.
	// See `centrifugo -h` and https://centrifugal.dev/docs/server/console_commands.
	root.AddCommand(
		cli.Version(), cli.CheckConfig(), cli.GenConfig(), cli.GenToken(),
		cli.GenSubToken(), cli.CheckToken(), cli.CheckSubToken(), cli.DefaultConfig(),
		cli.DefaultEnv(), cli.ConfigDoc(), cli.Serve(),
	)
	_ = root.Execute()
}
