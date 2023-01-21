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

var (
	ErrEmptyRequestBody = errors.New("empty request body")
)

// Maximum memory in bytes for file uploads
var MaxMultipartMemory int64 = 32 << 20

type Map map[string]any

type Writer struct {
	statusCode    int
	headerWritten bool
	http.ResponseWriter
}

// Implement WriteHeader to intercept the statusCode of the request for logging.
func (w *Writer) WriteHeader(statusCode int) {
	if w.headerWritten && statusCode != 0 {
		return
	}

	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(w.statusCode)
	w.headerWritten = true
}

// Context encapsulates request/response operations.
type Context struct {
	Request  *http.Request     // Incoming request
	Response *Writer           // http response writer
	Params   map[string]string // Path parameters

	// Signal that request has been aborted
	aborted bool

	// Validation for structs after data binding
	validator *Validator

	// mutex to guard the data
	mu sync.RWMutex

	// Context data
	data map[string]any

	// Logger
	Logger zerolog.Logger
}

// Returns a query parameter by key.
func (c *Context) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

var ErrInvalidParam = errors.New("invalid url parameter")

// Get parameter as an integer. If key does not exist
// not a valid integer, sends a 401 http response.
func (c *Context) IntParam(key string) (int, error) {
	if val, ok := c.Params[key]; ok {
		valInt, err := strconv.Atoi(val)
		if err != nil {
			return 0, ErrInvalidParam
		}
		return valInt, nil
	}
	return 0, ErrInvalidParam
}

// returns a parameter for the key. if it does not exist, returns an empty string.
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// Get parameter as an integer. If key does not exist
// not a valid integer, IntParam panics.
func (c *Context) UintParam(key string) (uint, error) {
	if val, ok := c.Params[key]; ok {
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
func (c *Context) IntQuery(key string) (int, error) {
	val := c.Query(key)
	valInt, err := strconv.Atoi(val)
	if err != nil {
		return 0, ErrInvalidParam
	}

	return valInt, nil
}

// Get parameter as an integer. If key does not exist
// not a valid integer, IntParam panics.
func (c *Context) UintQuery(key string) (uint, error) {
	val := c.Query(key)
	valInt, err := strconv.Atoi(val)
	if err != nil {
		return 0, ErrInvalidParam
	}
	return uint(valInt), nil
}

// Set value onto the context. Goroutine safe.
func (c *Context) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
}

// Get value from the context. Goroutine safe.
func (c *Context) Get(key string) (value any, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok = c.data[key]
	return value, ok
}

// Get value from the context or panic if value not in context. Goroutine safe.
func (c *Context) MustGet(key string) (value any) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if value, ok := c.data[key]; ok {
		return value
	}

	panic("value for key " + key + " not found in the context")
}

func (c *Context) Header(key string, value string) {
	c.Response.Header().Set(key, value)
}

// Write the status code of the response.
// Chainable.
func (c *Context) Status(statusCode int) *Context {
	c.Response.WriteHeader(statusCode)
	return c
}

// Write the data into the response.
// Makes Context a Writer interface.
func (c *Context) Write(data []byte) (int, error) {
	return c.Response.Write(data)
}

func (c *Context) StatusCode() int {
	return c.Response.statusCode
}

// Read the request body into buffer p.
// Makes Context a Reader interface.
func (c *Context) Read(p []byte) (int, error) {
	return c.Request.Body.Read(p)
}

// Send encoded JSON response.
// Sets conent-type header as application/json.
func (c *Context) JSON(data any) {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.Write(b)
}

// Send binary data as response.
// Sets appropriate content-type as application/octet-stream.
func (c *Context) Binary(data []byte) {
	c.Response.Header().Set("Content-Type", "application/octet-stream")
	c.Response.Write(data)
}

// Send a text response as text/plain.
func (c *Context) String(text string) {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.Write([]byte(text))
}

// Send an HTML response as text/html.
func (c *Context) HTML(html string) {
	c.Response.WriteHeader(http.StatusOK)
	c.Response.Header().Set("Content-Type", "text/html")
	c.Response.Write([]byte(html))
}

// Send a file at filePath as a binary response with http.ServeFile
func (c *Context) File(filePath string) {
	http.ServeFile(c.Response, c.Request, filePath)
}

