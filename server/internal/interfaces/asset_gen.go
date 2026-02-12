package interfaces

import "context"

// ImageRequest represents a request to generate an image
type ImageRequest struct {
	Prompt      string
	NegativePrompt string
	Width       int
	Height      int
	Steps       int
	CFGScale    float64
	Seed        int64
	LoRA        string // Optional LoRA model name
}

// ImageResponse represents the response from image generation
type ImageResponse struct {
	ImagePath  string
	ImageURL   string
	Seed       int64
	GenerationTime int64 // milliseconds
}

// AudioRequest represents a request to generate audio
type AudioRequest struct {
	Text      string
	VoiceName string // 声音模型名称
	Speed     float64
	Pitch     float64
}

// AudioResponse represents the response from audio generation
type AudioResponse struct {
	AudioPath  string
	AudioURL   string
	Duration   float64 // seconds
}

// GeneratorStatus represents the status of a generator
type GeneratorStatus struct {
	IsAvailable bool
	QueueSize   int
	LastError   string
}

// AssetGen defines the interface for asset generation (images, audio)
type AssetGen interface {
	// GenerateImage generates an image from a prompt
	GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResponse, error)

	// PreloadModels preloads models to reduce generation latency
	PreloadModels(ctx context.Context, models []string) error

	// GetStatus returns the current status of the generator
	GetStatus(ctx context.Context) (*GeneratorStatus, error)
}

// TTSGenerator defines the interface for text-to-speech
type TTSGenerator interface {
	// GenerateAudio generates audio from text
	GenerateAudio(ctx context.Context, req *AudioRequest) (*AudioResponse, error)

	// GetAvailableVoices returns list of available voice models
	GetAvailableVoices(ctx context.Context) ([]string, error)
}
