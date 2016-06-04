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
	bot            *gobot.Gobot
	api            *slack.Client
	rtm            *slack.RTM
	token          string
	defaultChannel string
	channelMap     map[string]slack.Channel
	userNameMap    map[string]slack.User
	userIdMap      map[string]slack.User
}

type SlackConfig struct {
	Token   string
	Channel string
}

func (sa *SlackAdapter) Init(bot *gobot.Gobot) error {
	log.Infof("SlackAdapter init.")
	var conf SlackConfig
	if _, err := toml.DecodeFile(bot.ConfigPath+"/slack.toml", &conf); err != nil {
		return err
	}
	log.Infof("SlackAdapter get config %#v", conf)
	sa.token = conf.Token
	sa.bot = bot
	sa.api = slack.New(sa.token)
	dc := "general"
	if conf.Channel != "" {
		dc = conf.Channel
	}
	sa.defaultChannel = dc
	log.Infof("SlackAdapter init done. %#v", sa)
	return nil
}

func (sa *SlackAdapter) Start() {
	log.Info("SlackAdapter start.")
	sa.initChannelMap()
	sa.initUserMap()
	sa.startRTM()
}

func (sa *SlackAdapter) initChannelMap() {
	chs, err := sa.api.GetChannels(false)
	if err != nil {
		log.Error(err)
		panic("SlackConfig load channels fail.")
	}
	sa.channelMap = map[string]slack.Channel{}
	for _, ch := range chs {
		log.Infof("[SlackAdapter] load channel %s Id: %s", ch.Name, ch.ID)
		sa.channelMap[ch.Name] = ch
	}
}

func (sa *SlackAdapter) initUserMap() {
	users, err := sa.api.GetUsers()
	if err != nil {
		log.Error(err)
		panic("SlackConfig load users fail.")
	}
	log.Infof("[SlackAdapter] load users %#v", users)
	sa.userNameMap = map[string]slack.User{}
	sa.userIdMap = map[string]slack.User{}
	for _, user := range users {
		sa.userNameMap[user.Name] = user
		sa.userIdMap[user.ID] = user
	}
}

func (sa *SlackAdapter) Send(text string) error {
	log.Infof("[SlackAdapter] Get new text to send.%s", text, sa.defaultChannel)
	if ch, ok := sa.channelMap[sa.defaultChannel]; ok {
		log.Infof("[SlackAdapter] Send new text %s. To %s", text, ch.Name)
		_, _, err := sa.api.PostMessage("#"+ch.Name, text, slack.PostMessageParameters{AsUser: true})
		if err != nil {
			log.Error(err)
		}
	} else {
		log.Errorf("[SlackAdapter] Channel name %s not found.", sa.defaultChannel)
	}
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
