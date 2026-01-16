package teapot

// RenderLogger is called to log render timing information.
// Set this to enable render logging.
var RenderLogger func(layer string, durationMs float64)
