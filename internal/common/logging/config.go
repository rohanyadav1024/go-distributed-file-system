package logging

// Config controls logger mode and minimum log level.
type Config struct {
	Level      string
	Production bool
}
