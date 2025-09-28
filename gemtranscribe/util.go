package gemtranscribe

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"google.golang.org/genai"
)

func trimMarkdown(s string) string {
	s = strings.TrimPrefix(s, "```json\n")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	return s
}

/*
3
00:00:39,770 --> 00:00:41,880
在经历了一场人生巨变之后
When I was lying there in the VA hospital ...

4
00:00:42,550 --> 00:00:44,690
我被送进了退伍军人管理局医院
... with a big hole blown through the middle of my life,
*/
func sentence2SRT(resp *AudioResponse) string {
	lines := make([]string, 0, len(resp.Segments))
	for i, sentence := range resp.Segments {
		line := fmt.Sprintf("%d\n%s\n%s",
			i+1,
			timestamp2time(sentence.Start, sentence.End, resp.Duration),
			sentence.Text,
		)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// 00:00:39,770 --> 00:00:41,880
func timestamp2time(start float64, end float64, total float64) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d --> %02d:%02d:%02d,%03d",
		int(start/3600), int(start/60)%60, int(start)%60, int(start*1000)%1000,
		int(end/3600), int(end/60)%60, int(end)%60, int(end*1000)%1000)
}

func NewClientWithProxy(apiKey string, proxy *http.Transport) (*genai.Client, error) {
	cfg := &genai.ClientConfig{
		APIKey: apiKey,
	}

	if proxy != nil {
		cfg.HTTPClient = &http.Client{
			Transport: proxy,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}
	return client, nil
}
