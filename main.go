// Command snxplore is a read-first, agent-native CLI for understanding an
// arbitrary ServiceNow instance through the documented Now Platform Table API.
package main

import (
	"os"

	"github.com/icymoonray-ui/snxplore/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
