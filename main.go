package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/davecgh/go-spew/spew"
)

var SpewPrinter = spew.ConfigState{Indent: "    ", MaxDepth: 5}

const inputSubsFileName = "./142-jap.srt"

// var prompt string = `I'll provide you with a Japanese text to you in a JSON array. Your job is to convert the Japanese text to hiragana (with spaces) plus romaji plus its English translation.
// Do not include the original Japanese text, only the Hiragana, Romaji and the English translation.
// ALWAYS CONVERT THE ENTIRE TEXT. DO NOT WRITE ANYTHING OTHER THAN THE OUTPUT JSON ARRAY, THERE SHOULDN'T BE ANY COMMENTS AT ALL.
// DON'T GIVE ME MARKDOWN OR ANY OTHER FORMAT, I WANT A PLAIN JSON ARRAY IN TEXT FORMAT.
// Example - INPUT = [ "私" ], output = ["わたし\nwatashi\nI"].
// DO NOT DO ANYTHING OTHER THAN WHAT I'VE SPECIFIED. ALWAYS FOLLOW THE EXAMPLE I'VE GIVEN YOU. DO NOT GIVE ME A LIST OF DICTIONARIES ETC ETC.` + "DO NOT FORMAT THE TEXT AS ```json```"

var prompt string = `
I'll provide you with a Japanese text. Your job is to convert the Japanese text to hiragana (with spaces) plus romaji plus its English translation.
If the provided text is not Japanese, return it as is.
Do not include the original Japanese text, only the Hiragana, Romaji and the English translation.
ALWAYS CONVERT THE ENTIRE TEXT. DON'T GIVE ME MARKDOWN OR ANY OTHER FORMAT, I WANT THE ANSWER IN PLAIN TEXT FORMAT. 
Example - INPUT = "私", OUTPUT = "わたし\nwatashi\nI". 
DO NOT GIVE ME MARKDOWN.`

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

func sendOpenAIRequest(body OpenAIAPIRequest) (GPTResponse, error) {
	gptResponse := GPTResponse{}

	bodyBytes, err := json.Marshal(body)

	if err != nil {
		return gptResponse, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)

	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(bodyBytes),
	)

	if err != nil {
		return gptResponse, fmt.Errorf("Error creating request: %+v\n", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", os.Getenv("API_KEY")))

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return gptResponse, fmt.Errorf("Error making request: %+v\n", err)
	}

	fmt.Printf("StatusCode: %d\n", resp.StatusCode)

	// size := int64(math.Pow(2, 23))
	// respBody := make([]byte, size)

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return gptResponse, fmt.Errorf("Error reading response: %+v\n", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		err = json.Unmarshal(respBody, &gptResponse)

		if err != nil {
			SpewPrinter.Dump(respBody)
			return gptResponse, fmt.Errorf(
				"Error unmarshalling response into GPTResponse: %+v\n",
				err,
			)
		}

		// SpewPrinter.Dump(gptResponse)

		return gptResponse, nil
	}

	response := map[string]interface{}{}

	err = json.Unmarshal(respBody, &response)

	if err != nil {
		SpewPrinter.Dump(response)
		return gptResponse, fmt.Errorf("Error unmarshalling response to map: %+v\n", err)
	}

	SpewPrinter.Dump(response)

	return gptResponse, fmt.Errorf("Welp")
}

func getSubtitles(path string) *astisub.Subtitles {
	subtitles, err := astisub.OpenFile(path)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	return subtitles
}

func getJSONArray(subtitles *astisub.Subtitles, start, end int) string {
	s := "["

	for _, item := range subtitles.Items[start:end] {
		s += fmt.Sprintf(`"%s",`, item.String())
	}

	s += "]"

	return s
}

func writeSubsStringArrayToFile(subs []string, start, batch int) {
	SpewPrinter.Dump(subs)

	f, err := os.OpenFile(
		fmt.Sprintf("subsStrings-(%d)-start-%d-end-%d.json", time.Now(), start, batch),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0644,
	)

	if err != nil {
		fmt.Printf("Failed to open file. Err: %+v\n", err)
		return
	}

	defer f.Close()

	b, err := json.Marshal(subs)

	if err != nil {
		fmt.Printf("Failed to marshal to json. Err: %+v\n", err)
		return
	}

	_, err = f.Write(b)

	if err != nil {
		fmt.Printf("Failed to write to file. Err: %+v\n", err)
		return
	}
}