// Render a template/templates using template.ParseFiles using data
// and sends the resulting output as a text/html response.
func (c *Context) Render(status int, data any, filenames ...string) {
	tpl, err := template.ParseFiles(filenames...)
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	err = tpl.Execute(buf, data)
	if err != nil {
		panic(err)
	}

	c.Response.WriteHeader(status)
	c.Response.Header().Set("Content-Type", "text/html")
	c.Response.Write(buf.Bytes())
}

// Redirect the client to a different URL.
// Sets redirect code to http.StatusMovedPermanently.
func (c *Context) Redirect(url string) {
	http.Redirect(c.Response, c.Request, url, http.StatusMovedPermanently)
}

// Abort the request and cancel all processing down the middleware chain.
// Sends a response with the given status code and message.
func (c *Context) Abort(status int, message string) {
	c.Response.WriteHeader(status)
	c.Response.Write([]byte(message))
	// Set a flag on the context to indicate that the request has been aborted
	c.aborted = true
}

// Marks the request as aborted without sending any response.
// All pending middleware will not run.
func (c *Context) AbortRequest() {
	c.aborted = true
}

// Abort the request and cancel all processing down the middleware chain.
// Sends a response with the given status code and message.
func (c *Context) AbortWithError(status int, err error) {
	c.Response.WriteHeader(status)
	c.Response.Write([]byte(err.Error()))

	// Set a flag on the context to indicate that the request has been aborted
	c.aborted = true
}

// Bind the request body to a struct.
func (c *Context) BindJSON(v any) error {
	decoder := json.NewDecoder(c.Request.Body)
	return decoder.Decode(v)
}

// Validates structs, pointers to structs and slices/arrays of structs.
// Validate will panic if obj is not struct, slice, array or pointers to the same.
func (c *Context) Validate(v any) validator.ValidationErrors {
	return c.validator.Validate(v)
}

// Alias to c.BindJSON followed by c.Validate.
// Panics if BindJSON on v fails.
func (c *Context) MustBindJSON(v any) validator.ValidationErrors {
	err := c.BindJSON(v)

	if err != nil {
		if errors.Is(err, io.EOF) {
			panic(ErrEmptyRequestBody)
		}
		panic(err)
	}

	return c.validator.Validate(v)
}

func (c *Context) BindXML(v any) error {
	decoder := xml.NewDecoder(c.Request.Body)
	return decoder.Decode(v)
}

// Alias to c.BindXML followed by c.Validate.
// Panics if BindXML on v fails.
// v should be a pointer to struct, slice or array.
func (c *Context) MustBindXML(v any) validator.ValidationErrors {
	decoder := xml.NewDecoder(c.Request.Body)
	err := decoder.Decode(v)
	if err != nil {
		panic(err)
	}
	return c.Validate(v)
}

/*
This function returns two maps: one for the form values, and one for the files.
The files map is a map of string slices of *multipart.FileHeader values,
representing the uploaded files.

Max Memory used is 32 << 20.
*/
func (c *Context) ParseMultipartForm() (map[string][]string, map[string][]*multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(MaxMultipartMemory); err != nil {
		return nil, nil, err
	}

	formValues := make(map[string][]string)
	for key, values := range c.Request.MultipartForm.Value {
		formValues[key] = values
	}

	formFiles := make(map[string][]*multipart.FileHeader)
	for key, files := range c.Request.MultipartForm.File {
		formFiles[key] = files
	}
	return formValues, formFiles, nil
}

// Save multiple multipart files to disk.
// Returns filenames of the saved files and an error if any of the os/io operations fail.
func (c *Context) SaveMultipartFiles(formFiles map[string][]*multipart.FileHeader,
	destDir string) (filnames []string, err error) {
	filenames := make([]string, 0)

	for _, files := range formFiles {
		for _, file := range files {
			filename, err := c.SaveMultipartFile(file, destDir)
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
func (c *Context) ParseMultipartFile(fieldName string) (*multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(MaxMultipartMemory); err != nil {
		return nil, err
	}

	files, ok := c.Request.MultipartForm.File[fieldName]
	if !ok || len(files) == 0 {
		return nil, fmt.Errorf("file with field name %s not found", fieldName)
	}
	return files[0], nil
}

// Save the multipart file to disk using a random name into destDir directory.
// Returns the path to the destination filename and error if any.
func (c *Context) SaveMultipartFile(file *multipart.FileHeader, destDir string) (string, error) {
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

// Extract Bearer Token from Authorization header.
// If the token is not in the correct format: Bearer xxxxxxx, returns an empty string.
func (c *Context) BearerToken() string {
	authorization := c.Request.Header.Get("Authorization")
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
