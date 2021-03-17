package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
	"github.com/sirupsen/logrus"
)

var log = &logrus.Logger{
	Out:       os.Stderr,
	Formatter: new(logrus.TextFormatter),
	Hooks:     make(logrus.LevelHooks),
	Level:     logrus.InfoLevel,
}

var noCtx = context.Background()

// checkErr logs errors if not nil, along with a user-specified trace
func checkErr(err error, trace string) {
	if err != nil {
		log.WithFields(logrus.Fields{
			"trace": trace,
		}).Error(err)
	}
}

func commandSuccessful(s disgord.Session, channelID, messageID disgord.Snowflake) {
	ReactToMessageOrSay(s, channelID, messageID, "✅", "DONE BISCH!")
}

func commandFailed(s disgord.Session, channelID, messageID disgord.Snowflake) {
	ReactToMessageOrSay(s, channelID, messageID, "❌", "NEIN!")
}

func unknownCommand(s disgord.Session, channelID, messageID disgord.Snowflake) {
	ReactToMessageOrSay(s, channelID, messageID, "❓", "unknown command")
}

func ReactToMessageOrSay(s disgord.Session, channelID, messageID disgord.Snowflake, reaction, fallbackMessage string) {
	r := &disgord.Emoji{Name: reaction}
	err := s.Channel(channelID).Message(messageID).Reaction(r).Create()
	if err != nil {
		log.Errorf("failed to react to message{%s}: %v", messageID, err)
		_, _ = s.SendMsg(channelID, fallbackMessage)
		return
	}
}

type Commando struct {
	dict *ReadOnlyDictionary
}

func (c *Commando) RenameMember(s disgord.Session, e *disgord.GuildMemberAdd) {
	words := c.dict.Combination(2, 5)
	id := e.Member.UserID
	err := s.Guild(e.Member.GuildID).Member(id).UpdateBuilder().SetNick(strings.Join(words, " ")).Execute()
	if err != nil {
		log.Errorf("unable to update member nick: %s, to &v", e.Member.String(), words)
	}
}

func (c *Commando) List() (string, error) {
	words := c.dict.List()
	response := fmt.Sprintf("Dictionary consists of %d words:\n", len(words))
	response += "```markdown\n"
	for _, word := range words {
		response += " * " + word + "\n"
	}
	response += "```"
	return response, nil
}

func (c *Commando) Demultiplexer(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	words := strings.Split(msg.Content, " ")
	if len(words) == 0 {
		unknownCommand(s, msg.ChannelID, msg.ID)
		return
	}

	switch words[0] {
	case "add":
		if len(words) < 2 {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed due to too few words: %s", msg.String())
			return
		}
		word := words[1]
		if err := c.dict.Add(word); err != nil {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("unable to add word '%s' to dicitonary: %s, %v", word, msg.String(), err)
			return
		}
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "list":
		response, err := c.List()
		if err != nil {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed: %s: %v", msg.String(), err)
			return
		}
		_, _ = s.SendMsg(msg.ChannelID, response)
		commandSuccessful(s, msg.ChannelID, msg.ID)
	default:
		log.Warning("unknown command")
		unknownCommand(s, msg.ChannelID, msg.ID)
		return
	}
}

func main() {
	const prefix = "!"

	client := disgord.New(disgord.Config{
		ProjectName:  "discord-auto-nicknamer",
		BotToken:     os.Getenv("DISCORD_TOKEN"),
		Logger:       log,
		RejectEvents: disgord.AllEventsExcept(disgord.EvtMessageCreate),
		// ! Non-functional due to a current bug, will be fixed.
		Presence: &disgord.UpdateStatusPayload{
			Game: []*disgord.Activity{
				{Name: "write " + prefix + "ping"},
			},
		},
	})
	client.AddPermission(disgord.PermissionSendMessages | disgord.PermissionAddReactions | disgord.PermissionReadMessages)
	u, err := client.BotAuthorizeURL()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.String())

	dict := &ReadOnlyDictionary{
		Storage: &FileStorage{
			FileName: "words.json",
			DirPath:  "",
		},
	}
	dict.Load()
	commando := &Commando{
		dict: dict,
	}

	defer client.Gateway().StayConnectedUntilInterrupted()

	logFilter, _ := std.NewLogFilter(client)
	filter, _ := std.NewMsgFilter(context.Background(), client)
	filter.SetPrefix(prefix)

	client.Gateway().WithMiddleware(
		filter.NotByBot,    // ignore bot messages
		filter.HasPrefix,   // message must have the given prefix
		logFilter.LogMsg,   // log command message
		filter.StripPrefix, // remove the command prefix from the message
	).MessageCreate(commando.Demultiplexer)

	client.Gateway().GuildMemberAdd(commando.RenameMember)

	// create a handler and bind it to the bot init
	// dummy log print
	client.Gateway().BotReady(func() {
		log.Info("Bot is ready!")
	})
}