func fixJson(content string) string {
	start := strings.Index(content, "[")

	i := len(content) - 1
	end := -1

	for i >= 0 {
		if content[i] == ']' {
			end = i
		}

		i--
	}

	return content[start : end+1]
}

func createNewSubsFile(subs *astisub.Subtitles, newSubtitlesStringArray []string) {
	newSubs := make([]*astisub.Item, len(subs.Items))
	copy(newSubs, subs.Items)

	for i := range len(newSubs) {
		lines := []astisub.Line{}

		for _, line := range strings.Split(newSubtitlesStringArray[i], "\n") {
			lines = append(lines, astisub.Line{
				Items: []astisub.LineItem{
					{Text: line},
				},
			})
		}

		newSubs[i].Lines = lines
	}

	s := astisub.Subtitles{Items: newSubs}
	s.Write("./newthing.srt")
}

func handleArgs() {
	arg := os.Args[1]

	switch arg {
	case "write":
		// Need a subs json file and a subtitles file and writes a new file
		{
			if len(os.Args) != 4 {
				fmt.Printf("Usage: ./exe write <original subs file> <subs json file>\n")
				return
			}

			subs := getSubtitles(os.Args[2])

			bytes, err := os.ReadFile(os.Args[3])

			if err != nil {
				fmt.Printf("Failed to read file '%s', with err: %+v\n", os.Args[3], err)
				return
			}

			newSubtitlesStringArray := []string{}

			err = json.Unmarshal(bytes, &newSubtitlesStringArray)

			if err != nil {
				fmt.Printf(
					"Failed to unmarshal contents of file '%s', with err: %+v\n",
					os.Args[3],
					err,
				)
				return
			}

			createNewSubsFile(subs, newSubtitlesStringArray)
		}

	default:
		{
			fmt.Printf("Arg: '%s' not handled\n", arg)
			return
		}
	}
}

func main() {
	if len(os.Args) > 1 {
		handleArgs()
		return
	}

	isJson := false

	batchSize := 15
	currentBatch := 0

	subs := getSubtitles(inputSubsFileName)

	if subs == nil {
		return
	}

	newSubtitlesStringArray := []string{}

	errCount := 0
	retriesThreshold := 5

	for currentBatch < len(subs.Items) {
		start := currentBatch * batchSize
		end := currentBatch*batchSize + batchSize

		if start >= len(subs.Items) && isJson {
			break
		}

		fmt.Printf("CurrentBatch: %d / %d\n", currentBatch, len(subs.Items))

		text := ""

		if isJson {
			text = getJSONArray(
				subs,
				start,
				min(end, len(subs.Items)),
			)
		} else {
			text = subs.Items[currentBatch].String()
		}

		resp, err := sendOpenAIRequest(OpenAIAPIRequest{
			Model: "chatgpt-4o-latest",
			Store: true,
			Messages: []RequestMessage{
				{
					Role: "user",
					Content: fmt.Sprintf(
						"%s\n\n%s",
						prompt,
						text,
					),
				},
			},
		})

		if err != nil {
			errCount++

			SpewPrinter.Dump(resp)
			fmt.Println(err)

			if errCount >= retriesThreshold {
				writeSubsStringArrayToFile(newSubtitlesStringArray, start-batchSize, start-1)
				return
			}

			continue
		}

		errCount = 0

		content := resp.Choices[0].Message.Content

		if isJson {
			content = fixJson(content)
			newSubsString := []string{}
			err = json.Unmarshal([]byte(content), &newSubsString)

			if err != nil {
				writeSubsStringArrayToFile(newSubtitlesStringArray, start-batchSize, start-1)
				fmt.Printf(
					"Failed to unmarshal GPT response. Err: %+v, Batch: %d\n",
					err,
					currentBatch,
				)
				return
			}

			newSubtitlesStringArray = append(newSubtitlesStringArray, newSubsString...)
		} else {
			newSubtitlesStringArray = append(newSubtitlesStringArray, content)
		}

		fmt.Printf("'%s'\n", content)

		currentBatch++

		if !isJson {
			time.Sleep(200 * time.Millisecond)
		}
	}

	writeSubsStringArrayToFile(newSubtitlesStringArray, currentBatch*batchSize, len(subs.Items))

	createNewSubsFile(subs, newSubtitlesStringArray)
}
