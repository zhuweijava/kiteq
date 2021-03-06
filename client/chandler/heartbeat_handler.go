package chandler

import (
	. "kiteq/pipe"
	"kiteq/protocol"
	rcclient "kiteq/remoting/client"
	"log"
	"time"
)

type HeartbeatHandler struct {
	BaseForwardHandler
	clientMangager   *rcclient.ClientManager
	heartbeatPeriod  time.Duration
	heartbeatTimeout time.Duration
}

//------创建heartbeat
func NewHeartbeatHandler(name string, heartbeatPeriod time.Duration,
	heartbeatTimeout time.Duration, clientMangager *rcclient.ClientManager) *HeartbeatHandler {
	phandler := &HeartbeatHandler{}
	phandler.BaseForwardHandler = NewBaseForwardHandler(name, phandler)
	phandler.clientMangager = clientMangager
	phandler.heartbeatPeriod = heartbeatPeriod
	phandler.heartbeatTimeout = heartbeatTimeout
	go phandler.keepAlive()
	return phandler
}

func (self *HeartbeatHandler) keepAlive() {

	for {
		select {
		case <-time.After(self.heartbeatPeriod):
			//心跳检测
			func() {
				id := time.Now().Unix()
				clients := self.clientMangager.ClientsClone()
				packet := protocol.MarshalHeartbeatPacket(id)
				for h, c := range clients {
					i := 0
					//关闭的时候发起重连
					if c.IsClosed() {
						i = 3
					} else {
						for ; i < 3; i++ {
							hp := protocol.NewPacket(protocol.CMD_HEARTBEAT, packet)
							err := c.Ping(hp, time.Duration(int64(self.heartbeatTimeout)*int64(i+1)))
							//如果有错误则需要记录
							if nil != err {
								log.Printf("HeartbeatHandler|KeepAlive|FAIL|%s|%s|%d\n", err, h, id)
								continue
							} else {
								log.Printf("HeartbeatHandler|KeepAlive|SUCC|%s|%d|tryCount:%d\n", h, id, i)
								break
							}
						}

					}
					if i >= 3 {
						//说明连接有问题需要重连
						c.Shutdown()
						self.clientMangager.SubmitReconnect(c)
						log.Printf("HeartbeatHandler|SubmitReconnect|%s\n", c.RemoteAddr())
					}
				}
			}()
		}
	}

}

func (self *HeartbeatHandler) TypeAssert(event IEvent) bool {
	_, ok := self.cast(event)
	return ok
}

func (self *HeartbeatHandler) cast(event IEvent) (val *HeartbeatEvent, ok bool) {
	val, ok = event.(*HeartbeatEvent)
	return
}

func (self *HeartbeatHandler) Process(ctx *DefaultPipelineContext, event IEvent) error {

	hevent, ok := self.cast(event)
	if !ok {
		return ERROR_INVALID_EVENT_TYPE
	}

	// log.Printf("HeartbeatHandler|%s|Process|Recieve|Pong|%s|%d\n", self.GetName(), hevent.RemoteClient.RemoteAddr(), hevent.Version)
	hevent.RemoteClient.Attach(hevent.Opaque, hevent.Version)
	return nil
}
