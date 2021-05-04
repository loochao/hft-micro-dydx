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

	done  chan interface{}

	PointCh  chan *client.Point
	PointsCh chan []*client.Point
	points   []*client.Point
	stopped  int32
}

func (iw *InfluxWriter) Done() chan interface{} {
	return iw.done
}

func (iw *InfluxWriter) Stop() {
	if atomic.LoadInt32(&iw.stopped) == 0 {
		atomic.StoreInt32(&iw.stopped, 1)
		logger.Debugf("stop influx")
		close(iw.done)
		err := iw.save()
		if err != nil {
			logger.Debugf("iw.save() error from stop event, %v", err)
		}
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
	logger.Debugf("save %d %d", len(iw.points), iw.batchSize)
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  iw.database,
		Precision: "ns",
	})
	if err != nil {
		return err
	}
	if len(iw.points) <= 100*iw.batchSize {
		bp.AddPoints(iw.points)
		err = iw.influxClient.Write(bp)
		if err != nil {
			return err
		}
		iw.points = make([]*client.Point, 0)
		return nil
	} else {
		bp.AddPoints(iw.points[:100*iw.batchSize])
		err = iw.influxClient.Write(bp)
		if err != nil {
			return err
		}
		iw.points = iw.points[100*iw.batchSize:]
		return iw.save()
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
	defer logger.Debugf("START watchPoints")
	defer func() {
		logger.Debugf("EXIT watchPoints")
	}()

	saveTimer := time.NewTimer(time.Second * 3)
	defer saveTimer.Stop()
	defer iw.Stop()
	saveSilent := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-iw.done:
			return
		case pts := <-iw.PointsCh:
			err := iw.savePoints(pts)
			if err != nil {
				logger.Debugf("iw.savePoints(pts) %v", err)
			}
			break
		case <-saveTimer.C:
			saveSilent = false
		case pt := <-iw.PointCh:
			iw.points = append(iw.points, pt)
			if len(iw.points) > iw.batchSize && !saveSilent {
				saveSilent = true
				saveTimer.Reset(time.Minute)
				err := iw.save()
				if err != nil {
					logger.Debugf("iw.save() error %d %f", len(iw.points), err)
				}
			}
		}
	}
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
	}
	go iw.watchPoints(ctx)
	return iw, nil
}

