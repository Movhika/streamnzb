package paths

import (
	"os"
)

func GetDataDir() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {

		return "/app/data"
	}
	return "."
}
