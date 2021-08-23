package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Message struct {
	Id   int    `json:"message_id"`
	Text string `json:"text"`
}
type Update struct {
	UpdateId int     `json:"update_id"`
	Message  Message `json:"message"`
}

type SendMessageResponse struct {
	Ok     bool              `json:"ok"`
	Result SendMessageResult `json:"result"`
}

type SendMessageResult struct {
	MessageId int `json:"message_id"`
}

type PinnedMessage struct {
	Date int64  `json:"date"`
	Text string `json:"text"`
}

var (
	telegramBotToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	chatId           = os.Getenv("CHAT_ID")
	botURL           = fmt.Sprintf("https://api.telegram.org/bot%s", telegramBotToken)
	client           = http.Client{}
)

func Handler(w http.ResponseWriter, r *http.Request) {
	var update Update

	err := json.NewDecoder(r.Body).Decode(&update)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	log.Println(update.UpdateId)
	log.Println(update.Message)
	result, err := sendMessage(update.Message.Text)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	log.Println(result.MessageId)
}

func sendMessage(text string) (SendMessageResult, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/sendMessage?chat_id=@%s&text=%s", botURL, chatId, text),
		nil,
	)
	if err != nil {
		return SendMessageResult{}, err
	}

	resp, err := client.Do(request)
	if err != nil {
		return SendMessageResult{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SendMessageResult{}, err
	}

	sendMessageResponse := SendMessageResponse{}
	json.Unmarshal(body, &sendMessageResponse)

	return sendMessageResponse.Result, nil
}
