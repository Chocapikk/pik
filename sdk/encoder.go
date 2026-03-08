package sdk

// Encoder describes a registered payload encoder.
type Encoder struct {
	Name     string
	Platform string // "linux", "windows", "" (any)
	Desc     string
	Fn       func(string) string
}

var encoders []*Encoder

// RegisterEncoder registers a payload encoder.
func RegisterEncoder(e *Encoder) {
	encoders = append(encoders, e)
}

// ListEncoders returns all encoders compatible with the given platform.
func ListEncoders(platform string) []*Encoder {
	var result []*Encoder
	for _, e := range encoders {
		if e.Platform == "" || e.Platform == platform || platform == "" {
			result = append(result, e)
		}
	}
	return result
}

// EncoderNames returns all encoder names for the given platform.
func EncoderNames(platform string) []string {
	enc := ListEncoders(platform)
	names := make([]string, len(enc))
	for i, e := range enc {
		names[i] = e.Name
	}
	return names
}

// GetEncoder returns an encoder by name, or nil if not found.
func GetEncoder(name string) *Encoder {
	for _, e := range encoders {
		if e.Name == name {
			return e
		}
	}
	return nil
}
