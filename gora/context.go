package gora

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
)

// Maximum memory in bytes for file uploads
var MaxMultipartMemory int64 = 32 << 20

// Context encapsulates request/response operations.
type Context struct {
	Request  *http.Request       // Incoming request
	Response http.ResponseWriter // http response writer
	Params   map[string]string   // Path parameters

	// Signal that request has been aborted
	aborted bool

	// Validation for structs after data binding
	validator *Validator

	// mutex to guard the data
	mu sync.RWMutex

	// Context data
	data map[string]any

	// Logger
	Logger *zerolog.Logger
}

// Returns a query parameter by key.
func (ctx *Context) Query(key string) string {
	return ctx.Request.URL.Query().Get(key)
}

var ErrInvalidParam = errors.New("invalid url parameter")

// Get parameter as an integer. If key does not exist
// not a valid integer, sends a 401 http response.
func (ctx *Context) IntParam(key string) (int, error) {
	if val, ok := ctx.Params[key]; ok {
		valInt, err := strconv.Atoi(val)
		if err != nil {
			return 0, ErrInvalidParam
		}
		return valInt, nil
	}
	return 0, ErrInvalidParam
}

// Get parameter as an integer. If key does not exist
// not a valid integer, IntParam panics.
func (ctx *Context) UintParam(key string) (uint, error) {
	if val, ok := ctx.Params[key]; ok {
		valInt, err := strconv.Atoi(val)
		if err != nil {
			return 0, ErrInvalidParam
		}
		return uint(valInt), nil
	}
	return 0, ErrInvalidParam
}

// Get parameter as an integer. If key does not exist
// not a valid integer, IntParam panics.
func (ctx *Context) IntQuery(key string) (int, error) {
	val := ctx.Query(key)
	valInt, err := strconv.Atoi(val)
	if err != nil {
		return 0, ErrInvalidParam
	}

	return valInt, nil
}

// Get parameter as an integer. If key does not exist
// not a valid integer, IntParam panics.
func (ctx *Context) UintQuery(key string) (uint, error) {
	val := ctx.Query(key)
	valInt, err := strconv.Atoi(val)
	if err != nil {
		return 0, ErrInvalidParam
	}
	return uint(valInt), nil
}

// Set value onto the context. Goroutine safe.
func (ctx *Context) Set(key string, value any) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.data[key] = value
}

// Get value from the context. Goroutine safe.
func (ctx *Context) Get(key string) (value any, ok bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	value, ok = ctx.data[key]
	return value, ok
}

// Get value from the context or panic if value not in context. Goroutine safe.
func (ctx *Context) MustGet(key string) (value any) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()

	if value, ok := ctx.data[key]; ok {
		return value
	}

	panic("value for key " + key + " not found in the context")
}

// Write the status of the response.
func (ctx *Context) Status(statusCode int) *Context {
	ctx.Response.WriteHeader(statusCode)
	return ctx
}

// Write the data into the response.
// Makes Context a Writer interface.
func (ctx *Context) Write(data []byte) (int, error) {
	return ctx.Response.Write(data)
}

// Read the request body into buffer p.
// Makes Context a Reader interface.
func (ctx *Context) Read(p []byte) (int, error) {
	return ctx.Request.Body.Read(p)
}

