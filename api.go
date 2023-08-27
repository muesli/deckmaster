// Package main Deckmaster API implements a REST API for controlling Stream Decks.
//
// Swagger file generated from source by the go-swagger package.
//
// `swagger generate spec -o swagger.json -i swagger-overrides.yml -m`
//
// https://goswagger.io/
// https://github.com/go-swagger/go-swagger/
//
// Allows controlling a single Stream Deck from anywhere using a REST-like API.
// The device is modeled using four main concepts:
//
//   - The device:
//     Holds read-only information about the connected device itself, such as
//     the unique serial number, key layout, awake state, and brightness.
//     Some properties can be changed to immediately update the device state.
//
//   - The deck:
//     Holds the configuration initially loaded from a deck file. Properties can
//     be completely or partially updated to reload the correspodning widget(s)
//     with the new confiuration. Only one deck is loaded at a time.
//
//   - A Key configuration:
//     Holds the configuration for a single key. Updating it immediately reloads
//     the corresponding widget with the new configuration. A key is referenced
//     using its one-dimensional index number, or a string name which must be
//     unique per deck. Some keys may not yet have a configuration.
//
//   - A widget:
//     Holds the active widget state of a configured key. The widget state
//     cannot be updated directly and contents vary by type of widget.
//
//     Schemes: http
//     Host: localhost:4321
//     BasePath: /v1
//     Version: 1.0.0
//
//     Consumes:
//
//   - application/json
//
//     Produces:
//
//   - application/json
//
//   - image/png
//
// swagger:meta
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-openapi/runtime/middleware"
	adapter "github.com/gwatts/gin-adapter"
	"github.com/jub0bs/fcors"
	"github.com/lesismal/nbio/nbhttp"
)

// Read-only properties of a Stream Deck device.
//
// swagger:model DeviceStateRead
type jsonDeviceRead struct {
	// Device serial number.
	Serial string `json:"serial"`
	// Platform-specific device path.
	ID string `json:"id"`
	// Current brightness level (previous brightness if asleep).
	Brightness *uint `json:"brightness"`
	// Is the device asleep (screen off)?
	Asleep bool `json:"asleep"`
	// Number of key columns.
	Columns uint8 `json:"columns"`
	// Number of key rows.
	Rows uint8 `json:"rows"`
	// Number of keys.
	Keys uint8 `json:"keys"`
	// Number of screen pixels.
	Pixels uint `json:"pixels"`
	// Screen resolution in dots per inches.
	DPI uint `json:"dpi"`
	// Padding around buttons in pixels.
	Padding uint `json:"padding"`
	// Fade out time in human readable form like "250ms".
	FadeDuration string `json:"fadeDuration"`
	// Sleep timeout in human readable form like "1m30s".
	SleepTimeout string `json:"sleepTimeout"`
}

// Writeable properties of a Stream Deck device.
//
// swagger:model DeviceStateWrite
type jsonDeviceWrite struct {
	// Brightness level.
	Brightness *uint `json:"brightness"`
	// Sleep toggle.
	Asleep bool `json:"asleep"`
	// Fade out time in human readable form like "250ms".
	FadeDuration string `json:"fadeDuration"`
	// Sleep timout in human readable form like "1m30s".
	SleepTimeout string `json:"sleepTimeout"`
}

// API representation of an active deck.
//
// swagger:model DeckState
type apiDeck struct {
	// The path to the loaded deck file.
	File string `json:"file"`
	// The path to the loaded background image.
	Background string `json:"background"`
	// The active widgets.
	Widgets []apiWidget `json:"widgets"`
}

// API representation of an active widget.
//
// swagger:model WidgetState
type apiWidget struct {
	// Type of widget used for the key.
	Type string `json:"type"`
	// An action to perform when the key is pressed and released.
	Action *ActionConfig `json:"action,omitempty"`
	// An action to perform when the key is held.
	ActionHold *ActionConfig `json:"action_hold,omitempty"`
	// Widget specific state.
	State Widget `json:"state"`
}

// An API error response.
//
// swagger:model ApiError
type apiError struct {
	// An error message.
	Message string `json:"error"`
	// An optional internal error string with more details.
	Description string `json:"description,omitempty"`
	// An optional internal error object with more details.
	Object error `json:"details,omitempty"`
}

