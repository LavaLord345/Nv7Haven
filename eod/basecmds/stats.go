package basecmds

import (
	"fmt"
	"time"

	"github.com/Nv7-Github/Nv7Haven/eod/types"
	"github.com/Nv7-Github/Nv7Haven/eod/util"
	"github.com/bwmarrin/discordgo"
)

func (b *BaseCmds) StatsCmd(m types.Msg, rsp types.Rsp) {
	rsp.Acknowledge()

	db, res := b.GetDB(m.GuildID)
	if !res.Exists {
		rsp.ErrorMessage(res.Message)
		return
	}
	gd, err := b.dg.State.Guild(m.GuildID)
	if rsp.Error(err) {
		return
	}

	found := 0
	db.RLock()
	for _, val := range db.Invs() {
		found += len(val.Elements)
	}

	categorized := 0
	for _, val := range db.Cats() {
		categorized += len(val.Elements)
	}

	rsp.Message(fmt.Sprintf(db.Config.LangProperty("Stats"), util.FormatInt(len(db.Elements)), util.FormatInt(db.ComboCnt()), util.FormatInt(gd.MemberCount), util.FormatInt(found), util.FormatInt(categorized)), discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label: db.Config.LangProperty("ViewMoreStats"),
				URL:   "https://nv7haven.com/?page=eod",
				Style: discordgo.LinkButton,
			},
		},
	})
	db.RUnlock()
}

func (b *BaseCmds) SaveStats() {
	var lastTime int64
	err := b.db.QueryRow("SELECT time FROM eod_stats ORDER BY time DESC LIMIT 1").Scan(&lastTime)
	if err != nil {
		fmt.Println(err)
	}

	if time.Since(time.Unix(lastTime, 0)).Hours() > 24 {
		b.Data.RLock()

		categorized := 0
		found := 0
		elemCnt := 0
		comboCnt := 0
		users := make(map[string]types.Empty)

		for _, dat := range b.Data.DB {
			dat.RLock()
			for _, val := range dat.Cats() {
				categorized += len(val.Elements)
			}
			for _, val := range dat.Invs() {
				found += len(val.Elements)
				users[val.User] = types.Empty{}
			}
			elemCnt += len(dat.Elements)
			comboCnt += dat.ComboCnt()
			dat.RUnlock()
		}
		b.Data.RUnlock()

		_, err = b.db.Exec("INSERT INTO eod_stats VALUES (?, ?, ?, ?, ?, ?, ?)", time.Now().Unix(), elemCnt, comboCnt, len(users), found, categorized, len(b.Data.DB))
		if err != nil {
			fmt.Println(err)
		}
	}
}
