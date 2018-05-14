package files

import (
	"net/http"
	"fmt"
	"os"
	"crypto/sha256"
)

func Download(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	body := make([]byte, 1024)
	n, err := resp.Body.Read(body)

	// hash filename
	h := sha256.New()
	h.Write([]byte(url))
	filename := fmt.Sprintf("%x", h.Sum(nil))

	text := fmt.Sprintf("%s\n\n%s\n", url, body[:n])

	f, err := os.OpenFile(fmt.Sprintf("%s/%s.txt", path, filename), os.O_APPEND|os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	if err != nil {
		return err
	}

	return nil
}