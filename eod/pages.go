package eod

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const leftArrow = "⬅️"
const rightArrow = "➡️"

const defaultPageLength = 10
const playPageLength = 30

const ldbQuery = `
SELECT rw, ` + "user" + `, ` + "%s" + `
FROM (
    SELECT 
         ROW_NUMBER() OVER (ORDER BY ` + "%s" + ` DESC) AS rw,
         ` + "user" + `, ` + "%s" + `
    FROM eod_inv WHERE guild=?
) sub
WHERE sub.user=?
`

func (b *EoD) invPageGetter(p pageSwitcher) (string, int, int, error) {
	length := int(math.Floor(float64(len(p.Items)-1) / float64(p.PageLength)))
	if p.PageLength*p.Page > (len(p.Items) - 1) {
		return "", 0, length, nil
	}

	if p.Page < 0 {
		return "", length, length, nil
	}

	items := p.Items[p.PageLength*p.Page:]
	if len(items) > p.PageLength {
		items = items[:p.PageLength]
	}
	return strings.Join(items, "\n"), p.Page, length, nil
}

func (b *EoD) lbPageGetter(p pageSwitcher) (string, int, int, error) {
	cnt := b.db.QueryRow("SELECT COUNT(1) FROM eod_inv WHERE guild=?", p.Guild)
	pos := b.db.QueryRow(fmt.Sprintf(ldbQuery, p.Sort, p.Sort, p.Sort), p.Guild, p.User)
	var count int
	var ps int
	var u string
	var ucnt int
	err := pos.Scan(&ps, &u, &ucnt)
	if err != nil {
		return "", 0, 0, err
	}
	cnt.Scan(&count)
	length := int(math.Floor(float64(count-1) / float64(p.PageLength)))
	if err != nil {
		return "", 0, 0, err
	}
	if p.PageLength*p.Page > (count - 1) {
		return "", 0, length, nil
	}

	if p.Page < 0 {
		return "", length, length, nil
	}

	text := ""
	res, err := b.db.Query(fmt.Sprintf("SELECT %s, `user` FROM eod_inv WHERE guild=? ORDER BY %s DESC LIMIT ? OFFSET ?", p.Sort, p.Sort), p.Guild, p.PageLength, p.Page*p.PageLength)
	if err != nil {
		return "", 0, 0, err
	}
	defer res.Close()
	i := p.PageLength*p.Page + 1
	var user string
	var ct int
	for res.Next() {
		err = res.Scan(&ct, &user)
		if err != nil {
			return "", 0, 0, err
		}
		you := ""
		if user == u {
			you = " *You*"
		}
		text += fmt.Sprintf("%d. <@%s>%s - %d\n", i, user, you, ct)
		i++
	}
	if !((p.PageLength*p.Page <= ps) && (ps <= (p.Page+1)*p.PageLength)) {
		text += fmt.Sprintf("\n%d. <@%s> *You* - %d\n", ps, u, ucnt)
	}
	return text, p.Page, length, nil
}

func (b *EoD) newPageSwitcher(ps pageSwitcher, m msg, rsp rsp) {
	rsp.Acknowledge()

	lock.RLock()
	dat, exists := b.dat[m.GuildID]
	lock.RUnlock()
	if !exists {
		rsp.ErrorMessage("Guild isn't setup yet!")
		return
	}

	ps.Channel = m.ChannelID
	ps.Guild = m.GuildID
	ps.Page = 0
	ps.PageLength = defaultPageLength
	_, exists = dat.playChannels[m.ChannelID]
	if exists {
		ps.PageLength = playPageLength
	}

	cont, _, length, err := ps.PageGetter(ps)
	if rsp.Error(err) {
		return
	}
	id := rsp.Embed(&discordgo.MessageEmbed{
		Title:       ps.Title,
		Description: cont,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: ps.Thumbnail,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d", ps.Page+1, length+1),
		},
	})
	b.dg.MessageReactionAdd(m.ChannelID, id, leftArrow)
	b.dg.MessageReactionAdd(m.ChannelID, id, rightArrow)
	if dat.pageSwitchers == nil {
		dat.pageSwitchers = make(map[string]pageSwitcher)
	}
	dat.pageSwitchers[id] = ps

	lock.Lock()
	b.dat[m.GuildID] = dat
	lock.Unlock()
}

