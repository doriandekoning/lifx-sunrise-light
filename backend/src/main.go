package main

import (
	"fmt"
	"log"
	"time"

	"flag"

	"github.com/jasonlvhit/gocron"
	"github.com/pdf/golifx"
	"github.com/pdf/golifx/common"
	"github.com/pdf/golifx/protocol"
	"github.com/pkg/errors"
)

func main() {

	location, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		log.Println("Unfortunately can't load a location")
		log.Println(err)
	} else {
		gocron.ChangeLoc(location)
	}

	nocron := flag.Bool("nocron", false, "Do not run wake up in cron but run it immediately")
	configLocation := flag.String("config", "config.yaml", "Location of the config file (default:config.yaml)")
	time := flag.String("time", "8:00", "Time to start the")
	flag.Parse()
	if configLocation == nil {
		panic(errors.New("Config location invalid"))
	}
	//TODO add command line argument for just running wakeup now
	if nocron != nil && !*nocron {
		fmt.Println("Cron started")
		gocron.Every(1).Day().At(*time).Do(wakeup, *configLocation)
		<-gocron.Start()
	} else {
		wakeup(*configLocation)
	}

}

func wakeup(configLocation string) {
	fmt.Println("Waking up!", time.Now())
	config, err := ReadConfig(configLocation)
	if err != nil {
		panic(errors.Wrap(err, "Cannot read config in wakeup"))
	}

	light, err := getLight(10*time.Second, config.LightID)
	if err != nil {
		panic(err)
	}
	err = initLight(*light, config.Initialcolor)
	if err != nil {
		panic(errors.Wrap(err, "Error occured during initializing light"))
	}

	startTime := time.Now()
	for startTime.Add(time.Second * time.Duration(config.Duration)).After(time.Now()) {
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
		color, err := findColor(int(time.Since(startTime).Seconds()), config)
		if err != nil {
			panic(err)
		}
		err = (*light).SetColor(*color, time.Duration(config.UpdateInterval)*time.Second)
		if err != nil {
			panic(err)
		}
	}
	//TODO do actual wakeup sequence
}

func findColor(offsetTime int, config *Config) (*common.Color, error) {
	color := common.Color{
		Saturation: 65535,
		Brightness: 10000,
		Kelvin:     6000,
	}
	values := map[string]uint16{}
	for _, transition := range config.Transitions {
		if offsetTime < transition.Starttime {
			values[transition.Type] = uint16(transition.Startvalue * 65535)
		} else if transition.Endtime < offsetTime {
			values[transition.Type] = uint16((transition.Endvalue) * 65535)
		} else {
			//Interpolate (linearely)
			scalefactor := float64(offsetTime-transition.Starttime) / float64(transition.Endtime-transition.Starttime)
			diff := float64(transition.Endvalue - transition.Startvalue)
			if transition.Type == "kelvin" {
				values[transition.Type] = uint16((float64(transition.Startvalue) + (scalefactor * diff)))
			} else {
				values[transition.Type] = uint16(((transition.Startvalue + (scalefactor * diff)) * 65535))
			}
		}
	}

	// color := common.Color{Hue: 100, Brightness: 100, Kelvin: 5000, Saturation: 100}
	color.Hue = uint16(values["hue"])
	color.Saturation = uint16(values["saturation"])
	color.Brightness = uint16(values["brightness"])
	color.Kelvin = uint16(values["kelvin"])
	return &color, nil
}

func getLight(timeout time.Duration, lampid uint64) (*common.Light, error) {
	client, err := golifx.NewClient(&protocol.V2{Reliable: true})
	if err != nil {
		panic(errors.Wrap(err, "Cannot create client"))
	}

	//Timeout to wait for discovery of light todo test if we can remove this
	time.Sleep(2 * time.Second)

	startTime := time.Now()
	for startTime.Add(timeout).After(time.Now()) {
		actLight, err := client.GetLightByID(lampid)
		if err == common.ErrNotFound {
			time.Sleep(1 * time.Second)
		} else if err != nil {
			panic(errors.Wrap(err, "Unexpected error not find wake up light"))
		} else {
			return &actLight, nil
		}

	}
	return nil, errors.New("Timeout occured when getting lamp")
}

func initLight(light common.Light, defaultColor common.Color) error {
	err := light.SetColor(defaultColor, 0)
	if err != nil {
		return err
	}
	// Small timeout to make sure the color is set before we turn the light on
	time.Sleep(500 * time.Millisecond)
	err = light.SetPowerDuration(true, 0)
	if err != nil {
		return err
	}
	return nil
}
