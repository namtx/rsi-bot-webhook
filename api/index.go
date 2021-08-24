package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Message struct {
	Id              int             `json:"message_id"`
	Text            string          `json:"text"`
	MessageEntities []MessageEntity `json:"entities"`
	Chat            Chat            `json:"chat"`
}

type Chat struct {
	Id int64 `json:"id"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
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

type Indicator struct {
	Rsi float64 `json:"rsi"`
}

type IndicatorRequest struct {
	Type     string
	Symbol   string
	Interval string
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

	if len(update.Message.MessageEntities) == 0 || update.Message.MessageEntities[0].Type != "bot_command" {
		return
	}

	indicatorRequest, err := parseIndicatorRequest(update.Message)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	indicator, err := getIndicator(update.Message)

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	result, err := sendMessage(fmt.Sprintf("RSI %s %s %f", strings.ToUpper(indicatorRequest.Symbol), indicatorRequest.Interval, indicator.Rsi), update.Message.Chat.Id)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	log.Println(result.MessageId)
}

func getIndicator(message Message) (Indicator, error) {
	log.Println(message.Text)
	indicatorRequest, err := parseIndicatorRequest(message)
	if err != nil {
		return Indicator{}, err
	}

	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://polar-cliffs-67704.herokuapp.com/indicators?symbol=%s&interval=%s", indicatorRequest.Symbol, indicatorRequest.Interval),
		nil,
	)
	if err != nil {
		return Indicator{}, err
	}
	resp, err := client.Do(request)
	if err != nil {
		return Indicator{}, err
	}

	log.Println(resp.Body)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Indicator{}, err
	}

	var indicator Indicator
	err = json.Unmarshal(body, &indicator)
	if err != nil {
		log.Println(err.Error())
	}

	log.Println(indicator)

	return indicator, nil
}

func parseIndicatorRequest(message Message) (IndicatorRequest, error) {
	log.Println(message.Text)
	supportedIndicators := getSupportedIndicators()
	indicatorType := message.Text[1:message.MessageEntities[0].Length]
	log.Println(indicatorType)

	if indexOf(supportedIndicators, indicatorType) == -1 {
		return IndicatorRequest{}, errors.New("Unsupported indicator")
	}

	remaining := message.Text[message.MessageEntities[0].Length+1 : len(message.Text)]

	entities := strings.Split(remaining, " ")

	var indicatorRequest IndicatorRequest

	if len(entities) == 2 {
		indicatorRequest = IndicatorRequest{Type: indicatorType, Symbol: entities[0], Interval: entities[1]}
	} else {
		indicatorRequest = IndicatorRequest{Type: indicatorType, Symbol: entities[0]}
	}
	if indicatorRequest.Interval == "" {
		indicatorRequest.Interval = "1d"
	}

	if strings.Index(indicatorRequest.Symbol, "/") == -1 {
		indicatorRequest.Symbol = fmt.Sprintf("%s/%s", indicatorRequest.Symbol, "USDT")
	}

	return indicatorRequest, nil
}

func indexOf(slice []string, item string) int {
	for i := range slice {
		if slice[i] == item {
			return i
		}
	}

	return -1
}

func getSupportedIndicators() []string {
	return []string{"rsi"}
}

func sendMessage(text string, chatId int64) (SendMessageResult, error) {
	request, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/sendMessage?chat_id=%d&text=%s", botURL, chatId, text),
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
