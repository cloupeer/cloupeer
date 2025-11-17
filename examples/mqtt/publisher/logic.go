package main

import (
	"fmt"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

// 每次成功连接后会被调用：在这里做订阅
func ConnectionUp(cm *autopaho.ConnectionManager, c *paho.Connack) {
	fmt.Println("mqtt connection up")
}

// 每次连接失败后会被调用
func ConnectError(err error) {
	fmt.Printf("连接失败 (OnConnectError)：%v\n", err)
}

func ClientError(err error) {
	fmt.Printf("client error: %s\n", err)
}

func ServerDisconnect(d *paho.Disconnect) {
	if d.Properties != nil {
		fmt.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
	} else {
		fmt.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
	}
}
