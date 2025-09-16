package gemtranscribe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

var defaultPrompt = `Transcribe the following audio clip. 

Requirements:
- Do not return word-level, only segment-level.
- Make sure segment boundaries are logical and natural.
- For each segment, provide the start and end timestamps in seconds. Return the result as a JSON array, where each object contains:
	- "text": the transcribed segment,
	- "start": the start time (in seconds),
	- "end": the end time (in seconds).

Example output:
{
	"language": "english", // french, spanish, chinese, etc.
	"duration": 48.47,
	"segments": [
		...
		{
			"start": 10.96,
			"end": 16.63,
			"text": "What are some class jobs and why are they important?"
		},
		{
			"start": 19.33,
			"end": 23.99,
			"text": "Students in my class have jobs to do."
		},
		...
	]
}
Only return valid JSON.
`

type TranscribeResult = openai.AudioResponse

// Transcribe returns the transcription of the audio file.
// audioType "mp3", "wav", "m4a", "ogg", "webm"
// format "json", "verbose_json", "text", "srt"
// https://platform.openai.com/docs/api-reference/audio/createTranscription
//
// Example:
// resp, err := Transcribe(context.Background(), c, "gemini-2.5-flash", "", "mp3", audioBytes, openai.AudioResponseFormatSRT)
func Transcribe(ctx context.Context, cli *genai.Client, model string, userPrompt string, audioType string, audioBytes []byte, format string) (*openai.AudioResponse, error) {
	p := defaultPrompt
	if len(userPrompt) > 0 {
		p = userPrompt + "\n\n------------\n" + defaultPrompt
	}

	mimeType := "audio/" + audioType
	parts := []*genai.Part{
		genai.NewPartFromText(p),
		&genai.Part{
			InlineData: &genai.Blob{
				MIMEType: mimeType,
				Data:     audioBytes,
			},
		},
	}
	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}
	result, err := cli.Models.GenerateContent(
		ctx,
		model,
		contents,
		nil,
	)
	if err != nil {
		return nil, err
	}
	text := trimMarkdown(result.Text())
	if text == "" {
		return nil, fmt.Errorf("")
	}

	var resp openai.AudioResponse
	err = json.Unmarshal([]byte(text), &resp)
	if err != nil {
		return nil, err
	}

	resp_format := openai.AudioResponseFormat(format)
	switch resp_format {
	case openai.AudioResponseFormatJSON, openai.AudioResponseFormatVerboseJSON, openai.AudioResponseFormatText:
		text := ""
		for _, segment := range resp.Segments {
			text += segment.Text + " "
		}
		resp.Text = text
		if resp_format == openai.AudioResponseFormatText {
			resp.Segments = nil
		} else {
			for i := range resp.Segments {
				resp.Segments[i].ID = i
			}
		}
		return &resp, nil
	case openai.AudioResponseFormatSRT:
		resp.Text = sentence2SRT(&resp)
		resp.Segments = nil
		return &resp, nil
	default:
		return &resp, fmt.Errorf("unsupported format: %s", format)
	}
}
