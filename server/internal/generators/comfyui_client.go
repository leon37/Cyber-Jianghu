package generators

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	comfyUIHost         = "localhost"
	comfyUIPort         = 8188
	comfyBaseURL        = "http://localhost:8188"
	defaultTimeout     = 300 * time.Second
	pollInterval      = 1 * time.Second
	maxPollAttempts   = 300 // 5 minutes max wait time
)

// ComfyUIClient connects to local ComfyUI instance
type ComfyUIClient struct {
	httpClient *http.Client
	baseURL    string
}

// Workflow represents a ComfyUI workflow - use integer node IDs
type Workflow map[int]*WorkflowNode

// MarshalJSON implements custom JSON marshaling for Workflow
func (w *Workflow) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[int]*WorkflowNode(*w))
}

// UnmarshalJSON implements custom JSON unmarshaling for Workflow
func (w *Workflow) UnmarshalJSON(data []byte) error {
	var nodes map[int]*WorkflowNode
	if err := json.Unmarshal(data, &nodes); err != nil {
		return err
	}
	*w = Workflow(nodes)
	return nil
}

// WorkflowNode represents a node in the workflow
type WorkflowNode struct {
	ClassType string                 `json:"class_type"`
	Inputs    map[string]interface{} `json:"inputs"`
}

// PromptRequest represents a prompt generation request
type PromptRequest struct {
	Prompt   Workflow `json:"prompt"`
	ClientID string   `json:"client_id"`
}

// QueueResponse represents the queue status
type QueueResponse struct {
	QueueRunning []QueueItem `json:"queue_running"`
	QueuePending []QueueItem `json:"queue_pending"`
}

// QueueItem represents an item in the queue
type QueueItem struct {
	PromptID    []int `json:"prompt"`
	Additional  map[string]interface{} `json:"additional_info"`
}

// HistoryResponse represents generation history
type HistoryResponse struct {
	Queue  map[string]HistoryItem `json:"queue_running"`
}

// HistoryItem represents a history item
type HistoryItem struct {
	Prompt  []map[string]interface{} `json:"prompt"`
	Outputs map[string]struct {
		Images []ImageInfo `json:"images"`
	} `json:"outputs"`
}

// ImageInfo represents an image in history
type ImageInfo struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

// GenerateOptions holds options for image generation
type GenerateOptions struct {
	Prompt        string
	NegativePrompt string
	Width         int
	Height        int
	Steps         int
	CFGScale      float64
	Seed          int
	Model         string
	Lora          string // LoRA model to use
	LoraStrength  float64
	SamplerName   string
	Scheduler     string
}

// GenerateResult represents the result of image generation
type GenerateResult struct {
	ImageID    string
	ImageData  []byte
	ImageBase64 string
	Filename   string
	Width      int
	Height     int
	Duration   time.Duration
}

// NewComfyUIClient creates a new ComfyUI client
func NewComfyUIClient() *ComfyUIClient {
	return &ComfyUIClient{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: comfyBaseURL,
	}
}

