package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"os/exec"

	"github.com/joho/godotenv"
)

type AskRequest struct {
	Question string `json:"question"`
}

type GeminiRequest struct {
	Question string `json:"question"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
	} `json:"candidates"`
	Error string `json:"error,omitempty"`
}

type AskResponse struct {
	Answer string `json:"answer"`
	Error  string `json:"error,omitempty"`
}

type StreamResponse struct {
	Type    string `json:"type"` // "chunk" atau "done"
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

type CreateAppRequest struct {
	Prompt string `json:"prompt"`
}

type CreateAppResponse struct {
	AppID string `json:"app_id"`
	URL   string `json:"url"`
	Error string `json:"error,omitempty"`
}

type CreateFolderRequest struct {
	Prompt string `json:"prompt"`
}

type CreateFolderResponse struct {
	JobID      string `json:"job_id"`
	FolderName string `json:"folder_name,omitempty"`
	FilePath   string `json:"file_path,omitempty"`
	URL        string `json:"url,omitempty"`
	Error      string `json:"error,omitempty"`
}

type JobStatus struct {
	JobID      string    `json:"job_id"`
	Status     string    `json:"status"` // "pending", "processing", "completed", "failed"
	Progress   string    `json:"progress"`
	FolderName string    `json:"folder_name,omitempty"`
	FilePath   string    `json:"file_path,omitempty"`
	URL        string    `json:"url,omitempty"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type JobStatusRequest struct {
	JobID string `json:"job_id"`
}

// APIKeyManager manages multiple API keys with fallback mechanism
type APIKeyManager struct {
	apiKeys []string
	current int
}

// NewAPIKeyManager creates a new API key manager from .env file
func NewAPIKeyManager() *APIKeyManager {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("[INFO] .env file not found, will use environment variables or defaults")
	}

	var keys []string

	// Try loading from CURSOR_API_KEYS (comma-separated)
	keysEnv := os.Getenv("CURSOR_API_KEYS")
	if keysEnv != "" {
		keyList := strings.Split(keysEnv, ",")
		for _, key := range keyList {
			trimmed := strings.TrimSpace(key)
			if trimmed != "" {
				keys = append(keys, trimmed)
			}
		}
	}

	// Also try individual keys CURSOR_API_KEY_1, CURSOR_API_KEY_2, etc.
	if len(keys) == 0 {
		for i := 1; i <= 10; i++ {
			key := os.Getenv(fmt.Sprintf("CURSOR_API_KEY_%d", i))
			if key != "" {
				keys = append(keys, key)
			}
		}
	}

	// Fallback to hardcoded keys if .env not found
	if len(keys) == 0 {
		log.Println("[WARNING] No API keys found in .env. Using fallback keys.")
		keys = []string{
			"key_96099a14caa94c694678055ebc1cb6253e68cde994e5d8ccff81aaaae1d4c3a2", //makedonald
			"key_48387d5a2592e38c88c096df090c8d9efad0fa5493418d3d1b8a1ab8318f76ed", //sekli
			"key_e4d3b699771d8cd983fa9d6d2b36c0803f84583ee01b698fabd67e58cba504cd", //telen
			"key_afda379ba8eec6608ccbdf880d3c06c1a20bd47ce79da786a47f1f2c42e7b332", //elang
			"key_9804b9282922da61a10f0db176e53f02dbfcab0005c6d0f676d851f7380d7b4b", //akuman
		}
	}

	log.Printf("[INFO] Loaded %d API key(s) for fallback mechanism", len(keys))

	return &APIKeyManager{
		apiKeys: keys,
		current: 0,
	}
}

// GetNextKey gets the next API key in rotation (round-robin for fresh chat)
func (m *APIKeyManager) GetNextKey() string {
	key := m.apiKeys[m.current]
	m.current = (m.current + 1) % len(m.apiKeys)
	return key
}

// GetAllKeys returns all API keys for fallback
func (m *APIKeyManager) GetAllKeys() []string {
	return m.apiKeys
}

var apiKeyManager = NewAPIKeyManager()

// App counter for unique IDs
var appCounter int64
var appCounterMutex sync.Mutex
var counterInitialized bool

// Job tracking for async operations
var jobStore = make(map[string]*JobStatus)
var jobStoreMutex sync.RWMutex

// generateJobID generates a unique job ID
func generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// initializeAppCounter reads existing folders and sets counter to the highest number + 1
func initializeAppCounter() {
	appCounterMutex.Lock()
	defer appCounterMutex.Unlock()

	if counterInitialized {
		return // Already initialized
	}

	generatedAppsDir := "generated-apps"

	// Ensure directory exists
	if err := os.MkdirAll(generatedAppsDir, 0755); err != nil {
		log.Printf("[WARNING] Failed to create generated-apps directory: %v", err)
		appCounter = 0
		counterInitialized = true
		return
	}

	// Read directory
	entries, err := os.ReadDir(generatedAppsDir)
	if err != nil {
		log.Printf("[WARNING] Failed to read generated-apps directory: %v", err)
		appCounter = 0
		counterInitialized = true
		return
	}

	maxNum := int64(0)
	appPattern := regexp.MustCompile(`^app_(\d+)$`)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		matches := appPattern.FindStringSubmatch(entry.Name())
		if len(matches) >= 2 {
			// Parse number from app_XXX
			var num int64
			if _, err := fmt.Sscanf(matches[1], "%d", &num); err == nil {
				if num > maxNum {
					maxNum = num
				}
			}
		}
	}

	appCounter = maxNum
	counterInitialized = true
	log.Printf("[INFO] App counter initialized. Last app number: app_%03d", maxNum)
}

// getNextAppID generates a unique app ID by incrementing counter
func getNextAppID() string {
	appCounterMutex.Lock()
	defer appCounterMutex.Unlock()

	if !counterInitialized {
		initializeAppCounter()
	}

	appCounter++
	return fmt.Sprintf("app_%03d", appCounter)
}

// getNextFolderName generates a unique folder name by incrementing counter
func getNextFolderName() string {
	appCounterMutex.Lock()
	defer appCounterMutex.Unlock()

	if !counterInitialized {
		initializeAppCounter()
	}

	appCounter++
	return fmt.Sprintf("app_%03d", appCounter)
}

// Configuration constants for resource optimization
const (
	// Max concurrent requests: 10
	MaxConcurrentRequests = 10
	// Max buffer size for streaming: 10 MB
	MaxBufferSize = 10 * 1024 * 1024
	// Buffer read size: 4 KB (reduced from 1 KB for better performance)
	BufferReadSize = 4 * 1024
)

// Semaphore untuk membatasi concurrent requests
var requestSemaphore = make(chan struct{}, MaxConcurrentRequests)

// Resource statistics tracking
type ResourceStats struct {
	TotalRequests       uint64    `json:"total_requests"`
	ActiveRequests      int32     `json:"active_requests"`
	TotalErrors         uint64    `json:"total_errors"`
	TotalBytesProcessed uint64    `json:"total_bytes_processed"`
	StartTime           time.Time `json:"start_time"`
	Uptime              string    `json:"uptime"`
}

var stats = ResourceStats{
	StartTime: time.Now(),
}

var statsMutex sync.RWMutex

// Update stats helper functions
func incrementTotalRequests() {
	atomic.AddUint64(&stats.TotalRequests, 1)
}

func incrementActiveRequests() {
	atomic.AddInt32(&stats.ActiveRequests, 1)
}

func decrementActiveRequests() {
	atomic.AddInt32(&stats.ActiveRequests, -1)
}

func incrementTotalErrors() {
	atomic.AddUint64(&stats.TotalErrors, 1)
}

func addBytesProcessed(bytes int) {
	atomic.AddUint64(&stats.TotalBytesProcessed, uint64(bytes))
}

