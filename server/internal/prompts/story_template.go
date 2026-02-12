package prompts

import (
	"Cyber-Jianghu/server/internal/interfaces"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// TemplateEngine manages prompt templates
type TemplateEngine struct {
	templates map[string]*Template
	mu        sync.RWMutex
}

// Template represents a prompt template with variables
type Template struct {
	Name        string            `json:"name"`
	Content     string            `json:"content"`
	Variables   []string          `json:"variables"`
	Description string            `json:"description"`
}

// TemplateContext holds variables for template rendering
type TemplateContext struct {
	// Story context
	CurrentScene  string `json:"current_scene"`
	PlayerAction  string `json:"player_action"`
	PreviousText  string `json:"previous_text"`
	StorySummary  string `json:"story_summary"`

	// Character context
	Protagonist   string `json:"protagonist"`
	NPCs          string `json:"npcs"`

	// RAG context
	RelatedMemories string `json:"related_memories"`
	RelatedDecisions string `json:"related_decisions"`

	// Genre and style
	Genre          string `json:"genre"`
	Tone           string `json:"tone"`
	Style          string `json:"style"`

	// Additional context
	Custom         map[string]string `json:"custom"`
}

// ImagePromptContext holds context for image generation prompts
type ImagePromptContext struct {
	SceneDescription string   `json:"scene_description"`
	Style           string   `json:"style"`
	Characters      []string `json:"characters"`
	Mood            string   `json:"mood"`
	TimeOfDay       string   `json:"time_of_day"`
	Weather         string   `json:"weather"`
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		templates: make(map[string]*Template),
	}
}

// RegisterTemplate registers a new template
func (e *TemplateEngine) RegisterTemplate(tmpl *Template) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.templates[tmpl.Name] = tmpl
	return nil
}

// GetTemplate retrieves a template by name
func (e *TemplateEngine) GetTemplate(name string) (*Template, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tmpl, ok := e.templates[name]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	return tmpl, nil
}

// Render renders a template with the given context
func (e *TemplateEngine) Render(templateName string, ctx *TemplateContext) (string, error) {
	tmpl, err := e.GetTemplate(templateName)
	if err != nil {
		return "", err
	}

	return e.renderTemplate(tmpl, ctx)
}

// renderTemplate performs the actual template rendering
func (e *TemplateEngine) renderTemplate(tmpl *Template, ctx *TemplateContext) (string, error) {
	result := tmpl.Content

	// Replace variables in the format {{variable_name}}
	varRegex := regexp.MustCompile(`\{\{(\w+)\}\}`)

	result = varRegex.ReplaceAllStringFunc(result, func(match string) string {
		varName := varRegex.FindStringSubmatch(match)[1]
		value, ok := e.getVariableValue(ctx, varName)
		if ok {
			return value
		}
		return match // Keep placeholder if not found
	})

	// Handle custom variables
	if ctx.Custom != nil {
		for key, value := range ctx.Custom {
			placeholder := fmt.Sprintf("{{%s}}", key)
			result = strings.ReplaceAll(result, placeholder, value)
		}
	}

	return result, nil
}

// getVariableValue retrieves a variable value from context
func (e *TemplateEngine) getVariableValue(ctx *TemplateContext, varName string) (string, bool) {
	switch varName {
	case "current_scene":
		return ctx.CurrentScene, ctx.CurrentScene != ""
	case "player_action":
		return ctx.PlayerAction, ctx.PlayerAction != ""
	case "previous_text":
		return ctx.PreviousText, ctx.PreviousText != ""
	case "story_summary":
		return ctx.StorySummary, ctx.StorySummary != ""
	case "protagonist":
		return ctx.Protagonist, ctx.Protagonist != ""
	case "npcs":
		return ctx.NPCs, ctx.NPCs != ""
	case "related_memories":
		return ctx.RelatedMemories, ctx.RelatedMemories != ""
	case "related_decisions":
		return ctx.RelatedDecisions, ctx.RelatedDecisions != ""
	case "genre":
		return ctx.Genre, ctx.Genre != ""
	case "tone":
		return ctx.Tone, ctx.Tone != ""
	case "style":
		return ctx.Style, ctx.Style != ""
	default:
		if ctx.Custom != nil {
			if val, ok := ctx.Custom[varName]; ok {
				return val, true
			}
		}
		return "", false
	}
}

