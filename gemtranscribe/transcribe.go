package gemtranscribe

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/sashabaranov/go-openai"
	"google.golang.org/genai"
)

var defaultPrompt = `Transcribe the following audio clip. 

Requirements:
- Do not return word-level, only segment-level.
- Make sure segment boundaries are logical and natural.
- For each segment, provide the start and end timestamps the format of timecode. Return the result as a JSON array, where each object contains:
	- "text": the transcribed segment,
	- "start": the start time (in timecode),
	- "end": the end time (in timecode).

Example output:
{
	"language": "english", // french, spanish, chinese, etc.
	"length": "00:10:48,47",
	"segments": [
		...
		{
			"start": "00:00:10,96",
			"end": "00:00:16,63",
			"text": "What are some class jobs and why are they important?"
		},
		{
			"start": "00:00:19,33",
			"end": "00:00:23,99",
			"text": "Students in my class have jobs to do."
		},
		...
	]
}
Only return valid JSON.
`

// NOTE: why don't return timestamp? because the timestamp is not accurate.
// only "00:00:19,33" format is accurate
type respJson struct {
	Language string `json:"language"`
	Length   string `json:"length"`
	Segments []struct {
		Start string `json:"start"`
		End   string `json:"end"`
		Text  string `json:"text"`
	} `json:"segments"`
}
type RespSegment struct {
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
	Transient        bool    `json:"transient"`
}
type AudioResponse struct {
	Task     string        `json:"task"`
	Language string        `json:"language"`
	Duration float64       `json:"duration"`
	Segments []RespSegment `json:"segments"`
	Words    []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"words"`
	Text string `json:"text"`
}

type _ = openai.AudioResponse

// Transcribe returns the transcription of the audio file.
// audioType "mp3", "wav", "m4a", "ogg", "webm"
// format "json", "verbose_json", "text", "srt"
// https://platform.openai.com/docs/api-reference/audio/createTranscription
//
// Example:
// resp, err := Transcribe(context.Background(), c, "gemini-2.5-flash", "", "mp3", audioBytes, openai.AudioResponseFormatSRT)
func Transcribe(ctx context.Context, cli *genai.Client, model string, userPrompt string, audioType string, audioBytes []byte, format string) (*AudioResponse, error) {
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

	var _resp respJson
	err = json.Unmarshal([]byte(text), &_resp)
	if err != nil {
		return nil, err
	}
	duration, err := timecode2Seconds(_resp.Length)
	if err != nil {
		return nil, err
	}
	resp := AudioResponse{
		Language: _resp.Language,
		Duration: duration,
		Segments: make([]RespSegment, len(_resp.Segments)),
	}
	for i, segment := range _resp.Segments {
		resp.Segments[i].ID = i
		resp.Segments[i].Start, err = timecode2Seconds(segment.Start)
		if err != nil {
			return nil, err
		}
		resp.Segments[i].End, err = timecode2Seconds(segment.End)
		if err != nil {
			return nil, err
		}
		resp.Segments[i].Text = segment.Text
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

// 00:00:13,99 -> 13.99
func timecode2Seconds(timecode string) (float64, error) {
	parts := strings.SplitN(timecode, ":", 3)
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid timecode: %s", timecode)
	}
	hours := parts[0]
	hoursInt, err := strconv.Atoi(hours)
	if err != nil {
		return 0, fmt.Errorf("invalid hours: %s", hours)
	}
	minutes := parts[1]
	minutesInt, err := strconv.Atoi(minutes)
	if err != nil {
		return 0, fmt.Errorf("invalid minutes: %s", minutes)
	}
	seconds := parts[2]
	seconds = strings.ReplaceAll(seconds, ",", ".")
	// 13,99
	secondsFloat, err := strconv.ParseFloat(seconds, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid seconds: %s", seconds)
	}
	return float64(hoursInt*3600+minutesInt*60) + secondsFloat, nil
}
