package gemtranscribe

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestTranscribe(t *testing.T) {
	godotenv.Overload()
	proxyURL, err := url.Parse(os.Getenv("SOCKS5_PROXY"))
	if err != nil {
		log.Fatal(err)
	}
	httpProxy := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	c, err := NewClientWithProxy(os.Getenv("GOOGLE_API_KEY"), httpProxy)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	audioBytes, _ := os.ReadFile("testdata.mp3")

	resp, err := Transcribe(context.Background(), c, "gemini-2.5-flash", "", "wav", audioBytes, "json")
	if err != nil {
		t.Fatalf("Failed to transcribe: %v", err)
	}

	for _, s := range resp.Segments {
		fmt.Println(s.Start, s.End, s.Text)
	}
	fmt.Println(resp)
}