// RenderImagePrompt renders an image generation prompt
func (e *TemplateEngine) RenderImagePrompt(templateName string, ctx *ImagePromptContext) (string, error) {
	tmpl, err := e.GetTemplate(templateName)
	if err != nil {
		return "", err
	}

	result := tmpl.Content

	// Build scene description
	sceneDesc := ctx.SceneDescription
	if len(ctx.Characters) > 0 {
		sceneDesc += ", with " + strings.Join(ctx.Characters, " and ")
	}
	if ctx.Mood != "" {
		sceneDesc += ", " + ctx.Mood + " mood"
	}
	if ctx.TimeOfDay != "" {
		sceneDesc += ", " + ctx.TimeOfDay
	}
	if ctx.Weather != "" {
		sceneDesc += ", " + ctx.Weather
	}

	// Replace variables
	replacements := map[string]string{
		"scene_description": sceneDesc,
		"style":            ctx.Style,
		"characters":       strings.Join(ctx.Characters, ", "),
		"mood":             ctx.Mood,
		"time_of_day":      ctx.TimeOfDay,
		"weather":          ctx.Weather,
	}

	for key, value := range replacements {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// InitializeDefaultTemplates initializes the default story templates
func (e *TemplateEngine) InitializeDefaultTemplates() error {
	templates := []*Template{
		{
			Name:        "story_continuation",
			Description: "Main template for continuing the story",
			Content: `你是一位深谙金庸武侠风格的专业小说家，正在创作一部纯正的古龙江湖互动小说。

## 故事背景
{{story_summary}}

## 当前场景
{{current_scene}}

## 之前的剧情
{{previous_text}}

## 玩家的行为
{{player_action}}

## 相关记忆与决策
{{related_memories}}

{{related_decisions}}

## 写作要求
1. 完全使用金庸古龙江湖风格的语言和描写方式
2. 严禁出现任何现代科技、霓虹、芯片、机械、电子、AI、虚拟等词汇
3. 严禁出现'赛博'、'科幻'、'未来'、'高科技'等概念
4. 环境描写使用古典武侠元素：古道、客栈、茶楼、古庙、山洞、竹林、江湖门派
5. 兵器描写使用武侠元素：刀、剑、棍、鞭、扇、暗器，避免'能量剑'、'激光'、'电磁'等
6. 人物对话使用古风白话文，如'少侠'、'阁下'、'在下'、'姑娘'、'道长'
7. 基于{{protagonist}}的性格设定，描述其行为和反应
8. 融合当前场景的环境描写
9. 针对玩家的行为给出合理的剧情分支
10. 保持{{tone}}的语调
11. 控制在300-500字以内
12. 结尾给出2-3个供玩家选择的行为选项，选项用 A. B. C. 或 ① ② ③ 格式

请继续创作故事：`,
			Variables: []string{"story_summary", "current_scene", "previous_text", "player_action", "related_memories", "related_decisions", "protagonist", "genre", "tone"},
		},
		{
			Name:        "scene_description",
			Description: "Template for describing a scene",
			Content: `## 场景描述生成任务

请根据以下信息，生成一个生动的场景描述（100-200字）：

场景名称：{{scene_name}}
时间：{{time_of_day}}
天气：{{weather}}
环境：{{environment}}
氛围：{{mood}}

要求：
1. 使用武侠风格的描写语言
2. 包含视觉、听觉、嗅觉等多感官描述
3. 符合{{genre}}的风格`,
			Variables: []string{"scene_name", "time_of_day", "weather", "environment", "mood", "genre"},
		},
		{
			Name:        "image_generation",
			Description: "Template for generating image prompts",
			Content: `Generate a detailed image prompt for a {{genre}} style scene.

Scene: {{scene_description}}
Style: {{style}}
Characters: {{characters}}
Mood: {{mood}}
Time of day: {{time_of_day}}
Weather: {{weather}}

The image should have:
- High quality, detailed art style
- Atmospheric lighting appropriate for the mood
- Character designs consistent with wuxia aesthetics
- Rich background details matching the scene description

Do not include any text in the image.`,
			Variables: []string{"scene_description", "style", "characters", "mood", "time_of_day", "weather"},
		},
		{
			Name:        "npc_response",
			Description: "Template for NPC dialogue",
			Content: `## NPC对话生成

NPC角色：{{npc_name}}
性格特点：{{npc_personality}}
说话风格：{{npc_speaking_style}}
当前处境：{{current_situation}}
玩家的行为：{{player_action}}

请生成一段NPC的回应（50-100字）：
1. 体现NPC的性格特点
2. 回应玩家的行为
3. 符合{{genre}}的语言风格
4. 包含情感色彩（{{mood}}）`,
			Variables: []string{"npc_name", "npc_personality", "npc_speaking_style", "current_situation", "player_action", "genre", "mood"},
		},
		{
			Name:        "decision_summary",
			Description: "Template for summarizing player decisions",
			Content: `## 玩家决策总结

当前剧情节点：{{story_node}}
玩家选择：{{player_choice}}
选择理由：{{choice_reason}}

请简要总结这个决策的意义（50字以内）：`,
			Variables: []string{"story_node", "player_choice", "choice_reason"},
		},
	}

	for _, tmpl := range templates {
		if err := e.RegisterTemplate(tmpl); err != nil {
			return fmt.Errorf("failed to register template %s: %w", tmpl.Name, err)
		}
	}

	return nil
}

// BuildStoryContext builds a story context from story state and danmaku
func BuildStoryContext(story *interfaces.Story, danmaku interfaces.Danmaku, relatedMemories []string, relatedDecisions []string) *TemplateContext {
	return &TemplateContext{
		CurrentScene:    story.CurrentScene,
		PlayerAction:    danmaku.Content,
		PreviousText:    story.PreviousText,
		StorySummary:    story.Summary,
		Protagonist:     story.Protagonist,
		NPCs:            story.NPCs,
		RelatedMemories: strings.Join(relatedMemories, "\n"),
		RelatedDecisions: strings.Join(relatedDecisions, "\n"),
		Genre:           story.Genre,
		Tone:            story.Tone,
		Style:           story.Style,
	}
}

// ParseTemplateVariables extracts variables from a template
func ParseTemplateVariables(templateContent string) []string {
	varRegex := regexp.MustCompile(`\{\{(\w+)\}\}`)
	matches := varRegex.FindAllStringSubmatch(templateContent, -1)

	uniqueVars := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			uniqueVars[match[1]] = true
		}
	}

	vars := make([]string, 0, len(uniqueVars))
	for v := range uniqueVars {
		vars = append(vars, v)
	}

	return vars
}

// ExportTemplate exports a template as JSON
func (e *TemplateEngine) ExportTemplate(name string) (string, error) {
	tmpl, err := e.GetTemplate(name)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal template: %w", err)
	}

	return string(data), nil
}

// ImportTemplate imports a template from JSON
func (e *TemplateEngine) ImportTemplate(jsonData string) error {
	var tmpl Template
	if err := json.Unmarshal([]byte(jsonData), &tmpl); err != nil {
		return fmt.Errorf("failed to unmarshal template: %w", err)
	}

	// Extract variables from content
	tmpl.Variables = ParseTemplateVariables(tmpl.Content)

	return e.RegisterTemplate(&tmpl)
}