// Impements JSON marshaling for an error response.
// Transforms the supplied error into a message with an optional details
// message if an internal error was supplied, and includes the JSON
// representation of the internal error object if possible.
func (e apiError) MarshalJSON() ([]byte, error) {
	var message = e.Message
	if len(message) == 0 {
		if e.Object != nil {
			message = e.Object.Error()
		} else {
			return nil, errors.New("apiError must have a message or error object")
		}
	} else if e.Object != nil {
		e.Description = e.Object.Error()
	}

	errorOut, errEncoding := json.Marshal(e.Object)

	if errEncoding == nil && string(errorOut) == "{}" {
		return json.Marshal(&struct {
			Message string `json:"error"`
			Details string `json:"details,omitempty"`
		}{
			Message: message,
			Details: e.Description,
		})
	}
	return json.Marshal(&struct {
		Message string `json:"error"`
		Details string `json:"details,omitempty"`
		Object  error  `json:"object"`
	}{
		Message: message,
		Details: e.Description,
		Object:  e.Object,
	})

}

func restGetDevice(c *gin.Context) {
	// swagger:route GET /device getDeviceState
	//
	// Gets the complete device state.
	//
	// Responses:
	// 200: DeviceStateRead
	c.JSON(http.StatusOK, jsonDeviceRead{
		Serial:       deck.dev.Serial,
		ID:           deck.dev.ID,
		Brightness:   brightness,
		Asleep:       deck.dev.Asleep(),
		Columns:      deck.dev.Columns,
		Rows:         deck.dev.Rows,
		Keys:         deck.dev.Keys,
		Pixels:       deck.dev.Pixels,
		DPI:          deck.dev.DPI,
		Padding:      deck.dev.Padding,
		FadeDuration: fadeDuration.String(),
		SleepTimeout: sleepTimeout.String(),
	})
}

func restPutDevice(c *gin.Context) {
	// swagger:route PUT /device putDeviceState
	//
	// Updates the device state.
	//
	// Parameters:
	//  + name: body
	//    in: body
	//	  type: DeviceStateWrite
	//
	// Responses:
	// 200: DeviceStateRead
	content := jsonDeviceWrite{
		Brightness: brightness,
		Asleep:     deck.dev.Asleep(),
	}
	c.BindJSON(&content)
	var err error
	newSleepTimeout := sleepTimeout
	// Adjust sleep timeout.
	if len(content.SleepTimeout) > 0 {
		newSleepTimeout, err = time.ParseDuration(content.SleepTimeout)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Message: "Could not parse sleep timeout.", Object: err})
			return
		}
	}
	// Adjust fade duration.
	newFadeDuration := fadeDuration
	if len(content.FadeDuration) > 0 {
		newFadeDuration, err = time.ParseDuration(content.FadeDuration)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Message: "Could not parse fade duration.", Object: err})
			return
		}
	}
	if newSleepTimeout != sleepTimeout {
		sleepTimeout = newSleepTimeout
		deck.dev.SetSleepTimeout(sleepTimeout)
	}
	fadeDuration = newFadeDuration
	deck.dev.SetSleepFadeDuration(fadeDuration)

	// Toggle wake state.
	if content.Asleep && !deck.dev.Asleep() {
		deck.dev.Sleep()
	} else if !content.Asleep && deck.dev.Asleep() {
		deck.dev.Wake()
	}
	// Adjust brightness.
	if *content.Brightness > 100 {
		*content.Brightness = 100
	}
	deck.dev.SetBrightness(uint8(*content.Brightness))

	restGetDevice(c)
}

func restGetDeck(c *gin.Context) {
	// swagger:route GET /deck getDeckState
	//
	// Gets the complete deck state.
	//
	// Responses:
	//   200: DeckState
	apiModel := apiDeck{
		File: deck.File,
	}
	for _, w := range deck.Widgets {
		apiModel.Widgets = append(apiModel.Widgets, apiConvertWidget(w))
	}
	c.JSON(http.StatusOK, apiModel)
}

