package generators

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	sovitsHost         = "localhost"
	sovitsPort         = 9880
	sovitsBaseURL     = "http://localhost:9880"
	defaultTimeout    = 60 * time.Second
)

// GPTSoVITSClient connects to local GPT-SoVITS instance
type GPTSoVITSClient struct {
	httpClient *http.Client
	baseURL    string
}

// TTSRequest represents a text-to-speech request
type TTSRequest struct {
	Text            string  `json:"text"`
	ReferenceAudio   string  `json:"reference_audio,omitempty"`
	Language        string  `json:"language,omitempty"`
	Speed           float64 `json:"speed,omitempty"`
	Tone            string  `json:"tone,omitempty"`
}

// TTSResponse represents the TTS response
type TTSResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message,omitempty"`
	AudioData []byte `json:"-"`
	Base64    string `json:"audio_data,omitempty"`
	Duration  float64 `json:"duration,omitempty"`
	SampleRate int    `json:"sample_rate,omitempty"`
}

// VoiceModel represents a voice model
type VoiceModel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Gender      string                 `json:"gender"`
	Language    string                 `json:"language"`
	Style       string                 `json:"style"`
	ReferencePath string                 `json:"reference_path"`
	Description string                 `json:"description"`
	Enabled     bool                   `json:"enabled"`
}

// VoiceRegistry manages voice models
type VoiceRegistry struct {
	voices     map[string]*VoiceModel
	directory  string
	mu          sync.RWMutex
}

// NewGPTSoVITSClient creates a new GPT-SoVITS client
func NewGPTSoVITSClient() *GPTSoVITSClient {
	return &GPTSoVITSClient{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: sovitsBaseURL,
	}
}

// Synthesize synthesizes text to speech
func (c *GPTSoVITSClient) Synthesize(ctx context.Context, text string, voiceID string, options *TTSRequest) ([]byte, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// Build request
	reqBody := &TTSRequest{
		Text:     text,
		Language: "zh", // Default to Chinese
		Speed:    1.0, // Default speed
	}

	// Apply options
	if options != nil {
		if options.ReferenceAudio != "" {
			reqBody.ReferenceAudio = options.ReferenceAudio
		}
		if options.Language != "" {
			reqBody.Language = options.Language
		}
		if options.Speed > 0 {
			reqBody.Speed = options.Speed
		}
		if options.Tone != "" {
			reqBody.Tone = options.Tone
		}
	}

	// Add voice reference if voiceID specified
	if voiceID != "" {
		reqBody.ReferenceAudio = voiceID
	}

	// Marshal request
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	url := fmt.Sprintf("%s/tts", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error response
	var ttsResp TTSResponse
	if resp.StatusCode == http.StatusOK {
		// Parse JSON response
		if err := json.Unmarshal(audioData, &ttsResp); err == nil {
			if !ttsResp.Success {
				return nil, fmt.Errorf("TTS failed: %s", ttsResp.Message)
			}

			// Decode base64 if present
			if ttsResp.Base64 != "" {
				decoded, err := base64.StdEncoding.DecodeString(ttsResp.Base64)
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64: %w", err)
				}
				return decoded, nil
			}
		}

		// Return raw audio data
		return audioData, nil
	}

	// Check for direct audio response (binary)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "audio/wav" || contentType == "audio/mpeg" || contentType == "audio/mp3" {
		return audioData, nil
	}

	return nil, fmt.Errorf("unexpected response: status %d, content-type %s", resp.StatusCode, contentType)
}

// SynthesizeAsync synthesizes text to speech asynchronously
func (c *GPTSoVITSClient) SynthesizeAsync(ctx context.Context, text string, voiceID string) (string, error) {
	// For async, return a task ID
	// GPT-SoVITS may need additional async endpoint
	// Simplified implementation
	return "", fmt.Errorf("async synthesis not yet implemented")
}

// GetAvailableVoices retrieves available voice models
func (c *GPTSoVITSClient) GetAvailableVoices(ctx context.Context) ([]*VoiceModel, error) {
	url := fmt.Sprintf("%s/voices", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get voices: status %d", resp.StatusCode)
	}

	var voices []*VoiceModel
	if err := json.NewDecoder(resp.Body).Decode(&voices); err != nil {
		return nil, err
	}

	return voices, nil
}

