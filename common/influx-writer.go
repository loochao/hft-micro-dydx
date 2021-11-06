package common

import (
	"context"
	"fmt"
	"github.com/geometrybase/hft-micro/influx/client"
	"github.com/geometrybase/hft-micro/logger"
	"sync/atomic"
	"time"
)

type InfluxWriter struct {
	address   string
	username  string
	password  string
	database  string
	batchSize int

	influxClient client.Client

	done chan interface{}

	PointCh  chan *client.Point
	PointsCh chan []*client.Point
	points   []*client.Point
	stopped  int32

	allSavedCh chan interface{}
}

func (iw *InfluxWriter) Done() chan interface{} {
	return iw.done
}

func (iw *InfluxWriter) Stop() {
	if atomic.LoadInt32(&iw.stopped) == 0 {
		atomic.StoreInt32(&iw.stopped, 1)
		logger.Debugf("stopping")
		close(iw.done)
		<-iw.allSavedCh
		logger.Debugf("stopped")
	}
}

func (iw *InfluxWriter) PushPoint(pt *client.Point) error {
	select {
	case iw.PointCh <- pt:
		return nil
	default:
		return fmt.Errorf("iw.PointCh <- pt failed, len(iw.PointCh) %d", len(iw.PointCh))
	}
}
func (iw *InfluxWriter) PushPoints(pts []*client.Point) error {
	select {
	case iw.PointsCh <- pts:
		return nil
	default:
		return fmt.Errorf("iw.PointsCh <- pts <- pt failed, len(iw.PointCh) %d", len(iw.PointCh))
	}
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
	//logger.Debugf("%d", len(iw.points))
	if len(iw.points) <= iw.batchSize {
		bp.AddPoints(iw.points)
		err = iw.influxClient.Write(bp)
		if err != nil {
			return err
		}
		iw.points = make([]*client.Point, 0)
		return nil
	} else {
		bp.AddPoints(iw.points[:iw.batchSize])
		err = iw.influxClient.Write(bp)
		if err != nil {
			return err
		}
		if len(iw.points) > iw.batchSize {
			iw.points = iw.points[iw.batchSize:]
			return iw.save()
		}
		return nil
	}
}

func (iw *InfluxWriter) savePoints(points []*client.Point) error {
	if len(points) == 0 {
		return nil
	}
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  iw.database,
		Precision: "ns",
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

func (iw *InfluxWriter) watchPoints(ctx context.Context) {
	logger.Debugf("START watchPoints")
	defer func() {
		logger.Debugf("EXIT watchPoints")
	}()

	saveSilentResetTimer := time.NewTimer(time.Second * 3)
	defer saveSilentResetTimer.Stop()
	defer iw.Stop()
	saveSilent := false
	stopped := false
	for !stopped || len(iw.PointsCh) > 0 || len(iw.PointCh) > 0 {
		select {
		case <-ctx.Done():
			stopped = true
			break
		case <-iw.done:
			stopped = true
			break
		case pts := <-iw.PointsCh:
			err := iw.savePoints(pts)
			if err != nil {
				logger.Debugf("iw.savePoints(pts) %v", err)
			}
			break
		case <-saveSilentResetTimer.C:
			saveSilent = false
		case pt := <-iw.PointCh:
			iw.points = append(iw.points, pt)
			if len(iw.points) > iw.batchSize && (!saveSilent || stopped) {
				saveSilent = true
				saveSilentResetTimer.Reset(time.Minute)
				err := iw.save()
				if err != nil {
					logger.Debugf("iw.save() error %d %v", len(iw.points), err)
				}
			}
		}
	}
	err := iw.save()
	if err != nil {
		logger.Debugf("iw.save() error %d %v", len(iw.points), err)
	}
	iw.allSavedCh <- nil
}

func NewInfluxWriter(ctx context.Context, address, username, password, database string, batchSize int) (*InfluxWriter, error) {
	if batchSize <= 0 {
		batchSize = 1
	}
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
		done:         make(chan interface{}, 1),
		PointCh:      make(chan *client.Point, 1000000),
		PointsCh:     make(chan []*client.Point, 100),
		points:       make([]*client.Point, 0),
		stopped:      0,
		allSavedCh:   make(chan interface{}),
	}
	go iw.watchPoints(ctx)
	return iw, nil
}