// Send encoded JSON response.
// Sets conent-type header as application/json.
func (ctx *Context) JSON(status int, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	ctx.Response.WriteHeader(status)
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.Write(b)

	if flusher, ok := ctx.Response.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Send binary data as response.
// Sets appropriate content-type as application/octet-stream.
func (ctx *Context) Binary(status int, data []byte) {
	ctx.Response.WriteHeader(status)
	ctx.Response.Header().Set("Content-Type", "application/octet-stream")
	ctx.Response.Write(data)
}

// Send a text response as text/plain.
func (ctx *Context) Text(status int, text string) {
	ctx.Response.WriteHeader(status)
	ctx.Response.Header().Set("Content-Type", "text/plain")
	ctx.Response.Write([]byte(text))

	if flusher, ok := ctx.Response.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Send an HTML response as text/html.
func (ctx *Context) HTML(status int, html string) {
	ctx.Response.WriteHeader(status)
	ctx.Response.Header().Set("Content-Type", "text/html")
	ctx.Response.Write([]byte(html))
}

// Send a file at filePath as a binary response with content-type application/octet-stream.
// If the file exists, it sends a 200 statusCode otherwise, it sends a 404 instead of panic.
// Will panic if io.Copy of file into ResponseWriter fails.
func (ctx *Context) File(filePath string) {
	http.ServeFile(ctx.Response, ctx.Request, filePath)
}

// Render a template/templates using template.ParseFiles using data
// and sends the resulting output as a text/html response.
func (ctx *Context) Render(status int, data any, filenames ...string) {
	tpl, err := template.ParseFiles(filenames...)
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	err = tpl.Execute(buf, data)
	if err != nil {
		panic(err)
	}

	ctx.Response.WriteHeader(status)
	ctx.Response.Header().Set("Content-Type", "text/html")
	ctx.Response.Write(buf.Bytes())
}

// Redirect the client to a different URL.
// Sets redirect code to http.StatusMovedPermanently.
func (ctx *Context) Redirect(url string) {
	http.Redirect(ctx.Response, ctx.Request, url, http.StatusMovedPermanently)
}

// Abort the request and cancel all processing down the middleware chain.
// Sends a response with the given status code and message.
func (ctx *Context) Abort(status int, message string) {
	ctx.Response.WriteHeader(status)
	ctx.Response.Write([]byte(message))
	// Set a flag on the context to indicate that the request has been aborted
	ctx.aborted = true
}

// Abort the request and cancel all processing down the middleware chain.
// Sends a response with the given status code and message.
func (ctx *Context) AbortWithError(status int, err error) {
	ctx.Response.WriteHeader(status)
	ctx.Response.Write([]byte(err.Error()))

	// Set a flag on the context to indicate that the request has been aborted
	ctx.aborted = true
}

// Bind the request body to a struct.
func (ctx *Context) BindJSON(v any) error {
	decoder := json.NewDecoder(ctx.Request.Body)
	return decoder.Decode(v)

}

// Validates structs, pointers to structs and slices/arrays of structs.
// Validate will panic if obj is not struct, slice, array or pointers to the same.
func (ctx *Context) Validate(v any) validator.ValidationErrors {
	return ctx.validator.Validate(v)
}

// Alias to ctx.BindJSON followed by ctx.Validate.
// Panics if BindJSON on v fails.
func (ctx *Context) MustBindJSON(v any) validator.ValidationErrors {
	err := ctx.BindJSON(v)
	if err != nil {
		panic("unable to bind JSON: " + err.Error())
	}
	return ctx.validator.Validate(v)
}

func (ctx *Context) BindXML(v any) error {
	decoder := xml.NewDecoder(ctx.Request.Body)
	return decoder.Decode(v)
}

// Alias to ctx.BindXML followed by ctx.Validate.
// Panics if BindXML on v fails.
// v should be a pointer to struct, slice or array.
func (ctx *Context) MustBindXML(v any) validator.ValidationErrors {
	decoder := xml.NewDecoder(ctx.Request.Body)
	err := decoder.Decode(v)
	if err != nil {
		panic(err)
	}

	return ctx.Validate(v)
}

/*
This function returns two maps: one for the form values, and one for the files.
The files map is a map of string slices of *multipart.FileHeader values,
representing the uploaded files.

Max Memory used is 32 << 20.
*/
func (ctx *Context) ParseMultipartForm() (map[string][]string, map[string][]*multipart.FileHeader, error) {
	if err := ctx.Request.ParseMultipartForm(MaxMultipartMemory); err != nil {
		return nil, nil, err
	}

	formValues := make(map[string][]string)
	for key, values := range ctx.Request.MultipartForm.Value {
		formValues[key] = values
	}

	formFiles := make(map[string][]*multipart.FileHeader)
	for key, files := range ctx.Request.MultipartForm.File {
		formFiles[key] = files
	}
	return formValues, formFiles, nil
}

// Save multiple multipart files to disk.
// Returns filenames of the saved files and an error if any of the os/io operations fail.
func (ctx *Context) SaveMultipartFiles(formFiles map[string][]*multipart.FileHeader,
	destDir string) (filnames []string, err error) {
	filenames := make([]string, 0)

	for _, files := range formFiles {
		for _, file := range files {
			filename, err := ctx.SaveMultipartFile(file, destDir)
			if err != nil {
				return nil, err
			}
			filenames = append(filenames, filename)
		}
	}
	return filenames, nil
}

// randString generates a random string of the specified length
func randString(length int) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Parse a single file from the request body.
// If no file exists for the field name, returns an error.
func (ctx *Context) ParseMultipartFile(fieldName string) (*multipart.FileHeader, error) {
	if err := ctx.Request.ParseMultipartForm(MaxMultipartMemory); err != nil {
		return nil, err
	}

	files, ok := ctx.Request.MultipartForm.File[fieldName]
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("file with field name %s not found", fieldName)
	}
	return files[0], nil
}

// Save the multipart file to disk using a random name into destDir directory.
// Returns the path to the destination filename and error if any.
func (ctx *Context) SaveMultipartFile(file *multipart.FileHeader, destDir string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Add randomness to the filename to avoid collisions
	filename := fmt.Sprintf("%s-%d-%s", file.Filename, time.Now().UnixNano(), randString(10))
	dst, err := os.Create(filepath.Join(destDir, filename))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return filename, nil
}

type TransformRule func(interface{}) (interface{}, error)

// This function takes an http.Request and a map of transformation rules as arguments.
// The transformation rules are represented by the TransformRule type,
// which is a function that takes a value of type interface{} and returns
// the transformed value and an error
func TransformRequestBody(req *http.Request, rules map[string]TransformRule) error {
	// Parse the request body as JSON and map it to a map[string]interface{}
	var body map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		return err
	}

	// Loop through the rules and apply the transformations
	for key, rule := range rules {
		// Check if the key exists in the body
		if val, ok := body[key]; ok {
			// Apply the transformation
			transformedVal, err := rule(val)
			if err != nil {
				return err
			}
			// Update the value in the body
			body[key] = transformedVal
		}
	}

	// Encode the transformed body back to JSON and update the request body
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(b))
	return nil
}

// Extract Bearer Token from Authorization header.
// If the token is not in the correct format: Bearer xxxxxxx, returns an empty string.
func (ctx *Context) BearerToken() string {
	authorization := ctx.Request.Header.Get("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
