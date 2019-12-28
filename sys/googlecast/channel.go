/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2019
  All Rights Reserved
  Documentation http://djthorpe.github.io/gopi/
  For Licensing and Usage information, please see LICENSE.md
*/

package googlecast

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
	event "github.com/djthorpe/gopi/util/event"
	proto "github.com/gogo/protobuf/proto"

	// Protocol buffers
	pb "github.com/djthorpe/googlecast/rpc/protobuf/googlecast"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type Channel struct {
	Addr    string
	Port    uint16
	Timeout time.Duration
}

type castchannel struct {
	log       gopi.Logger
	conn      *tls.Conn
	timeout   time.Duration
	messageid int

	// The current status of the device
	app    *application
	volume *volume
	media  []*media

	sync.Mutex
	event.Tasks
	event.Publisher
}

////////////////////////////////////////////////////////////////////////////////
// CONSTANTS

const (
	DEFAULT_TIMEOUT       = 5 * time.Second
	READ_TIMEOUT          = 500 * time.Millisecond
	STATUS_INTERVAL       = 10 * time.Second
	CAST_DEFAULT_SENDER   = "sender-0"
	CAST_DEFAULT_RECEIVER = "receiver-0"
	CAST_NS_CONN          = "urn:x-cast:com.google.cast.tp.connection"
	CAST_NS_HEARTBEAT     = "urn:x-cast:com.google.cast.tp.heartbeat"
	CAST_NS_RECV          = "urn:x-cast:com.google.cast.receiver"
	CAST_NS_MEDIA         = "urn:x-cast:com.google.cast.media"
)

////////////////////////////////////////////////////////////////////////////////
// OPEN AND CLOSE

func (config Channel) Open(log gopi.Logger) (gopi.Driver, error) {
	log.Debug("<googlecast.Channel.Open>{ %+v }", config)

	this := new(castchannel)
	this.log = log
	if config.Timeout == 0 {
		this.timeout = DEFAULT_TIMEOUT
	} else {
		this.timeout = config.Timeout
	}

	addrport := fmt.Sprintf("%s:%d", config.Addr, config.Port)
	if conn, err := tls.DialWithDialer(&net.Dialer{
		Timeout:   this.timeout,
		KeepAlive: this.timeout,
	}, "tcp", addrport, &tls.Config{
		InsecureSkipVerify: true,
	}); err != nil {
		return nil, fmt.Errorf("%s: %w", addrport, err)
	} else {
		this.conn = conn
	}

	// Task to receive messages
	this.Tasks.Start(this.receive)

	// Call connect
	if err := this.Connect(); err != nil {
		return nil, err
	}

	// Return success
	return this, nil
}

