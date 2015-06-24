package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/tideland/golib/cells"
	"github.com/tideland/golib/identifier"
)

type user struct {
	nullBehavior
	name string
	ctx  cells.Context
}

func newUser(name string) *user {
	u := &user{name: name}
	return u
}

func makeUserID(name string) string {
	return identifier.Identifier("user", name)
}

func (u *user) Init(ctx cells.Context) (err error) {
	u.ctx = ctx
	return
}

func (u *user) ProcessEvent(event cells.Event) error {
	switch event.Topic() {
	case SAYS_TO:
		to, ok := event.Payload().Get("to")
		if !ok || to.(string) != makeUserID(u.name) {
			return nil
		}
		log.Info(event.Payload())
	case SAY:
		env := u.ctx.Environment()
		roomID, _ := event.Payload().Get("room")

		if !env.HasCell(roomID.(string)) {
			log.Panicf("%s not found", roomID)
			return fmt.Errorf("room %s not found", roomID)
		}

		ok, err := env.Request(roomID.(string), IN_ROOM, cells.PayloadValues{"user": makeUserID(u.name)}, nil, time.Second)

		if err != nil {
			log.Panic(err)
			return err
		}

		if !ok.(bool) {
			return fmt.Errorf("%s is not in %s", u.name, roomID)
		}

		payload := event.Payload()
		payload = payload.Apply(cells.PayloadValues{"from": makeUserID(u.name)})

		env.EmitNew(roomID.(string), SAYS_ALL, payload, nil)
	case USER_DISCONNECTED:
		log.Info(USER_DISCONNECTED)
	}
	return nil
}

type logUser struct {
	user
}

func (lu *logUser) ProcessEvent(event cells.Event) (err error) {
	// TODO: would be nice to have an event.Source()
	if event.Topic() == SAYS_ALL {
		user, _ := event.Payload().Get("from")
		msg, _ := event.Payload().Get("message")
		log.Infof("%s heard %v say \"%v\"", lu.name, user, msg)
	}
	return
}