// executeWithFallback executes command with API key fallback mechanism
// Returns output, error, and which API key was used (index)
func executeWithFallback(ctx context.Context, question string) (string, error, int) {
	keys := apiKeyManager.GetAllKeys()
	var lastErr error
	var usedKeyIndex int

	for i, apiKey := range keys {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return "", ctx.Err(), usedKeyIndex
		default:
		}

		usedKeyIndex = i
		templatePrompt := "You are a helpful assistant that can answer questions dont make file or create update just answer as text the this question: "

		// Build args: cursor-agent --api-key <key> <question> -p --output-format text
		args := []string{"--api-key", apiKey, question, templatePrompt, "-p", "--output-format", "text"}

		// Execute with context timeout (will be cancelled if timeout or client disconnect)
		cmd := exec.CommandContext(ctx, "cursor-agent", args...)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err == nil {
			// Check if output contains textual error indicators
			lower := strings.ToLower(outputStr)
			if strings.Contains(lower, "connecterror") || strings.Contains(lower, "resource_exhausted") || strings.Contains(lower, "resource exhausted") ||
				strings.Contains(lower, "rate limit") || strings.Contains(lower, "unauthorized") || strings.Contains(lower, "quota") {
				log.Printf("[WARNING] API key %d returned error text, trying next...", i+1)
				lastErr = fmt.Errorf("API key %d textual error: %s", i+1, outputStr[:min(200, len(outputStr))])
				continue
			}
			log.Printf("[SUCCESS] API key %d worked successfully", i+1)
			return outputStr, nil, i
		}

		// Log detailed failure with truncated output
		trunc := outputStr
		if len(trunc) > 200 {
			trunc = trunc[:200]
		}
		// No timeout checks
		log.Printf("[ERROR] API key %d failed: %v | output: %q", i+1, err, trunc)
		lastErr = fmt.Errorf("API key %d last failure: %v | output: %s", i+1, err, trunc)
	}

	return "", fmt.Errorf("all %d API keys failed. Last error: %v", len(keys), lastErr), usedKeyIndex
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// loggingMiddleware logs request details
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get client IP
		clientIP := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		// Log incoming request
		log.Printf("[REQUEST] %s %s from %s", r.Method, r.URL.Path, clientIP)

		// Create a response writer wrapper to capture status code
		lw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lw, r)

		// Log response
		duration := time.Since(start)
		log.Printf("[RESPONSE] %s %s - Status: %d - Duration: %v", r.Method, r.URL.Path, lw.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush delegates to the underlying http.Flusher when available
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// compressionMiddleware adds gzip compression for responses
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func compressionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
	})
}

// cors middleware adds CORS headers and handles preflight requests
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept-Encoding")
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func askHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Acquire semaphore (limit concurrent requests)
	select {
	case requestSemaphore <- struct{}{}:
		incrementTotalRequests()
		incrementActiveRequests()
		defer func() {
			<-requestSemaphore
			decrementActiveRequests()
		}() // Release on return
	case <-r.Context().Done():
		// Client disconnected before acquiring semaphore
		return
	}

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask - Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] /ask - Question: %s", req.Question)

	// Use client request context only (no server-side timeout)
	ctx := r.Context()

	// Use fallback mechanism with context (supports timeout and cancellation)
	output, err, usedKeyIndex := executeWithFallback(ctx, req.Question)

	resp := AskResponse{
		Answer: output,
	}

	if err != nil {
		log.Printf("[ERROR] /ask - Limit telah habis, silahkan upgrade ke premium untuk melanjutkan penggunaan. Last error: %v", err)
		resp.Error = err.Error()
		if err == context.DeadlineExceeded {
			w.WriteHeader(http.StatusRequestTimeout)
			resp.Error = "Request timeout after 60 seconds"
		} else if err == context.Canceled {
			w.WriteHeader(http.StatusRequestTimeout)
			resp.Error = "Request cancelled"
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		addBytesProcessed(len(output))
		log.Printf("[INFO] /ask - Command executed successfully using API key %d, output length: %d bytes", usedKeyIndex+1, len(output))
	}

	if err != nil {
		incrementTotalErrors()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// askGeminiHandler handles POST /ask-gemini - regular Gemini API request
func askGeminiHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Acquire semaphore
	select {
	case requestSemaphore <- struct{}{}:
		incrementTotalRequests()
		incrementActiveRequests()
		defer func() {
			<-requestSemaphore
			decrementActiveRequests()
		}()
	case <-r.Context().Done():
		return
	}

	var req GeminiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "Question is required", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] /ask-gemini - Question: %s", req.Question)

	// Gemini API endpoint
	apiKey := "AIzaSyDdfzMI2uzIWaHx9dD5H2dNm28N-LE0Wys"
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent?key=%s", apiKey)

	// Prepare request body
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"text": req.Question,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - Failed to marshal request: %v", err)
		http.Error(w, "Failed to prepare request", http.StatusInternalServerError)
		return
	}

	// Create HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - Failed to create request: %v", err)
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - Request failed: %v", err)
		geminiResp := GeminiResponse{
			Error: fmt.Sprintf("Request failed: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(geminiResp)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - Failed to read response: %v", err)
		geminiResp := GeminiResponse{
			Error: fmt.Sprintf("Failed to read response: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(geminiResp)
		return
	}

	// Parse response
	var geminiResp GeminiResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - Failed to parse response: %v", err)
		geminiResp = GeminiResponse{
			Error: fmt.Sprintf("Failed to parse response: %v", err),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(geminiResp)
		return
	}

	if resp.StatusCode != http.StatusOK {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini - API returned status %d: %s", resp.StatusCode, string(body))
		geminiResp.Error = fmt.Sprintf("API returned status %d", resp.StatusCode)
		w.WriteHeader(resp.StatusCode)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(geminiResp)
}

// askGeminiStreamHandler handles POST /ask-gemini-stream - streaming Gemini API request
func askGeminiStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Acquire semaphore
	select {
	case requestSemaphore <- struct{}{}:
		incrementTotalRequests()
		incrementActiveRequests()
		defer func() {
			<-requestSemaphore
			decrementActiveRequests()
		}()
	case <-r.Context().Done():
		return
	}

	var req GeminiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini-stream - Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "Question is required", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] /ask-gemini-stream - Question: %s", req.Question)

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send immediate started signal
	started := StreamResponse{Type: "started"}
	startedJSON, _ := json.Marshal(started)
	fmt.Fprintf(w, "data: %s\n\n", startedJSON)
	flusher.Flush()

	// Gemini API endpoint for streaming
	apiKey := "AIzaSyDdfzMI2uzIWaHx9dD5H2dNm28N-LE0Wys"
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:streamGenerateContent?key=%s", apiKey)

	// Prepare request body
	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{
						"text": req.Question,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini-stream - Failed to marshal request: %v", err)
		errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("Failed to prepare request: %v", err)}
		jsonData, _ := json.Marshal(errResp)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return
	}

	// Create HTTP request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini-stream - Failed to create request: %v", err)
		errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("Failed to create request: %v", err)}
		jsonData, _ := json.Marshal(errResp)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute request
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(httpReq)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-gemini-stream - Request failed: %v", err)
		errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("Request failed: %v", err)}
		jsonData, _ := json.Marshal(errResp)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		incrementTotalErrors()
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[ERROR] /ask-gemini-stream - API returned status %d: %s", resp.StatusCode, string(body))
		errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("API returned status %d", resp.StatusCode)}
		jsonData, _ := json.Marshal(errResp)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return
	}

	// Stream response from Gemini
	// Gemini stream response format: Each line is a complete JSON object
	// Example: {"candidates":[...]}\n{"candidates":[...]}\n
	// But response may come in chunks, so we need to buffer and find complete JSON objects
	reader := bufio.NewReader(resp.Body)
	var buffer strings.Builder

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Read chunk
		chunk := make([]byte, 4096)
		n, err := reader.Read(chunk)
		if n > 0 {
			buffer.Write(chunk[:n])

			// Process complete JSON objects from buffer
			bufferStr := buffer.String()
			for {
				// Find complete JSON object by tracking braces
				jsonStart := -1
				jsonEnd := -1
				braceCount := 0
				inString := false
				escapeNext := false

				// Find start of JSON object (opening brace)
				for i, char := range bufferStr {
					if escapeNext {
						escapeNext = false
						continue
					}

					if char == '\\' {
						escapeNext = true
						continue
					}

					if char == '"' {
						inString = !inString
						continue
					}

					if !inString {
						if char == '{' {
							if jsonStart == -1 {
								jsonStart = i
							}
							braceCount++
						} else if char == '}' {
							braceCount--
							if braceCount == 0 && jsonStart != -1 {
								// Found complete JSON object
								jsonEnd = i + 1
								break
							}
						}
					}
				}

				if jsonStart >= 0 && jsonEnd > jsonStart {
					// Extract complete JSON object
					jsonStr := bufferStr[jsonStart:jsonEnd]
					bufferStr = bufferStr[jsonEnd:]
					buffer.Reset()
					buffer.WriteString(bufferStr)

					// Parse Gemini stream response
					var geminiResp GeminiResponse
					if err := json.Unmarshal([]byte(jsonStr), &geminiResp); err == nil {
						// Send as SSE to frontend
						jsonData, _ := json.Marshal(geminiResp)
						fmt.Fprintf(w, "data: %s\n\n", jsonData)
						flusher.Flush()
					} else {
						log.Printf("[WARNING] /ask-gemini-stream - Failed to parse JSON: %v", err)
					}
				} else {
					// No complete JSON object found, wait for more data
					// Limit buffer size to prevent memory issues
					if buffer.Len() > 1024*1024 {
						log.Printf("[ERROR] /ask-gemini-stream - Buffer too large, clearing")
						buffer.Reset()
					}
					break
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				// Try to parse any remaining buffer
				if buffer.Len() > 0 {
					bufferStr := buffer.String()
					// Try to find and parse any remaining JSON
					for {
						jsonStart := -1
						jsonEnd := -1
						braceCount := 0
						inString := false
						escapeNext := false

						for i, char := range bufferStr {
							if escapeNext {
								escapeNext = false
								continue
							}
							if char == '\\' {
								escapeNext = true
								continue
							}
							if char == '"' {
								inString = !inString
								continue
							}
							if !inString {
								if char == '{' {
									if jsonStart == -1 {
										jsonStart = i
									}
									braceCount++
								} else if char == '}' {
									braceCount--
									if braceCount == 0 && jsonStart != -1 {
										jsonEnd = i + 1
										break
									}
								}
							}
						}

						if jsonStart >= 0 && jsonEnd > jsonStart {
							jsonStr := bufferStr[jsonStart:jsonEnd]
							bufferStr = bufferStr[jsonEnd:]

							var geminiResp GeminiResponse
							if err := json.Unmarshal([]byte(jsonStr), &geminiResp); err == nil {
								jsonData, _ := json.Marshal(geminiResp)
								fmt.Fprintf(w, "data: %s\n\n", jsonData)
								flusher.Flush()
							}
						} else {
							break
						}
					}
				}

				// Send done signal
				done := StreamResponse{Type: "done"}
				doneJSON, _ := json.Marshal(done)
				fmt.Fprintf(w, "data: %s\n\n", doneJSON)
				flusher.Flush()
				return
			}
			log.Printf("[ERROR] /ask-gemini-stream - Read error: %v", err)
			errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("Read error: %v", err)}
			jsonData, _ := json.Marshal(errResp)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
			return
		}
	}
}

func askStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Acquire semaphore (limit concurrent requests)
	select {
	case requestSemaphore <- struct{}{}:
		incrementTotalRequests()
		incrementActiveRequests()
		defer func() {
			<-requestSemaphore
			decrementActiveRequests()
		}() // Release on return
	case <-r.Context().Done():
		// Client disconnected before acquiring semaphore
		return
	}

	var req AskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-stream - Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] /ask-stream - Question: %s", req.Question)

	// Use client request context only (no server-side timeout)
	ctx := r.Context()

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Prepare SSE streaming
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send immediate started signal to reduce TTFB
	started := StreamResponse{Type: "started"}
	startedJSON, _ := json.Marshal(started)
	fmt.Fprintf(w, "data: %s\n\n", startedJSON)
	flusher.Flush()

	// Try each API key with fallback mechanism
	keys := apiKeyManager.GetAllKeys()
	var lastErr error
	var success bool

	for i, apiKey := range keys {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			errResp := StreamResponse{
				Type:  "error",
				Error: fmt.Sprintf("Request cancelled or timeout: %v", ctx.Err()),
			}
			jsonData, _ := json.Marshal(errResp)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
			return
		default:
		}

		// Use streaming JSON output form: cursor-agent --api-key <key> -p --output-format stream-json --stream-partial-output ask <question>
		templatePrompt := "You are a helpful assistant that can answer questions dont make file or create update just answer as text the this question: "
		args := []string{"--api-key", apiKey, "-p", "--output-format", "stream-json", "--stream-partial-output", "ask", req.Question, templatePrompt}
		// Execute with context (supports timeout and cancellation)
		cmd := exec.CommandContext(ctx, "cursor-agent", args...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			lastErr = err
			log.Printf("[ERROR] /ask-stream - API key %d: Failed to create stdout pipe: %v", i+1, err)
			continue
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			lastErr = err
			log.Printf("[ERROR] /ask-stream - API key %d: Failed to create stderr pipe: %v", i+1, err)
			continue
		}

		if err := cmd.Start(); err != nil {
			lastErr = err
			log.Printf("[ERROR] /ask-stream - API key %d: Failed to start command: %v", i+1, err)
			errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("API key %d failed to start: %v", i+1, err)}
			jsonData, _ := json.Marshal(errResp)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
			continue
		}

		// Stream stderr lines as regular chunks too (surface errors inline)
		errorOccurred := false
		go func(keyIndex int) {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				// Check for error indicators
				if strings.Contains(strings.ToLower(line), "error") ||
					strings.Contains(strings.ToLower(line), "rate limit") ||
					strings.Contains(strings.ToLower(line), "unauthorized") ||
					strings.Contains(strings.ToLower(line), "quota") {
					errorOccurred = true
				}
				// Forward stderr as chunk content so client sees messages immediately
				chunk := StreamResponse{Type: "chunk", Content: line}
				jsonData, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				flusher.Flush()
			}
		}(i)

		// Stream stdout in raw chunks (flush ASAP)
		reader := bufio.NewReader(stdout)
		buf := make([]byte, BufferReadSize)
		hasOutput := false

		for {
			n, rerr := reader.Read(buf)
			if n > 0 {
				hasOutput = true
				segment := string(buf[:n])
				addBytesProcessed(n)
				chunk := StreamResponse{Type: "chunk", Content: segment}
				jsonData, _ := json.Marshal(chunk)
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
				flusher.Flush()
			}
			if rerr != nil {
				if rerr != io.EOF {
					lastErr = rerr
					log.Printf("[ERROR] /ask-stream - API key %d: Read error: %v", i+1, rerr)
				}
				break
			}
		}

		err = cmd.Wait()
		if err != nil || errorOccurred || !hasOutput {
			lastErr = err
			if err != nil {
				log.Printf("[ERROR] /ask-stream - API key %d: Command wait failed: %v", i+1, err)
			}
			if errorOccurred {
				log.Printf("[ERROR] /ask-stream - API key %d: Error %v, trying next key", i+1, err)
			}
			if !hasOutput {
				log.Printf("[ERROR] /ask-stream - API key %d: No output received, trying next key %v", i+1, err)
			}
			log.Printf("[ERROR] /ask-stream - API key %d: Last error: %v", i+1, lastErr)
			continue
		}

		// Success!
		success = true
		log.Printf("[SUCCESS] /ask-stream - Stream completed successfully using API key %d", i+1)
		break
	}

	if !success {
		incrementTotalErrors()
		log.Printf("[ERROR] /ask-stream - All %d Limit telah habis, silahkan upgrade ke premium untuk melanjutkan penggunaan. Last error: %v", len(keys), lastErr)
		errResp := StreamResponse{Type: "error", Error: fmt.Sprintf("All %d Limit telah habis, silahkan upgrade ke premium untuk melanjutkan penggunaan. Last error: %v", len(keys), lastErr)}
		jsonData, _ := json.Marshal(errResp)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	}

	doneResp := StreamResponse{Type: "done"}
	jsonData, _ := json.Marshal(doneResp)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

// ParsedFile represents a file extracted from generated code
type ParsedFile struct {
	Path    string
	Content string
}

