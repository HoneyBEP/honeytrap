package files

import (
	"net/http"
	"fmt"
	"os"
	"io/ioutil"
)

func Download(url string, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	body := make([]byte, 1024)
	n, err := resp.Body.Read(body)

	err = ioutil.WriteFile(fmt.Sprintf("%s", path), body[:n], os.FileMode(os.O_RDWR))
	if err != nil {
		return err
	}

	return nil
}