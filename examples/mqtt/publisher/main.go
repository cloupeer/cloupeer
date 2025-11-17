package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/eclipse/paho.golang/paho/session/state"
)

// Config
const (
	broker    = "tcp://localhost:1883"
	clientID  = "publisher-001"
	keepAlive = 60

	topic = "test/topic/go"
	qos   = 1
)

func main() {
	// Session
	sessionState := state.NewInMemory()

	serverUrl, err := url.Parse(broker)
	if err != nil {
		log.Fatalf("解析 broker URL 失败: %v", err)
	}

	cfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverUrl},
		KeepAlive:                     keepAlive,
		CleanStartOnInitialConnection: false,
		SessionExpiryInterval:         60,
		ReconnectBackoff:              autopaho.NewConstantBackoff(5 * time.Second),
		OnConnectionUp:                ConnectionUp,
		OnConnectError:                ConnectError,
		ClientConfig: paho.ClientConfig{
			ClientID:           clientID,
			Session:            sessionState,
			OnClientError:      ClientError,
			OnServerDisconnect: ServerDisconnect,
		},
	}

	cfg.Debug = logger{prefix: "autoPaho"}
	cfg.PahoDebug = logger{prefix: "paho"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cm, err := autopaho.NewConnection(ctx, cfg)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var count uint64
		for {
			err = cm.AwaitConnection(ctx)
			if err != nil { // Should only happen when context is cancelled
				fmt.Printf("publisher done (AwaitConnection: %s)\n", err)
				return
			}

			count += 1
			// The message could be anything; lets make it JSON containing a simple count (makes it simpler to track the messages)
			msg, err := json.Marshal(struct {
				Count uint64
			}{Count: count})
			if err != nil {
				panic(err)
			}

			// Publish will block, so we run it in a goRoutine
			// Note that we could use PublishViaQueue if we wanted to trust that the library will deliver (and, ideally,
			// use a file-based queue and state).
			go func(msg []byte) {
				pr, err := cm.Publish(ctx, &paho.Publish{
					QoS:     qos,
					Topic:   topic,
					Payload: msg,
				})
				if err != nil {
					fmt.Printf("error publishing message %s: %s\n", msg, err)
				} else if pr.ReasonCode != 0 && pr.ReasonCode != 16 { // 16 = Server received message but there are no subscribers
					fmt.Printf("reason code %d received for message %s\n", pr.ReasonCode, msg)
				} else {
					fmt.Printf("sent message: %s\n", msg)
				}
			}(msg)

			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
				fmt.Println("publisher done")
				return
			}
		}
	}()

	// Wait for a signal before exiting
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	signal.Notify(sig, syscall.SIGTERM)

	<-sig
	fmt.Println("signal caught - exiting")
	cancel()

	wg.Wait()
	fmt.Println("shutdown complete")
}
