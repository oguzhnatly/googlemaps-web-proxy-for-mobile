package proxy

import (
    "appengine"
    "appengine/memcache"
    "appengine/urlfetch"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "strconv"
    "strings"
    "time"
    "io"
)

// New constants for location rounding and cache expiry.
const locationPrecision = 4  // Up to 11 metres
const cacheExpiry = time.Second * 600  // 5 minutes
const placesAPIKey = "API_KEY"
const placesURL = "https://maps.googleapis.com/maps/api/place/nearbysearch/json?key=%s&location=%s&radius=%s"

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

// Rounds off the latitude and longitude of a location.
func normalizeLocation(location string) (string, error) {
    var lat, lng float64
    var err error
    latLng := strings.Split(location, ",")
    if len(latLng) != 2 {
        return "", errors.New("Invalid location")
    }
    if lat, err = strconv.ParseFloat(latLng[0], locationPrecision); err != nil {
        return "", errors.New("Invalid location")
    }
    if lng, err = strconv.ParseFloat(latLng[1], locationPrecision); err != nil {
        return "", errors.New("Invalid location")
    }
    return fmt.Sprintf("%.2f,%.2f", lat, lng), nil
}

func formatPlaces(body io.Reader)([]byte, error) {
    var places placeResults
    if err := json.NewDecoder(body).Decode(&places); err != nil {
        return nil, err
    }
    return json.Marshal(places)
}

// fetchPlaces now stores results in the cache.
func fetchPlaces(ctx appengine.Context, location, radius string) ([]byte, error) {
    client := urlfetch.Client(ctx)
    resp, err := client.Get(fmt.Sprintf(placesURL, placesAPIKey, location, radius))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    places, err := formatPlaces(resp.Body)
    if (err == nil) {
        memcache.Set(ctx, &memcache.Item{
            Key:        location,
            Value:      places,
            Expiration: cacheExpiry,
        })
    }
    return places, err
}

// handler now retrieves results from the cache if they exist.
func handler(w http.ResponseWriter, r *http.Request) {
    radius := r.FormValue("radius")
    location, err := normalizeLocation(r.FormValue("location"))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    ctx := appengine.NewContext(r)
    var places []byte
    if cached, err := memcache.Get(ctx, location); err == nil {
        places = cached.Value
        // We use Golang's goroutines here to call fetchPlaces in the background,
        // without having to wait for the result it returns. This ensures that
        // both the cache remains fresh, and that we also remain in compliance
        // with the Google Places API Policies.
        go fetchPlaces(ctx, location, radius)
    } else if places, err = fetchPlaces(ctx, location, radius); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Add("Content-Type", "application/json; charset=utf-8")
    w.Write(places)
}

func init() {
    http.HandleFunc("/", handler)
}