# GEMINIPROXY

A proxy for Gemini API.

## Features
- Proxy Gemini API
- Proxy OpenAI API
- Support OpenAI style transcription
- Support HTTP_PROXY


## Build

```bash
CGO_ENABLED=0 go build .
```

## Usage

```bash
./geminiproxy --proxy=http://127.0.0.1:1080
```
