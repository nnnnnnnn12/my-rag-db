package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

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
func calculateScore(doc, query string) float64 {
	score := 0.0
	doc = strings.ToLower(doc)
	query = strings.ToLower(query)

	// 1. 硬匹配：直接包含关键词，给最高分
	if strings.Contains(doc, query) {
		score += 10.0
	}

	// 2. 模拟语义联想：这是 RAG 实习面试常考的“知识图谱”或“语义搜索”思想
	// 我们手动模拟一些向量数据库能自动识别的“意思相近”词
	synonyms := map[string][]string{
		"冷":  {"气温", "温度", "冬季", "寒冷", "冰", "湿冷"},
		"ai": {"人工智能", "机器人", "模型", "deepseek", "rag", "vla"},
		"go": {"编程", "后端", "开发", "并发", "计算机"},
	}

	for key, words := range synonyms {
		// 逻辑：如果用户搜的词(query)在我们的关联词表(words)里
		// 或者用户搜的词就是 key 本身
		userTalkingAboutThisTopic := false
		if strings.Contains(query, key) {
			userTalkingAboutThisTopic = true
		}
		for _, word := range words {
			if strings.Contains(query, word) {
				userTalkingAboutThisTopic = true
			}
		}

		// 如果确定用户在聊这个话题，就去文档里找对应的关键词
		if userTalkingAboutThisTopic {
			if strings.Contains(doc, key) {
				score += 5.0
			}
			for _, word := range words {
				if strings.Contains(doc, word) {
					score += 2.0 // 命中相关词也加分
				}
			}
		}
	}
	return score
}

// --- 主逻辑 ---
func main() {
	apiKey := "sk-7fc194096e114465a32221fe902c4ea0" // 替换为真实的 Key

	// 1. 加载本地知识库 (data.txt)
	file, _ := os.Open("data.txt")
	defer file.Close()
	var knowledgeBase []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		knowledgeBase = append(knowledgeBase, scanner.Text())
	}

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
		score := calculateScore(doc, query)
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
