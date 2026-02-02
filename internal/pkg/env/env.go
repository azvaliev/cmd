package env

import "os"

var DEBUG = func() bool {
	debug := os.Getenv("DEBUG")

	return debug == "true" || debug == "1"
}()
