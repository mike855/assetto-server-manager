package servermanager

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Car struct {
	Name  string
	Skins []string
}

func (c Car) PrettyName() string {
	return prettifyName(c.Name, true)
}

type Cars []Car

func (cs Cars) AsMap() map[string][]string {
	out := make(map[string][]string)

	for _, car := range cs {
		out[car.Name] = car.Skins
	}

	return out
}

func ListCars() (Cars, error) {
	var cars Cars

	carFiles, err := ioutil.ReadDir(filepath.Join(ServerInstallPath, "content", "cars"))

	if err != nil {
		return nil, err
	}

	for _, carFile := range carFiles {
		if !carFile.IsDir() {
			continue
		}

		skinFiles, err := ioutil.ReadDir(filepath.Join(ServerInstallPath, "content", "cars", carFile.Name(), "skins"))

		if err != nil && !os.IsNotExist(err) {
			// just load without skins. non-fatal
			logrus.Errorf("couldn't read car dir, err: %s", err)
		}

		var skins []string

		for _, skinFile := range skinFiles {
			if !skinFile.IsDir() {
				continue
			}

			skins = append(skins, skinFile.Name())
		}

		cars = append(cars, Car{
			Name:  carFile.Name(),
			Skins: skins,
		})
	}

	sort.Slice(cars, func(i, j int) bool {
		return cars[i].PrettyName() < cars[j].PrettyName()
	})

	return cars, nil
}

func carsHandler(w http.ResponseWriter, r *http.Request) {
	cars, err := ListCars()

	if err != nil {
		logrus.Errorf("could not get car list, err: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("content", "cars.html"), map[string]interface{}{
		"cars": cars,
	})
}

func apiCarUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Car")
}

func carDeleteHandler(w http.ResponseWriter, r *http.Request) {
	carName := mux.Vars(r)["name"]
	carsPath := filepath.Join(ServerInstallPath, "content", "cars")

	existingCars, err := ListCars()

	if err != nil {
		logrus.Errorf("could not get car list, err: %s", err)

		AddFlashQuick(w, r, "couldn't get car list")

		http.Redirect(w, r, r.Referer(), http.StatusFound)

		return
	}

	var found bool

	for _, car := range existingCars {
		if car.Name == carName {
			// Delete car
			found = true

			err := os.RemoveAll(filepath.Join(carsPath, carName))

			if err != nil {
				found = false
				logrus.Errorf("could not remove car files, err: %s", err)
			}

			break
		}
	}

	var message string

	if found {
		// confirm deletion
		message = "Car successfully deleted!"
	} else {
		// inform car wasn't found
		message = "Sorry, car could not be deleted. Are you sure it was installed?"
	}

	AddFlashQuick(w, r, message)

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
