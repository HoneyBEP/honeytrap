package files

import (
	"net/http"
)

func Download(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	body := make([]byte, 1024)
	n, err := resp.Body.Read(body)

	fc, err := NewFileCloser(path)
	if err != nil {
		return err
	}
	defer fc.Close()

	_, err = fc.Write(body[:n])
	if err != nil {
		return err
	}

	return nil
}