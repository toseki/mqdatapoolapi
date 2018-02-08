package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/codegangsta/cli"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/toseki/mqttdatapoolapi/auth"
	"github.com/toseki/mqttdatapoolapi/httphandler"
	"github.com/toseki/mqttdatapoolapi/storage"
	"github.com/toseki/mqttdatapoolapi/sub"
)

var version string // set by the compiler
var noslack bool

func run(c *cli.Context) error {
	log.SetLevel(log.Level(uint8(c.Int("log-level"))))

	if runtime.GOOS == "windows" {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02T15:04:05.000000", DisableColors: true})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02T15:04:05.000000"})

	}

	log.WithFields(log.Fields{
		"version":   version,
		"log-level": c.Int("log-level"),
	}).Info("MQTTDATAPOOL API")

	// param check
	subtopic := c.String("mqtt-subtopic")
	if subtopic == "" || c.String("kvs-path") == "" || c.String("mysql-dsn") == "" || c.String("user-table") == "" {
		log.Errorf("missing some parameter")
		fmt.Println("ex) ")
		fmt.Printf("%s --mqtt-server tcp://<mqttserver>:1883 --mqtt-username <username> --mqtt-password <password> --mqtt-subtopic <topic> --kvs-path <path> --api-port 8080 --mysql-dsn \"user:password@tcp(host:port)/dbname\" --user-table <usertable name>\n", os.Args[0])
		os.Exit(0)
	}

	//submutex := new(sync.RWMutex)

	var mqttHandler *sub.MqttBackend
	for {
		var err error
		//mqttHandler, err = sub.NewMqttBackend(c.String("mqtt-server"), c.String("mqtt-username"), c.String("mqtt-password"), c.String("mqtt-ca-cert"), submutex, c.Int("mqtt-buf-cnt"))
		mqttHandler, err = sub.NewMqttBackend(c.String("mqtt-server"), c.String("mqtt-username"), c.String("mqtt-password"), c.String("mqtt-ca-cert"), c.Int("mqtt-buf-cnt"))
		if err == nil {
			break
		}

		log.Errorf("could not connect mqtt backend, retry in 2 seconds: %s", err)
		time.Sleep(2 * time.Second)
	}
	defer mqttHandler.Close()

	// mqtt subscribe stopic conf
	mqttHandler.SubscribeTopic(subtopic)

	// mqttmsg -> leveldb
	kvsh := storage.NewMsgKvsHandler(c.String("kvs-path"))
	defer kvsh.CloseKVSHandler()

	go func() {
		cnt := 1
		for batch := range mqttHandler.SubPayloadChan() {
			if err := kvsh.PutKVS(batch); err != nil {
				log.Warningf("PutKVS error:", err)
			}
			log.WithField("payload", fmt.Sprint(batch)).Debug("putKVS was done")

			cnt++

		}
	}()

	// echo Server
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	sqlh := storage.NewSQLHandler(c.String("mysql-dsn"), c.String("user-table"))
	defer sqlh.CloseSQLHandler()

	e.Use(auth.BasicAuth(sqlh))

	httph := httphandler.NewhttpHandler(kvsh)
	e.GET("/:base/:userparam/:param1/:param2", httph.GetData)

	go func() {
		e.Start(":" + c.String("api-port"))
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	log.WithField("signal", <-sigChan).Info("signal received")

	mqttHandler.UnSubscribeTopic(c.String("mqtt-subtopic"))

	log.Warning("shutting down application")
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "mqdatapoolapi"
	app.Usage = "MQTT to LevelDB. Data Access RestAPI"
	app.Version = version
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "mqtt-server",
			Usage:  "mqtt server (e.g. scheme://host:port where scheme is tcp, ssl or ws)",
			Value:  "tcp://127.0.0.1:1883",
			EnvVar: "MQTT_SERVER",
		},
		cli.StringFlag{
			Name:   "mqtt-subtopic",
			Usage:  "mqtt topic name",
			EnvVar: "MQTT_SUBTOPIC",
		},
		cli.StringFlag{
			Name:   "mqtt-username",
			Usage:  "mqtt server username (optional)",
			EnvVar: "MQTT_USERNAME",
		},
		cli.StringFlag{
			Name:   "mqtt-password",
			Usage:  "mqtt server password (optional)",
			EnvVar: "MQTT_PASSWORD",
		},
		cli.StringFlag{
			Name:   "mqtt-ca-cert",
			Usage:  "mqtt CA certificate file (optional)",
			EnvVar: "MQTT_CA_CERT",
		},
		cli.IntFlag{
			Name:   "mqtt-buf-cnt",
			Usage:  "mqtt msg buffer count (optional) buf required reason is limitation of slackwebhook 1msg/sec",
			Value:  100,
			EnvVar: "MQTT_BUF_CNT",
		},
		cli.StringFlag{
			Name:   "kvs-path",
			Usage:  "leveldb data path.",
			EnvVar: "KVS_PATH",
		},
		cli.StringFlag{
			Name:   "mysql-dsn",
			Usage:  "mysql connection param. ex) \"user:password@tcp(host:port)/dbname\"",
			EnvVar: "MYSQL_DSN",
		},
		cli.StringFlag{
			Name:   "user-table",
			Usage:  "mysqldb user table name ex) usertable",
			EnvVar: "USER_TABLE",
		},
		cli.StringFlag{
			Name:   "api-port",
			Usage:  "API Server Port",
			Value:  "8080",
			EnvVar: "API_PORT",
		},
		cli.IntFlag{
			Name:   "log-level",
			Value:  4,
			Usage:  "debug=5, info=4, warning=3, error=2, fatal=1, panic=0",
			EnvVar: "LOG_LEVEL",
		},
	}
	app.Run(os.Args)
}
