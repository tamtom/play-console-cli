package audit

import "os"

// appendRaw writes raw bytes to path; used in tests to simulate malformed rows.
func appendRaw(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
