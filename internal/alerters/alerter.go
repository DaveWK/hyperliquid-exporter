package alerters

import (
	"context"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/telegram"
	"log"
)

type AlerterConfig struct {
	AlertsEnabled   bool    `toml:"alerts_enabled"`
	TelegramAPIKey  string  `toml:"telegram_api_key"`
	TelegramChatIDs []int64 `toml:"telegram_rx_chat_ids"`
	TelegramEnabled bool    `toml:"telegram_enabled"`
}

func SendAlertMessage(config AlerterConfig, message string) {
	if config.AlertsEnabled != true {
		log.Println("alerters disabled")
		return
	}
	if config.TelegramEnabled {
		sendTelegramAlert(config.TelegramAPIKey, config.TelegramChatIDs, message)
	}
}
func sendTelegramAlert(apiToken string, chatIDs []int64, description string) {
	log.Printf("Sending telegram alert to chatIds: %d", chatIDs)
	telegramService, _ := telegram.New(apiToken)
	for _, chatID := range chatIDs {
		telegramService.AddReceivers(chatID)
	}
	notify.UseServices(telegramService)
	err := notify.Send(context.Background(), "Hyperliquid Monitor", description)
	if err != nil {
		log.Printf(
			"Error sending telegram notification: %s",
			err.Error(),
		)
	}
}
