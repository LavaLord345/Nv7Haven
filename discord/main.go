package discord

import (
	"database/sql"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	dbUser     = "u57_fypTHIW9t8"
	dbPassword = "C7HgI6!GF0NaHCrdUi^tEMGy"
	dbName     = "s57_nv7haven"
	token      = "Nzg4MTg1MzY1NTMzNTU2NzM2.X9f00g.krA6cjfFWYdzbqOPXq8NvRjxb3k"
	clientID   = "788185365533556736"
)

var helpText string
var currHelp string

// Bot is a discord bot
type Bot struct {
	dg    *discordgo.Session
	db    *sql.DB
	props map[string]property

	memerefreshtime time.Time
	memedat         []meme
	memecache       map[string]map[int]empty
	cmemedat        []meme
	cmemecache      map[string]map[int]empty
	pmemedat        []meme
	pmemecache      map[string]map[int]empty

	mathvars map[string]map[string]interface{} // should be map[string]map[string]float64 but govaluate wants interface{}

	prefixcache map[string]string
}

func (b *Bot) handlers() {
	b.dg.AddHandler(b.giveNum)
	b.dg.AddHandler(b.help)
	b.dg.AddHandler(b.memes)
	b.dg.AddHandler(b.currencyBasics)
	b.dg.AddHandler(b.properties)
	b.dg.AddHandler(b.specials)
	b.dg.AddHandler(b.mod)
	b.dg.AddHandler(b.other)
	b.dg.AddHandler(b.memeGen)
	b.dg.AddHandler(b.math)
	for _, v := range commands {
		_, err := b.dg.ApplicationCommandCreate(clientID, "806258286043070545", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
	}
}

// InitDiscord creates a discord bot
func InitDiscord(db *sql.DB) Bot {
	// Init
	rand.Seed(time.Now().UnixNano())

	// Discord bot
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	// Help message
	data, err := ioutil.ReadFile("discord/help.txt")
	if err != nil {
		panic(err)
	}
	helpText = string(data)
	data, err = ioutil.ReadFile("discord/currency.txt")
	if err != nil {
		panic(err)
	}
	currHelp = string(data)

	// Init properties
	props := make(map[string]property, 0)
	for _, prop := range upgrades {
		props[prop.ID] = prop
	}

	// Set up bot
	b := Bot{
		dg:    dg,
		db:    db,
		props: props,

		mathvars: make(map[string]map[string]interface{}),

		prefixcache: make(map[string]string, 0),
	}
	b.handlers()
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	err = dg.Open()
	if err != nil {
		panic(err)
	}
	dg.UpdateGameStatus(0, "Run 7help to get help on this bot's commands!")
	return b
}

// Close cleans up
func (b *Bot) Close() {
	b.dg.Close()
}

func (b *Bot) help(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if b.startsWith(m, "7help currency") {
		s.ChannelMessageSend(m.ChannelID, currHelp)
		return
	}

	if b.startsWith(m, "7help") {
		s.ChannelMessageSend(m.ChannelID, helpText)
		return
	}
}
