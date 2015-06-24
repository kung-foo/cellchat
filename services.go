package main

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/tideland/golib/cells"
)

type censor struct {
	nullBehavior
	ctx      cells.Context
	words    map[string]bool
	warnings map[string]int
	id       string
}

func newCensor(id string, words []string) *censor {
	c := &censor{
		id:       id,
		words:    make(map[string]bool, len(words)),
		warnings: make(map[string]int),
	}

	for _, w := range words {
		c.words[w] = true
	}

	return c
}

func (c *censor) filter(event cells.Event) bool {
	message, _ := event.Payload().Get("message")
	user, _ := event.Payload().Get("from")
	for _, w := range strings.Split(message.(string), " ") {
		if _, ok := c.words[w]; ok {
			log.Infof("%s said a bad word: %s", user, w)
			c.warnings[user.(string)]++
			return false
		}
	}
	return true
}

func (c *censor) Init(ctx cells.Context) (err error) {
	c.ctx = ctx
	return
}

func (c *censor) ProcessEvent(event cells.Event) (err error) {
	switch event.Topic() {
	case CENSOR:
		ok := c.filter(event)
		userID, _ := event.Payload().Get("from")
		event.Respond(ok)

		if _, ok := c.warnings[userID.(string)]; !ok {
			c.warnings[userID.(string)] = 0
		}

		if !ok {
			return c.ctx.EmitNew(SAYS_TO, cells.PayloadValues{
				"message": fmt.Sprintf("You can't say that! You've been warned %d times.", c.warnings[userID.(string)]),
				"from":    c.id,
				"to":      userID.(string),
			}, nil)
		}
	}
	return
}