// parseCodeFromOutput extracts HTML, CSS, and JS files from cursor-agent output
func parseCodeFromOutput(output string) []ParsedFile {
	var files []ParsedFile

	// Pattern untuk match code blocks dengan filenames
	// Format: ```html:index.html atau ```css:style.css atau ```js:script.js
	codeBlockPattern := regexp.MustCompile(`(?s)` + "`" + "`" + "`" + `(\w+):([^\n]+)\n(.*?)` + "`" + "`" + "`")
	matches := codeBlockPattern.FindAllStringSubmatch(output, -1)

	for _, match := range matches {
		if len(match) >= 4 {
			fileType := match[1]
			fileName := strings.TrimSpace(match[2])
			content := strings.TrimSpace(match[3])

			// Ensure proper file extension
			if !strings.Contains(fileName, ".") {
				switch fileType {
				case "html":
					fileName = fileName + ".html"
				case "css":
					fileName = fileName + ".css"
				case "js", "javascript":
					fileName = fileName + ".js"
				}
			}

			files = append(files, ParsedFile{
				Path:    fileName,
				Content: content,
			})
		}
	}

	// Fallback: jika tidak ada pattern, coba cari standalone code blocks
	if len(files) == 0 {
		// Pattern untuk standard markdown code blocks
		standardPattern := regexp.MustCompile(`(?s)` + "`" + "`" + "`" + `(\w+)\n(.*?)` + "`" + "`" + "`")
		standardMatches := standardPattern.FindAllStringSubmatch(output, -1)

		// Default files
		defaultFiles := map[string]bool{
			"index.html": false,
			"style.css":  false,
			"script.js":  false,
		}

		for _, match := range standardMatches {
			if len(match) >= 3 {
				fileType := strings.ToLower(match[1])
				content := strings.TrimSpace(match[2])

				var fileName string
				switch fileType {
				case "html":
					fileName = "index.html"
					defaultFiles["index.html"] = true
				case "css":
					fileName = "style.css"
					defaultFiles["style.css"] = true
				case "js", "javascript":
					fileName = "script.js"
					defaultFiles["script.js"] = true
				default:
					continue
				}

				files = append(files, ParsedFile{
					Path:    fileName,
					Content: content,
				})
			}
		}

		// Jika tidak ada file yang ditemukan, coba extract dari full output
		if len(files) == 0 {
			// Coba extract HTML
			htmlPattern := regexp.MustCompile(`(?s)(<html[^>]*>.*?</html>)`)
			if htmlMatch := htmlPattern.FindString(output); htmlMatch != "" {
				files = append(files, ParsedFile{
					Path:    "index.html",
					Content: htmlMatch,
				})
			}
		}
	}

	// Ensure at least index.html exists
	hasHTML := false
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".html") {
			hasHTML = true
			break
		}
	}

	if !hasHTML {
		// Create minimal HTML
		files = append(files, ParsedFile{
			Path: "index.html",
			Content: `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Generated App</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <h1>Generated App</h1>
    <script src="script.js"></script>
</body>
</html>`,
		})
	}

	return files
}

// saveAppFiles saves parsed files to filesystem
func saveAppFiles(appID string, files []ParsedFile) error {
	appDir := filepath.Join("generated-apps", appID)

	// Create directory
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %v", err)
	}

	// Save each file
	for _, file := range files {
		filePath := filepath.Join(appDir, file.Path)

		// Create subdirectories if needed
		dir := filepath.Dir(filePath)
		if dir != "." && dir != appDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create subdirectory: %v", err)
			}
		}

		// Write file
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %v", file.Path, err)
		}

		log.Printf("[INFO] Saved file: %s", filePath)
	}

	return nil
}

// generateAppFromPrompt generates app files from prompt using cursor-agent
func generateAppFromPrompt(ctx context.Context, prompt string) ([]ParsedFile, error) {
	keys := apiKeyManager.GetAllKeys()
	var lastErr error

	// Enhanced prompt for code generation
	codeGenPrompt := fmt.Sprintf("You are a web development assistant. Create a complete web application based on this prompt: \"%s\"\n\n"+
		"Generate the application with the following structure:\n"+
		"1. Create an HTML file (index.html) with a complete, modern, and responsive structure\n"+
		"2. Create a CSS file (style.css) with modern styling\n"+
		"3. Create a JavaScript file (script.js) with any necessary interactivity\n\n"+
		"Format your response as code blocks with filenames like this:\n"+
		"```html:index.html\n"+
		"[HTML content here]\n"+
		"```\n\n"+
		"```css:style.css\n"+
		"[CSS content here]\n"+
		"```\n\n"+
		"```js:script.js\n"+
		"[JavaScript content here]\n"+
		"```\n\n"+
		"Make sure the application is complete, functional, and modern. Use best practices for HTML5, CSS3, and vanilla JavaScript.", prompt)

	for i, apiKey := range keys {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Build args: cursor-agent --api-key <key> -p --output-format text <prompt>
		args := []string{"--api-key", apiKey, codeGenPrompt, "-p", "--output-format", "text"}

		// Execute with context
		cmd := exec.CommandContext(ctx, "cursor-agent", args...)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err == nil {
			// Check if output contains textual error indicators
			lower := strings.ToLower(outputStr)
			if strings.Contains(lower, "connecterror") || strings.Contains(lower, "resource_exhausted") ||
				strings.Contains(lower, "resource exhausted") || strings.Contains(lower, "rate limit") ||
				strings.Contains(lower, "unauthorized") || strings.Contains(lower, "quota") {
				log.Printf("[WARNING] API key %d returned error text, trying next...", i+1)
				lastErr = fmt.Errorf("API key %d textual error: %s", i+1, outputStr[:min(200, len(outputStr))])
				continue
			}

			// Parse code from output
			files := parseCodeFromOutput(outputStr)
			if len(files) > 0 {
				log.Printf("[SUCCESS] API key %d generated %d file(s)", i+1, len(files))
				return files, nil
			}

			// If no files parsed, try again with next key
			log.Printf("[WARNING] API key %d generated output but no files parsed", i+1)
			lastErr = fmt.Errorf("no files parsed from output")
			continue
		}

		// Check if error is due to signal killed (context timeout or OS kill)
		// Check if it's a signal killed error
		if strings.Contains(err.Error(), "signal: killed") || strings.Contains(err.Error(), "killed") {
			log.Printf("[WARNING] API key %d process was killed, trying next key...", i+1)
			lastErr = fmt.Errorf("API key %d process killed (timeout or resource limit)", i+1)
			continue
		}

		// Log detailed failure
		trunc := outputStr
		if len(trunc) > 200 {
			trunc = trunc[:200]
		}
		log.Printf("[ERROR] API key %d failed: %v | output: %q", i+1, err, trunc)
		lastErr = fmt.Errorf("API key %d last failure: %v | output: %s", i+1, err, trunc)
	}

	return nil, fmt.Errorf("all %d Limit telah habis, silahkan upgrade ke premium untuk melanjutkan penggunaan. Last error: %v", len(keys), lastErr)
}

// createAppHandler handles POST /create-app
func createAppHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Acquire semaphore
	select {
	case requestSemaphore <- struct{}{}:
		incrementTotalRequests()
		incrementActiveRequests()
		defer func() {
			<-requestSemaphore
			decrementActiveRequests()
		}()
	case <-r.Context().Done():
		return
	}

	var req CreateAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /create-app - Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] /create-app - Prompt: %s", req.Prompt)

	// Generate app ID
	appID := getNextAppID()

	// Create context with longer timeout for app generation (5 minutes)
	// Don't cancel immediately if client disconnects - let it finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Generate app files
	files, err := generateAppFromPrompt(ctx, req.Prompt)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /create-app - Failed to generate app: %v", err)
		resp := CreateAppResponse{
			AppID: appID,
			Error: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Save files to filesystem
	if err := saveAppFiles(appID, files); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /create-app - Failed to save app files: %v", err)
		resp := CreateAppResponse{
			AppID: appID,
			Error: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Success response
	previewURL := fmt.Sprintf("/preview/%s", appID)
	resp := CreateAppResponse{
		AppID: appID,
		URL:   previewURL,
	}

	log.Printf("[INFO] /create-app - App created successfully: %s with %d file(s)", appID, len(files))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// createFolderHandler handles POST /create-folder - creates folder and saves HTML file (async)
func createFolderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] /create-folder - Failed to decode JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, "Prompt is required", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] /create-folder - Prompt: %s", req.Prompt)

	// Generate job ID for async tracking
	jobID := generateJobID()

	// Auto-generate folder name based on increment
	// Format: app_001, app_002, etc.
	cleanFolderName := getNextFolderName()
	log.Printf("[INFO] /create-folder - Generated folder name: %s, Job ID: %s", cleanFolderName, jobID)

	// Create job status (pending)
	jobStatus := &JobStatus{
		JobID:      jobID,
		Status:     "pending",
		Progress:   "Job created, waiting to start...",
		FolderName: cleanFolderName,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	jobStoreMutex.Lock()
	jobStore[jobID] = jobStatus
	jobStoreMutex.Unlock()

	// Return job ID immediately (async)
	resp := CreateFolderResponse{
		JobID: jobID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)

	// Start async processing in goroutine
	go processCreateFolderJob(jobID, req.Prompt, cleanFolderName)
}

