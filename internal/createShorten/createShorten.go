package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const endpointURL = "http://localhost:8080"

func main() {
	originalURL := ""

	for {
		log.Println("Insert originalURL")
		fmt.Fscan(os.Stdin, &originalURL)

		bodyReader := bytes.NewReader([]byte(originalURL))
		req, err := http.NewRequest(http.MethodPost, endpointURL+"/", bodyReader)
		if err != nil {
			log.Fatalln(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalln(err)
		}

		log.Println(resp)

		if resp.StatusCode == http.StatusCreated {
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}

			shortURL := string(bodyBytes)
			log.Println("Short URL is", shortURL)
		}

		resp.Body.Close()
	}
}