// GenerateImage generates an image using ComfyUI
func (c *ComfyUIClient) GenerateImage(ctx context.Context, opts *GenerateOptions) (*GenerateResult, error) {
	// Build workflow from options
	workflow := c.buildSDXLWorkflow(opts)

	// Create prompt request
	req := &PromptRequest{
		Prompt:   *workflow,
		ClientID: generateClientID(),
	}

	// Send prompt to queue
	promptID, err := c.queuePrompt(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to queue prompt: %w", err)
	}

	// Poll for completion
	result, err := c.pollForResult(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	result.Duration = time.Since(time.Now())

	return result, nil
}

// GenerateImageAsync generates an image asynchronously
func (c *ComfyUIClient) GenerateImageAsync(ctx context.Context, opts *GenerateOptions) (string, error) {
	workflow := c.buildSDXLWorkflow(opts)

	req := &PromptRequest{
		Prompt:   *workflow,
		ClientID: generateClientID(),
	}

	return c.queuePrompt(ctx, req)
}

// GetQueueStatus returns the current queue status
func (c *ComfyUIClient) GetQueueStatus(ctx context.Context) (*QueueResponse, error) {
	url := fmt.Sprintf("%s/queue", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var queueResp QueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&queueResp); err != nil {
		return nil, err
	}

	return &queueResp, nil
}

// GetHistory returns generation history
func (c *ComfyUIClient) GetHistory(ctx context.Context) (*HistoryResponse, error) {
	url := fmt.Sprintf("%s/history", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var historyResp HistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&historyResp); err != nil {
		return nil, err
	}

	return &historyResp, nil
}

// GetImage retrieves an image by filename
func (c *ComfyUIClient) GetImage(ctx context.Context, filename, subfolder string) ([]byte, error) {
	url := fmt.Sprintf("%s/view?filename=%s", c.baseURL, filename)
	if subfolder != "" {
		url += fmt.Sprintf("&subfolder=%s", subfolder)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

// GetImageBase64 retrieves an image as base64 string
func (c *ComfyUIClient) GetImageBase64(ctx context.Context, filename, subfolder string) (string, error) {
	data, err := c.GetImage(ctx, filename, subfolder)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// queuePrompt sends a prompt to the queue
func (c *ComfyUIClient) queuePrompt(ctx context.Context, req *PromptRequest) (string, error) {
	url := fmt.Sprintf("%s/prompt", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	log.Printf("ComfyUI queuePrompt: Sending request to %s", url)
	log.Printf("ComfyUI queuePrompt: Request body: %s", string(reqBody))

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("ComfyUI queuePrompt: HTTP error: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	// Read body for logging
	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("ComfyUI queuePrompt: Response status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	var result map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return "", err
	}

	promptID, ok := result["prompt_id"].(float64)
	if !ok {
		log.Printf("ComfyUI queuePrompt: Response missing prompt_id, full response: %+v", result)
		return "", fmt.Errorf("invalid response: missing prompt_id")
	}

	log.Printf("ComfyUI queuePrompt: Got prompt_id=%.0f", promptID)
	return fmt.Sprintf("%.0f", promptID), nil
}

// pollForResult polls for the generation result
func (c *ComfyUIClient) pollForResult(ctx context.Context, promptID string) (*GenerateResult, error) {
	for attempt := 0; attempt < maxPollAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
			// Check history for our prompt
			history, err := c.GetHistory(ctx)
			if err != nil {
				continue
			}

			// Search for our prompt in history
			for key, item := range history.Queue {
				if key == promptID || len(item.Prompt) == 0 {
					// Found our result
					if len(item.Outputs) > 0 {
						for _, output := range item.Outputs {
							if len(output.Images) > 0 {
								img := output.Images[0]
								imageData, err := c.GetImage(ctx, img.Filename, img.Subfolder)
								if err != nil {
									return nil, fmt.Errorf("failed to get image: %w", err)
								}

								return &GenerateResult{
									ImageID:     promptID,
									ImageData:   imageData,
									ImageBase64: "",
									Filename:    img.Filename,
								}, nil
							}
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("timeout waiting for image generation")
}

// buildSDXLWorkflow builds a workflow for SDXL Turbo
func (c *ComfyUIClient) buildSDXLWorkflow(opts *GenerateOptions) *Workflow {
	// Apply defaults
	if opts.Width == 0 {
		opts.Width = 1024
	}
	if opts.Height == 0 {
		opts.Height = 1024
	}
	if opts.Steps == 0 {
		opts.Steps = 8 // SDXL Turbo needs fewer steps
	}
	if opts.CFGScale == 0 {
		opts.CFGScale = 7.0
	}

	// Create workflow nodes
	workflow := make(Workflow)

	// Load checkpoint (provides MODEL and CLIP)
	workflow[4] = &WorkflowNode{
		ClassType: "CheckpointLoaderSimple",
		Inputs: map[string]interface{}{
			"ckpt_name": opts.Model,
		},
	}

	// Load VAE separately (use the same checkpoint file for SDXL)
	workflow[1] = &WorkflowNode{
		ClassType: "VAELoader",
		Inputs: map[string]interface{}{
			"vae_name": opts.Model,
		},
	}

	workflow[3] = &WorkflowNode{
		ClassType: "KSampler",
		Inputs: map[string]interface{}{
			"seed":         opts.Seed,
			"steps":        opts.Steps,
			"cfg":          opts.CFGScale,
			"sampler_name": opts.SamplerName,
			"scheduler":    opts.Scheduler,
			"denoise":      1,
			"model":        []interface{}{4, 0},
			"positive":     []interface{}{6, 0},
			"negative":     []interface{}{7, 0},
			"latent_image": []interface{}{5, 0},
		},
	}

	workflow[5] = &WorkflowNode{
		ClassType: "EmptyLatentImage",
		Inputs: map[string]interface{}{
			"width":      opts.Width,
			"height":     opts.Height,
			"batch_size": 1,
		},
	}

	workflow[6] = &WorkflowNode{
		ClassType: "CLIPTextEncode",
		Inputs: map[string]interface{}{
			"text": opts.Prompt,
			"clip": []interface{}{4, 1},
		},
	}

	workflow[7] = &WorkflowNode{
		ClassType: "CLIPTextEncode",
		Inputs: map[string]interface{}{
			"text": opts.NegativePrompt,
			"clip": []interface{}{4, 1},
		},
	}

	workflow[8] = &WorkflowNode{
		ClassType: "VAEDecode",
		Inputs: map[string]interface{}{
			"samples": []interface{}{3, 0},
			"vae":      []interface{}{1, 0},
		},
	}

	workflow[9] = &WorkflowNode{
		ClassType: "SaveImage",
		Inputs: map[string]interface{}{
			"images":          []interface{}{8, 0},
			"filename_prefix": generateFilenamePrefix(),
		},
	}

	// Add LoRA if specified
	if opts.Lora != "" {
		workflow[10] = &WorkflowNode{
			ClassType: "LoraLoader",
			Inputs: map[string]interface{}{
				"lora_name":      opts.Lora,
				"strength_model":  opts.LoraStrength,
				"strength_clip":   opts.LoraStrength,
			},
		}
		// Add model and clip inputs to LoraLoader
		workflow[10].Inputs["model"] = []interface{}{4, 0}
		workflow[10].Inputs["clip"] = []interface{}{4, 1}
		// Update KSampler to use Lora output
		workflow[3].Inputs["model"] = []interface{}{10, 0}
		// Update CLIP encoders to use Lora output
		workflow[6].Inputs["clip"] = []interface{}{10, 1}
		workflow[7].Inputs["clip"] = []interface{}{10, 1}
	}

	return &workflow
}

// HealthCheck checks if ComfyUI is accessible
func (c *ComfyUIClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/queue", c.baseURL)

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
		return fmt.Errorf("ComfyUI returned status %d", resp.StatusCode)
	}

	return nil
}

// Helper functions
func generateClientID() string {
	return fmt.Sprintf("cyber_jianghu_%d", time.Now().UnixNano())
}

func generateFilenamePrefix() string {
	return fmt.Sprintf("cyber_jianghu_%d", time.Now().Unix())
}
