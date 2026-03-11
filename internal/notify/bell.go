package notify

import (
	"fmt"
	"os"
)

// Bell sends a terminal bell character to stdout.
func Bell() {
	fmt.Fprint(os.Stdout, "\a")
}
