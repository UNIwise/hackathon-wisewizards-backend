package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/UNIwise/go-template/internal/rest/contexts"
	"github.com/UNIwise/go-template/internal/rest/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/labstack/echo"
)

const (
	awsRegion           = "eu-central-1"
	knowledgeBaseID     = "T4RVHJL8EL"
	modelARN            = "arn:aws:bedrock:eu-central-1::foundation-model/anthropic.claude-3-5-sonnet-20240620-v1:0"
	regularModelARN     = "anthropic.claude-v2:1"
	knowledgeBasePrompt = `Please provide detailed guidelines for evaluating a student's submission for this specific exam,
	based on the assignment documents (where purpose is assignment), and guideline documents (where purpose is marking_guidance).
	Make sure to list the required knowledge and skill objectives.
	Start by listing the names of the available documents.`
)

// Event represents Server-Sent Event.
// SSE explanation: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#event_stream_format
type Event struct {
	// ID is used to set the EventSource object's last event ID value.
	ID []byte
	// Data field is for the message. When the EventSource receives multiple consecutive lines
	// that begin with data:, it concatenates them, inserting a newline character between each one.
	// Trailing newlines are removed.
	Data []byte
	// Event is a string identifying the type of event described. If this is specified, an event
	// will be dispatched on the browser to the listener for the specified event name; the website
	// source code should use addEventListener() to listen for named events. The onmessage handler
	// is called if no event name is specified for a message.
	Event []byte
	// Retry is the reconnection time. If the connection to the server is lost, the browser will
	// wait for the specified time before attempting to reconnect. This must be an integer, specifying
	// the reconnection time in milliseconds. If a non-integer value is specified, the field is ignored.
	Retry []byte
	// Comment line can be used to prevent connections from timing out; a server can send a comment
	// periodically to keep the connection alive.
	Comment []byte
}

// MarshalTo marshals Event to given Writer
func (ev *Event) MarshalTo(w io.Writer) error {
	// Marshalling part is taken from: https://github.com/r3labs/sse/blob/c6d5381ee3ca63828b321c16baa008fd6c0b4564/http.go#L16
	if len(ev.Data) == 0 && len(ev.Comment) == 0 {
		return nil
	}

	if len(ev.Data) > 0 {

		if ev.ID != nil {
			if _, err := fmt.Fprintf(w, "id: %s\n", ev.ID); err != nil {
				return err
			}
		}

		sd := bytes.Split(ev.Data, []byte("\n"))
		for i := range sd {
			if _, err := fmt.Fprintf(w, "data: %s\n", sd[i]); err != nil {
				return err
			}
		}

		if len(ev.Event) > 0 {
			if _, err := fmt.Fprintf(w, "event: %s\n", ev.Event); err != nil {
				return err
			}
		}

		if len(ev.Retry) > 0 {
			if _, err := fmt.Fprintf(w, "retry: %s\n", ev.Retry); err != nil {
				return err
			}
		}
	}

	if len(ev.Comment) > 0 {
		if _, err := fmt.Fprintf(w, ": %s\n", ev.Comment); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}

	return nil
}

type handlePromptRequest struct {
	StudentName string `json:"studentName" validate:"required"`
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
	StopReason string `json:"stop_reason"`
	Stop       string `json:"stop"`
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
	fmt.Println("Knowledgebase response: ", knowledgeBaseString)

	streamResp, err := callClaude(sdkConfig, req.StudentName, req.Grade, knowledgeBaseString)

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

				fmt.Print(string(response.Completion))
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

func callClaude(sdkConfig aws.Config, studentName string, grade string, knowledgeBaseString string) (*bedrockruntime.InvokeModelWithResponseStreamOutput, error) {
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

	resp, err := client.InvokeModelWithResponseStream(context.Background(), &bedrockruntime.InvokeModelWithResponseStreamInput{
		ModelId:     aws.String(regularModelARN),
		ContentType: aws.String("application/json"),
		Body:        body,
	})

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such host") {
			fmt.Printf("Error: The Bedrock service is not available in the selected region. Please double-check the service availability for your region at https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/.\n")
		} else if strings.Contains(errMsg, "Could not resolve the foundation model") {
			fmt.Printf("Error: Could not resolve the foundation model from model identifier: \"%v\". Please verify that the requested model exists and is accessible within the specified region.\n", regularModelARN)
		} else {
			fmt.Printf("Error: Couldn't invoke Anthropic Claude. Here's why: %v\n", err)
		}
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
