package assistant

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	openai "github.com/sashabaranov/go-openai"
)

var client *openai.Client

func Init(app *fiber.App) {

	app.Get("/ws/assistant", websocket.New(func(c *websocket.Conn) {

		var msg []byte
		var err error

		for {
			if _, msg, err = c.ReadMessage(); err != nil {
				log.Println("error read", err)
				break
			}

			log.Printf("recv: %s", msg)

			req := ClientReq{}
			err = json.Unmarshal(msg, &req)

			if err != nil {
				log.Println("error parsing :", err)
				continue
			}

			switch req.ReqType {
			case "assistantCreate":
				createAssistant(c, msg)
			case "upload":
				upload(c, msg)
			case "chat":
				chat(c, msg)
			}
		}
	}))
}

var assistantId string
var threadId string
var vecId string

func getClient() *openai.Client {

	if client == nil {
		client = openai.NewClient(os.Getenv("API_KEY"))
	}

	return client
}

func getThread() (string, error) {

	if threadId != "" {
		return threadId, nil
	}

	thr, err := getClient().CreateThread(context.Background(), openai.ThreadRequest{})

	if err != nil {
		return "", err
	}

	threadId = thr.ID
	return thr.ID, nil
}

func chat(c *websocket.Conn, msg []byte) {

	if assistantId == "" {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: "Assistant is not created"})
		return
	}

	req := ClientReq{}
	err := json.Unmarshal(msg, &req)

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	thrID, err := getThread()

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	_, err = getClient().CreateMessage(context.Background(), thrID, openai.MessageRequest{
		Role:    openai.ChatMessageRoleUser,
		Content: req.Payload,
	})

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	run, err := getClient().CreateRun(context.Background(), thrID, openai.RunRequest{
		AssistantID: assistantId,
		Model:       openai.GPT4oMini,
	})

	for {

		status, err := getClient().RetrieveRun(context.Background(), thrID, run.ID)

		if err != nil {
			c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
			return
		}

		switch status.Status {
		case "completed":
			// get latest thread message generated by run
			listMsg, err := getClient().ListMessage(context.Background(), thrID, nil, nil, nil, nil, &run.ID)

			if err != nil {
				c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
				return
			} else if len(listMsg.Messages) > 0 && len(listMsg.Messages[0].Content) > 0 {
				ret := listMsg.Messages[0].Content[0].Text.Value

				c.WriteJSON(ClientRes{
					ReqType: "chat",
					Payload: ret,
				})
				return
			} else {

				c.WriteJSON(ClientRes{
					ReqType: "error",
					Payload: "invalid response",
				})
				return
			}
		case "cancelled", "failed":

			c.WriteJSON(ClientRes{
				ReqType: "error",
				Payload: "failed",
			})
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func getVectorStore() (string, error) {

	if vecId != "" {
		return vecId, nil
	}

	store, err := getClient().CreateVectorStore(context.Background(), openai.VectorStoreRequest{
		Name: "assvectorstore",
	})

	if err != nil {
		return "", err
	}

	vecId = store.ID
	return store.ID, nil
}

func upload(c *websocket.Conn, msg []byte) {

	if assistantId == "" {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: "Assistant is not created"})
		return
	}

	req := ClientReq{}
	err := json.Unmarshal(msg, &req)

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	vecId, err := getVectorStore()

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	fl, err := getClient().CreateFileBytes(context.Background(), openai.FileBytesRequest{
		Name:    "data.txt",
		Bytes:   []byte(req.Payload),
		Purpose: openai.PurposeAssistants,
	})

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	vecFl, err := getClient().CreateVectorStoreFile(context.Background(), vecId, openai.VectorStoreFileRequest{
		FileID: fl.ID,
	})

	getClient().ModifyAssistant(context.Background(), assistantId, openai.AssistantRequest{
		ToolResources: &openai.AssistantToolResource{
			FileSearch: &openai.AssistantToolFileSearch{
				VectorStoreIDs: []string{vecId},
			},
		},
	})

	c.WriteJSON(ClientRes{
		ReqType: "uploadRes",
		Payload: vecFl.ID,
	})
}

func createAssistant(c *websocket.Conn, msg []byte) {

	assReq := AssistantReq{}
	err := json.Unmarshal(msg, &assReq)

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	ast, err := getClient().CreateAssistant(context.Background(), openai.AssistantRequest{
		Name:         &assReq.Name,
		Instructions: &assReq.Instruction,
		Model:        openai.GPT4oMini,
		Tools: []openai.AssistantTool{
			{
				Type: openai.AssistantToolTypeFileSearch,
			},
		},
	})

	if err != nil {
		c.WriteJSON(ClientRes{ReqType: "error", Payload: err.Error()})
		return
	}

	assistantId = ast.ID

	c.WriteJSON(ClientRes{
		ReqType: "assistantResponse",
		Payload: ast.ID,
	})
}