// processCreateFolderJob processes the create folder job asynchronously
func processCreateFolderJob(jobID string, prompt string, cleanFolderName string) {
	// Update job status to processing
	updateJobStatus(jobID, "processing", "Creating folder...", "", "", "", "")

	// Acquire semaphore for processing
	select {
	case requestSemaphore <- struct{}{}:
		incrementTotalRequests()
		incrementActiveRequests()
		defer func() {
			<-requestSemaphore
			decrementActiveRequests()
		}()
	default:
		// If semaphore is full, wait a bit
		time.Sleep(100 * time.Millisecond)
		select {
		case requestSemaphore <- struct{}{}:
			incrementTotalRequests()
			incrementActiveRequests()
			defer func() {
				<-requestSemaphore
				decrementActiveRequests()
			}()
		case <-time.After(30 * time.Second):
			updateJobStatus(jobID, "failed", "Timeout waiting for semaphore", "", "", "", "Timeout waiting for processing slot")
			return
		}
	}

	// Ensure generated-apps directory exists
	generatedAppsDir := "generated-apps"
	if err := os.MkdirAll(generatedAppsDir, 0755); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] Job %s - Failed to create generated-apps directory: %v", jobID, err)
		updateJobStatus(jobID, "failed", "Failed to create directory", "", "", "", fmt.Sprintf("Failed to create generated-apps directory: %v", err))
		return
	}

	folderPath := filepath.Join(generatedAppsDir, cleanFolderName)

	// Additional security: ensure path is within generated-apps (prevent directory traversal)
	absFolderPath, err := filepath.Abs(folderPath)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] Job %s - Failed to get absolute path: %v", jobID, err)
		updateJobStatus(jobID, "failed", "Invalid folder path", "", "", "", fmt.Sprintf("Invalid folder path: %v", err))
		return
	}

	absGeneratedApps, err := filepath.Abs("generated-apps")
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] Job %s - Failed to get generated-apps path: %v", jobID, err)
		updateJobStatus(jobID, "failed", "Invalid path", "", "", "", fmt.Sprintf("Invalid path: %v", err))
		return
	}

	// Ensure folder path is within generated-apps directory
	if !strings.HasPrefix(absFolderPath, absGeneratedApps) {
		incrementTotalErrors()
		log.Printf("[ERROR] Job %s - Path traversal attempt detected: %s", jobID, absFolderPath)
		updateJobStatus(jobID, "failed", "Invalid folder path", "", "", "", "Path traversal detected")
		return
	}

	// Create folder
	updateJobStatus(jobID, "processing", "Creating folder...", cleanFolderName, "", "", "")
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] Job %s - Failed to create folder: %v", jobID, err)
		updateJobStatus(jobID, "failed", "Failed to create folder", cleanFolderName, "", "", fmt.Sprintf("Failed to create folder: %v", err))
		return
	}

	// Create context with longer timeout for HTML generation (15 minutes)
	// Increased timeout to handle multiple API key retries
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Generate HTML from prompt INSIDE the folder directory
	updateJobStatus(jobID, "processing", "Generating HTML files...", cleanFolderName, "", "", "")
	log.Printf("[INFO] Job %s - Generating HTML in directory: %s", jobID, folderPath)

	// Generate HTML from prompt with working directory set to folder
	err = generateHTMLInDirectoryWithProgress(ctx, prompt, folderPath, jobID)
	if err != nil {
		incrementTotalErrors()
		log.Printf("[ERROR] Job %s - Failed to generate HTML: %v", jobID, err)
		updateJobStatus(jobID, "failed", "Failed to generate HTML", cleanFolderName, "", "", err.Error())
		return
	}

	// Find HTML file created by cursor-agent (could be any HTML file)
	htmlFileName := ""
	htmlFilePath := ""

	// First, try to find the generated filename from prompt
	generatedFileName := generateFileNameFromPrompt(prompt)
	potentialPath := filepath.Join(folderPath, generatedFileName)
	if _, err := os.Stat(potentialPath); err == nil {
		htmlFileName = generatedFileName
		htmlFilePath = potentialPath
		log.Printf("[INFO] Job %s - Found generated HTML file: %s", jobID, htmlFileName)
	} else {
		// If generated filename doesn't exist, try index.html
		indexPath := filepath.Join(folderPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			htmlFileName = "index.html"
			htmlFilePath = indexPath
			log.Printf("[INFO] Job %s - Found index.html", jobID)
		} else {
			// If index.html doesn't exist, try to find any HTML file
			entries, readErr := os.ReadDir(folderPath)
			if readErr == nil {
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
						htmlFileName = entry.Name()
						htmlFilePath = filepath.Join(folderPath, htmlFileName)
						log.Printf("[INFO] Job %s - Found HTML file: %s", jobID, htmlFileName)
						break
					}
				}
			}
		}
	}

	// If still no HTML file found, use generated filename as fallback
	if htmlFileName == "" {
		htmlFileName = generatedFileName
		htmlFilePath = filepath.Join(folderPath, htmlFileName)
		log.Printf("[INFO] Job %s - Using generated filename: %s", jobID, htmlFileName)
	}

	// Success
	previewURL := fmt.Sprintf("/preview/%s/%s", cleanFolderName, htmlFileName)
	updateJobStatus(jobID, "completed", "App created successfully!", cleanFolderName, htmlFilePath, previewURL, "")
	log.Printf("[INFO] Job %s - Folder created successfully: %s with files in directory", jobID, cleanFolderName)
}

// updateJobStatus updates the job status in the store
func updateJobStatus(jobID string, status string, progress string, folderName string, filePath string, url string, errorMsg string) {
	jobStoreMutex.Lock()
	defer jobStoreMutex.Unlock()

	if job, exists := jobStore[jobID]; exists {
		job.Status = status
		job.Progress = progress
		job.UpdatedAt = time.Now()
		if folderName != "" {
			job.FolderName = folderName
		}
		if filePath != "" {
			job.FilePath = filePath
		}
		if url != "" {
			job.URL = url
		}
		if errorMsg != "" {
			job.Error = errorMsg
		}
	}
}

// getJobStatus returns the job status
func getJobStatus(jobID string) (*JobStatus, bool) {
	jobStoreMutex.RLock()
	defer jobStoreMutex.RUnlock()

	job, exists := jobStore[jobID]
	if exists {
		// Return a copy to avoid race conditions
		jobCopy := *job
		return &jobCopy, true
	}
	return nil, false
}

// jobStatusHandler handles GET /job-status - returns job status
func jobStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("job_id")
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	job, exists := getJobStatus(jobID)
	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// generateFileNameFromPrompt generates a safe filename from prompt
func generateFileNameFromPrompt(prompt string) string {
	// Take first 50 characters of prompt
	text := prompt
	if len(text) > 50 {
		text = text[:50]
	}

	// Convert to lowercase
	text = strings.ToLower(text)

	// Extract meaningful words (remove common words)
	words := strings.Fields(text)
	var meaningfulWords []string
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "create": true, "make": true, "build": true,
		"app": true, "application": true, "website": true, "web": true,
	}

	for _, word := range words {
		// Remove punctuation and special characters
		cleaned := ""
		for _, r := range word {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				cleaned += string(r)
			}
		}

		// Skip empty or stop words
		if cleaned != "" && len(cleaned) > 2 && !stopWords[cleaned] {
			meaningfulWords = append(meaningfulWords, cleaned)
		}

		// Limit to 3-4 words max
		if len(meaningfulWords) >= 4 {
			break
		}
	}

	// Join words with dash
	if len(meaningfulWords) > 0 {
		fileName := strings.Join(meaningfulWords, "-")
		// Ensure it's not too long
		if len(fileName) > 40 {
			fileName = fileName[:40]
		}
		return fileName + ".html"
	}

	// Fallback: use first few characters
	cleaned := ""
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned += string(r)
		} else if unicode.IsSpace(r) && cleaned != "" && !strings.HasSuffix(cleaned, "-") {
			cleaned += "-"
		}
		if len(cleaned) >= 30 {
			break
		}
	}

	if cleaned == "" {
		return "app.html"
	}

	// Remove trailing dash
	cleaned = strings.TrimSuffix(cleaned, "-")
	if len(cleaned) > 30 {
		cleaned = cleaned[:30]
	}
	return cleaned + ".html"
}

