package controllers

import (
	"context"
	"encoding/json"
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
	"github.com/labstack/echo"
)

type handlePromptRequest struct {
	StudentName string `json:"student_name" validate:"required"`
	Grade       string `json:"grade" validate:"required"`
}

type dostuffResponse struct {
	Response string `json:"response"`
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

	response := callClaude(req.StudentName, req.Grade)

	data := &dostuffResponse{
		Response: response,
	}

	return ctx.JSON(http.StatusOK, &helpers.APIResponse{
		Success: true,
		Data:    data,
	})
}

func callClaude(studentName string, grade string) string {
	var region = aws.String("eu-central-1")
	sdkConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(*region))
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return "Error"
	}

	client := bedrockruntime.NewFromConfig(sdkConfig)

	modelId := "anthropic.claude-v2:1"

	file1, err := os.ReadFile("guideline1.txt")
	if err != nil {
		log.Fatal(err)
	}

	file2, err := os.ReadFile("guideline2.txt")
	if err != nil {
		log.Fatal(err)
	}

	submission, err := os.ReadFile("submission.txt")
	if err != nil {
		log.Fatal(err)
	}

	// Old prompt
	// Anthropic Claude requires you to enclose the prompt as follows:
	// prompt := `You are an assessor, evaluating a student's submission.
	// 	The student received the final exam result: ` + grade + `.
	// 	The student's whole submission is also attached.
	// 	Attached are some guidelines for how to evaluate the student submission, or feedback from another assessor.

	prompt := `
		Here are some documents to use as reference for the following questions:

		<documents>
			<document index="1">
				<document_title>Guideline</document_title>
				<document_content>
				` + string(file1) + `
				</document_content>
			</document>
			<document index="2">
				<document_title>Guideline</document_title>
				<document_content>
				` + string(file2) + `
				</document_content>
			</document>
			<document index="3">
				<document_title>Student submission</document_title>
				<document_content>
				` + string(submission) + `
				</document_content>
			</document>
		</documents>

		The student's grade for this submission is: ` + grade + `
		Based on the document and the grade provided, please analyze the student's performance. Consider the following:
		1. What are the key strengths and weaknesses of the student's submission?
		2. How does the grade align with the submission content?
		3. What recommendations would you make for the student's future improvement?
		4. Please refer to the student by the name: ` + studentName + `

		As an assessor, provide a detailed explanation (in bullet form) for why that specific grade was given to the student, based on the attached documents.
		Please ensure your explanation covers both strengths and areas for improvement in the student's work.
	`

	wrappedPrompt := "Human: " + prompt + "\n\nAssistant:"

	request := ClaudeRequest{
		Prompt:            wrappedPrompt,
		MaxTokensToSample: 2000,
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
	fmt.Println("Anthropic claude responded.")
	return response.Completion
}
