package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const RawInfoSize = 9
const RawReadingSize = 13

var (
	RESTURI  string
	CLOUDURI string
	MQTTURI  string
	TOPICS   string
)

var GSDVal struct {
	gasValue string
	gasType  string
}

// IAQDevice structure
type IAQDevice struct {
	ID int `json:"id"`
	//IDRegion int    `json:"id_region"`
	Device string `json:"device"`
	//Name        string `json:"device_name"`
	//ImageFile   string `json:"device_imagefile"`

	H2SAdjust string `json:"h2s_adjust"`
	AQIAdjust string `json:"aqi_adjust"`

	SO2Adjust  string `json:"so2_adjust"`
	NO2Adjust  string `json:"no2_adjust"`
	O3Adjust   string `json:"o3_adjust"`
	HCHOAdjust string `json:"hcho_adjust"`

	PM2p5Adjust string `json:"pm2p5_adjust"`
	COAdjust    string `json:"co_adjust"`
	CO2Adjust   string `json:"co2_adjust"`
	C2H2Adjust  string `json:"c2h2_adjust"`
	HumAdjust   string `json:"hum_adjust"`
	TempAdjust  string `json:"temp_adjust"`
	TVOCAdjust  string `json:"tvoc_adjust"`
	PM10Adjust  string `json:"pm10_adjust"`
	PM100Adjust string `json:"pm100_adjust"`

	DateCreated string `json:"date_created"`
	DateUpdated string `json:"date_updated"`
}

type CEEPayLoad struct {
	MAC string `json:"mac"`

	//H2S json.Number `json:"h2s"`
	//AQI json.Number `json:"aqi"`

	SO2  json.Number `json:"so2"`
	NO2  json.Number `json:"no2"`
	O3   json.Number `json:"o3"`
	HCHO json.Number `json:"hcho"`

	CO    json.Number `json:"co"`
	TEMP  json.Number `json:"temp"`
	HUM   json.Number `json:"hum"`
	PM2P5 json.Number `json:"pm2p5"`
	PM10  json.Number `json:"pm10"`
	TVOC  json.Number `json:"tvoc"`
	CO2   json.Number `json:"co2"`
	PM100 json.Number `json:"pm100"`
	C2H2  json.Number `json:"c2h2"`
}

/*
type CEEPayLoad struct {
	Sensors []struct {
		RawInfo    []int64 `json:"rawInfo"`
		RawReading []int64 `json:"rawReading"`
		SensorType int64   `json:"sensorType"`
	} `json:"sensors"`
	MAC string `json:"mac"`
}
*/

type Adapter struct {
	l   *log.Logger
	adj map[string]IAQDevice
}

func Init(l *log.Logger, adj map[string]IAQDevice) *Adapter {
	return &Adapter{l, adj}
}

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

