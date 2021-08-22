package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/byronng/iaq_adapter_cee/handlers"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func connect(clientID string, uri *url.URL) mqtt.Client {
	opts := createClientOptions(clientID, uri)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	for !token.WaitTimeout(3 * time.Second) {
	}
	if err := token.Error(); err != nil {
		log.Fatal(err)
	}
	return client
}

func createClientOptions(clientID string, uri *url.URL) *mqtt.ClientOptions {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	password, _ := uri.User.Password()
	opts.SetPassword(password)
	opts.SetClientID(clientID)
	return opts
}

/*
func listen(uri *url.URL, topic string) {
	client := connect("sub", uri)
	client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		fmt.Printf("Topic:%s, Payload:%s\n", msg.Topic(), string(msg.Payload()))
		//handleData(msg.Payload())
	})
}
*/

func getDeviceAdjust(restServer string, devAdj map[string]handlers.IAQDevice, l *log.Logger) error {
	if len(restServer) > 0 {
		resp, err := http.Get(restServer + "/iaqdeviceadj")
		//resp, err := http.Get("https://air.woofaa.com/api/iaqdevice")
		if err != nil {
			l.Println("http client error, ", err)
			l.Printf("%s\n", restServer)
			return err
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		iaqDeviceMap := make(map[string][]handlers.IAQDevice)

		//l.Printf("%s\n", string(body))
		err = json.Unmarshal(body, &iaqDeviceMap)
		if err != nil {
			l.Println("GetDeviceAdjust error:, ", err)
			return err
		}

		for _, ID := range iaqDeviceMap["result"] {
			//_deviceAdjust[ID.Device] = ID
			devAdj[ID.Device] = ID
		}

		if len(devAdj) == 0 {
			l.Println("response Body:", string(body))
			return errors.New("GetDeviceAdjust response Body: " + string(body))
		}

		l.Println("Get devices: ", len(devAdj))
	}
	return nil
}

func main() {
	if len(os.Getenv("REST_URI")) > 0 {
		handlers.RESTURI = os.Getenv("REST_URI")
	} else {
		// for debug
		//handlers.RESTURI = `https://iaq.creaxtive.com/api`
		handlers.RESTURI = `http://localhost:4000`
	}

	if len(os.Getenv("CLOUD_URI")) > 0 {
		handlers.CLOUDURI = os.Getenv("CLOUD_URI")
	} else {
		// for debug
		handlers.CLOUDURI = ""
	}

	if len(os.Getenv("MQTT_URI")) > 0 {
		handlers.MQTTURI = os.Getenv("MQTT_URI")
	} else {
		// for debug
		handlers.MQTTURI = `mqtt://pi:woofaa@localhost:1883`
	}

	if len(os.Getenv("MQTT_TOPICS")) > 0 {
		handlers.TOPICS = os.Getenv("MQTT_TOPICS")
	} else {
		// for debug
		handlers.TOPICS = `GASDATA;UVSTATUS/CURRENTLIFETIME;UVSTATUS/SENDVALUE`
	}

	l := log.New(os.Stdout, "cee_adapter", log.LstdFlags)
	uri, err := url.Parse(handlers.MQTTURI)
	if err != nil {
		l.Printf("%+v\n", err)
		l.Fatal(err)
	}

	adj := make(map[string]handlers.IAQDevice)
	err = getDeviceAdjust(handlers.RESTURI, adj, l)
	if err != nil {
		l.Printf("Get Device Adjustment Error: %+v\n", err)
	}
	a := handlers.Init(l, adj)

	var topics = make(map[string]byte)
	entries := strings.Split(handlers.TOPICS, ";")
	for _, e := range entries {
		text := strings.TrimSpace(e)
		if len(text) > 0 {
			topics[text+"/#"] = 0
		}
	}

	l.Printf("Listen to topic(s) %v \n", topics)

	go a.ListenMultiple(uri, topics)

	// Sending test message every minute

	a.SendTestingData(uri)
}