// generateHTMLInDirectoryWithProgress generates HTML files in directory with progress updates
func generateHTMLInDirectoryWithProgress(ctx context.Context, prompt string, workingDir string, jobID string) error {
	keys := apiKeyManager.GetAllKeys()
	var lastErr error

	// Generate filename from prompt
	htmlFileName := generateFileNameFromPrompt(prompt)
	log.Printf("[INFO] Job %s - Generated HTML filename from prompt: %s", jobID, htmlFileName)

	// Enhanced prompt for HTML generation that creates files directly in current directory
	htmlGenPrompt := fmt.Sprintf("You are a web development assistant. Create a complete web application based on this prompt: \"%s\"\n\n"+
		"IMPORTANT: You are currently in a folder that has been created for this project. The current working directory is the project folder.\n"+
		"All files MUST be created in this current directory (the folder where you are now).\n\n"+
		"Generate the application with the following structure:\n"+
		"1. Create an HTML file named \"%s\" (or any descriptive HTML filename based on the app theme) with a complete, modern, and responsive structure - save it in the current directory\n"+
		"2. Create a CSS file (style.css) if needed with modern styling - save it in the current directory\n"+
		"3. Create a JavaScript file (script.js) if needed with any necessary interactivity - save it in the current directory\n\n"+
		"CRITICAL: The folder has already been created. You are INSIDE that folder now. Create all files directly in this current directory.\n"+
		"Do NOT create a new folder. Save files directly here using standard file creation commands.\n"+
		"The HTML file should be named something descriptive based on the app (e.g., if it's a coffee shop app, name it 'toko-kopi.html' or similar).\n"+
		"Make sure the HTML is complete, functional, and modern. Use best practices for HTML5, CSS3, and vanilla JavaScript.", prompt, htmlFileName)

	for i, apiKey := range keys {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Update progress
		updateJobStatus(jobID, "processing", fmt.Sprintf("Trying API key %d/%d...", i+1, len(keys)), "", "", "", "")

		// Build args: cursor-agent --api-key <key> -p --output-format text <prompt>
		args := []string{"--api-key", apiKey, htmlGenPrompt, "-p", "--output-format", "text"}

		// Execute with context and set working directory to the folder
		cmd := exec.CommandContext(ctx, "cursor-agent", args...)
		cmd.Dir = workingDir // Set working directory to the folder

		updateJobStatus(jobID, "processing", fmt.Sprintf("Generating with API key %d/%d...", i+1, len(keys)), "", "", "", "")
		log.Printf("[INFO] Job %s - Executing cursor-agent in directory: %s", jobID, workingDir)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err == nil {
			// Check if output contains textual error indicators
			lower := strings.ToLower(outputStr)
			if strings.Contains(lower, "connecterror") || strings.Contains(lower, "resource_exhausted") ||
				strings.Contains(lower, "resource exhausted") || strings.Contains(lower, "rate limit") ||
				strings.Contains(lower, "unauthorized") || strings.Contains(lower, "quota") {
				log.Printf("[WARNING] Job %s - API key %d returned error text, trying next...", jobID, i+1)
				lastErr = fmt.Errorf("API key %d textual error: %s", i+1, outputStr[:min(200, len(outputStr))])
				updateJobStatus(jobID, "processing", fmt.Sprintf("API key %d failed, trying next...", i+1), "", "", "", "")
				continue
			}

			// Check if files were created in the directory
			updateJobStatus(jobID, "processing", "Checking for generated files...", "", "", "", "")
			entries, readErr := os.ReadDir(workingDir)
			if readErr == nil {
				hasHTML := false
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
						hasHTML = true
						log.Printf("[SUCCESS] Job %s - API key %d created HTML file: %s", jobID, i+1, entry.Name())
					}
				}
				if hasHTML {
					log.Printf("[SUCCESS] Job %s - API key %d generated files in directory", jobID, i+1)
					return nil
				}
			}

			// If no files found, try extracting HTML from output and save it
			updateJobStatus(jobID, "processing", "Extracting HTML from output...", "", "", "", "")
			htmlContent := extractHTMLFromOutput(outputStr)
			if htmlContent != "" {
				// Use generated filename or fallback to index.html
				htmlFilePath := filepath.Join(workingDir, htmlFileName)
				if err := os.WriteFile(htmlFilePath, []byte(htmlContent), 0644); err != nil {
					log.Printf("[ERROR] Job %s - Failed to save extracted HTML: %v", jobID, err)
					lastErr = fmt.Errorf("failed to save HTML: %v", err)
					continue
				}
				log.Printf("[SUCCESS] Job %s - API key %d generated HTML and saved to %s", jobID, i+1, htmlFileName)
				return nil
			}

			log.Printf("[WARNING] Job %s - API key %d generated output but no files created", jobID, i+1)
			lastErr = fmt.Errorf("no files created in directory")
			continue
		}

		// Handle errors - err is not nil here since we passed the if err == nil check above
		errStr := err.Error()

		// Check if error is due to signal killed
		if strings.Contains(errStr, "signal: killed") || strings.Contains(errStr, "killed") {
			log.Printf("[WARNING] Job %s - API key %d process was killed, trying next key...", jobID, i+1)
			lastErr = fmt.Errorf("API key %d process killed (timeout or resource limit)", i+1)
			updateJobStatus(jobID, "processing", fmt.Sprintf("API key %d killed, trying next...", i+1), "", "", "", "")

			// Check if context is still valid before continuing
			select {
			case <-ctx.Done():
				log.Printf("[ERROR] Job %s - Context expired before trying all keys", jobID)
				return fmt.Errorf("context deadline exceeded before trying all %d API keys. Last error: %v", len(keys), lastErr)
			default:
				// Context still valid, continue to next key
				time.Sleep(2 * time.Second) // Brief pause before retry
				continue
			}
		}

		// Check if error is due to context deadline
		if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "context canceled") {
			log.Printf("[WARNING] Job %s - API key %d hit context deadline, trying next key...", jobID, i+1)
			lastErr = fmt.Errorf("API key %d context deadline: %v", i+1, err)
			updateJobStatus(jobID, "processing", fmt.Sprintf("API key %d timeout, trying next...", i+1), "", "", "", "")

			// If this is not the last key, continue
			if i < len(keys)-1 {
				time.Sleep(2 * time.Second) // Brief pause before retry
				continue
			}
			// If this is the last key, return error
			return fmt.Errorf("all %d API keys failed due to timeouts. Last error: %v", len(keys), lastErr)
		}

		// Log detailed failure
		trunc := outputStr
		if len(trunc) > 200 {
			trunc = trunc[:200]
		}
		log.Printf("[ERROR] Job %s - API key %d failed: %v | output: %q", jobID, i+1, err, trunc)
		lastErr = fmt.Errorf("API key %d last failure: %v | output: %s", i+1, err, trunc)
	}

	// If we get here, all keys failed
	if lastErr != nil {
		return fmt.Errorf("all %d API keys failed. Last error: %v", len(keys), lastErr)
	}
	return fmt.Errorf("all %d API keys failed with unknown errors", len(keys))
}

