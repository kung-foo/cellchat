package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/tideland/golib/cells"
	"github.com/tideland/golib/identifier"
	"github.com/tideland/golib/loop"
)

var (
	PublicAddressUserID = makeUserID("public address")
)

type building struct {
	nullBehavior
	ctx   cells.Context
	name  string
	rooms map[string]bool
}

func makeBuildingID(name string) string {
	return identifier.Identifier("building", name)
}

func (b *building) Init(ctx cells.Context) (err error) {
	b.ctx = ctx
	return
}

func (b *building) ProcessEvent(event cells.Event) (err error) {
	switch event.Topic() {
	case ROOM_ADDED:
		room, _ := event.Payload().Get("room")
		b.rooms[room.(string)] = true
		log.Infof("added %v", room)
	case LIST_ROOMS:
		rooms := []string{}
		for room := range b.rooms {
			rooms = append(rooms, room)
		}
		event.Respond(rooms)
	case SAYS_ALL:
		user, ok := event.Payload().Get("from")
		if !ok || user.(string) != PublicAddressUserID {
			return fmt.Errorf("invalid public address request")
		}
		b.ctx.Emit(event)
	}
	return
}

func addBuilding(env cells.Environment, buildingName string) string {
	b := &building{
		name:  buildingName,
		rooms: make(map[string]bool),
	}
	buildingID := makeBuildingID(buildingName)
	env.StartCell(buildingID, b)

	pa := &publicAddress{interval: time.Second * 10}
	paID := identifier.Identifier("building", buildingName, "pa")
	env.StartCell(paID, pa)

	subscribe(env, paID, buildingID)

	return buildingID
}

func addRoom(env cells.Environment, buildingName string, roomName string) {
	r := newRoom(roomName, buildingName, []string{"hell"})
	roomID := identifier.Identifier("room", buildingName, roomName)
	env.StartCell(roomID, r)

	paID := identifier.Identifier("building", buildingName, "pa")
	subscribe(env, paID, roomID)

	env.EmitNew(identifier.Identifier("building", buildingName), ROOM_ADDED, cells.PayloadValues{
		"room": roomID,
	}, nil)
}

type publicAddress struct {
	nullBehavior
	ctx      cells.Context
	interval time.Duration
	loop     loop.Loop
}

func (pa *publicAddress) Init(ctx cells.Context) error {
	pa.ctx = ctx
	pa.loop = loop.Go(pa.publishLoop)
	return nil
}

func (pa *publicAddress) Terminate() error {
	return pa.loop.Stop()
}

func (pa *publicAddress) publishLoop(l loop.Loop) error {
	for {
		select {
		case <-l.ShallStop():
			return nil
		case <-time.After(pa.interval):
			/*
				pa.ctx.EmitNew(SAYS_ALL, cells.PayloadValues{
					"message": fmt.Sprintf("The current time is %v", time.Now().Format(time.RFC850)),
					"from":    PublicAddressUserID,
				}, nil)
			*/
		}
	}
}
