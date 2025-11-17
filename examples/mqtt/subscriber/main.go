package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/session/state"
)

// Config
const (
	broker    = "tcp://localhost:1883"
	clientID  = "subscriber-001"
	keepAlive = 60

	topic = "test/topic/go"
	qos   = 1
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverUrl, err := url.Parse(broker)
	if err != nil {
		log.Fatalf("解析 broker URL 失败: %v", err)
	}

	// Create a handler that will deal with incoming messages
	h := NewHandler()

	// Session
	sessionState := state.NewInMemory()

	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverUrl},
		KeepAlive:                     keepAlive,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		ReconnectBackoff:              autopaho.NewConstantBackoff(5 * time.Second),
		OnConnectionUp: func(cm *autopaho.ConnectionManager, c *paho.Connack) {
			fmt.Println("mqtt connection up")
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: topic, QoS: qos},
				},
			}); err != nil {
				fmt.Printf("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
				return
			}
			fmt.Println("mqtt subscription made")
		},
		OnConnectError: func(err error) { fmt.Printf("error whilst attempting connection: %s\n", err) },
		ClientConfig: paho.ClientConfig{
			ClientID: clientID,
			Session:  sessionState,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					h.handle(pr.Packet)
					return true, nil
				},
			},
			OnClientError: func(err error) { fmt.Printf("client error: %s\n", err) },
			OnServerDisconnect: func(d *paho.Disconnect) {

				if d.ReasonCode == 0x8E {
					fmt.Println("检测到会话被接管 (Session Taken Over)，将停止重连并退出。")
					cancel()
				}

				if d.Properties != nil {
					fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		},
	}

	cfg.Debug = logger{prefix: "autoPaho"}
	cfg.PahoDebug = logger{prefix: "paho"}

	// =======================
	// Connect to the server
	cm, err := autopaho.NewConnection(ctx, cfg)
	if err != nil {
		panic(err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig
	fmt.Println("signal caught - exiting")

	// We could cancel the context at this point but will call Disconnect instead (this waits for autopaho to shutdown)
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = cm.Disconnect(ctx)

	fmt.Println("shutdown complete")
}
