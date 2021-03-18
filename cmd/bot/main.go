package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/andersfylling/nicknamer"
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
	dict *nicknamer.ReadOnlyDictionary
}

func (c *Commando) RenameMember(s disgord.Session, e *disgord.GuildMemberAdd) {
	nickName, err := c.dict.PopName()
	if errors.Is(err, nicknamer.ErrOutOfNames) {
		words := c.dict.RandWords(2, 5)
		nickName = strings.Join(words, " ")
		log.Infof("assigned random name '%s' to member: %s", nickName, e.Member.String())
	} else {
		log.Infof("assigned pre-defined name '%s' to member: %s", nickName, e.Member.String())
	}

	id := e.Member.UserID
	if err = s.Guild(e.Member.GuildID).Member(id).UpdateBuilder().SetNick(nickName).Execute(); err != nil {
		log.Errorf("unable to update member nick: %s, to %v. %v", e.Member.String(), nickName, err)
	}
}

func (c *Commando) ListWords() (string, error) {
	return c.ListTemplate(c.dict.ListWords(), "words")
}

func (c *Commando) ListNames() (string, error) {
	return c.ListTemplate(c.dict.ListNames(), "names")
}

func (c *Commando) ListTemplate(list []string, title string) (string, error) {
	response := fmt.Sprintf("Storage has %d %s:\n", len(list), title)
	response += "```markdown\n"
	for _, entry := range list {
		response += " * " + entry + "\n"
	}
	if len(list) == 0 {
		response += " # empty\n"
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
	case "add-word", "addword", "aw":
		if len(words) < 2 {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed due to too few words: %s", msg.String())
			return
		}
		word := words[1]
		if err := disgord.ValidateUsername(word); err != nil {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("illegal discord name '%s': %s, %v", word, msg.String(), err)
			return
		}
		if err := c.dict.AddWord(word); err != nil && !errors.Is(err, nicknamer.ErrAlreadyExists) {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("unable to add word '%s' to dicitonary: %s, %v", word, msg.String(), err)
			return
		}
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "add-name", "addname", "an":
		if len(words) < 2 {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed due to too few words: %s", msg.String())
			return
		}
		name := strings.Join(words[1:], " ")
		if strings.HasPrefix(name, `"`) && strings.HasSuffix(name, `"`) {
			name = strings.TrimPrefix(name, `"`)
			name = strings.TrimSuffix(name, `"`)
		}
		if err := disgord.ValidateUsername(name); err != nil {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("illegal discord name '%s': %s, %v", name, msg.String(), err)
			return
		}
		if err := c.dict.AddName(name); err != nil && !errors.Is(err, nicknamer.ErrAlreadyExists) {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("unable to add name '%s' to dicitonary: %s, %v", name, msg.String(), err)
			return
		}
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "list-words", "listwords", "lsw":
		response, err := c.ListWords()
		if err != nil {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed: %s: %v", msg.String(), err)
			return
		}
		_, _ = s.SendMsg(msg.ChannelID, response)
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "list-names", "listnames", "lsn":
		response, err := c.ListNames()
		if err != nil {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed: %s: %v", msg.String(), err)
			return
		}
		_, _ = s.SendMsg(msg.ChannelID, response)
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "remove-name", "removename", "rmn":
		if len(words) < 2 {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed due to too few words: %s", msg.String())
			return
		}
		_ = c.dict.RemoveName(strings.Join(words[1:], " "))
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "remove-word", "removeword", "rmw":
		if len(words) < 2 {
			commandFailed(s, msg.ChannelID, msg.ID)
			log.Errorf("command failed due to too few words: %s", msg.String())
			return
		}
		_ = c.dict.RemoveWord(words[1])
		commandSuccessful(s, msg.ChannelID, msg.ID)
	case "help", "h":
		response := "```markdown\n"
		response += " * list-words (listwords, lsw): list all words in dictionary used to generate random names\n"
		response += " * list-names (listnames, lsn): list all pre-defined names\n"
		response += " * add-word (addword, aw): add a new word to the dictionary\n"
		response += " * add-name (addname, an): add a new pre-defined name\n"
		response += " * remove-name (removename, rmn): remove a name from the pre-defined list\n"
		response += " * remove-word (removeword, rmw): remove a word from the dictionary\n"
		response += " * help (h): display this message\n"
		response += "```\n"
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
	
	nicknamer.Log = log

	client := disgord.New(disgord.Config{
		ProjectName:  "discord-auto-nicknamer",
		BotToken:     os.Getenv("DISCORD_TOKEN"),
		Logger:       log,
		RejectEvents: disgord.AllEventsExcept(disgord.EvtMessageCreate, disgord.EvtGuildMemberAdd),
		// ! Non-functional due to a current bug, will be fixed.
		Presence: &disgord.UpdateStatusPayload{
			Game: []*disgord.Activity{
				{Name: "write " + prefix + "help"},
			},
		},
	})
	client.AddPermission(disgord.PermissionSendMessages | disgord.PermissionAddReactions | disgord.PermissionReadMessages | disgord.PermissionManageNicknames)
	u, err := client.BotAuthorizeURL()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(u.String())

	dict := &nicknamer.ReadOnlyDictionary{
		Storage: &nicknamer.FileStorage{
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
