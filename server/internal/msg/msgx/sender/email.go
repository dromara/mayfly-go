package sender

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"net/smtp"
	"strings"

	"github.com/may-fly/cast"
)

type EmailSender struct{}

func (e EmailSender) Send(channel *msgx.Channel, msg *msgx.Msg) error {
	return e.SendEmail(channel, msg)
}

func (e EmailSender) SendEmail(channel *msgx.Channel, msg *msgx.Msg) error {
	subject := msg.Title
	content, err := stringx.TemplateResolve(msg.Content, msg.Params)
	if err != nil {
		return err
	}

	to := collx.ArrayMapFilter(msg.Receivers, func(a msgx.Receiver) (string, bool) {
		return a.Email, a.Email != ""
	})

	if len(to) == 0 {
		return errors.New("no receiver")
	}

	systemName := "mayfly-go"

	serverAndPort := strings.Split(channel.URL, ":")
	smtpServer := serverAndPort[0]
	smtpPort := 465
	if len(serverAndPort) == 2 {
		smtpPort = cast.ToInt(serverAndPort[1])
	}

	smtpAccount := channel.GetExtraString("smtpAccount")
	smtpPassword := channel.GetExtraString("smtpPassword")

	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	mail := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s<%s>\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n",
		strings.Join(to, ";"), systemName, smtpAccount, encodedSubject, content))
	auth := smtp.PlainAuth("", smtpAccount, smtpPassword, smtpServer)
	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)

	if smtpPort == 465 {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         smtpServer,
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", smtpServer, smtpPort), tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, smtpServer)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(smtpAccount); err != nil {
			return err
		}
		for _, receiver := range to {
			if err = client.Rcpt(receiver); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(mail)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	} else {
		err = smtp.SendMail(addr, auth, smtpAccount, to, mail)
	}
	return err
}
