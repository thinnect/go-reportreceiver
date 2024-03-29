// Author  Raido Pahtma
// License MIT

// Package reportreceiver ReportReceiver
package reportreceiver

import (
	"errors"
	"fmt"
	"time"

	"github.com/proactivity-lab/go-loggers"
	"github.com/proactivity-lab/go-moteconnection"
)

const AMID_REPORTS = 9
const AM_DEFAULT_GROUP = 0x22

const HEADER_REPORTMESSAGE = 1
const HEADER_REPORTMESSAGE_ACK = 2

type ReportWriter interface {
	Append(*Report) error
}

type ReportMsg struct {
	Header   uint8
	Report   uint32
	Fragment uint8
	Total    uint8
	Data     []byte
}

type ReportMsgAck struct {
	Header  uint8
	Report  uint32
	Missing []uint8
}

type ReportData struct {
	Channel        uint8
	Id             uint32
	LocalTimeMilli uint32
	ClockTime      uint32
	Data           []byte
}

type Report struct {
	Source         moteconnection.AMAddr
	Report         uint32
	Channel        uint8
	Id             uint32
	LocalTimeMilli uint32 // MilliSeconds from boot
	ClockTime      uint32 // Seconds from the beginning of the century or 0xFFFFFFFF
	Data           []byte

	FirstRcvd time.Time
	LastRcvd  time.Time
	FragsRcvd int
}

type QueueElement struct {
	Source moteconnection.AMAddr
}


func (self *Report) StorageStringHeader() string {
	return "timestamp ff, timestamp lf, ADDR, reportnum, CHANNEL, reportid, clocktime, localtime, data"
}

func (self *Report) StorageString() string {
	tsl := "2006-01-02 15:04:05.999"
	return fmt.Sprintf("%s,%s,%s,%d,%02X,%d,%d,%d,%X", self.FirstRcvd.Format(tsl), self.LastRcvd.Format(tsl), self.Source, self.Report, self.Channel, self.Id, self.ClockTime, self.LocalTimeMilli, self.Data)
}

func (self *Report) String() string {
	return fmt.Sprintf("%s rprt %d([%02X]%d) %X", self.Source, self.Report, self.Channel, self.Id, self.Data)
}

type PartialReport struct {
	Source moteconnection.AMAddr
	Report uint32

	Total uint8

	FirstRcvd time.Time
	LastRcvd  time.Time
	FragsRcvd int

	LastContact time.Time

	Fragments map[uint8]ReportMsg
}

func (self *PartialReport) String() string {
	return fmt.Sprintf("%s rprt %d %d/%d", self.Source, self.Report, self.Total, len(self.Fragments))
}

func (self *PartialReport) AddFragment(fragment *ReportMsg) {
	self.Total = fragment.Total
	self.FragsRcvd++
	self.LastRcvd = time.Now().UTC()
	self.Fragments[fragment.Fragment] = *fragment
}

func (self *PartialReport) IsComplete() bool {
	if fragment, ok := self.Fragments[0]; ok {
		if len(self.Fragments) == int(fragment.Total) {
			return true
		}
	}
	return false
}

func (self *PartialReport) GetReport() (*Report, error) {
	if self.IsComplete() {
		report := new(Report)
		report.Source = self.Source
		report.Report = self.Report

		data := make([]byte, len(self.Fragments[0].Data)*len(self.Fragments))
		data = data[:0]
		for i := uint8(0); i < uint8(len(self.Fragments)); i++ {
			data = append(data, self.Fragments[i].Data...)
		}

		rpd := new(ReportData)
		if err := moteconnection.DeserializePacket(rpd, data); err != nil {
			return nil, err
		}

		report.Channel = rpd.Channel
		report.Id = rpd.Id
		report.ClockTime = rpd.ClockTime
		report.LocalTimeMilli = rpd.LocalTimeMilli
		report.Data = rpd.Data

		report.FirstRcvd = self.FirstRcvd
		report.LastRcvd = self.LastRcvd
		report.FragsRcvd = self.FragsRcvd

		return report, nil
	}
	return nil, errors.New("Report not complete")
}

func (self *PartialReport) Missing() []uint8 {
	missing := make([]uint8, self.Total)
	missing = missing[:0]
	for i := uint8(0); i < self.Total; i++ {
		if _, ok := self.Fragments[i]; !ok {
			missing = append(missing, i)
		}
	}
	return missing
}

func NewPartialReport(source moteconnection.AMAddr, rm *ReportMsg) *PartialReport {
	pr := new(PartialReport)
	pr.Source = source
	pr.Report = rm.Report
	pr.Fragments = make(map[uint8]ReportMsg)
	pr.FirstRcvd = time.Now().UTC()
	return pr
}

type ReportReceiver struct {
	loggers.DIWEloggers

	mconn        moteconnection.MoteConnection
	dsp          *moteconnection.MessageDispatcher
	receive      chan moteconnection.Packet
	reports      map[moteconnection.AMAddr]*PartialReport
	reportwriter ReportWriter
}