func restPutDeck(c *gin.Context) {
	// swagger:route PUT /deck putDeckState
	//
	// Reloads the current configration from the deck file on disk.
	//
	// Decks which have overwritten the file property can not be reloaded.
	//
	// Responses:
	// 200: DeckState
	// 500: ApiError
	if deck.File == "" {
		restAbortWithErrorStatus(http.StatusInternalServerError, errors.New("The current deck was not loaded from a file"), c)
		return
	}
	newDeck, err := LoadDeck(deck.dev, ".", deck.File)
	if err != nil {
		restAbortWithErrorStatus(http.StatusInternalServerError, err, c)
		return
	}
	deck = newDeck
	deck.updateWidgets()
	restGetDeck(c)
}

func restGetDeckBackground(c *gin.Context) {
	// swagger:route GET /deck/background putDeckBackground
	//
	// Get the active deck background.
	//
	// Set the Accepts header to get a PNG or the native representation as JSON.
	//
	// Produces:
	//  - application/json
	//  - image/png
	//
	// Responses:
	// 200: image.Image
	// 404: ApiError
	// 500: ApiError
	switch c.Request.Header.Get("Accepts") {
	case "image/png":
		c.Status(200)
		png.Encode(c.Writer, deck.Background)
		break

	case "application/json":
		c.JSON(http.StatusOK, deck.Background)
		break

	default:
		c.Status(http.StatusUnsupportedMediaType)
	}
	return
}

func restPutDeckBackground(c *gin.Context) {
	// swagger:route GET /deck/background putDeckBackground
	//
	// Set the active deck background.
	//
	// Accepts the file in a multipart/form-data property named "background".
	//
	// Responses:
	// 200:
	// 404: ApiError
	// 500: ApiError
	upload, err := c.FormFile("background")
	if err != nil {
		restAbortWithErrorStatus(http.StatusBadRequest, err, c)
		return
	}
	file, err := upload.Open()
	if err != nil {
		restAbortWithErrorStatus(http.StatusBadRequest, err, c)
		return
	}
	image, _, err := image.Decode(file)
	if err != nil {
		restAbortWithErrorStatus(http.StatusBadRequest, err, c)
		return
	}

	if err != nil {
		restAbortWithErrorStatus(http.StatusBadRequest, err, c)
		return
	}
	err = deck.replaceBackground(image)
	if err != nil {
		restAbortWithErrorStatus(http.StatusBadRequest, err, c)
		return
	}
	c.JSON(http.StatusOK, nil)
}

func restGetDeckConfig(c *gin.Context) {
	// swagger:route GET /deck/config getDeckConfig
	//
	// Gets the deck config.
	//
	// Outputs the entire configuration for the displayed deck.
	//
	// Responses:
	// 200: DeckConfig
	c.JSON(http.StatusOK, deck.Config)
}

func restPostDeckConfig(c *gin.Context) {
	// swagger:route POST /deck/config postDeckConfig
	//
	// Sets the deck config.
	//
	// Overwrites the entire configuration for the displayed deck.
	// Parameters:
	//  + name: body
	//    in: body
	//	  type: DeckConfig
	//
	// Responses:
	// 200: DeckConfig
	restHandleDeckConfigRequest(c, false)
}

func restPutDeckConfig(c *gin.Context) {
	// swagger:route PUT /deck/config putDeckConfig
	//
	// Updates the deck config.
	//
	// Merges in changes to the configuration for the displayed deck.
	//
	// Parameters:
	//  + name: body
	//    in: body
	//	  type: DeckConfig
	//
	// Responses:
	// 200: DeckConfig
	restHandleDeckConfigRequest(c, true)
}

// Handle a request to update the deck config
// The merge parameter decides if to merge the incoming config into the existing
// config, else replace it completely.
func restHandleDeckConfigRequest(c *gin.Context, merge bool) {
	var newConfig DeckConfig
	err := c.BindJSON(&newConfig)
	if err != nil {
		restAbortWithError(err, c)
		return
	}
	if newConfig.Parent != "" {
		parentConfig, err := LoadConfig(newConfig.Parent)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Message: "The parent config could not be loaded.", Object: err})
			return
		}
		newConfig = MergeDeckConfig(&newConfig, &parentConfig)
	}
	if merge {
		newConfig = MergeDeckConfig(&newConfig, &deck.Config)
	}
	if err = setDeckConfig(newConfig, deck); err != nil {
		restAbortWithError(err, c)
		return
	}
	deck.updateWidgets()
	c.JSON(http.StatusOK, deck.Config)
}

