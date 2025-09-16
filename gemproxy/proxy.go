package gemproxy

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/byebyebruce/geminiproxy/gemtranscribe"
)

const (
	OpenAIPrefix = "/v1/"
	GeminiURL    = "https://generativelanguage.googleapis.com"
	OpenAIURL    = "https://generativelanguage.googleapis.com/v1beta/openai"
)

var (
	geminiURL *url.URL
	openaiURL *url.URL
)

func init() {
	var err error
	geminiURL, err = url.Parse(GeminiURL)
	if err != nil {
		panic(err)
	}
	openaiURL, err = url.Parse(OpenAIURL)
	if err != nil {
		panic(err)
	}
}

// IsOpenAI checks if the path is an OpenAI path.
func IsOpenAI(path string) bool {
	return strings.HasPrefix(path, OpenAIPrefix)
}

func newReverseProxy(targetUrl *url.URL, pathFn func(path string) string) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	// 修改请求前的处理
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 你可以在这里处理响应，比如日志记录
		return nil
	}
	proxy.Director = func(req *http.Request) {
		// 保留原始路径和查询参数
		req.URL.Scheme = targetUrl.Scheme
		req.URL.Host = targetUrl.Host
		// 保证路径不会重复
		path := req.URL.Path
		if pathFn != nil {
			path = pathFn(path)
		}
		req.URL.Path = singleJoiningSlash(targetUrl.Path, path)
		//fmt.Println("req.URL.Path", req.URL.Path)
		// 解析 apikey

		// 如果需要把 apikey 换成 Authorization 之类，也可以在这里修改
		// req.Header.Set("Authorization", "Bearer "+apikey)

		// 可选：删除 Host 头（或按需设置）
		req.Host = targetUrl.Host
	}
	return proxy
}

// 工具函数，合并路径
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// NewOpenHandler creates a new OpenAI handler.
func NewOpenHandler(transport *http.Transport) http.HandlerFunc {
	proxy := newReverseProxy(openaiURL, func(path string) string {
		return strings.TrimPrefix(path, OpenAIPrefix)
	})
	if transport != nil {
		proxy.Transport = transport
	}
	return func(w http.ResponseWriter, r *http.Request) {
		// 如果路径包含 /audio/transcriptions，则调用 transcribe 函数
		if strings.Contains(r.URL.Path, "/audio/transcriptions") {
			transcribe(w, r, transport)
		} else {
			proxy.ServeHTTP(w, r)
		}
	}
}

// NewGeminiHandler creates a new Gemini handler.
func NewGeminiHandler(transport *http.Transport) http.HandlerFunc {
	proxy := newReverseProxy(geminiURL, nil)
	if transport != nil {
		proxy.Transport = transport
	}
	return proxy.ServeHTTP
}

/*
  - curl --request POST \
    --url https://api.openai.com/v1/audio/transcriptions \
    --header "Authorization: Bearer $OPENAI_API_KEY" \
    --header 'Content-Type: multipart/form-data' \
    --form file=@/path/to/file/audio.mp3 \
    --form model=gpt-4o-transcribe
*/
//https://platform.openai.com/docs/guides/speech-to-text
func transcribe(w http.ResponseWriter, r *http.Request, transport *http.Transport) {
	apiKey := r.Header.Get("Authorization")
	if apiKey == "" {
		return
	}
	apiKey = strings.TrimPrefix(apiKey, "Bearer ")
	if apiKey == "" {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	cli, err := gemtranscribe.NewClientWithProxy(apiKey, transport)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Parse multipart form data
	err = r.ParseMultipartForm(100 << 20) // 100MB max memory
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No file uploaded")
		return
	}
	defer file.Close()

	// Read file bytes
	audioBytes, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read file")
		return
	}

	// Get file extension to determine audio type
	filename := header.Filename
	ext := strings.ToLower(filepath.Ext(filename))
	audioType := strings.TrimPrefix(ext, ".")
	model := r.FormValue("model")
	prompt := r.FormValue("prompt")
	responseFormat := r.FormValue("response_format")
	if responseFormat == "" {
		responseFormat = "json" // default format
	}
	if audioType == "" {
		audioType = "mp3" // default
	}

	resp, err := gemtranscribe.Transcribe(r.Context(), cli, model, prompt, audioType, audioBytes, responseFormat)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode response", "error", err)
		return
	}
}

func writeError(w http.ResponseWriter, status int, err string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err})
}
