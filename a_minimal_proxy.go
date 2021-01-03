package proxy

import (
    "appengine"
    "appengine/urlfetch"
    "fmt"
    "io/ioutil"
    "net/http"
)

// Define constants for the API key and the web service URL.
const placesAPIKey = "API_KEY"
const placesURL = "https://maps.googleapis.com/maps/api/place/nearbysearch/json?key=%s&location=%s&radius=%s"

// Makes the request to the Places API.
func fetchPlaces(ctx appengine.Context, location, radius string) ([]byte, error) {
    client := urlfetch.Client(ctx)
    resp, err := client.Get(fmt.Sprintf(placesURL, placesAPIKey, location, radius))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return ioutil.ReadAll(resp.Body)
}

// Calls the fetchPlaces function and returns the results to the browser.
func handler(w http.ResponseWriter, r *http.Request) {
    ctx := appengine.NewContext(r)
    places, err := fetchPlaces(ctx, r.FormValue("location"), r.FormValue("radius"))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Add("Content-Type", "application/json; charset=utf-8")
    w.Write(places)
}

func init() {
    http.HandleFunc("/", handler)
}