func restGetWidget(c *gin.Context) {
	// swagger:route GET /deck/widgets/{id} getWidget
	//
	// Gets the state of a single widget.
	//
	// Parameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	//
	// Responses:
	// 200: WidgetState
	// 404: ApiError
	// 500: ApiError
	index, err := apiGetKeyIndex(c.Param("id"))
	if err != nil {
		restAbortWithErrorStatus(http.StatusNotFound, err, c)
		return
	}
	for _, w := range deck.Widgets {
		if int(w.Key()) == index {
			c.JSON(http.StatusOK, apiConvertWidget(w))
			return
		}
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, apiError{Message: "Widget not found."})
}

func restGetWidgetButtonIcon(c *gin.Context) {
	// swagger:route GET /deck/widgets/{id}/icon getWidgetButtonIcon
	//
	// Get the active icon of a button widget.
	//
	// Set the Accepts header to get a PNG or the native representation as JSON.
	//
	// Parameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	// Produces:
	//  - application/json
	//  - image/png
	//
	// Responses:
	// 200: image.Image
	// 404: ApiError
	// 500: ApiError
	index, err := apiGetKeyIndex(c.Param("id"))
	if err != nil {
		restAbortWithErrorStatus(http.StatusNotFound, err, c)
		return
	}
	for _, w := range deck.Widgets {
		if int(w.Key()) == index {
			button, ok := w.(*ButtonWidget)
			if ok {
				switch c.Request.Header.Get("Accept") {
				case "image/png":
					c.Status(200)
					png.Encode(c.Writer, button.icon)
					break

				case "application/json":
					c.JSON(http.StatusOK, button.icon)
					break

				default:
					c.Status(http.StatusUnsupportedMediaType)
				}
				return
			} else {
				c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Message: "Widget is not a button with icon"})
				return
			}
		}
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, apiError{Message: "Widget not found."})
}

func restPutWidgetButtonIcon(c *gin.Context) {
	// swagger:route PUT /deck/widgets/{id}/icon putWidgetButtonIcon
	//
	// Replace the active icon of a button widget.
	//
	// Accepts the file in a multipart/form-data property named "icon".
	//
	// sParameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	// Responses:
	// 200:
	// 404: ApiError
	// 500: ApiError
	index, err := apiGetKeyIndex(c.Param("id"))
	if err != nil {
		restAbortWithErrorStatus(http.StatusNotFound, err, c)
		return
	}
	for _, w := range deck.Widgets {
		if int(w.Key()) == index {
			button, ok := w.(*ButtonWidget)
			if ok {
				upload, err := c.FormFile("icon")
				if err != nil {
					restAbortWithErrorStatus(http.StatusBadRequest, err, c)
					return
				}
				file, err := upload.Open()
				if err != nil {
					restAbortWithErrorStatus(http.StatusBadRequest, err, c)
					return
				}
				image, _, err := image.Decode(file)
				if err != nil {
					restAbortWithErrorStatus(http.StatusBadRequest, err, c)
					return
				}

				if err != nil {
					restAbortWithErrorStatus(http.StatusBadRequest, err, c)
					return
				}
				button.SetImage(image)
				button.Update()
				c.JSON(http.StatusOK, nil)
				return
			} else {
				c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Message: "Widget is not a button with icon"})
				return
			}
		}
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, apiError{Message: "Widget not found."})
}

func restGetKeyConfig(c *gin.Context) {
	// swagger:route GET /deck/keys/{id}/config getKeyConfig
	//
	// Gets the config for a single key.
	//
	// Parameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	// Responses:
	// 200: KeyConfig
	// 404: ApiError
	// 500: ApiError
	indexParam, err := apiGetKeyIndex(c.Param("id"))
	if err != nil {
		restAbortWithErrorStatus(http.StatusNotFound, err, c)
		return
	}
	index := uint8(indexParam)
	for _, config := range deck.Config.Keys {
		if config.Index == index {
			c.JSON(http.StatusOK, config)
			return
		}
	}
	restAbortWithErrorStatus(http.StatusInternalServerError, errors.New("Widget not found."), c)
}

