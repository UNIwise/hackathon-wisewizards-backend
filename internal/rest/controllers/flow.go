package controllers

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/UNIwise/go-template/internal/rest/contexts"
	"github.com/UNIwise/go-template/internal/rest/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/awsdocs/aws-doc-sdk-examples/gov2/bedrock-runtime/scenarios"
	"github.com/awsdocs/aws-doc-sdk-examples/gov2/demotools"
	"github.com/labstack/echo"
)

type handlePromptRequest struct {
	Prompt string `json:"prompt" validate:"required"`
}

type dostuffResponse struct {
	ID string `json:"id"`
}

type ClaudeRequest struct {
	Prompt            string `json:"prompt"`
	MaxTokensToSample int    `json:"max_tokens_to_sample"`
	// Omitting optional request parameters
}

type ClaudeResponse struct {
	Completion string `json:"completion"`
}

func (handlers *Handlers) dostuff(ctx contexts.AuthenticatedContext) error {
	req, err := helpers.Bind[handlePromptRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to bind createFlowRequest")

		return echo.ErrBadRequest
	}

	// bedrock()
	callClaude(req.Prompt)

	ctx.Log.Info("Create Flow success")

	data := &dostuffResponse{
		// ID: flow.ID,
		ID: req.Prompt,
	}

	return ctx.JSON(http.StatusOK, &helpers.APIResponse{
		Success: true,
		Data:    data,
	})
}

func bedrock() {
	scenarioMap := map[string]func(sdkConfig aws.Config){
		"invokemodels": runInvokeModelsScenario,
	}
	choices := make([]string, len(scenarioMap))
	choiceIndex := 0
	for choice := range scenarioMap {
		choices[choiceIndex] = choice
		choiceIndex++
	}
	scenario := flag.String(
		"scenario", "",
		fmt.Sprintf("The scenario to run. Must be one of %v.", choices))

	// var region = flag.String("region", "us-east-1", "The AWS region")
	var region = aws.String("eu-central-1")
	flag.Parse()

	fmt.Printf("Using AWS region: %s\n", *region)

	if runScenario, ok := scenarioMap[*scenario]; !ok {
		fmt.Printf("'%v' is not a valid scenario.\n", *scenario)
		flag.Usage()
	} else {
		sdkConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(*region))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}

		log.SetFlags(0)
		runScenario(sdkConfig)
	}
}

func runInvokeModelsScenario(sdkConfig aws.Config) {
	scenario := scenarios.NewInvokeModelsScenario(sdkConfig, demotools.NewQuestioner())
	scenario.Run()
}

func callClaude(prompt string) {
	var region = aws.String("eu-central-1")
	sdkConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(*region))
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return
	}

	client := bedrockruntime.NewFromConfig(sdkConfig)

	modelId := "anthropic.claude-v2"

	// Anthropic Claude requires you to enclose the prompt as follows:
	prefix := "Human: "
	postfix := "\n\nAssistant:"
	wrappedPrompt := prefix + prompt + postfix

	request := ClaudeRequest{
		Prompt:            wrappedPrompt,
		MaxTokensToSample: 200,
	}

	body, err := json.Marshal(request)
	if err != nil {
		log.Panicln("Couldn't marshal the request: ", err)
	}

	result, err := client.InvokeModel(context.Background(), &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelId),
		ContentType: aws.String("application/json"),
		Body:        body,
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such host") {
			fmt.Printf("Error: The Bedrock service is not available in the selected region. Please double-check the service availability for your region at https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/.\n")
		} else if strings.Contains(errMsg, "Could not resolve the foundation model") {
			fmt.Printf("Error: Could not resolve the foundation model from model identifier: \"%v\". Please verify that the requested model exists and is accessible within the specified region.\n", modelId)
		} else {
			fmt.Printf("Error: Couldn't invoke Anthropic Claude. Here's why: %v\n", err)
		}
		os.Exit(1)
	}

	var response ClaudeResponse

	err = json.Unmarshal(result.Body, &response)

	if err != nil {
		log.Fatal("failed to unmarshal", err)
	}
	fmt.Println("Prompt:\n", prompt)
	fmt.Println("Response from Anthropic Claude:\n", response.Completion)
}
