package sub

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	log "github.com/sirupsen/logrus"
)

// MqttBackend Backend implements a MQTT sub backend.
type MqttBackend struct {
	conn           mqtt.Client
	subPayloadChan chan map[string][]byte
	subTopic       string
	mutex          sync.RWMutex
	// submutex       *sync.RWMutex
}

// NewMqttBackend creates a new Backend.
//func NewMqttBackend(server, username, password, cafile string, submutex *sync.RWMutex, bufcnt int) (*MqttBackend, error) {
func NewMqttBackend(server, username, password, cafile string, bufcnt int) (*MqttBackend, error) {

	b := MqttBackend{
		subPayloadChan: make(chan map[string][]byte, bufcnt),
		//submutex:       submutex,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(server)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetOnConnectHandler(b.onConnected)
	opts.SetConnectionLostHandler(b.onConnectionLost)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(300 * time.Second)

	if cafile != "" {
		tlsconfig, err := NewTLSConfig(cafile)
		if err == nil {
			opts.SetTLSConfig(tlsconfig)
		}
	}

	log.WithField("server", server).Info("mqtt: connecting to mqtt broker")
	b.conn = mqtt.NewClient(opts)
	if token := b.conn.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &b, nil
}

// NewTLSConfig returns the TLS configuration.
func NewTLSConfig(cafile string) (*tls.Config, error) {
	// Import trusted certificates from CAfile.pem.

	cert, err := ioutil.ReadFile(cafile)
	if err != nil {
		log.Errorf("mqtt: couldn't load cafile: %s", err)
		return nil, err
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(cert)

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
	}, nil
}

// Close closes the backend.
func (b *MqttBackend) Close() {
	b.conn.Disconnect(250) // wait 250 milisec to complete pending actions
}

// SubPayloadChan returns the mqtt.Message channel.
func (b *MqttBackend) SubPayloadChan() chan map[string][]byte {
	return b.subPayloadChan
}

func (b *MqttBackend) SubscribeTopic(topic string) {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	b.subTopic = topic
}

func (b *MqttBackend) UnSubscribeTopic(topic string) error {
	defer b.mutex.Unlock()
	b.mutex.Lock()

	log.WithField("topic", topic).Info("mqtt: unsubscribing from topic")
	if token := b.conn.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (b *MqttBackend) subPacketHandler(c mqtt.Client, msg mqtt.Message) {
	//b.submutex.Lock() // waiting for previous send slack.
	//defer b.submutex.Unlock()

	log.WithField("topic", msg.Topic()).Info("mqtt: received message")
	//fmt.Println(string(msg.Payload()))

	// check queue high usage.
	if len(b.subPayloadChan) > cap(b.subPayloadChan)-10 {
		log.WithFields(log.Fields{
			"queue-count": len(b.subPayloadChan),
			"queue-max":   cap(b.subPayloadChan),
		}).Warning("High Usage: subPayload-queue space.")
		//time.Sleep(200 * time.Millisecond)
	}

	log.WithFields(log.Fields{
		"queue-count": len(b.subPayloadChan),
		"queue-max":   cap(b.subPayloadChan),
	}).Debug("subPayload queue usage")

	batch := make(map[string][]byte)
	batch[string(msg.Topic())] = msg.Payload()

	//b.subPayloadChan <- string(msg.Payload())
	b.subPayloadChan <- batch
	log.Debug("mqtt: updated payload channel")
}

func (b *MqttBackend) onConnected(c mqtt.Client) {
	defer b.mutex.RUnlock()
	b.mutex.RLock()

	for {
		log.WithField("topic", b.subTopic).Info("mqtt: subscribing to topic")

		if token := b.conn.Subscribe(b.subTopic, 1, b.subPacketHandler); token.Wait() && token.Error() != nil {
			log.Errorf("mqtt: Subscribe error: %s", token.Error())
			time.Sleep(time.Second)
			continue
		}
		break
	}
	log.Info("mqtt: connected to mqtt broker")
}

func (b *MqttBackend) onConnectionLost(c mqtt.Client, reason error) {
	log.Errorf("mqtt: mqtt connection error: %s", reason)
}