// Transforms a key index string or key name to a widget index.
func apiGetKeyIndex(id string) (int, error) {
	indexParam, err := strconv.Atoi(id)
	if err != nil {
		for _, k := range deck.Config.Keys {
			if k.Name != "" && k.Name == id {
				return int(k.Index), nil
			}
		}
		return -1, fmt.Errorf("Key name '%s' not found.", id)
	}
	if indexParam < 0 || indexParam > len(deck.Widgets)-1 {
		return -1, fmt.Errorf("Key id '%s' is out of range.", id)
	}
	return indexParam, nil
}

func restPutKeyConfig(c *gin.Context) {
	// swagger:route PUT /deck/keys/{id}/config putKeyConfig
	//
	// Updates the config for a single key.
	//
	// Parameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	// Parameters:
	//  + name: body
	//    in: body
	//	  type: KeyConfig
	//
	// Responses:
	// 200: KeyConfig
	// 404: ApiError
	// 500: ApiError
	restHandleWidgetConfigRequest(c, true)
}

func restPostWidgetConfig(c *gin.Context) {
	// swagger:route POST /deck/keys/{id}/config postKeyConfig
	//
	// Sets the config for a single key.
	//
	// Parameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	// Parameters:
	//  + name: body
	//    in: body
	//	  type: KeyConfig
	//
	// Responses:
	// 200: KeyConfig
	// 404: ApiError
	// 500: ApiError
	restHandleWidgetConfigRequest(c, false)
}

func restHandleWidgetConfigRequest(c *gin.Context, merge bool) {
	indexParam, err := apiGetKeyIndex(c.Param("id"))
	if err != nil {
		restAbortWithErrorStatus(http.StatusNotFound, err, c)
		return
	}
	index := uint8(indexParam)
	// Default to a new empty widget configuration.
	config := KeyConfig{Index: index}
	names := map[string]uint8{}
	configIndex := -1
	// Collect widget names to check for duplicates and map to configs.
	for i, c := range deck.Config.Keys {
		if c.Name != "" {
			names[c.Name] = c.Index
		}
		// Find existing configuration to merge into, if allowed.
		if merge && c.Index == index {
			config = c
			configIndex = i
		}
	}
	// Merge the request config into a copy of the original config.
	configCopy, _ := json.Marshal(config)
	var newConfig KeyConfig
	json.Unmarshal(configCopy, &newConfig)
	err = c.BindJSON(&newConfig)
	if err != nil || newConfig.Index != index {
		restAbortWithError(errors.New("The index can not be changed."), c)
		return
	}
	if newConfig.Name != "" {
		i, ok := names[newConfig.Name]
		if ok && i != newConfig.Index {
			restAbortWithError(fmt.Errorf("The name '%s' is already taken.", newConfig.Name), c)
			return
		}
	}
	// Replace existing config and recreate the widget.
	var w Widget
	w, err = LoadWidget(deck, index, newConfig)
	if err != nil {
		restAbortWithErrorStatus(http.StatusBadRequest, err, c)
		return
	}
	if configIndex == -1 {
		deck.Config.Keys = append(deck.Config.Keys, newConfig)
	} else {
		deck.Config.Keys[configIndex] = newConfig
	}
	deck.Widgets[index] = w
	// Reload the widget.
	updateMutex.Lock()
	w.Update()
	updateMutex.Unlock()
	c.JSON(http.StatusOK, newConfig)
}

func restDeleteKeyConfig(c *gin.Context) {
	// swagger:route DELETE /deck/keys/{id}/config deleteKeyConfig
	//
	// Deletes the config for a single key.
	//
	// Parameters:
	//	+ name: id
	//    in: path
	//    description: A 0-based key index (left to right, top to bottom) or the unique name of a key
	//    required: true
	//    type: string
	//
	// Responses:
	// 	204:
	// 	404: ApiError
	indexParam, err := apiGetKeyIndex(c.Param("id"))
	if err != nil {
		restAbortWithErrorStatus(http.StatusNotFound, err, c)
		return
	}
	index := uint8(indexParam)
	configIndex := -1
	// Map to configs.
	for i, c := range deck.Config.Keys {
		// Find existing configuration to merge into, if allowed.
		if c.Index == index {
			configIndex = i
			break
		}
	}
	var w Widget
	w, err = LoadWidget(deck, index, KeyConfig{})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError{Message: "Unable to load widgets.", Object: err})
		return
	}
	if configIndex != -1 {
		deck.Config.Keys = append(deck.Config.Keys[:configIndex], deck.Config.Keys[configIndex+1:]...)
	}
	deck.Widgets[index] = w
	c.JSON(http.StatusNoContent, nil)
}

