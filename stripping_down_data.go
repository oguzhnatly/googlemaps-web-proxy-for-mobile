package proxy

import (
    "appengine"
    "appengine/urlfetch"
    "encoding/json"
    "fmt"
    "net/http"
    "io"
)

const placesAPIKey = "API_KEY"
const placesURL = "https://maps.googleapis.com/maps/api/place/nearbysearch/json?key=%s&location=%s&radius=%s"

// Represents the structure of the Places API JSON response.
type placeResults struct {
    Results []struct {
        Geometry struct {
            Location struct {
                Lat float64 `json:"lat"`
                Lng float64 `json:"lng"`
            } `json:"location"`
        } `json:"geometry"`
    } `json:"results"`
}

// Takes the JSON response from the Google Places API, and converts it into a variable of type placeResult
func formatPlaces(body io.Reader)([]byte, error) {
    var places placeResults
    if err := json.NewDecoder(body).Decode(&places); err != nil {
        return nil, err
    }
    return json.Marshal(places)
}

func fetchPlaces(ctx appengine.Context, location, radius string) ([]byte, error) {
    client := urlfetch.Client(ctx)
    resp, err := client.Get(fmt.Sprintf(placesURL, placesAPIKey, location, radius))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    // Change the return value to use the new formatPlaces function.
    return formatPlaces(resp.Body)
}

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