// sanitizeFolderName removes invalid characters from folder name
func sanitizeFolderName(name string) string {
	// Remove invalid characters for folder names
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalidChars {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Remove leading/trailing spaces and dots
	result = strings.TrimSpace(result)
	result = strings.Trim(result, ".")
	// Replace spaces with underscores
	result = strings.ReplaceAll(result, " ", "_")
	// Ensure not empty
	if result == "" {
		result = "folder_" + fmt.Sprintf("%d", time.Now().Unix())
	}
	return result
}

// generateHTMLInDirectory generates HTML files in the specified directory using cursor-agent
// This function runs cursor-agent with working directory set to the target folder
func generateHTMLInDirectory(ctx context.Context, prompt string, workingDir string) error {
	keys := apiKeyManager.GetAllKeys()
	var lastErr error

	// Enhanced prompt for HTML generation that creates files directly in current directory
	htmlGenPrompt := fmt.Sprintf("You are a web development assistant. Create a complete web application based on this prompt: \"%s\"\n\n"+
		"IMPORTANT: You are currently in a folder that has been created for this project. The current working directory is the project folder.\n"+
		"All files MUST be created in this current directory (the folder where you are now).\n\n"+
		"Generate the application with the following structure:\n"+
		"1. Create an HTML file (index.html or any HTML file) with a complete, modern, and responsive structure - save it in the current directory\n"+
		"2. Create a CSS file (style.css) if needed with modern styling - save it in the current directory\n"+
		"3. Create a JavaScript file (script.js) if needed with any necessary interactivity - save it in the current directory\n\n"+
		"CRITICAL: The folder has already been created. You are INSIDE that folder now. Create all files directly in this current directory.\n"+
		"Do NOT create a new folder. Save files directly here using standard file creation commands.\n"+
		"Make sure the HTML is complete, functional, and modern. Use best practices for HTML5, CSS3, and vanilla JavaScript.", prompt)

	for i, apiKey := range keys {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Build args: cursor-agent --api-key <key> -p --output-format text <prompt>
		args := []string{"--api-key", apiKey, htmlGenPrompt, "-p", "--output-format", "text"}

		// Execute with context and set working directory to the folder
		cmd := exec.CommandContext(ctx, "cursor-agent", args...)
		cmd.Dir = workingDir // Set working directory to the folder

		log.Printf("[INFO] Executing cursor-agent in directory: %s", workingDir)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err == nil {
			// Check if output contains textual error indicators
			lower := strings.ToLower(outputStr)
			if strings.Contains(lower, "connecterror") || strings.Contains(lower, "resource_exhausted") ||
				strings.Contains(lower, "resource exhausted") || strings.Contains(lower, "rate limit") ||
				strings.Contains(lower, "unauthorized") || strings.Contains(lower, "quota") {
				log.Printf("[WARNING] API key %d returned error text, trying next...", i+1)
				lastErr = fmt.Errorf("API key %d textual error: %s", i+1, outputStr[:min(200, len(outputStr))])
				continue
			}

			// Check if files were created in the directory
			entries, readErr := os.ReadDir(workingDir)
			if readErr == nil {
				hasHTML := false
				for _, entry := range entries {
					if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
						hasHTML = true
						log.Printf("[SUCCESS] API key %d created HTML file: %s", i+1, entry.Name())
					}
				}
				if hasHTML {
					log.Printf("[SUCCESS] API key %d generated files in directory", i+1)
					return nil
				}
			}

			// If no files found, try extracting HTML from output and save it
			htmlContent := extractHTMLFromOutput(outputStr)
			if htmlContent != "" {
				htmlFilePath := filepath.Join(workingDir, "index.html")
				if err := os.WriteFile(htmlFilePath, []byte(htmlContent), 0644); err != nil {
					log.Printf("[ERROR] Failed to save extracted HTML: %v", err)
					lastErr = fmt.Errorf("failed to save HTML: %v", err)
					continue
				}
				log.Printf("[SUCCESS] API key %d generated HTML and saved to index.html", i+1)
				return nil
			}

			log.Printf("[WARNING] API key %d generated output but no files created", i+1)
			lastErr = fmt.Errorf("no files created in directory")
			continue
		}

		// Check if error is due to signal killed
		if strings.Contains(err.Error(), "signal: killed") || strings.Contains(err.Error(), "killed") {
			log.Printf("[WARNING] API key %d process was killed, trying next key...", i+1)
			lastErr = fmt.Errorf("API key %d process killed (timeout or resource limit)", i+1)
			continue
		}

		// Log detailed failure
		trunc := outputStr
		if len(trunc) > 200 {
			trunc = trunc[:200]
		}
		log.Printf("[ERROR] API key %d failed: %v | output: %q", i+1, err, trunc)
		lastErr = fmt.Errorf("API key %d last failure: %v | output: %s", i+1, err, trunc)
	}

	return fmt.Errorf("all %d Limit telah habis, silahkan upgrade ke premium untuk melanjutkan penggunaan. Last error: %v", len(keys), lastErr)
}

// generateHTMLFromPrompt generates HTML content from prompt using cursor-agent (legacy function)
func generateHTMLFromPrompt(ctx context.Context, prompt string) (string, error) {
	keys := apiKeyManager.GetAllKeys()
	var lastErr error

	// Enhanced prompt for HTML generation
	htmlGenPrompt := fmt.Sprintf("You are a web development assistant. Create a complete HTML page based on this prompt: \"%s\"\n\n"+
		"Generate a complete, modern, and responsive HTML page with inline CSS and JavaScript if needed.\n"+
		"Make sure the HTML is complete, functional, and modern. Use best practices for HTML5, CSS3, and vanilla JavaScript.\n"+
		"Return only the HTML code without markdown code blocks or explanations.", prompt)

	for i, apiKey := range keys {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Build args: cursor-agent --api-key <key> -p --output-format text <prompt>
		args := []string{"--api-key", apiKey, htmlGenPrompt, "-p", "--output-format", "text"}

		// Execute with context
		cmd := exec.CommandContext(ctx, "cursor-agent", args...)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err == nil {
			// Check if output contains textual error indicators
			lower := strings.ToLower(outputStr)
			if strings.Contains(lower, "connecterror") || strings.Contains(lower, "resource_exhausted") ||
				strings.Contains(lower, "resource exhausted") || strings.Contains(lower, "rate limit") ||
				strings.Contains(lower, "unauthorized") || strings.Contains(lower, "quota") {
				log.Printf("[WARNING] API key %d returned error text, trying next...", i+1)
				lastErr = fmt.Errorf("API key %d textual error: %s", i+1, outputStr[:min(200, len(outputStr))])
				continue
			}

			// Extract HTML from output (remove markdown code blocks if present)
			htmlContent := extractHTMLFromOutput(outputStr)
			if htmlContent != "" {
				log.Printf("[SUCCESS] API key %d generated HTML", i+1)
				return htmlContent, nil
			}

			// If no HTML extracted, try using raw output
			log.Printf("[WARNING] API key %d generated output but no HTML extracted, using raw output", i+1)
			lastErr = fmt.Errorf("no HTML extracted from output")
			continue
		}

		// Check if error is due to signal killed (context timeout or OS kill)
		// Check if it's a signal killed error
		if strings.Contains(err.Error(), "signal: killed") || strings.Contains(err.Error(), "killed") {
			log.Printf("[WARNING] API key %d process was killed, trying next key...", i+1)
			lastErr = fmt.Errorf("API key %d process killed (timeout or resource limit)", i+1)
			continue
		}

		// Log detailed failure
		trunc := outputStr
		if len(trunc) > 200 {
			trunc = trunc[:200]
		}
		log.Printf("[ERROR] API key %d failed: %v | output: %q", i+1, err, trunc)
		lastErr = fmt.Errorf("API key %d last failure: %v | output: %s", i+1, err, trunc)
	}

	return "", fmt.Errorf("all %d Limit telah habis, silahkan upgrade ke premium untuk melanjutkan penggunaan. Last error: %v", len(keys), lastErr)
}

// extractHTMLFromOutput extracts HTML content from cursor-agent output
func extractHTMLFromOutput(output string) string {
	// Try to extract HTML from markdown code blocks
	htmlPattern := regexp.MustCompile(`(?s)` + "`" + "`" + "`" + `html\s*\n(.*?)` + "`" + "`" + "`")
	matches := htmlPattern.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}

	// Try to extract HTML tags directly
	htmlTagPattern := regexp.MustCompile(`(?s)(<html[^>]*>.*?</html>)`)
	if htmlMatch := htmlTagPattern.FindString(output); htmlMatch != "" {
		return htmlMatch
	}

	// Try standard code blocks
	standardPattern := regexp.MustCompile(`(?s)` + "`" + "`" + "`" + `\s*\n(.*?)` + "`" + "`" + "`")
	standardMatches := standardPattern.FindStringSubmatch(output)
	if len(standardMatches) >= 2 {
		content := strings.TrimSpace(standardMatches[1])
		// Check if it looks like HTML
		if strings.Contains(content, "<html") || strings.Contains(content, "<!DOCTYPE") {
			return content
		}
	}

	// If output contains HTML tags, return the whole output
	if strings.Contains(output, "<html") || strings.Contains(output, "<!DOCTYPE") {
		return strings.TrimSpace(output)
	}

	return ""
}

// previewHandler serves preview of generated app
func previewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract app ID from path /preview/:appId
	path := strings.TrimPrefix(r.URL.Path, "/preview/")
	if path == "" {
		http.Error(w, "App ID is required", http.StatusBadRequest)
		return
	}

	// Split path to get app ID and subpath
	parts := strings.SplitN(path, "/", 2)
	appID := parts[0]

	if appID == "" {
		http.Error(w, "App ID is required", http.StatusBadRequest)
		return
	}

	appDir := filepath.Join("generated-apps", appID)

	// Check if directory exists
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		http.Error(w, "App not found", http.StatusNotFound)
		return
	}

	// If root path or ends with /, try to find HTML file (prefer any HTML file, fallback to index.html)
	if len(parts) == 1 || parts[1] == "" || strings.HasSuffix(r.URL.Path, "/") {
		// First, try to find any HTML file
		entries, err := os.ReadDir(appDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
					htmlPath := filepath.Join(appDir, entry.Name())
					http.ServeFile(w, r, htmlPath)
					return
				}
			}
		}

		// Fallback to index.html
		indexPath := filepath.Join(appDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}
	}

	// Strip /preview/:appId prefix and serve file
	// Rewrite URL to point to the file in app directory
	subPath := ""
	if len(parts) > 1 {
		subPath = parts[1]
	}

	filePath := filepath.Join(appDir, subPath)

	// Security: prevent directory traversal
	if !strings.HasPrefix(filePath, appDir) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Serve the file
	http.ServeFile(w, r, filePath)
}

// FileInfo represents a file or directory in the explorer
type FileInfo struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"` // "file" or "directory"
	URL      string     `json:"url,omitempty"`
	Children []FileInfo `json:"children,omitempty"`
}

