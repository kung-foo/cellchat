package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/tideland/golib/cells"
	golib_logger "github.com/tideland/golib/logger"
)

type nullBehavior struct{}

func (n *nullBehavior) Init(ctx cells.Context) (err error)         { return }
func (n *nullBehavior) ProcessEvent(event cells.Event) (err error) { return }
func (n *nullBehavior) Recover(r interface{}) (err error)          { return }
func (n *nullBehavior) Terminate() (err error)                     { return }

var rootEnvironment cells.Environment

func subscribe(env cells.Environment, src string, dst string) (err error) {
	log.Infof("subscribing %s to %s", dst, src)
	err = env.Subscribe(src, dst)
	if err != nil {
		log.Error(err)
	}
	return
}

func payloadToValues(payload cells.Payload) cells.PayloadValues {
	output := cells.PayloadValues{}

	payload.Do(func(key string, value interface{}) error {
		output[key] = value
		return nil
	})

	return output
}

func main() {
	golib_logger.SetLevel(golib_logger.LevelInfo)
	log.Print("start")

	rootEnvironment = cells.NewEnvironment()
	defer rootEnvironment.Stop()

	addBuilding(rootEnvironment, "school")
	addRoom(rootEnvironment, "school", "cafeteria")

	startServer(rootEnvironment)
	/*
		addRoom(rootEnvironment, "school", "playground")

		time.Sleep(time.Millisecond * 100)

		addLogUser(rootEnvironment, "school", "cafeteria")

		addUser(rootEnvironment, "school", "cafeteria", "willy")
		addUser(rootEnvironment, "school", "cafeteria", "bart")
		addUser(rootEnvironment, "school", "cafeteria", "lisa")

		time.Sleep(time.Millisecond * 100)

		bart := makeUserID("bart")

		rootEnvironment.EmitNew(bart, SAY, cells.PayloadValues{
			"room":    makeRoomID("cafeteria", "school"),
			"message": "Don't have a cow, Man!",
		}, nil)

		time.Sleep(time.Millisecond * 100)

		rootEnvironment.EmitNew(bart, SAY, cells.PayloadValues{
			"room":    makeRoomID("cafeteria", "school"),
			"message": "I'm Bart Simpson, who the hell are you?",
		}, nil)

		time.Sleep(time.Millisecond * 100)

		rootEnvironment.EmitNew(bart, SAY, cells.PayloadValues{
			"room":    makeRoomID("cafeteria", "school"),
			"message": "what teh hell ?",
		}, nil)

		time.Sleep(time.Second * 5)
	*/
}
