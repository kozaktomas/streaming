package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/k0kubun/go-ansi"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

//go:embed .env
var secrets []byte

//go:embed prompts/*.prompt
var promptFolder embed.FS

var rootCmd = &cobra.Command{
	Use:   "streaming",
	Short: "Streaming helper. Check it out at https://www.twitch.tv/worldofyaml",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the stream.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := personalPageHook(true); err != nil {
			return err
		}
		return run("Stream is starting...", "starting_seq.prompt", 60*10)
	},
}

var coffeeCmd = &cobra.Command{
	Use:   "kafe [seconds]",
	Short: "Small break - coffee preparation.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		seconds, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid number of seconds: %s", args[0])
		}

		return run("Coffee prep...", "coffee_seq.prompt", seconds)
	},
}

var breakCmd = &cobra.Command{
	Use:   "break [seconds]",
	Short: "Small break during the stream.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		seconds, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid number of seconds: %s", args[0])
		}

		return run("Small break...", "break_seq.prompt", seconds)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the stream.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := personalPageHook(false); err != nil {
			return err
		}
		return run("Stream is ending...", "ending_seq.prompt", 60*5)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(coffeeCmd)
	rootCmd.AddCommand(breakCmd)
	rootCmd.AddCommand(stopCmd)
}

func main() {
	loadEnvironment()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func run(title, promptFile string, seconds int) error {
	ctx := context.Background()
	openAiKey := os.Getenv("OPENAI_API_KEY")

	client := openai.NewClient(
		option.WithAPIKey(openAiKey),
	)

	content, err := promptFolder.ReadFile(fmt.Sprintf("prompts/%s", promptFile))
	if err != nil {
		return err
	}

	question := strings.ReplaceAll(string(content), "%COUNT%", fmt.Sprintf("%d", seconds/6))

	messages := openai.F([]openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(question),
	})

	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.F(openai.ChatModelGPT4oMini),
	}

	running := true
	iter := 0
	var items []string
	for running && iter < 5 {
		iter++
		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			panic(err)
		}

		resp := completion.Choices[0].Message.Content
		params.Messages.Value = append(params.Messages.Value, openai.AssistantMessage(resp))

		items, err = parseJsonFromResponse(resp)
		if err != nil {
			params.Messages.Value = append(params.Messages.Value, openai.UserMessage(err.Error()))
			continue
		}

		if len(items) < seconds/6 {
			msg := fmt.Sprintf("Not enough items. I need at least %d items, but got %d", seconds/6, len(items))
			params.Messages.Value = append(params.Messages.Value, openai.UserMessage(msg))
			continue
		}

		running = false
	}

	n := seconds
	f := n / len(items)
	p := 0

	bar := progressbar.NewOptions(n,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()), //you should install "github.com/k0kubun/go-ansi"
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetSpinnerChangeInterval(0),
		progressbar.OptionSetWidth(20),
		progressbar.OptionSetDescription("Booting..."),
	)

	fmt.Printf("\n   %s\n", title)
	fmt.Println()

	for i := 0; i < n; i++ {
		bar.Add(1)

		if i%f == 0 {
			bar.Describe(items[p] + "   ")
			p++
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Print("Done!")

	return nil
}

var reJson = regexp.MustCompile(`(\[[^]]+])`)

func parseJsonFromResponse(resp string) ([]string, error) {
	matches := reJson.FindAllString(resp, -1)
	if len(matches) != 1 {
		return nil, fmt.Errorf("could not find json array in response")
	}

	var result []string
	err := json.Unmarshal([]byte(matches[0]), &result)
	if err != nil {
		return nil, fmt.Errorf("could not parse json: %s", err)
	}

	return result, nil
}

// sends hook to my personal page
func personalPageHook(live bool) error {
	const URL = "https://kozak.in/api/live"

	data := struct {
		Live bool `json:"live"`
	}{
		Live: live,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal json: %s", err)
	}

	request, err := http.NewRequest("POST", URL, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("could not create request: %s", err)
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", "Bearer "+os.Getenv("PERSONAL_PAGE_API_KEY"))

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("could not send request: %s", err)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	return nil
}

func loadEnvironment() {
	envMap, err := godotenv.Parse(bytes.NewReader(secrets))
	if err != nil {
		fmt.Println("could not load .env file")
		panic(err)
	}

	for key, value := range envMap {
		_ = os.Setenv(key, value)
	}
}