func (b *EoD) pageSwitchHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if r.UserID == b.dg.State.User.ID {
		return
	}

	lock.RLock()
	dat, exists := b.dat[r.GuildID]
	lock.RUnlock()
	if !exists {
		return
	}
	ps, exists := dat.pageSwitchers[r.MessageID]
	if !exists {
		return
	}

	if r.Emoji.Name == rightArrow {
		ps.Page++
	} else if r.Emoji.Name == leftArrow {
		ps.Page--
	} else {
		return
	}

	cont, page, length, err := ps.PageGetter(ps)
	if err != nil {
		return
	}
	if page != ps.Page {
		ps.Page = page
		cont, _, length, err = ps.PageGetter(ps)
		if err != nil {
			return
		}
	}

	color, _ := b.getColor(r.GuildID, r.UserID)
	b.dg.ChannelMessageEditEmbed(ps.Channel, r.MessageID, &discordgo.MessageEmbed{
		Title:       ps.Title,
		Description: cont,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: ps.Thumbnail,
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d", ps.Page+1, length+1),
		},
		Color: color,
	})
	b.dg.MessageReactionRemove(ps.Channel, r.MessageID, r.Emoji.Name, r.UserID)
	dat.pageSwitchers[r.MessageID] = ps

	lock.Lock()
	b.dat[r.GuildID] = dat
	lock.Unlock()
}

func (b *EoD) invCmd(user string, m msg, rsp rsp, sorter string) {
	lock.RLock()
	dat, exists := b.dat[m.GuildID]
	lock.RUnlock()
	if !exists {
		return
	}
	dat.lock.RLock()
	inv, exists := dat.invCache[user]
	dat.lock.RUnlock()
	if !exists {
		if user == m.Author.ID {
			rsp.ErrorMessage("You don't have an inventory!")
		} else {
			rsp.ErrorMessage(fmt.Sprintf("User <@%s> doesn't have an inventory!", user))
		}
		return
	}
	items := make([]string, len(inv))
	i := 0
	for k := range inv {
		dat.lock.RLock()
		items[i] = dat.elemCache[k].Name
		dat.lock.RUnlock()
		i++
	}

	switch sorter {
	case "id":
		sort.Slice(items, func(i, j int) bool {
			dat.lock.RLock()
			elem1, exists := dat.elemCache[strings.ToLower(items[i])]
			if !exists {
				return false
			}

			elem2, exists := dat.elemCache[strings.ToLower(items[j])]
			if !exists {
				return false
			}
			dat.lock.RUnlock()
			return elem1.CreatedOn.Before(elem2.CreatedOn)
		})

	case "madeby":
		count := 0
		outs := make([]string, len(items))
		for _, val := range items {
			creator := ""
			dat.lock.RLock()
			elem, exists := dat.elemCache[strings.ToLower(val)]
			dat.lock.RUnlock()
			if exists {
				creator = elem.Creator
			}
			if creator == user {
				outs[count] = val
				count++
			}
		}
		outs = outs[:count]
		sort.Strings(outs)
		items = outs

	default:
		sort.Strings(items)
	}

	name := m.Author.Username
	if m.Author.ID != user {
		u, err := b.dg.User(user)
		if rsp.Error(err) {
			return
		}
		name = u.Username
	}
	b.newPageSwitcher(pageSwitcher{
		Kind:       pageSwitchInv,
		Title:      fmt.Sprintf("%s's Inventory (%d, %s%%)", name, len(items), formatFloat(float32(len(items))/float32(len(dat.elemCache))*100, 2)),
		PageGetter: b.invPageGetter,
		Items:      items,
	}, m, rsp)
}

func (b *EoD) lbCmd(m msg, rsp rsp, sort string) {
	lock.RLock()
	dat, exists := b.dat[m.GuildID]
	lock.RUnlock()
	if !exists {
		return
	}
	dat.lock.RLock()
	_, exists = dat.invCache[m.Author.ID]
	dat.lock.RUnlock()
	if !exists {
		rsp.ErrorMessage("You don't have an inventory!")
		return
	}

	b.newPageSwitcher(pageSwitcher{
		Kind:       pageSwitchLdb,
		Title:      "Top Most Elements",
		PageGetter: b.lbPageGetter,
		Sort:       sort,
		User:       m.Author.ID,
	}, m, rsp)
}
