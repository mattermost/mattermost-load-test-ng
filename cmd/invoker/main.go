package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type LambdaEvent struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	Debug    bool   `json:"debug"`
	Reload   bool   `json:"reload"`
	Delay    int    `json:"delay"`
	JoinTeam string `json:"join_team,omitempty"`
}

type LambdaResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

type ResponseBody struct {
	Message    string `json:"message"`
	Screenshot string `json:"screenshot,omitempty"`
	Error      string `json:"error,omitempty"`
}

func main() {
	// Define command line flags
	username := flag.String("username", "", "Username for login (with number suffix after '-')")
	password := flag.String("password", "", "Password for login")
	url := flag.String("url", "", "URL of the Mattermost instance")
	debug := flag.Bool("debug", false, "Enable debug mode (includes screenshot)")
	reload := flag.Bool("reload", false, "Enable reload after delay")
	delay := flag.Int("delay", 5000, "Delay in milliseconds after login")
	joinTeam := flag.String("join_team", "", "Team name to join")
	functionName := flag.String("function", "mattermost-login", "Name of the Lambda function")
	region := flag.String("region", "us-east-1", "AWS region")
	outputFile := flag.String("output", "", "File to save screenshot (if debug is enabled)")
	count := flag.Int("count", 1, "Number of concurrent Lambda invocations")

	flag.Parse()

	// Validate required parameters
	if *username == "" || *password == "" || *url == "" {
		fmt.Println("Error: username, password, and url are required")
		flag.Usage()
		return
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*region))
	if err != nil {
		fmt.Printf("Error loading AWS config: %v\n", err)
		return
	}

	// Create Lambda client
	client := lambda.NewFromConfig(cfg)

	// Create the event payload
	event := LambdaEvent{
		Username: *username,
		Password: *password,
		URL:      *url,
		Debug:    *debug,
		Reload:   *reload,
		Delay:    *delay,
		JoinTeam: *joinTeam,
	}

	if *count == 1 {
		invokeSync(client, *functionName, event, *outputFile)
		return
	}
	invokeConcurrent(client, *functionName, event, *count, *outputFile)
}

// parseUsername extracts the prefix and number from a username like "testuser-1"
func parseUsername(username string) (string, int, error) {
	parts := strings.Split(username, "-")
	if len(parts) < 2 {
		return "", 0, fmt.Errorf("username must be in format 'prefix-number'")
	}

	prefix := strings.Join(parts[:len(parts)-1], "-")
	var startNum int
	_, err := fmt.Sscanf(parts[len(parts)-1], "%d", &startNum)
	if err != nil {
		return "", 0, fmt.Errorf("invalid number suffix in username: %v", err)
	}

	return prefix, startNum, nil
}

// invokeSync performs a synchronous Lambda invocation
func invokeSync(client *lambda.Client, functionName string, event LambdaEvent, outputFile string) {
	fmt.Printf("Invoking Lambda function for %s...\n", event.Username)

	// Convert event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		fmt.Printf("Error marshaling event: %v\n", err)
		return
	}

	// Invoke Lambda function
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(event.Delay)*time.Millisecond+5*time.Minute)
	defer cancel()
	result, err := client.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(functionName),
		Payload:      payload,
	})
	if err != nil {
		fmt.Printf("Error invoking Lambda function: %v\n", err)
		return
	}

	// Check for Lambda execution errors
	if result.FunctionError != nil {
		fmt.Printf("Lambda execution error for %s: %s, %s\n", event.Username, *result.FunctionError, string(result.Payload))
		return
	}

	// Parse the Lambda response
	var response LambdaResponse
	if err := json.Unmarshal(result.Payload, &response); err != nil {
		fmt.Printf("Error parsing Lambda response: %v\n", err)
		return
	}

	// Parse the response body
	var responseBody ResponseBody
	if err := json.Unmarshal([]byte(response.Body), &responseBody); err != nil {
		fmt.Printf("Error parsing response body (%q): %v\n", response.Body, err)
		return
	}

	// Print the response
	if response.StatusCode == http.StatusOK {
		fmt.Printf("[%s] Message: %s\n", event.Username, responseBody.Message)
	} else {
		fmt.Printf("[%s] Error: %s\n", event.Username, responseBody.Error)
	}

	if outputFile != "" && responseBody.Screenshot != "" {
		fmt.Printf("Saving screenshot to %s\n", outputFile)
		// Decode base64 screenshot and save to file
		err := saveScreenshot(outputFile, responseBody.Screenshot)
		if err != nil {
			fmt.Printf("Error saving screenshot: %v\n", err)
		}
	}
}

// invokeConcurrent performs multiple concurrent Lambda invocations
func invokeConcurrent(client *lambda.Client, functionName string, baseEvent LambdaEvent, count int, outputFileTemplate string) {
	fmt.Printf("Invoking %d Lambda functions concurrently...\n", count)

	// Parse username to extract prefix and starting number
	prefix, startNum, err := parseUsername(baseEvent.Username)
	if err != nil {
		fmt.Printf("Error parsing username: %v\n", err)
		return
	}

	var wg sync.WaitGroup

	// Launch goroutines for each invocation
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int, event LambdaEvent) {
			defer wg.Done()

			// Add random sleep proportional to count
			// The higher the count, the more spread out the starts will be
			maxSleepMs := count * 200 // 200ms per count as a scaling factor
			sleepTime := time.Duration(rand.Intn(maxSleepMs)) * time.Millisecond
			time.Sleep(sleepTime)

			userNum := startNum + index
			username := fmt.Sprintf("%s-%d", prefix, userNum)

			event.Username = username

			// Generate unique filename for this invocation if needed
			var outputFile string
			if outputFileTemplate != "" {
				ext := filepath.Ext(outputFileTemplate)
				base := strings.TrimSuffix(outputFileTemplate, ext)
				outputFile = fmt.Sprintf("%s-%s%s", base, username, ext)
			}

			invokeSync(client, functionName, event, outputFile)
		}(i, baseEvent)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	fmt.Println("All Lambda invocations completed")
}

func saveScreenshot(filename, base64Image string) error {
	// Decode base64 string to bytes
	imageBytes, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return err
	}

	// Write bytes to file
	return os.WriteFile(filename, imageBytes, 0644)
}