// GetVoice retrieves a specific voice model
func (c *GPTSoVITSClient) GetVoice(ctx context.Context, voiceID string) (*VoiceModel, error) {
	voices, err := c.GetAvailableVoices(ctx)
	if err != nil {
		return nil, err
	}

	for _, voice := range voices {
		if voice.ID == voiceID {
			return voice, nil
		}
	}

	return nil, fmt.Errorf("voice not found: %s", voiceID)
}

// HealthCheck checks if GPT-SoVITS is accessible
func (c *GPTSoVITSClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GPT-SoVITS returned status %d", resp.StatusCode)
	}

	return nil
}

// NewVoiceRegistry creates a new voice registry
func NewVoiceRegistry(directory string) *VoiceRegistry {
	return &VoiceRegistry{
		voices:    make(map[string]*VoiceModel),
		directory: directory,
	}
}

// LoadVoices loads voices from the specified directory
func (r *VoiceRegistry) LoadVoices(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Scan directory for voice reference files
	// This is a simplified implementation
	// In production, this would parse actual GPT-SoVITS voice format

	voices := []*VoiceModel{
		{
			ID:          "narrator",
			Name:        "说书人",
			Gender:      "male",
			Language:    "zh",
			Style:       "classic",
			Description: "传统说书人音色，适合武侠故事",
			Enabled:     true,
		},
		{
			ID:          "male_youth",
			Name:        "青年男声",
			Gender:      "male",
			Language:    "zh",
			Style:       "modern",
			Description: "现代青年男声",
			Enabled:     true,
		},
		{
			ID:          "female",
			Name:        "女声",
			Gender:      "female",
			Language:    "zh",
			Style:       "soft",
			Description: "柔和女声",
			Enabled:     true,
		},
	}

	// Load voices
	for _, voice := range voices {
		r.voices[voice.ID] = voice
	}

	return nil
}

// GetVoice retrieves a voice by ID
func (r *VoiceRegistry) GetVoice(id string) (*VoiceModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	voice, ok := r.voices[id]
	if !ok {
		return nil, fmt.Errorf("voice not found: %s", id)
	}

	voiceCopy := *voice
	return &voiceCopy, nil
}

// ListVoices returns all voices
func (r *VoiceRegistry) ListVoices() []*VoiceModel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	voices := make([]*VoiceModel, 0, len(r.voices))
	for _, voice := range r.voices {
		voiceCopy := *voice
		voices = append(voices, &voiceCopy)
	}

	return voices
}

// EnableVoice enables a voice
func (r *VoiceRegistry) EnableVoice(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	voice, ok := r.voices[id]
	if !ok {
		return fmt.Errorf("voice not found: %s", id)
	}

	voice.Enabled = true
	return nil
}

// DisableVoice disables a voice
func (r *VoiceRegistry) DisableVoice(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	voice, ok := r.voices[id]
	if !ok {
		return fmt.Errorf("voice not found: %s", id)
	}

	voice.Enabled = false
	return nil
}

// GetDefaultVoice returns the default voice
func (r *VoiceRegistry) GetDefaultVoice() (*VoiceModel, error) {
	return r.GetVoice("narrator")
}

// TTSOptions represents synthesis options
type TTSOptions struct {
	Speed    float64 // Playback speed (0.5 to 2.0)
	Language  string   // Language code (zh, en, etc.)
	Tone      string   // Tone (classic, modern, soft, etc.)
	Format    string   // Output format (wav, mp3, etc.)
}

// NewTTSOptions creates default TTS options
func NewTTSOptions() *TTSOptions {
	return &TTSOptions{
		Speed:   1.0,
		Language: "zh",
		Format:   "wav",
	}
}

// SynthesizeWithOptions synthesizes with additional options
func (c *GPTSoVITSClient) SynthesizeWithOptions(ctx context.Context, text string, voiceID string, opts *TTSOptions) ([]byte, error) {
	if opts == nil {
		opts = NewTTSOptions()
	}

	req := &TTSRequest{
		Text:     text,
		Language: opts.Language,
		Speed:    opts.Speed,
		Tone:     opts.Tone,
	}

	return c.Synthesize(ctx, text, voiceID, req)
}

// GetAudioFormat returns the audio format based on extension
func GetAudioFormat(data []byte) string {
	// Check for WAV header
	if len(data) >= 4 && string(data[0:4]) == "RIFF" {
		return "wav"
	}

	// Check for MP3 header
	if len(data) >= 3 {
		if data[0] == 0xFF && data[1] == 0xFB && data[2] == 0x90 {
			return "mp3"
		}
	}

	return "unknown"
}
