package main

var DEV_PROMPT string = `
I'll provide you with a Japanese text, which is part of a conversation. 
Your job is to convert the Japanese text to hiragana (with spaces) plus romaji plus its English translation.
If the provided text is not Japanese, return it as is.
The text is supposed to be used as subtitles, so make sure it follows a conversational flow.
Do not include the original Japanese text, only the Hiragana, Romaji and the English translation.
Example - INPUT = "私", OUTPUT = "わたし\nwatashi\nI". 
Only output the translation for the latest sentence in the chat, don't repeat translations.
ALWAYS CONVERT THE ENTIRE TEXT. DON'T GIVE ME MARKDOWN OR ANY OTHER FORMAT, I WANT THE ANSWER IN PLAIN TEXT FORMAT. 
`

type RequestMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIAPIRequest struct {
	Model    string           `json:"model"`
	Store    bool             `json:"store"`
	Messages []RequestMessage `json:"messages"`
}

// GPTResponse represents the structure of the API response
type GPTResponse struct {
	Choices           []Choice `json:"choices"`
	Created           float64  `json:"created"`
	ID                string   `json:"id"`
	Model             string   `json:"model"`
	Object            string   `json:"object"`
	ServiceTier       string   `json:"service_tier"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Usage             Usage    `json:"usage"`
}

// Choice represents each choice returned in the response
type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Logprobs     *string `json:"logprobs"`
	Message      Message `json:"message"`
}

// Message represents the message content in the response
type Message struct {
	Content string  `json:"content"`
	Refusal *string `json:"refusal"`
	Role    string  `json:"role"`
}

// Usage represents token usage details
type Usage struct {
	CompletionTokens        int          `json:"completion_tokens"`
	CompletionTokensDetails TokenDetails `json:"completion_tokens_details"`
	PromptTokens            int          `json:"prompt_tokens"`
	PromptTokensDetails     TokenDetails `json:"prompt_tokens_details"`
	TotalTokens             int          `json:"total_tokens"`
}

// TokenDetails represents details of token usage
type TokenDetails struct {
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
	AudioTokens              int `json:"audio_tokens"`
	ReasoningTokens          int `json:"reasoning_tokens"`
	RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
	CachedTokens             int `json:"cached_tokens,omitempty"`
}

var chatMessages []RequestMessage = []RequestMessage{
	{
		Role:    "developer",
		Content: DEV_PROMPT,
	},
}

// Will send maximum of 10 messages with rolling window
const MaxConvLen = 10

func GetConverstaionMessages(newMessage RequestMessage) []RequestMessage {
	if len(chatMessages)-1 > MaxConvLen {
		chatMessages = append(chatMessages[:1], chatMessages[2:]...)
	}

	chatMessages = append(chatMessages, newMessage)

	return chatMessages
}