func NewReportReceiver(mconn moteconnection.MoteConnection, source moteconnection.AMAddr, group moteconnection.AMGroup) *ReportReceiver {
	rl := new(ReportReceiver)
	rl.InitLoggers()

	rl.receive = make(chan moteconnection.Packet)
	rl.reports = make(map[moteconnection.AMAddr]*PartialReport)

	rl.dsp = moteconnection.NewMessageDispatcher(moteconnection.NewMessage(group, source))
	rl.dsp.RegisterMessageReceiver(AMID_REPORTS, rl.receive)

	rl.mconn = mconn
	rl.mconn.AddDispatcher(rl.dsp)

	return rl
}

func (self *ReportReceiver) SendAck(destination moteconnection.AMAddr, report uint32, missing []uint8) {
	ack := new(ReportMsgAck)
	ack.Header = 2
	ack.Report = report
	ack.Missing = missing

	msg := self.dsp.NewMessage()
	msg.SetDestination(destination)
	msg.SetType(AMID_REPORTS)
	msg.Payload = moteconnection.SerializePacket(ack)

	self.mconn.Send(msg)
}

func (self *ReportReceiver) SetOutput(rw ReportWriter) {
	self.reportwriter = rw
}

func (self *ReportReceiver) Run() {
	self.Debug.Printf("run main loop\n")
	for {
		select {
		case packet := <-self.receive:
			msg := packet.(*moteconnection.Message)
			self.Debug.Printf("%s\n", msg)
			if len(msg.Payload) > 0 {
				if msg.Payload[0] == HEADER_REPORTMESSAGE {
					rpm := new(ReportMsg)

					if err := moteconnection.DeserializePacket(rpm, msg.Payload); err != nil {
						self.Error.Printf("%s %s\n", msg, err)
						break
					}
					if rpm.Report == 0 {
						self.Info.Printf("RESET %s\n", msg.Source())
						delete(self.reports, msg.Source())
					}

					pr, ok := self.reports[msg.Source()]
					if !ok {
						self.reports[msg.Source()] = NewPartialReport(msg.Source(), rpm)
						pr = self.reports[msg.Source()]
					}

					if rpm.Report != pr.Report {
						// Getting a different report than expected for some reason
						if rpm.Report > pr.Report { // Report skip
							if pr.IsComplete() == false {
								self.Debug.Printf("skip %s rprt %d->%d\n", msg.Source(), pr.Report, rpm.Report)
							} else {
								self.Debug.Printf("new %s rprt %d\n", msg.Source(), rpm.Report)
							}
							self.reports[msg.Source()] = NewPartialReport(msg.Source(), rpm)
							pr = self.reports[msg.Source()]
						} else { // Older report
							self.SendAck(msg.Source(), rpm.Report, nil)
							// TODO Should only ack like once really, if there are several fragments
							break
						}
					} else {
						if pr.IsComplete() {
							self.Debug.Printf("repeat %s rprt %d\n", msg.Source(), rpm.Report)
							self.SendAck(msg.Source(), rpm.Report, nil)
							break
						}
					}

					pr.AddFragment(rpm)
					self.Debug.Printf("%s\n", pr)
					if pr.IsComplete() {
						if report, err := pr.GetReport(); err == nil {
							self.Info.Printf("%s\n", report)
							err := self.reportwriter.Append(report)
							if err != nil {
								self.Error.Printf("%s\n", err)
							}
							//sends ACK if all fragments have arrived
							self.SendAck(msg.Source(), rpm.Report, nil)
						} else {
							self.Error.Printf("%s\n", err)
						}
					} else {
						self.Debug.Printf("Missing fragments!")
					}
				}
			}
		}
	}
}

// RunResetResender func
// Resends ACK packets to nodes which are awaiting for routing and have only sent RESET
func (self *ReportReceiver) RunResetResender() {
	self.Debug.Println("Run RESET ACK sender")
	for {
		time.Sleep(2 * time.Minute)
		for key, element := range self.reports {
			if 2 > element.Total {
				self.Debug.Printf("Reportlogger shall send ACK from reset queue")
				self.Debug.Printf("Sending to: %s", key)
				self.SendAck(key, element.Report, nil)
				break
			}
		}
	}
}

// RunMissingFragmentResender func
// Resends ACK packets to nodes which have not sent all fragments at current report
func (self *ReportReceiver) RunMissingFragmentResender() {
	self.Debug.Println("Run missing fragment sender")
	for {
		time.Sleep(1 * time.Minute)
		for key, element := range self.reports {
			if !element.IsComplete() {
				self.Debug.Printf("missing fragment, sending to: %s", key)
				self.SendAck(key, element.Report, nil)
				time.Sleep(5 * time.Second)
			}
		}

	}
}
