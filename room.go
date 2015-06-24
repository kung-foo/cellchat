package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/tideland/golib/cells"
	"github.com/tideland/golib/identifier"
)

type room struct {
	nullBehavior
	ctx           cells.Context
	name          string
	buildingName  string
	users         map[string]bool
	censoredWords []string
	censor        *censor
	censorID      string
}

func makeRoomID(name string, buildingName string) string {
	return identifier.Identifier("room", buildingName, name)
}

func newRoom(name string, buildingName string, censoredWords []string) *room {
	return &room{
		name:          name,
		buildingName:  buildingName,
		users:         make(map[string]bool),
		censoredWords: censoredWords,
	}
}

func (r *room) Init(ctx cells.Context) (err error) {
	r.ctx = ctx
	r.censorID = identifier.Identifier("room", r.buildingName, r.name, "censor")
	r.censor = newCensor(r.censorID, r.censoredWords)

	// TODO: hangs without goroutine
	go func() {
		roomID := identifier.Identifier("room", r.buildingName, r.name)
		env := r.ctx.Environment()
		env.StartCell(r.censorID, r.censor)
		subscribe(env, r.censorID, roomID)
	}()

	return
}

func (r *room) ProcessEvent(event cells.Event) (err error) {
	switch event.Topic() {
	case SAYS_ALL, SAYS_TO:
		user, _ := event.Payload().Get("from")
		switch user.(string) {
		case PublicAddressUserID, r.censorID:
			r.ctx.Emit(event)
		default:
			env := r.ctx.Environment()
			ok, err := env.Request(r.censorID, CENSOR, event.Payload(), nil, time.Second)

			if err != nil {
				log.Error(err)
				return err
			}

			if ok.(bool) {
				r.ctx.Emit(event)
			}
		}
	case IN_ROOM:
		userID, _ := event.Payload().Get("user")
		ok, _ := r.users[userID.(string)]
		event.Respond(ok)
	case USER_ADDED:
		userID, _ := event.Payload().Get("user")
		if ok, _ := r.users[userID.(string)]; !ok {
			r.users[userID.(string)] = true
			r.ctx.Environment().EmitNew(r.ctx.ID(), SAYS_ALL, cells.PayloadValues{
				"message": fmt.Sprintf("%s has entered the room", userID),
				"from":    PublicAddressUserID,
			}, nil)
		}
	case USER_COUNT:
		event.Respond(len(r.users))
	}
	return
}

func addUser(env cells.Environment, buildingName string, roomName string, userName string) string {
	roomID := identifier.Identifier("room", buildingName, roomName)
	userID := makeUserID(userName)

	u := newUser(userName)
	env.StartCell(userID, u)

	subscribe(env, roomID, userID)

	env.EmitNew(roomID, USER_ADDED, cells.PayloadValues{
		"user": userID,
	}, nil)

	return userID
}

func addLogUser(env cells.Environment, buildingName string, roomName string) {
	roomID := identifier.Identifier("room", buildingName, roomName)
	userID := makeUserID("logger")

	lu := &logUser{}
	lu.name = "logger"
	env.StartCell(userID, lu)

	subscribe(env, roomID, userID)

	env.EmitNew(roomID, "user-added", cells.PayloadValues{
		"user": userID,
	}, nil)
}
