# GEMINIPROXY

A proxy for Gemini API.

## Features
- Proxy Gemini API
- Proxy OpenAI API
- Support OpenAI style transcription
- Support HTTP_PROXY


## Build

1. Go
```bash
CGO_ENABLED=0 go build .
```
2. Docker
```bash
docker build -t geminiproxy .
```

## Usage

1. Binary
```bash
./geminiproxy --proxy=http://127.0.0.1:1080
```
2. Docker
- use proxy
```bash
docker run -it -p 8085:8085 ghcr.io/byebyebruce/geminiproxy --proxy=http://127.0.0.1:1080
```
- no proxy
```bash
docker run -it -p 8085:8085 ghcr.io/byebyebruce/geminiproxy 
```

## Test

```bash
curl http://localhost:8085/v1/audio/transcriptions \
  -H "Authorization: Bearer $GEMINI_API_KEY" \
  -H "Content-Type: multipart/form-data" \
  -F file="@/path/to/file/audio.mp3" \
  -F model="gemini-2.5-flash"
```

## License
MIT
