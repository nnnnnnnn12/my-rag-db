package main

import (

	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Config struct {
	Synonyms map[string][]string `json:"synonyms"`
}

// --- AI API 相关结构体 ---
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func loadConfig(fileName string) (*Config, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
// 加载文件夹下的所有文本文件内容
func loadAllDocs(folderPath string) ([]string, error) {
	var allDocs []string
	// 读取文件夹下的所有文件
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// 只读取以 .txt 结尾的文件
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			filePath := folderPath + "/" + file.Name()
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("读取文件 %s 失败: %v\n", file.Name(), err)
				continue
			}
			// 将内容按行拆分并加入知识库
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					allDocs = append(allDocs, line)
				}
			}
		}
	}
	return allDocs, nil
}
// 调用 AI 的函数
func askAI(apiKey, prompt string) string {
	apiUrl := "https://api.deepseek.com/chat/completions"
	reqBody := ChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "网络错误: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var chatResp ChatResponse
	json.Unmarshal(body, &chatResp)

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content
	}
	return "AI 没能给出回复"
}

// 模拟语义打分：根据关键词和预设的语义关系计算匹配度
func calculateScore(doc, query string, config *Config) float64 {
	score := 0.0
	doc = strings.ToLower(doc)
	query = strings.ToLower(query)

	// 基础匹配：如果直接包含关键词，给最高分
	if strings.Contains(doc, query) {
		score += 10.0
	}

	// --- 也就是你刚才问的那段代码，添加在这里 ---
	for key, words := range config.Synonyms {
		// 只要 query 包含 key（例如“冷”），或者包含 words 里的任何一个（例如“气温”），就视为命中
		match := strings.Contains(query, key)
		if !match {
			for _, w := range words {
				if strings.Contains(query, w) {
					match = true
					break
				}
			}
		}

		// 如果用户提问命中了语义词，且文档(doc)里含有核心词(key)，则加分
		if match && strings.Contains(doc, key) {
			score += 5.0
		}
	}
	return score
}

// --- 主逻辑 ---
func main() {
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Println("加载配置失败:", err)
		return
	}
	apiKey := "sk-7fc194096e114465a32221fe902c4ea0" // 替换为真实的 Key


	// --- 粘贴这段新代码 ---
// 2. 加载 docs 文件夹下的所有知识 (确保你已经写好了 loadAllDocs 函数)
knowledgeBase, err := loadAllDocs("docs")
if err != nil {
    fmt.Println("加载知识库失败:", err)
    return
}
fmt.Printf(">>> 成功加载了 %d 条知识条目。\n", len(knowledgeBase))
// ---------------------
	// 2. 获取用户提问
	var query string
	fmt.Print("请输入您想咨询的问题关键词: ")
	fmt.Scanln(&query)

	// 3. 检索最相关的上下文
	// 3. 检索最相关的上下文（升级版：从 Contains 变为 Score 打分）
	var bestContext string
	var maxScore float64

	fmt.Println(">>> 正在进行智能语义匹配...")
	for _, doc := range knowledgeBase {
		score := calculateScore(doc, query, config)
		if score > maxScore {
			maxScore = score
			bestContext = doc
		}
	}

	// 结果判断
	if maxScore == 0 {
		fmt.Println("⚠️ 本地未检索到相关内容，将由 AI 自由发挥...")
		bestContext = "无相关本地背景知识。"
	} else {
		fmt.Printf("🎯 命中本地知识 (匹配分: %.1f): %s\n", maxScore, bestContext)
	}

	// 4. 构造 RAG 专属 Prompt
	// 这是 RAG 的核心：告诉 AI，根据我给你的背景资料来回答
	finalPrompt := fmt.Sprintf(`你是我的私人助理。
背景资料：
"""
%s
"""
用户问题：%s
请结合背景资料，用亲切的语气回答用户。`, bestContext, query)

	fmt.Println("\n>>> 正在检索并请求 AI 生成回答...")

	// 5. 获取 AI 回复
	answer := askAI(apiKey, finalPrompt)

	fmt.Println("\n--------------------------------")
	fmt.Println("AI 助手的回答：")
	fmt.Println(answer)
	fmt.Println("--------------------------------")
}
