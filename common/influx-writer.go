package common

import (
	"fmt"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"time"
)

type InfluxWriter struct {
	address   string
	username  string
	password  string
	database  string
	batchSize int

	influxClient client.Client

	done  chan interface{}
	stop  chan interface{}
	saved chan error

	PushCh chan *client.Point
	saveCh chan []*client.Point
	points []*client.Point
}

func (iw *InfluxWriter) Done() chan interface{} {
	return iw.done
}

func (iw *InfluxWriter) Stop() error {
	select {
	case iw.stop <- nil:
		close(iw.done)
		return <-iw.saved
	default:
		return fmt.Errorf("already stopped")
	}
}

func (iw *InfluxWriter) Push(pt *client.Point) {
	select {
	case <-iw.done:
		return
	case iw.PushCh <- pt:
	}
}
func (iw *InfluxWriter) Save(pts []*client.Point) {
	logger.Debugf("Save %d", len(pts))
	select {
	case <-iw.done:
		return
	case iw.saveCh <- pts:
	}
}

func (iw *InfluxWriter) watchStop() {
	<-iw.stop
}

func (iw *InfluxWriter) save() error {
	if len(iw.points) == 0 {
		return nil
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  iw.database,
		Precision: "ns",
	})
	if err != nil {
		return err
	}
	if len(iw.points) <= 100 {
		bp.AddPoints(iw.points)
		//logger.Debugf("SAVING %d POINTS", len(bp.Points()))
		err := iw.influxClient.Write(bp)
		if err != nil {
			return err
		}
		//logger.Debugf("SAVED %d POINTS", len(bp.Points()))
		iw.points = make([]*client.Point, 0)
		return nil
	} else {
		//logger.Debugf("SAVING %d POINTS", 300)
		bp.AddPoints(iw.points[:100])
		err := iw.influxClient.Write(bp)
		if err != nil {
			return err
		}
		//logger.Debugf("SAVED %d POINTS", 300)
		iw.points = iw.points[100:]
		return iw.save()
	}
}

func (iw *InfluxWriter) savePoints(points []*client.Point) error {
	if len(points) == 0 {
		return nil
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  iw.database,
		Precision: "s",
	})
	if err != nil {
		return err
	}
	bp.AddPoints(points)
	err = iw.influxClient.Write(bp)
	if err != nil {
		return err
	}
	return nil
}

func (iw *InfluxWriter) watchPoints() {
	defer func() {
		logger.Debugf("EXIT watchPoints")
	}()
	// 控制save的频率
	timer := time.NewTimer(time.Second * 3)
	defer timer.Stop()
	saveSilent := false
	for {
		select {
		case <-iw.done:
			//logger.Debugf("watchPoints writer is DONE, exit")
			//time.Sleep(time.Second * 3)
			close(iw.PushCh)
			for pt := range iw.PushCh {
				iw.points = append(iw.points, pt)
			}
			iw.saved <- iw.save()
			return
		case pts := <-iw.saveCh:
			logger.Debugf("savingPoints %d", len(pts))
			err := iw.savePoints(pts)
			if err != nil {
				logger.Debugf("save points error %v", err)
			} else {
				logger.Debugf("savedPoints %d", len(pts))
			}
			//time.Sleep(time.Second * 10)
		case <-timer.C:
			saveSilent = false
		case pt := <-iw.PushCh:
			iw.points = append(iw.points, pt)
			if len(iw.points) > iw.batchSize && !saveSilent {
				saveSilent = true
				timer.Reset(time.Minute)
				err := iw.save()
				if err != nil {
					logger.Debugf("save %d points error %v", len(iw.points), err)
				}
			}
		}
	}
}

func NewInfluxWriter(address, username, password, database string, batchSize int) (*InfluxWriter, error) {
	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     address,
		Username: username,
		Password: password,
		Timeout:  time.Minute * 5,
	})
	if err != nil {
		return nil, err
	}
	iw := &InfluxWriter{
		address:      address,
		username:     username,
		password:     password,
		database:     database,
		batchSize:    batchSize,
		influxClient: influxClient,
		stop:         make(chan interface{}, 1),
		done:         make(chan interface{}, 1),
		saved:        make(chan error, 1),
		PushCh:       make(chan *client.Point, 1000000),
		saveCh:       make(chan []*client.Point, 100),
		points:       make([]*client.Point, 0),
	}
	go iw.watchStop()
	go iw.watchPoints()
	return iw, nil
}
