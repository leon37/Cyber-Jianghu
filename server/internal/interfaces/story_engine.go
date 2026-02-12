package interfaces

import "context"

// StoryContext represents the current story state
type StoryContext struct {
	SessionID    string                 // 直播会话ID
	CurrentScene string                 // 当前场景
	Characters   map[string]*Character  // 角色状态
	Inventory    map[string]int         // 物品栏
	Flags        map[string]interface{} // 剧情标记
	Metadata     map[string]interface{} // 其他元数据
}

// Character represents a character in the story
type Character struct {
	ID       string
	Name     string
	State    map[string]interface{} // 角色状态（血量、位置等）
	Relation string                  // 与主角关系
}

// StoryRequest represents a request to generate story content
type StoryRequest struct {
	Danmaku      *Danmaku
	Context      *StoryContext
	UserAction   string // 用户指令（如 "/attack", "/talk"）
	RetrievedMemories []*Memory // RAG检索到的历史记忆
}

// StoryResponse represents the generated story response
type StoryResponse struct {
	Narrative     string // 剧情文本
	Action        string // 动作描述
	SceneChange   bool   // 是否切换场景
	NewScene      string // 新场景名
	CharacterUpdates map[string]*Character // 角色状态更新
	MemoriesToStore []*Memory // 需要存储的记忆
	ImagePrompt   string // 生成的图像提示词
	AudioText     string // 需要朗读的文本
}

// StoryEngine defines the interface for story generation
type StoryEngine interface {
	// GenerateStory generates story content based on user input
	GenerateStory(ctx context.Context, req *StoryRequest) (*StoryResponse, error)

	// InitializeStory initializes a new story session
	InitializeStory(ctx context.Context, sessionID string) (*StoryContext, error)

	// GetContext retrieves the current story context
	GetContext(ctx context.Context, sessionID string) (*StoryContext, error)

	// UpdateContext updates the story context
	UpdateContext(ctx context.Context, sessionID string, updates map[string]interface{}) error
}