func (a Adapter) GetDeviceAdjust(restServer string, devAdj map[string]IAQDevice) error {
	if len(restServer) > 0 {
		resp, err := http.Get(restServer + "/iaqdevice")
		if err != nil {
			a.l.Println("http client error, ", err)
			a.l.Printf("%s\n", restServer)
			return err
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		iaqDeviceMap := make(map[string][]IAQDevice)
		err = json.Unmarshal(body, &iaqDeviceMap)
		if err != nil {
			a.l.Println("GetDeviceAdjust error:, ", err)
			return err
		}

		for _, ID := range iaqDeviceMap["result"] {
			//_deviceAdjust[ID.Device] = ID
			devAdj[ID.Device] = ID
		}

		if len(a.adj) == 0 {
			a.l.Println("response Body:", string(body))
			return errors.New("GetDeviceAdjust response Body: " + string(body))
		}

		a.l.Println("Get devices: ", len(a.adj))
	}
	return nil
}

func (a Adapter) ValueNumberAdjustment(f json.Number, adjust string, sign bool) string {
	strVal := ""
	valFloat, err := f.Float64()
	if err != nil {
		return ""
	}

	adjustValue, err := strconv.ParseFloat(adjust, 64)
	if err != nil {
		adjustValue = 0.00
	}
	//a.l.Printf("Adjust Value: %.2f\n", adjustValue)

	valFloat += adjustValue
	//a.l.Println(valFloat)
	strVal = fmt.Sprintf("%.2f", valFloat)

	if !sign && valFloat <= 0 {
		strVal = "0"
	}
	//a.l.Println(strVal)

	return strVal
}

func (a Adapter) ListenMultiple(uri *url.URL, topics map[string]byte) {
	client := connect("sub", uri)
	//client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
	client.SubscribeMultiple(topics, func(client mqtt.Client, msg mqtt.Message) {

		var payload CEEPayLoad

		//err := json.Unmarshal([]byte(jsonString), &payload)
		err := json.Unmarshal(msg.Payload(), &payload)
		if err != nil {
			a.l.Println("json.Unmarshal error, ", err)
			a.l.Println("Payload data: " + string(msg.Payload()))
		}

		if len(payload.MAC) > 0 {
			a.StoreData(payload)
		}

	})
}

func (a Adapter) StoreData(payload CEEPayLoad) {
	macAddress := payload.MAC
	if len(macAddress) >= 12 {
		timeNow := time.Now().UTC().Format("2006-01-02 15:04:05")
		json := fmt.Sprintf(`
			{
				"device": "%s",
				"humidity": "%s",
				"temperature": "%s",
				"co": "%s",
				"tvoc": "%s",
				"pm2p5": "%s",
				"co2": "%s",
				"c2h2": "%s",
				"pm10": "%s",
				"pm100": "%s",

				"o3": "%s",
				"so2": "%s",
				"no2": "%s",
				"hcho": "%s",
				"date_created": "%s"
			}`, payload.MAC,
			a.ValueNumberAdjustment(payload.HUM, a.adj[macAddress].HumAdjust, false),
			a.ValueNumberAdjustment(payload.TEMP, a.adj[macAddress].TempAdjust, true),
			a.ValueNumberAdjustment(payload.CO, a.adj[macAddress].COAdjust, false),
			a.ValueNumberAdjustment(payload.TVOC, a.adj[macAddress].TVOCAdjust, false),
			a.ValueNumberAdjustment(payload.PM2P5, a.adj[macAddress].PM2p5Adjust, false),
			a.ValueNumberAdjustment(payload.CO2, a.adj[macAddress].CO2Adjust, false),
			a.ValueNumberAdjustment(payload.C2H2, a.adj[macAddress].C2H2Adjust, false),
			a.ValueNumberAdjustment(payload.PM10, a.adj[macAddress].PM10Adjust, false),
			a.ValueNumberAdjustment(payload.PM100, a.adj[macAddress].PM100Adjust, false),

			a.ValueNumberAdjustment(payload.O3, a.adj[macAddress].O3Adjust, false),
			a.ValueNumberAdjustment(payload.SO2, a.adj[macAddress].SO2Adjust, false),
			a.ValueNumberAdjustment(payload.NO2, a.adj[macAddress].NO2Adjust, false),
			a.ValueNumberAdjustment(payload.HCHO, a.adj[macAddress].HCHOAdjust, false),
			//a.ValueNumberAdjustment(payload.PM10, a.adj[macAddress].PM10Adjust, false),
			//a.ValueNumberAdjustment(payload.PM100, a.adj[macAddress].PM100Adjust, false),
			/*
				func(f sql.NullFloat64) string {
					if f.Valid {
						return fmt.Sprintf("%.2f", f.Float64)
					} else {
						return ""
					}
				}(payload.O3),
				func(f sql.NullFloat64) string {
					if f.Valid {
						return fmt.Sprintf("%.2f", f.Float64)
					} else {
						return ""
					}
				}(payload.H2S),
				func(f sql.NullFloat64) string {
					if f.Valid {
						return fmt.Sprintf("%.2f", f.Float64)
					} else {
						return ""
					}
				}(payload.SO2),
				func(f sql.NullFloat64) string {
					if f.Valid {
						return fmt.Sprintf("%.2f", f.Float64)
					} else {
						return ""
					}
				}(payload.NO2),
			*/
			timeNow)
		//fmt.Println(json)
		var jsonStr = []byte(json)

		req, err := http.NewRequest("POST", RESTURI+"/iaq/createrecord", bytes.NewBuffer(jsonStr))
		if err != nil {
			a.l.Println("Unable to create record: " + string(err.Error()))
		}
		req.Header.Set("X-Custom-Header", "creaXtive")
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			a.l.Println(err)
			return
		}
		defer resp.Body.Close()

		//fmt.Println("response Status:", resp.Status)
		/* for debug header and body responses
		fmt.Println("response Headers:", resp.Header)
		*/
		body, _ := ioutil.ReadAll(resp.Body)
		a.l.Println("response Body:", string(body))

		if len(CLOUDURI) > len("localhost") {
			req, err = http.NewRequest("POST", CLOUDURI+"/iaq/createrecord", bytes.NewBuffer(jsonStr))
			if err != nil {
				a.l.Println("Unable to create record: " + string(err.Error()))
			}
			req.Header.Set("X-Custom-Header", "creaXtive")
			req.Header.Set("Content-Type", "application/json")

			client = &http.Client{}
			resp, err = client.Do(req)
			if err != nil {
				a.l.Println(err)
				return
			}
			defer resp.Body.Close()

			//fmt.Println("response Status:", resp.Status)
			/* for debug header and body responses
			fmt.Println("response Headers:", resp.Header)
			*/
			body, _ = ioutil.ReadAll(resp.Body)
			a.l.Println("Cloud response Body:", string(body))
		}

	}
}

