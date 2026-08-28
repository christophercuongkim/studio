// Command studio is the single binary covering the full life of a YouTube
// video: new → ingest → serve → apply → scaffold → chapters → qc → thumbs →
// upload → archive, plus search. This file is dispatch only; every command's
// logic lives in internal/<area> and is wired up through internal/cli.
package main

import (
	"os"

	"github.com/christophercuongkim/studio/internal/cli"
)

// version is overridable at build time via -ldflags "-X main.version=…".
var version = "dev"

func main() {
	os.Exit(cli.Run(version, os.Args[1:]))
}
