package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/UNIwise/go-template/internal/rest/contexts"
	"github.com/UNIwise/go-template/internal/rest/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/labstack/echo"
)

const (
	awsRegion           = "eu-central-1"
	knowledgeBaseID     = "T4RVHJL8EL"
	modelARN            = "arn:aws:bedrock:eu-central-1::foundation-model/anthropic.claude-3-5-sonnet-20240620-v1:0"
	regularModelARN     = "anthropic.claude-3-5-sonnet-20240620-v1:0"
	knowledgeBasePrompt = `Please provide detailed guidelines for evaluating a student's submission for this specific exam,
	based on the assignment documents (where purpose is assignment), and guideline documents (where purpose is marking_guidance).
	Make sure to list the required knowledge and skill objectives.
	Start by listing the names of the available documents.`
)

type handlePromptRequest struct {
	StudentName string `json:"studentName" validate:"required"`
	Grade       string `json:"grade" validate:"required"`
	Language    string `json:"language" validate:"required"`
}

type dostuffResponse struct {
	Response string `json:"response"`
}

type ClaudeRequest struct {
	AnthropicVersion string     `json:"anthropic_version"`
	MaxTokens        int        `json:"max_tokens"`
	Temperature      float64    `json:"temperature,omitempty"`
	Messages         []Messages `json:"messages"`
}

type ClaudeReponse struct {
	Content []Content `json:"content"`
}

type Messages struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type   string  `json:"type"`
	Source *Source `json:"source,omitempty"`
	Text   string  `json:"text,omitempty"`
}

type Source struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type ImageAnalysis struct {
	Interest     float64  `json:"interest"`
	Confidence   float64  `json:"confidence"`
	Keywords     []string `json:"keywords"`
	Reason       string   `json:"reason"`
	Applications []string `json:"applications"`
}

type Delta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ClaudeResponse struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
	Delta Delta  `json:"delta"`
}

func (handlers *Handlers) dostuff(ctx contexts.AuthenticatedContext) error {
	req, err := helpers.Bind[handlePromptRequest](ctx)
	if err != nil {
		ctx.Log.WithError(err).Error("failed to bind createFlowRequest")

		return echo.ErrBadRequest
	}

	var region = aws.String(awsRegion)
	sdkConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(*region))
	if err != nil {
		fmt.Println("Couldn't load default configuration. Have you set up your AWS account?")
		fmt.Println(err)
		return err
	}

	knowledgeBaseResponse, err := callKnowledgeBase(context.Background(), sdkConfig)
	if err != nil {
		log.Fatal("failed to call knowledgebase", err)
		return err
	}

	knowledgeBaseString := *knowledgeBaseResponse.Output.Text

	streamResp, err := callClaude(sdkConfig, req.StudentName, req.Grade, req.Language, knowledgeBaseString)

	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, &helpers.APIResponse{
			Success: false,
			Error:   &helpers.APIError{Code: http.StatusInternalServerError, Message: "Failed to call Anthropic Claude"},
		})
	}

	w := ctx.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	stream := streamResp.GetStream()
	events := stream.Events()

	fmt.Println("Starting to stream events...")

	for {
		event := <-events
		if event != nil {
			if v, ok := event.(*bedrocktypes.ResponseStreamMemberChunk); ok {
				var response ClaudeResponse
				if err := json.Unmarshal(v.Value.Bytes, &response); err != nil {
					return err
				}

				fmt.Print(string(response.Delta.Text))
				if resp, err := json.Marshal(response); err != nil {
					return err
				} else {
					if _, err := fmt.Fprintf(w, "%s", resp); err != nil {
						return err
					}
				}

				w.Flush()

			} else if v, ok := event.(*bedrocktypes.UnknownUnionMember); ok {
				fmt.Print(v.Value)
			}
		} else {
			break
		}
	}

	return nil
}

func callClaude(sdkConfig aws.Config, studentName string, grade string, language string, knowledgeBaseString string) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
	fmt.Println("Calling Anthropic Claude...")

	client := bedrockruntime.NewFromConfig(sdkConfig)

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
				<document_title>Assessor guidelines</document_title>
				<document_content>
				` + knowledgeBaseString + `
				</document_content>
			<document index="2">
				<document_title>Student submission</document_title>
				<document_content>
				` + string(submission) + `
				</document_content>
			</document>
		</documents>

		The student's grade for this submission is: ` + grade + `
		Your response should be written in the language: ` + language + `.
		
		Based on the document and the grade provided, please analyze the student's performance. Consider the following:
		1. What are the key strengths and weaknesses of the Student submission file in relation to Guideline 1 and Guideline 2?
		2. Does the grade align with the Student submission file and if so, how?
		3. What recommendations would you make for the student's future improvement?
		4. Please refer to the student by the name: ` + studentName + `
		5. The justification should include both the Student submission file strengths and weaknesses
		6. Include line numbers and three word quotes from the Student submission file to support your analysis
		7. Add quote "<span style="color: red">This feedback was generated with AI assistance. Remember to check through the response!</span>" at the end of the feedback


		As an assessor, provide a detailed explanation (in bullet form) for why that specific grade was given to the student, based on the attached documents.
		Please ensure your explanation covers both strengths and areas for improvement in the student's work.
	`

	wrappedPrompt := "Human: " + prompt + "\n\nAssistant:"

	request := ClaudeRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        1000,
		Temperature:      0.2,
		Messages: []Messages{
			{
				Role: "user",
				Content: []Content{
					{
						Type: "text",
						Text: wrappedPrompt,
					},
				},
			},
		},
	}

	body, err := json.Marshal(request)
	if err != nil {
		log.Panicln("Couldn't marshal the request: ", err)
	}

	resp, err := client.InvokeModelWithResponseStream(context.Background(), &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(regularModelARN),
		ContentType: aws.String("application/json"),
		Body:        body,
	})

	if err != nil {
		errMsg := err.Error()
		fmt.Println("Error calling Claude: ", errMsg)
		os.Exit(1)
	}

	return resp, nil
}

func callKnowledgeBase(ctx context.Context, sdkConfig aws.Config) (*bedrockagentruntime.RetrieveAndGenerateOutput, error) {
	fmt.Println("Calling KnowledgeBase...")
	bedrockClient := bedrockagentruntime.NewFromConfig(sdkConfig)

	input := &bedrockagentruntime.RetrieveAndGenerateInput{
		Input: &types.RetrieveAndGenerateInput{
			Text: aws.String(knowledgeBasePrompt),
		},
		RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
			Type: types.RetrieveAndGenerateTypeKnowledgeBase,
			KnowledgeBaseConfiguration: &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
				KnowledgeBaseId: aws.String(knowledgeBaseID),
				ModelArn:        aws.String(modelARN),
			},
		},
	}

	return bedrockClient.RetrieveAndGenerate(ctx, input)
}
