package channel

import (
	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	DiscordBot *discordgo.Session
	ChannelID  string
	Token      string
}

func NewDiscord(token, channelId string) Channel {
	return &Discord{
		ChannelID: channelId,
		Token:     token,
	}
}

func (channel *Discord) InitChannel() error {
	// Create a new Discord session using the provided bot token.
	var err error
	discordBot, err := discordgo.New("Bot " + channel.Token)
	if err != nil {
		return err
	}

	err = discordBot.Open()
	if err != nil {
		return err
	}

	channel.DiscordBot = discordBot

	return nil
}

func (channel *Discord) Send(msg string) error {
	discordBot := channel.DiscordBot
	discordChannelId := channel.ChannelID

	_, err := discordBot.ChannelMessageSend(discordChannelId, msg)
	if err != nil {
		return err
	}

	return nil
}

func (channel *Discord) Close() error {
	if channel.DiscordBot == nil {
		return nil
	}

	return channel.DiscordBot.Close()
}
