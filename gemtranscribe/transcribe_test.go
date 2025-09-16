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

	audioBytes, _ := os.ReadFile("test.mp3")

	resp, err := Transcribe(context.Background(), c, "gemini-2.5-flash", "", "mp3", audioBytes, "json")
	if err != nil {
		t.Fatalf("Failed to transcribe: %v", err)
	}
	fmt.Println(resp)
}