// listAppsHandler returns list of generated apps (folders and files)
func listAppsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	generatedAppsDir := "generated-apps"

	// Ensure directory exists
	if err := os.MkdirAll(generatedAppsDir, 0755); err != nil {
		log.Printf("[ERROR] /list-apps - Failed to create directory: %v", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	// Read directory
	entries, err := os.ReadDir(generatedAppsDir)
	if err != nil {
		log.Printf("[ERROR] /list-apps - Failed to read directory: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]FileInfo{})
		return
	}

	var apps []FileInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files in root
		}

		appPath := filepath.Join(generatedAppsDir, entry.Name())
		appInfo := FileInfo{
			Name: entry.Name(),
			Path: appPath,
			Type: "directory",
			URL:  fmt.Sprintf("/preview/%s/", entry.Name()),
		}

		// Read files in the app directory
		appEntries, err := os.ReadDir(appPath)
		if err != nil {
			log.Printf("[WARNING] /list-apps - Failed to read app directory %s: %v", appPath, err)
			// Continue with empty children
			appInfo.Children = []FileInfo{}
		} else {
			var files []FileInfo
			for _, fileEntry := range appEntries {
				if !fileEntry.IsDir() {
					fileName := fileEntry.Name()
					// Only include HTML files or allow all files - let's include all
					filePath := filepath.Join(appPath, fileName)
					fileInfo := FileInfo{
						Name: fileName,
						Path: filePath,
						Type: "file",
						URL:  fmt.Sprintf("/preview/%s/%s", entry.Name(), fileName),
					}
					files = append(files, fileInfo)
				}
			}
			// Sort files by name
			sort.Slice(files, func(i, j int) bool {
				return files[i].Name < files[j].Name
			})
			appInfo.Children = files
		}

		apps = append(apps, appInfo)
	}

	// Sort apps by name descending (newest first: app_999, app_998, etc.)
	// Extract numeric suffix for proper numeric sorting
	sort.Slice(apps, func(i, j int) bool {
		// Extract number from app_XXX format
		extractNum := func(name string) int {
			// Remove "app_" prefix and parse number
			if strings.HasPrefix(name, "app_") {
				numStr := strings.TrimPrefix(name, "app_")
				num, err := strconv.Atoi(numStr)
				if err != nil {
					log.Printf("[WARNING] /list-apps - Failed to parse number from %s: %v", name, err)
					return 0
				}
				return num
			}
			return 0
		}

		numI := extractNum(apps[i].Name)
		numJ := extractNum(apps[j].Name)

		// Sort descending (higher number first)
		// Return true if i should come before j (i > j means i comes first)
		return numI > numJ
	})

	// Log sorted order for debugging
	if len(apps) > 0 {
		names := make([]string, len(apps))
		for i, app := range apps {
			names[i] = app.Name
		}
		// log.Printf("[DEBUG] /list-apps - Sorted apps: %v", names)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apps); err != nil {
		log.Printf("[ERROR] /list-apps - Failed to encode JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// statsHandler returns resource usage statistics
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	statsMutex.RLock()
	currentStats := ResourceStats{
		TotalRequests:       atomic.LoadUint64(&stats.TotalRequests),
		ActiveRequests:      atomic.LoadInt32(&stats.ActiveRequests),
		TotalErrors:         atomic.LoadUint64(&stats.TotalErrors),
		TotalBytesProcessed: atomic.LoadUint64(&stats.TotalBytesProcessed),
		StartTime:           stats.StartTime,
		Uptime:              time.Since(stats.StartTime).String(),
	}
	statsMutex.RUnlock()

	// Get system memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	response := map[string]interface{}{
		"stats": currentStats,
		"system": map[string]interface{}{
			"memory": map[string]interface{}{
				"allocated_mb":       float64(m.Alloc) / 1024 / 1024,
				"total_allocated_mb": float64(m.TotalAlloc) / 1024 / 1024,
				"system_mb":          float64(m.Sys) / 1024 / 1024,
				"num_gc":             m.NumGC,
			},
			"goroutines":              runtime.NumGoroutine(),
			"max_concurrent_requests": MaxConcurrentRequests,
			"buffer_read_size_kb":     BufferReadSize / 1024,
			"max_buffer_size_mb":      MaxBufferSize / 1024 / 1024,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// rootHandler serves ai.html as the main page
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// Only serve HTML for GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Serve ai.html as the main page
	// Try ai.html first, then fallback to index.html
	aiHTMLPath := "ai.html"
	if _, err := os.Stat(aiHTMLPath); err == nil {
		http.ServeFile(w, r, aiHTMLPath)
		return
	}

	// Fallback to index.html if ai.html doesn't exist
	indexHTMLPath := "index.html"
	if _, err := os.Stat(indexHTMLPath); err == nil {
		http.ServeFile(w, r, indexHTMLPath)
		return
	}

	// If neither exists, serve static files from current directory
	fs := http.FileServer(http.Dir("."))
	fs.ServeHTTP(w, r)
}

func main() {
	// Initialize app counter by reading existing folders
	initializeAppCounter()

	// Register API endpoints first (more specific routes)
	http.Handle("/ask", compressionMiddleware(loggingMiddleware(cors(http.HandlerFunc(askHandler)))))
	http.Handle("/ask-stream", loggingMiddleware(cors(http.HandlerFunc(askStreamHandler))))
	http.Handle("/ask-gemini", compressionMiddleware(loggingMiddleware(cors(http.HandlerFunc(askGeminiHandler)))))
	http.Handle("/ask-gemini-stream", loggingMiddleware(cors(http.HandlerFunc(askGeminiStreamHandler))))
	http.Handle("/stats", compressionMiddleware(cors(http.HandlerFunc(statsHandler))))
	http.Handle("/create-app", compressionMiddleware(loggingMiddleware(cors(http.HandlerFunc(createAppHandler)))))
	http.Handle("/create-folder", compressionMiddleware(loggingMiddleware(cors(http.HandlerFunc(createFolderHandler)))))
	http.Handle("/job-status", compressionMiddleware(cors(http.HandlerFunc(jobStatusHandler))))
	http.Handle("/list-apps", compressionMiddleware(cors(http.HandlerFunc(listAppsHandler))))
	http.Handle("/preview/", loggingMiddleware(cors(http.HandlerFunc(previewHandler))))

	// Serve ai.html as main page at root (catch-all, must be last)
	http.Handle("/", compressionMiddleware(cors(http.HandlerFunc(rootHandler))))

	fmt.Println("🎉 Cursor API Server running at http://localhost:8080")
	fmt.Println("")
	fmt.Println("📱 Web Interface:")
	fmt.Println("  - GET  / (main UI - ai.html)")
	fmt.Println("  - GET  /ai.html (main UI)")
	fmt.Println("")
	fmt.Println("🔌 API Endpoints:")
	fmt.Println("  - POST /ask (Cursor regular response)")
	fmt.Println("  - POST /ask-stream (Cursor streaming response)")
	fmt.Println("  - POST /ask-gemini (Gemini regular response)")
	fmt.Println("  - POST /ask-gemini-stream (Gemini streaming response)")
	fmt.Println("  - POST /create-app (generate app from prompt)")
	fmt.Println("  - POST /create-folder (create folder and save HTML file - async)")
	fmt.Println("  - GET  /job-status?job_id=xxx (get job status)")
	fmt.Println("  - GET  /list-apps (list all generated apps)")
	fmt.Println("  - GET  /preview/:appId (preview generated app)")
	fmt.Println("  - GET  /stats (resource usage statistics)")
	fmt.Println("")
	fmt.Println("?? API Key Configuration:")
	fmt.Printf("  - Loaded %d API key(s) for fallback mechanism\n", len(apiKeyManager.GetAllKeys()))
	fmt.Println("  - API keys are declared directly in code (APIKeys variable)")
	fmt.Println("")
	fmt.Println("? Features:")
	fmt.Println("  - Automatic API key fallback (if one fails, tries next)")
	fmt.Println("  - Fresh chat for every request (no chat history)")
	fmt.Println("  - Round-robin key rotation for load balancing")
	fmt.Println("  - Resource usage monitoring via /stats endpoint")
	fmt.Println("")
	fmt.Println("?? Resource Limits:")
	fmt.Printf("  - Max concurrent requests: %d\n", MaxConcurrentRequests)
	fmt.Printf("  - Max buffer size: %d MB\n", MaxBufferSize/1024/1024)
	fmt.Printf("  - Buffer read size: %d KB\n", BufferReadSize/1024)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
