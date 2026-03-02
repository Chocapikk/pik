package sdk

// Multipart builds a multipart/form-data body from named parts (unordered).
// Returns the body string and content-type header.
func Multipart(parts map[string]string) (string, string) {
	boundary := "----pik" + RandTextDefault(16)
	var body string
	for name, value := range parts {
		body += "--" + boundary + "\r\nContent-Disposition: form-data; name=\"" + name + "\"\r\n\r\n" + value + "\r\n"
	}
	body += "--" + boundary + "--"
	return body, "multipart/form-data; boundary=" + boundary
}

// MultipartOrdered builds a multipart/form-data body from ordered name-value pairs.
// Parts are provided as alternating name, value strings.
// Returns the body string and content-type header.
func MultipartOrdered(boundary string, parts ...string) (string, string) {
	if len(parts)%2 != 0 {
		panic("MultipartOrdered: odd number of arguments")
	}
	var body string
	for i := 0; i < len(parts); i += 2 {
		body += "--" + boundary + "\r\nContent-Disposition: form-data; name=\"" + parts[i] + "\"\r\n\r\n" + parts[i+1] + "\r\n"
	}
	body += "--" + boundary + "--"
	return body, "multipart/form-data; boundary=" + boundary
}
