package gobotslack

import (
	"fmt"
	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/eternnoir/gobot"
	"github.com/eternnoir/gobot/payload"
	"github.com/nlopes/slack"
)

const AdapterName string = "gobotslack"

func init() {
	gobot.RegisterAdapter(AdapterName, &SlackAdapter{})
}

type SlackAdapter struct {
	bot   *gobot.Gobot
	api   *slack.Client
	rtm   *slack.RTM
	token string
}

type SlackConfig struct {
	Token string
}

func (sa *SlackAdapter) Init(bot *gobot.Gobot) error {
	log.Infof("SlackAdapter init.")
	var conf SlackConfig
	if _, err := toml.DecodeFile(bot.ConfigPath+"/slack.toml", &conf); err != nil {
		return err
	}
	sa.token = conf.Token
	sa.bot = bot
	sa.api = slack.New(sa.token)
	return nil
}

func (sa *SlackAdapter) Start() {
	log.Info("SlackAdapter start.")
	sa.startRTM()
}

func (sa *SlackAdapter) Send(text string) error {
	log.Infof("Get new text to send.%s", text)
	return nil
}

func (sa *SlackAdapter) Reply(orimessage *payload.Message, text string) error {
	log.Infof("Get Replay message. Origin message is %s. Text %s", orimessage.Text, text)
	ev := orimessage.Payload.(*slack.MessageEvent)
	resMsg := sa.rtm.NewOutgoingMessage(text, ev.Channel)
	sa.rtm.SendMessage(resMsg)
	return nil
}

func (sa *SlackAdapter) startRTM() {
	rtm := sa.api.NewRTM()
	sa.rtm = rtm
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			log.Info("Event Received: ")
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
				// Ignore hello

			case *slack.ConnectedEvent:
				fmt.Println("Infos:", ev.Info)
				fmt.Println("Connection counter:", ev.ConnectionCount)
				// Replace #general with your Channel ID
				rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "#general"))

			case *slack.MessageEvent:
				log.Infof("[SlackAdapter] Get message %#v", ev)
				msg := &payload.Message{}
				msg.SourceAdapter = AdapterName
				msg.Text = ev.Msg.Text
				msg.Payload = ev
				sa.bot.Receive(msg)

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				fmt.Printf("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				log.Errorf("[SlackAdapter] Error: %s", ev.Error())

			case *slack.InvalidAuthEvent:
				log.Error("[SlackAdapter] InvalidAuthEvent Error")
				break Loop

			default:

				// Ignore other events..
				// fmt.Printf("Unexpected: %v\n", msg.Data)
			}
		}
	}

}