func (this *castchannel) Close() error {
	this.log.Debug("<googlecast.Channel.Close>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Call disconnect, warn on any errors
	if err := this.Disconnect(); err != nil {
		this.log.Warn("Close: %v", err)
	}

	// Kill background tasks
	if err := this.Tasks.Close(); err != nil {
		return err
	}

	// Unsubscribe
	this.Publisher.Close()

	// Close connection
	if this.conn != nil {
		if err := this.conn.Close(); err != nil {
			return err
		}
	}

	// Release resoruces
	this.conn = nil
	this.app = nil
	this.volume = nil
	this.media = nil

	// Success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *castchannel) String() string {
	return fmt.Sprintf("<googlecast.Channel>{ remote_addr=%v app=%v volume=%v media=%v }", strconv.Quote(this.RemoteAddr()), this.app, this.volume, this.media)
}

////////////////////////////////////////////////////////////////////////////////
// PROPERTIES

func (this *castchannel) LocalAddr() string {
	if this.conn != nil {
		return this.conn.LocalAddr().String()
	} else {
		return "<nil>"
	}
}

func (this *castchannel) RemoteAddr() string {
	if this.conn != nil {
		return this.conn.RemoteAddr().String()
	} else {
		return "<nil>"
	}
}

////////////////////////////////////////////////////////////////////////////////
// CONNECT AND DISCONNECT MESSAGES

func (this *castchannel) Connect() error {
	this.log.Debug2("<googlecast.Channel.Connect>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Release resources
	this.app = nil
	this.volume = nil
	this.media = nil

	// Send CONNECT message
	payload := &PayloadHeader{Type: "CONNECT"}
	reqid := this.nextMessageId()
	if err := this.send(CAST_DEFAULT_SENDER, CAST_DEFAULT_RECEIVER, CAST_NS_CONN, payload.WithId(reqid)); err != nil {
		return err
	} else {
		this.Emit(&castevent{
			googlecast.CAST_EVENT_CHANNEL_CONNECT, this, nil, this, reqid,
		})
	}

	// Success
	return nil
}

func (this *castchannel) Disconnect() error {
	this.log.Debug2("<googlecast.Channel.Disconnect>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Send close message
	if reqid, err := this.DisconnectEx(); err != nil {
		return err
	} else {
		this.Emit(&castevent{
			googlecast.CAST_EVENT_CHANNEL_DISCONNECT, this, nil, this, reqid,
		})
	}

	// Success
	return nil
}

func (this *castchannel) DisconnectEx() (int, error) {
	this.log.Debug2("<googlecast.Channel.DisconnectEx>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Send close message
	payload := &PayloadHeader{Type: "CLOSE"}
	reqid := this.nextMessageId()
	if err := this.send(CAST_DEFAULT_SENDER, CAST_DEFAULT_RECEIVER, CAST_NS_CONN, payload.WithId(reqid)); err != nil {
		return 0, err
	}

	// Release resources
	this.app = nil
	this.volume = nil
	this.media = nil

	// Success
	return reqid, nil
}

////////////////////////////////////////////////////////////////////////////////
// RETURN PROPERTIES

func (this *castchannel) Application() googlecast.Application {
	return this.app
}

func (this *castchannel) Volume() googlecast.Volume {
	return this.volume
}

func (this *castchannel) Media() []googlecast.Media {
	media := make([]googlecast.Media, len(this.Media()))
	for i, m := range this.media {
		media[i] = m
	}
	return media
}

////////////////////////////////////////////////////////////////////////////////
// GET STATUS

func (this *castchannel) GetStatus() (int, error) {
	this.log.Debug2("<googlecast.Channel.GetStatus>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Get Receiver Status
	payload := &PayloadHeader{Type: "GET_STATUS"}
	if err := this.send(CAST_DEFAULT_SENDER, CAST_DEFAULT_RECEIVER, CAST_NS_RECV, payload.WithId(this.nextMessageId())); err != nil {
		return 0, err
	} else {
		return payload.RequestId, nil
	}
}

func (this *castchannel) GetMediaStatus() (int, error) {
	this.log.Debug2("<googlecast.Channel.GetMediaStatus>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Get Media Status
	payload := &PayloadHeader{Type: "GET_STATUS"}
	if this.app == nil {
		return 0, gopi.ErrOutOfOrder
	} else if err := this.send(CAST_DEFAULT_SENDER, this.app.TransportId, CAST_NS_MEDIA, payload.WithId(this.nextMessageId())); err != nil {
		return 0, err
	} else {
		return payload.RequestId, nil
	}
}

func (this *castchannel) ConnectMedia() (int, error) {
	this.log.Debug2("<googlecast.Channel.ConnectMedia>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Connect to media to begin receiving events
	payload := &PayloadHeader{Type: "CONNECT"}
	if this.app == nil {
		return 0, gopi.ErrOutOfOrder
	} else if err := this.send(CAST_DEFAULT_SENDER, this.app.TransportId, CAST_NS_CONN, payload.WithId(this.nextMessageId())); err != nil {
		return 0, err
	} else {
		return payload.RequestId, nil
	}
}

func (this *castchannel) DisconnectMedia() (int, error) {
	this.log.Debug2("<googlecast.Channel.DisconnectMedia>{ remote_addr=%v }", strconv.Quote(this.RemoteAddr()))

	// Disconnect from receiving media events
	payload := &PayloadHeader{Type: "CLOSE"}
	if this.app == nil {
		return 0, gopi.ErrOutOfOrder
	} else if err := this.send(CAST_DEFAULT_SENDER, this.app.TransportId, CAST_NS_CONN, payload.WithId(this.nextMessageId())); err != nil {
		return 0, err
	} else {
		return payload.RequestId, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// SEND MESSAGES

func (this *castchannel) send(source, dest, ns string, payload Payload) error {
	this.log.Debug2("<googlecast.Channel.Send>{ source=%v dest=%v ns=%v payload=%v }", strconv.Quote(source), strconv.Quote(dest), strconv.Quote(ns), payload)

	if json, err := json.Marshal(payload); err != nil {
		return err
	} else {
		payload_str := string(json)
		message := &pb.CastMessage{
			ProtocolVersion: pb.CastMessage_CASTV2_1_0.Enum(),
			SourceId:        &source,
			DestinationId:   &dest,
			Namespace:       &ns,
			PayloadType:     pb.CastMessage_STRING.Enum(),
			PayloadUtf8:     &payload_str,
		}
		proto.SetDefaults(message)
		if data, err := proto.Marshal(message); err != nil {
			return err
		} else if err := binary.Write(this.conn, binary.BigEndian, uint32(len(data))); err != nil {
			return err
		} else if _, err := this.conn.Write(data); err != nil {
			return err
		}
	}

	// Success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// RECEIVE MESSAGES

func (this *castchannel) receive(start chan<- event.Signal, stop <-chan event.Signal) error {
	status := time.NewTimer(500 * time.Millisecond)
	start <- gopi.DONE

	this.log.Debug("receive: Started")
FOR_LOOP:
	for {
		select {
		case <-status.C:
			if this.app == nil && this.volume == nil {
				if _, err := this.GetStatus(); err != nil {
					this.log.Warn("GetStatus: %v", err)
				}
			} else if this.app != nil && this.media == nil {
				if _, err := this.ConnectMedia(); err != nil {
					this.log.Warn("ConnectMedia: %v", err)
				} else if _, err := this.GetMediaStatus(); err != nil {
					this.log.Warn("GetMediaStatus: %v", err)
				}
			}
			// Update receiver status if empty
			status.Reset(STATUS_INTERVAL)
		case <-stop:
			status.Stop()
			break FOR_LOOP
		default:
			var length uint32
			if err := this.conn.SetReadDeadline(time.Now().Add(READ_TIMEOUT)); err != nil {
				this.log.Error("receive: %v", err)
			} else if err := binary.Read(this.conn, binary.BigEndian, &length); err != nil {
				if err == io.EOF || os.IsTimeout(err) {
					// Ignore error
				} else {
					this.log.Error("receive: %v", err)
				}
			} else if length == 0 {
				this.log.Warn("receive: Received zero-sized data")
			} else {
				payload := make([]byte, length)
				if bytes_read, err := io.ReadFull(this.conn, payload); err != nil {
					this.log.Warn("receive: %v", err)
				} else if bytes_read != int(length) {
					this.log.Warn("receive: Received different number of bytes %v read, expected %v", bytes_read, length)
				} else if err := this.receive_message(payload); err != nil {
					this.log.Warn("receive: %v", err)
				}
			}
		}
	}

	this.log.Debug("receive: Stopped")

	// Success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func (this *castchannel) nextMessageId() int {
	this.Lock()
	defer this.Unlock()
	// Cycle messages from 1 to 99999
	this.messageid = (this.messageid + 1) % 100000
	return this.messageid
}

func (this *castchannel) receive_message(data []byte) error {
	message := &pb.CastMessage{}
	if err := proto.Unmarshal(data, message); err != nil {
		return err
	}
	ns := message.GetNamespace()
	switch ns {
	case CAST_NS_RECV:
		return this.receive_message_receiver(message)
	case CAST_NS_HEARTBEAT:
		return this.receive_message_heartbeat(message)
	case CAST_NS_CONN:
		return this.receive_message_connection(message)
	case CAST_NS_MEDIA:
		return this.receive_message_media(message)
	default:
		return fmt.Errorf("Ignoring message with namespace %v", strconv.Quote(ns))
	}
}

func (this *castchannel) receive_message_receiver(message *pb.CastMessage) error {
	var header PayloadHeader
	var receiver_status ReceiverStatusResponse

	if err := json.Unmarshal([]byte(*message.PayloadUtf8), &header); err != nil {
		return err
	}

	this.log.Debug2("receive_message_receiver: %v (from %v)", header.Type, this.RemoteAddr())

	switch header.Type {
	case "RECEIVER_STATUS":
		if err := json.Unmarshal([]byte(message.GetPayloadUtf8()), &receiver_status); err != nil {
			return fmt.Errorf("RECEIVER_STATUS: %w", err)
		}
		// Set application and volume
		this.set_application(header.RequestId, receiver_status.Status.Applications)
		this.set_volume(header.RequestId, receiver_status.Status.Volume)
		// Return success
		return nil
	default:
		return fmt.Errorf("Ignoring message %v in namespace %v", strconv.Quote(header.Type), strconv.Quote(message.GetNamespace()))
	}
}

func (this *castchannel) receive_message_heartbeat(message *pb.CastMessage) error {
	var header PayloadHeader

	if err := json.Unmarshal([]byte(*message.PayloadUtf8), &header); err != nil {
		return err
	}

	this.log.Debug2("receive_message_heartbeat: %v (from %v)", header.Type, this.RemoteAddr())

	switch header.Type {
	case "PING":
		payload := &PayloadHeader{Type: "PONG", RequestId: -1}
		if err := this.send(message.GetDestinationId(), message.GetSourceId(), message.GetNamespace(), payload); err != nil {
			return fmt.Errorf("Ping error: %w", err)
		}
	default:
		return fmt.Errorf("Ignoring message %v in namespace %v", strconv.Quote(header.Type), strconv.Quote(message.GetNamespace()))
	}
	// Return success
	return nil
}

func (this *castchannel) receive_message_connection(message *pb.CastMessage) error {
	var header PayloadHeader

	if err := json.Unmarshal([]byte(*message.PayloadUtf8), &header); err != nil {
		return err
	}

	this.log.Debug2("receive_message_connection: %v (from %v)", header.Type, this.RemoteAddr())

	switch header.Type {
	default:
		return fmt.Errorf("Ignoring message %v in namespace %v", strconv.Quote(header.Type), strconv.Quote(message.GetNamespace()))
	}
	// Return success
	return nil
}

func (this *castchannel) receive_message_media(message *pb.CastMessage) error {
	var header PayloadHeader
	var media_status MediaStatusResponse

	if err := json.Unmarshal([]byte(*message.PayloadUtf8), &header); err != nil {
		return err
	}

	this.log.Debug2("receive_message_media: %v (from %v)", header.Type, this.RemoteAddr())

	switch header.Type {
	case "MEDIA_STATUS":
		if err := json.Unmarshal([]byte(message.GetPayloadUtf8()), &media_status); err != nil {
			return err
		}
		// Update media
		this.set_media(header.RequestId, media_status.Status)
	default:
		return fmt.Errorf("Ignoring message %v in namespace %v", strconv.Quote(header.Type), strconv.Quote(message.GetNamespace()))
	}
	// Return success
	return nil
}

func (this *castchannel) set_application(reqid int, values []application) {
	var set bool
	if len(values) == 0 && this.app == nil {
		// Do nothing
	} else if len(values) == 0 && this.app != nil {
		set = true
		this.app = nil
	} else if len(values) > 0 && this.app == nil {
		set = true
		this.app = &values[0]
	} else if len(values) > 0 && this.app != nil {
		if values[0].Equals(this.app) == false {
			this.app = &values[0]
			set = true
		}
	}
	if set {
		this.set_media(reqid, nil)
		this.Emit(&castevent{
			googlecast.CAST_EVENT_APPLICATION_UPDATED, this, nil, this, reqid,
		})
	}
}

func (this *castchannel) set_volume(reqid int, value volume) {
	var set bool
	if this.volume == nil {
		this.volume = &value
		set = true
	} else if value.Equals(this.volume) == false {
		this.volume = &value
		set = true
	}
	if set {
		this.Emit(&castevent{
			googlecast.CAST_EVENT_VOLUME_UPDATED, this, nil, this, reqid,
		})
	}
}

func (this *castchannel) set_media(reqid int, value []media) {
	var set bool
	if value == nil && this.media != nil {
		this.media = nil
		set = true
	} else {
		this.media = make([]*media, len(value))
		set = true
		for i, m := range value {
			this.media[i] = &m
		}
	}
	if set {
		this.Emit(&castevent{
			googlecast.CAST_EVENT_MEDIA_UPDATED, this, nil, this, reqid,
		})
	}
}