// Convert a Widget into its API representation, including a type name.
func apiConvertWidget(w Widget) apiWidget {
	return apiWidget{
		Type:       strings.Trim(reflect.TypeOf(w).String(), "*"),
		Action:     w.Action(),
		ActionHold: w.ActionHold(),
		State:      w,
	}
}

func restPostDeviceWake(c *gin.Context) {
	// swagger:route POST /device/wake wakeDevice
	//
	// Wake the device if asleep.
	//
	// Responses:
	// 	200:
	if deck.dev.Asleep() {
		err := deck.dev.Wake()
		if err != nil {
			restAbortWithErrorStatus(http.StatusInternalServerError, err, c)
			return
		}
	}
	c.Status(http.StatusOK)
}

func restPostDeviceSleep(c *gin.Context) {
	// swagger:route POST /device/sleep wakeDevice
	//
	// Put the device to sleep if awake.
	//
	// Responses:
	// 	200:
	if !deck.dev.Asleep() {
		err := deck.dev.Sleep()
		if err != nil {
			restAbortWithErrorStatus(http.StatusInternalServerError, err, c)
			return
		}
	}
	c.Status(http.StatusOK)
}

// Helper to quickly generate a BadRequest response from an error.
func restAbortWithError(err error, c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, apiError{Object: err})
}

// Helper to quickly generate a specific response code from an error.
func restAbortWithErrorStatus(code int, err error, c *gin.Context) {
	c.AbortWithStatusJSON(code, apiError{Object: err})
}

// Initialize the REST API.
func initRestApi(listenAddress string, trustedProxies string, enabledDocs bool) *nbhttp.Server {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	if len(trustedProxies) > 0 {
		router.SetTrustedProxies(strings.Split(trustedProxies, ","))
	}

	// Configure the CORS middleware.
	cors, err := fcors.AllowAccess(
		fcors.FromAnyOrigin(),
		fcors.WithMethods(
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		),
		fcors.WithRequestHeaders(
			"Content-Type",
		),
		fcors.MaxAgeInSeconds(86400),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Apply the CORS middleware to the engine.
	router.Use(adapter.Wrap(cors))

	v1 := router.Group("/v1")
	{
		v1.GET("/device", restGetDevice)
		v1.PUT("/device", restPutDevice)
		v1.POST("/device/wake", restPostDeviceWake)
		v1.POST("/device/sleep", restPostDeviceSleep)
		v1.GET("/deck/background", restGetDeckBackground)
		v1.PUT("/deck/background", restPutDeckBackground)
		v1.GET("/deck/widgets/:id", restGetWidget)
		v1.GET("/deck/widgets/:id/icon", restGetWidgetButtonIcon)
		v1.PUT("/deck/widgets/:id/icon", restPutWidgetButtonIcon)
		v1.GET("/deck", restGetDeck)
		v1.PUT("/deck", restPutDeck)
		v1.GET("/deck/config", restGetDeckConfig)
		v1.PUT("/deck/config", restPutDeckConfig)
		v1.POST("/deck/config", restPostDeckConfig)
		v1.GET("/deck/keys/:id/config", restGetKeyConfig)
		v1.PUT("/deck/keys/:id/config", restPutKeyConfig)
		v1.POST("/deck/keys/:id/config", restPostWidgetConfig)
		v1.DELETE("/deck/keys/:id/config", restDeleteKeyConfig)
	}

	if enabledDocs {
		verbosef("Documentation served on /docs.")
		router.StaticFile("/swagger.json", "./swagger.json")
		router.GET("/docs", gin.WrapH(middleware.SwaggerUI(middleware.SwaggerUIOpts{
			BasePath: "/",
			SpecURL:  "/swagger.json",
		}, http.NotFoundHandler())))
	}

	return nbhttp.NewServer(nbhttp.Config{
		Network: "tcp",
		Addrs:   []string{listenAddress},
	}, router, nil, nil)
}
