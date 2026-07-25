package ocr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	aspose "github.com/aspose-pdf-foss/aspose-pdf-foss-for-go"
)

const defaultModel = "gpt-4o-mini"

// Client is a generic OpenAI/Vision OCR client.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	model      string
}

// NewClientFromEnv initializes the OCR client using environment variables.
func NewClientFromEnv() (*Client, error) {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return nil, ErrUnavailable
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("OCR_OPENAI_MODEL"))
	if model == "" {
		model = defaultModel
	}
	return &Client{
		httpClient: &http.Client{Timeout: 180 * time.Second},
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
	}, nil
}

// Extract sends an input document or image along with a generic prompt and schema to OpenAI Vision API,
// then decodes the resulting structured JSON into the provided result interface.
func (c *Client) Extract(ctx context.Context, reqData ExtractionRequest, result interface{}) error {
	if !SupportedMIME(reqData.Input.MimeType) {
		return ErrUnsupportedFile
	}
	if reqData.Input.MimeType == "application/pdf" {
		pages, err := pdfToImages(ctx, reqData.Input.Data)
		if err != nil {
			return fmt.Errorf("convert pdf: %w", err)
		}
		reqData.Input.Pages = pages
		reqData.Input.MimeType = "image/png"
	}

	content, err := c.callAPI(ctx, reqData)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal([]byte(content), result); err != nil {
			return fmt.Errorf("decode ocr json: %w", err)
		}
	}
	return nil
}

// ExtractText sends an input document or image along with a prompt to OpenAI Vision API,
// returning the raw string content (e.g., Markdown transcription or plain text) without enforcing JSON structure.
func (c *Client) ExtractText(ctx context.Context, reqData ExtractionRequest) (string, error) {
	if !SupportedMIME(reqData.Input.MimeType) {
		return "", ErrUnsupportedFile
	}
	if reqData.Input.MimeType == "application/pdf" {
		pages, err := pdfToImages(ctx, reqData.Input.Data)
		if err != nil {
			return "", fmt.Errorf("convert pdf: %w", err)
		}
		reqData.Input.Pages = pages
		reqData.Input.MimeType = "image/png"
	}

	return c.callAPIRaw(ctx, reqData)
}

func (c *Client) callAPI(ctx context.Context, reqData ExtractionRequest) (string, error) {
	content, err := c.callAPIRaw(ctx, reqData)
	if err != nil {
		return "", err
	}
	jsonContent, err := firstJSONObject(content)
	if err != nil {
		return "", err
	}
	return jsonContent, nil
}

func (c *Client) callAPIRaw(ctx context.Context, reqData ExtractionRequest) (string, error) {
	payload, err := json.Marshal(c.requestBody(reqData))
	if err != nil {
		return "", fmt.Errorf("marshal ocr request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create ocr request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send ocr request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		log.Printf("ocr api error: status=%d body=%s", resp.StatusCode, string(errBody))
		return "", ErrFailed
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ocr response: %w", err)
	}
	return completionContent(raw)
}

func (c *Client) requestBody(reqData ExtractionRequest) chatCompletionRequest {
	input := reqData.Input
	content := []chatContent{{Type: "text", Text: reqData.Prompt}}
	if len(input.Pages) > 0 {
		for _, pageData := range input.Pages {
			encoded := base64.StdEncoding.EncodeToString(pageData)
			content = append(content, chatContent{
				Type:     "image_url",
				ImageURL: &imageURL{URL: "data:image/png;base64," + encoded},
			})
		}
	} else {
		encoded := base64.StdEncoding.EncodeToString(input.Data)
		content = append(content, chatContent{
			Type:     "image_url",
			ImageURL: &imageURL{URL: "data:" + input.MimeType + ";base64," + encoded},
		})
	}

	req := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{{
			Role:    "user",
			Content: content,
		}},
	}
	if reqData.Schema != nil {
		req.ResponseFormat = &responseFormat{
			Type: "json_schema",
			JSONSchema: jsonSchema{
				Name:   reqData.SchemaName,
				Strict: true,
				Schema: reqData.Schema,
			},
		}
	}
	return req
}

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []chatContent `json:"content"`
}

type chatContent struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type responseFormat struct {
	Type       string     `json:"type"`
	JSONSchema jsonSchema `json:"json_schema"`
}

type jsonSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type chatCompletionChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func pdfToImages(ctx context.Context, pdfData []byte) ([][]byte, error) {
	doc, err := aspose.OpenStream(bytes.NewReader(pdfData))
	if err != nil {
		return nil, fmt.Errorf("open pdf: %w", err)
	}
	pageCount := doc.PageCount()
	if pageCount == 0 {
		return nil, fmt.Errorf("pdf has no pages")
	}
	var pages [][]byte
	for i := 1; i <= pageCount; i++ {
		page, err := doc.Page(i)
		if err != nil {
			return nil, fmt.Errorf("get pdf page %d: %w", i, err)
		}
		var buf bytes.Buffer
		if err := page.RenderPNG(&buf, aspose.RenderOptions{DPI: 200}); err != nil {
			return nil, fmt.Errorf("render pdf page %d: %w", i, err)
		}
		pages = append(pages, buf.Bytes())
	}
	return pages, nil
}
