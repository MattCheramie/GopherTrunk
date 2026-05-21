package broadcast

import (
	"bytes"
	"mime/multipart"
)

// multipartField is one entry in a multipart/form-data body. When
// Filename is non-empty the entry is written as a file part carrying
// Data; otherwise it is a plain text field carrying Value.
type multipartField struct {
	Name     string
	Value    string
	Filename string
	Data     []byte
}

// buildMultipart assembles a multipart/form-data body from fields and
// returns the encoded buffer together with the Content-Type header
// value (which carries the generated boundary).
func buildMultipart(fields []multipartField) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range fields {
		if f.Filename != "" {
			part, err := w.CreateFormFile(f.Name, f.Filename)
			if err != nil {
				return nil, "", err
			}
			if _, err := part.Write(f.Data); err != nil {
				return nil, "", err
			}
			continue
		}
		if err := w.WriteField(f.Name, f.Value); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}
