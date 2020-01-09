package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"iconsole/tunnel"
	"math"
	"strconv"

	"github.com/urfave/cli"
)

const (
	X_PI   = math.Pi * 3000.0 / 180.0
	OFFSET = 0.00669342162296594323
	AXIS   = 6378245.0
)

func isOutOFChina(lon, lat float64) bool {
	return !(lon > 73.66 && lon < 135.05 && lat > 3.86 && lat < 53.55)
}

func delta(lon, lat float64) (float64, float64) {
	dlat := transformlat(lon-105.0, lat-35.0)
	dlon := transformlng(lon-105.0, lat-35.0)

	radlat := lat / 180.0 * math.Pi
	magic := math.Sin(radlat)
	magic = 1 - OFFSET*magic*magic
	sqrtmagic := math.Sqrt(magic)

	dlat = (dlat * 180.0) / ((AXIS * (1 - OFFSET)) / (magic * sqrtmagic) * math.Pi)
	dlon = (dlon * 180.0) / (AXIS / sqrtmagic * math.Cos(radlat) * math.Pi)

	mgLat := lat + dlat
	mgLon := lon + dlon

	return mgLon, mgLat
}

func transformlat(lon, lat float64) float64 {
	var ret = -100.0 + 2.0*lon + 3.0*lat + 0.2*lat*lat + 0.1*lon*lat + 0.2*math.Sqrt(math.Abs(lon))
	ret += (20.0*math.Sin(6.0*lon*math.Pi) + 20.0*math.Sin(2.0*lon*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(lat*math.Pi) + 40.0*math.Sin(lat/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(lat/12.0*math.Pi) + 320*math.Sin(lat*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformlng(lon, lat float64) float64 {
	var ret = 300.0 + lon + 2.0*lat + 0.1*lon*lon + 0.1*lon*lat + 0.1*math.Sqrt(math.Abs(lon))
	ret += (20.0*math.Sin(6.0*lon*math.Pi) + 20.0*math.Sin(2.0*lon*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(lon*math.Pi) + 40.0*math.Sin(lon/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(lon/12.0*math.Pi) + 300.0*math.Sin(lon/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}

func GCJ02ToWGS84(lon, lat float64) (float64, float64) {
	if isOutOFChina(lon, lat) {
		return lon, lat
	}

	mgLon, mgLat := delta(lon, lat)

	return lon*2 - mgLon, lat*2 - mgLat
}

func BD09ToGCJ02(lon, lat float64) (float64, float64) {
	x := lon - 0.0065
	y := lat - 0.006

	z := math.Sqrt(x*x+y*y) - 0.00002*math.Sin(y*X_PI)
	theta := math.Atan2(y, x) - 0.000003*math.Cos(x*X_PI)

	gLon := z * math.Cos(theta)
	gLat := z * math.Sin(theta)

	return gLon, gLat
}

func BD09ToWGS84(lon, lat float64) (float64, float64) {
	lon, lat = BD09ToGCJ02(lon, lat)
	return GCJ02ToWGS84(lon, lat)
}

func startSimAction(ctx *cli.Context) error {
	udid := ctx.GlobalString("UDID")
	lon := ctx.Float64("lon")
	lat := ctx.Float64("lat")
	coor := ctx.String("coor")

	if lat <= 0 && lon <= 0 {
		return errors.New("lat and lon can't be zero")
	}

	switch coor {
	case "gcj02":
		lon, lat = GCJ02ToWGS84(lon, lat)
	case "bd09":
		lon, lat = BD09ToWGS84(lon, lat)
	}

	return session("com.apple.dt.simulatelocation", udid, func(conn *tunnel.MixConnection) error {
		buf := new(bytes.Buffer)
		if err := binary.Write(buf, binary.BigEndian, uint32(0)); err != nil {
			return err
		}

		latS := []byte(strconv.FormatFloat(lat, 'E', -1, 64))
		if err := binary.Write(buf, binary.BigEndian, uint32(len(latS))); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.BigEndian, latS); err != nil {
			return err
		}

		lonS := []byte(strconv.FormatFloat(lon, 'E', -1, 64))
		if err := binary.Write(buf, binary.BigEndian, uint32(len(lonS))); err != nil {
			return err
		}
		if err := binary.Write(buf, binary.BigEndian, lonS); err != nil {
			return err
		}

		if _, err := conn.Write(buf.Bytes()); err != nil {
			return err
		}
		return nil
	})
}

func stopSimAction(ctx *cli.Context) error {
	udid := ctx.GlobalString("UDID")
	return session("com.apple.dt.simulatelocation", udid, func(conn *tunnel.MixConnection) error {
		if _, err := conn.Write([]byte{0x00, 0x00, 0x00, 0x01}); err != nil {
			return err
		}
		return nil
	})
}

func initSimCommond() cli.Command {
	return cli.Command{
		Name:      "simlocation",
		ShortName: "sim",
		Usage:     "A mounted developer disk image is required on the device, otherwise the DTSimulateLocation service is not available.",
		Subcommands: []cli.Command{
			{
				Name:   "start",
				Action: startSimAction,
				Flags: append(globalFlags, []cli.Flag{
					cli.Float64Flag{
						Name:     "latitude, lat",
						EnvVar:   "LATITUDE",
						Required: true,
						Value:    0,
					},
					cli.Float64Flag{
						Name:     "longtitude, lon",
						EnvVar:   "LONGTITUDE",
						Required: true,
						Value:    0,
					},
					cli.StringFlag{
						Name:   "coordinate, coor",
						EnvVar: "COORDINATE",
						Usage:  "coordinate name `gcj02` `wsg84` `bd09` default `gcj02`",
						Value:  "gcj02",
					},
				}...),
			},
			{
				Name:   "stop",
				Action: stopSimAction,
				Flags:  globalFlags,
			},
		},
	}
}
