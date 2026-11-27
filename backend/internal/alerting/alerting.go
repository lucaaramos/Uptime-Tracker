package alerting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/lv99/uptime-tracker/internal/messaging"
	"github.com/lv99/uptime-tracker/internal/model"
	"github.com/lv99/uptime-tracker/internal/repository"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AlertWorker struct {
	mq       *messaging.RabbitMQ
	monitor  *repository.MonitorRepo
	smtpHost string
	smtpPort string
	smtpUser string
	smtpPass string
	fromAddr string
}

func NewAlertWorker(mq *messaging.RabbitMQ, monitor *repository.MonitorRepo, smtpHost, smtpPort, smtpUser, smtpPass, fromAddr string) *AlertWorker {
	return &AlertWorker{
		mq:       mq,
		monitor:  monitor,
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		smtpUser: smtpUser,
		smtpPass: smtpPass,
		fromAddr: fromAddr,
	}
}

func (a *AlertWorker) Start(ctx context.Context) error {
	msgs, err := a.mq.Consume(messaging.QueueAlerts)
	if err != nil {
		return fmt.Errorf("alert worker: failed to consume queue: %w", err)
	}

	log.Info().Msg("alert worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("alert worker stopped")
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("alert worker: deliveries channel closed")
			}

			var alert model.AlertMessage
			if err := json.Unmarshal(msg.Body, &alert); err != nil {
				log.Error().Err(err).Msg("alert worker: failed to unmarshal")
				msg.Nack(false, false)
				continue
			}

			a.processAlert(ctx, alert)
			msg.Ack(false)
		}
	}
}

func (a *AlertWorker) processAlert(ctx context.Context, alert model.AlertMessage) {
	alertType := alert.Type
	if alertType == "" {
		alertType = model.AlertTypeDown
	}

	log.Warn().
		Str("type", alertType).
		Str("monitor_id", alert.MonitorID).
		Str("name", alert.MonitorName).
		Int("fail_count", alert.FailCount).
		Msg("processing alert")

	monitorID, err := primitive.ObjectIDFromHex(alert.MonitorID)
	if err != nil {
		log.Error().Err(err).Msg("alert worker: invalid monitor id")
		return
	}

	monitor, err := a.monitor.GetByID(ctx, monitorID)
	if err != nil {
		log.Error().Err(err).Msg("alert worker: failed to get monitor")
		return
	}

	subject, body := formatAlert(monitor, alert)

	if monitor.AlertEmail != "" && a.smtpHost != "" {
		if err := a.sendEmail(monitor.AlertEmail, subject, body); err != nil {
			log.Error().Err(err).Str("email", monitor.AlertEmail).Msg("alert worker: failed to send email")
		} else {
			log.Info().Str("email", monitor.AlertEmail).Msg("alert worker: email sent")
		}
	}

	if monitor.AlertWebhook != "" {
		if err := a.sendWebhook(monitor.AlertWebhook, subject, body); err != nil {
			log.Error().Err(err).Str("webhook", monitor.AlertWebhook).Msg("alert worker: failed to send webhook")
		} else {
			log.Info().Str("webhook", monitor.AlertWebhook).Msg("alert worker: webhook sent")
		}
	}
}

func formatAlert(monitor *model.Monitor, alert model.AlertMessage) (string, string) {
	ts := time.UnixMilli(alert.Timestamp).UTC().Format(time.RFC3339)

	if alert.Type == model.AlertTypeRecovery {
		subject := fmt.Sprintf("[Uptime Tracker] %s is BACK UP", monitor.Name)
		body := fmt.Sprintf(
			"Monitor: %s\nURL: %s\nStatus: Operational\nDowntime: %s\nRecovered At: %s\n",
			monitor.Name, monitor.URL, formatDuration(alert.DurationSec), ts,
		)
		return subject, body
	}

	subject := fmt.Sprintf("[Uptime Tracker] %s is DOWN", monitor.Name)
	body := fmt.Sprintf(
		"Monitor: %s\nURL: %s\nFail Count: %d\nLast Error: %s\nTime: %s\n",
		monitor.Name, monitor.URL, alert.FailCount, alert.LastError, ts,
	)
	return subject, body
}

func formatDuration(sec int64) string {
	if sec < 60 {
		return fmt.Sprintf("%ds", sec)
	}
	if sec < 3600 {
		return fmt.Sprintf("%dm %ds", sec/60, sec%60)
	}
	hours := sec / 3600
	mins := (sec % 3600) / 60
	rem := sec % 60
	return fmt.Sprintf("%dh %dm %ds", hours, mins, rem)
}

func (a *AlertWorker) sendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", a.smtpHost, a.smtpPort)

	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", a.fromAddr, to, subject, body))

	auth := smtp.PlainAuth("", a.smtpUser, a.smtpPass, a.smtpHost)
	return smtp.SendMail(addr, auth, a.fromAddr, []string{to}, msg)
}

func (a *AlertWorker) sendWebhook(url, subject, body string) error {
	payload := map[string]string{
		"subject": subject,
		"body":    body,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