func (a Adapter) SendTestingData(uri *url.URL) {
	client := connect("pub", uri)
	timer := time.NewTicker(10 * time.Second)
	topic := "GASDATATEST/8caab58daaa1"
	for range timer.C {
		payload := `{"mac": "8caab58daaa1", "co": 10.0, "co2": 11, "o3": 4, "so2": 25.98, "no2": 51.07, "hcho": 1.07}`
		client.Publish(topic, 0, false, payload)
	}
}

// Data Examples:
/*
m := make(
	map[8caab58daa99:
		[
			map[
				rawInfo:[255 215 28 7 208 4 0 0 50]
				rawReading:[255 135 0 95 7 208 0 68 12 41 21 79 102] sensorType:28
			]
			map[
				rawInfo:[255 215 33 0 5 2 49 0 208]
				rawReading:[255 135 0 0 0 5 0 0 12 61 19 236 44] sensorType:33
			]
		]
	])
*/

/*
key[8caab58dac55]
value
[
	[
		map[
			rawInfo:[%!s(float64=255) %!s(float64=215) %!s(float64=36) %!s(float64=0) %!s(float64=5) %!s(float64=2) %!s(float64=48) %!s(float64=0) %!s(float64=206)
			]
			rawReading:[%!s(float64=255) %!s(float64=135) %!s(float64=0) %!s(float64=176) %!s(float64=0) %!s(float64=5) %!s(float64=0) %!s(float64=68) %!s(float64=12) %!s(float64=85) %!s(float64=19) %!s(float64=187) %!s(float64=81)
			]
			sensorType:%!s(float64=36)
		]
	]
]


value
[
	[
		map[
			rawInfo:[%!s(float64=255) %!s(float64=215) %!s(float64=28) %!s(float64=7) %!s(float64=208) %!s(float64=4) %!s(float64=0) %!s(float64=0) %!s(float64=50)
			]
			rawReading:[%!s(float64=255) %!s(float64=135) %!s(float64=0) %!s(float64=102) %!s(float64=7) %!s(float64=208) %!s(float64=0) %!s(float64=73) %!s(float64=12) %!s(float64=47) %!s(float64=21) %!s(float64=168) %!s(float64=251)
			]
			sensorType:%!s(float64=28)
		]
		map[
			rawInfo:[%!s(float64=255) %!s(float64=215) %!s(float64=33) %!s(float64=0) %!s(float64=5) %!s(float64=2) %!s(float64=49) %!s(float64=0) %!s(float64=208)
			]
			rawReading:[%!s(float64=255) %!s(float64=135) %!s(float64=0) %!s(float64=0) %!s(float64=0) %!s(float64=5) %!s(float64=0) %!s(float64=0) %!s(float64=12) %!s(float64=69) %!s(float64=20) %!s(float64=75) %!s(float64=196)
			]
			sensorType:%!s(float64=33)
		]
	]
]


*/
