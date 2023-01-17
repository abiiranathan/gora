package gora

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/goccy/go-json"
)

func TestRouterUse(t *testing.T) {
	t.Parallel()
	router := &Router{}

	var middlewareCalled bool
	var middlewareFunc MiddlewareFunc = func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			middlewareCalled = true
		}
	}

	router.Use(middlewareFunc, middlewareFunc)
	if len(router.middleware) != 2 {
		t.Error("Expected router to have 2 middleware functions")
	}

	router.GET("/", func(ctx *Context) {
		ctx.String("Hello World")
	})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	if !middlewareCalled {
		t.Fatalf("middleware not called")
	}
}

func TestInlineMiddleware(t *testing.T) {
	t.Parallel()

	router := &Router{}
	var middlewareCalled bool

	router.GET("/", func(ctx *Context) {
		ctx.String("Hello World")
	}, func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			middlewareCalled = true
		}
	})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	if !middlewareCalled {
		t.Fatalf("middleware not called")
	}
}

func TestRouterGroup(t *testing.T) {
	t.Parallel()
	router := &Router{}
	prefix := "/test"

	group := router.Group(prefix, func(next HandlerFunc) HandlerFunc {
		return func(ctx *Context) {
			next(ctx)
		}
	})

	if group.prefix != prefix {
		t.Error("Expected group to have the correct prefix")
	}

	if len(group.middleware) != 1 {
		t.Error("Expected group to have 1 middleware function")
	}

	if group.router != router {
		t.Error("Expected group to have the correct router")
	}
}

func TestRouterGroupGET(t *testing.T) {
	t.Parallel()
	router := &Router{}
	prefix := "/test"

	handlerFunc := func(ctx *Context) {}

	group := router.Group(prefix)
	group.GET("/", handlerFunc)

	if len(router.routes) != 1 {
		t.Error("Expected router to have 1 route")
	}

	route := router.routes[0]
	if route.method != http.MethodGet {
		t.Error("Expected route to have a GET method")
	}

	if route.pattern.String() != "^"+prefix+"/$" {
		t.Errorf("Expected route to have the correct pattern, got: %s", route.pattern)
	}

	if len(route.middleware) != 0 {
		t.Error("Expected route to have no middleware functions")
	}
}

func TestRouterGroupPUT(t *testing.T) {
	t.Parallel()
	router := &Router{}
	prefix := "/test"

	handlerFunc := func(ctx *Context) {}

	group := router.Group(prefix)
	group.PUT("/", handlerFunc)

	if len(router.routes) != 1 {
		t.Error("Expected router to have 1 route")
	}

	route := router.routes[0]
	if route.method != http.MethodPut {
		t.Error("Expected route to have a PUT method")
	}

	if route.pattern.String() != "^"+prefix+"/$" {
		t.Error("Expected route to have the correct pattern")
	}

	if len(route.middleware) != 0 {
		t.Error("Expected route to have no middleware functions")
	}
}

func TestRouterGroupDELETE(t *testing.T) {
	t.Parallel()
	router := &Router{}
	prefix := "/test"

	handlerFunc := func(ctx *Context) {}

	group := router.Group(prefix)
	group.DELETE("/", handlerFunc)

	if len(router.routes) != 1 {
		t.Error("Expected router to have 1 route")
	}

	route := router.routes[0]
	if route.method != http.MethodDelete {
		t.Error("Expected route to have a DELETE method")

	}
}

func TestHelloWorld(t *testing.T) {
	t.Parallel()
	HelloWorld := func(ctx *Context) {
		ctx.HTML("Hello, World!")
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	handler := Router{}
	handler.GET("/", HelloWorld)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := "Hello, World!"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestJSONParsing(t *testing.T) {
	t.Parallel()

	type User struct {
		ID   int
		Name string
	}

	JSONHandler := func(ctx *Context) {
		users := []User{
			{1, "Abiira N"},
			{2, "Dan K"},
		}
		ctx.JSON(users)
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	handler := Router{}
	handler.GET("/", JSONHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	var users []User
	json.Unmarshal(rr.Body.Bytes(), &users)

	if len(users) != 2 {
		t.Errorf("Expected users to be 2, got %d", len(users))
	}
}

func TestDataBinding(t *testing.T) {
	t.Parallel()

	type User struct {
		ID    int    `json:"id" validate:"required"`
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"email,omitempty"`
		Valid bool   `json:"valid"`
	}

	JSONHandler := func(ctx *Context) {
		var u []User

		err := ctx.BindJSON(&u)
		if err != nil {
			panic(err)
		}

		valError := ctx.Validate(u)
		if valError != nil {
			ctx.ValidationError(valError)
			return
		}

		ctx.JSON(u)
	}

	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	users := []User{
		{1, "Abiira N", "email.co.uk", true},
		{2, "Dan K", "invalid email", false},
		{3, "", "john.doe@gmail.com", false},
		{0, "Kakura", "kak.jk@gmail.com", false},
		{},
	}

	b, _ := json.Marshal(users)
	req, err := http.NewRequest("POST", "/", bytes.NewReader(b))

	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	handler := New(io.Discard)
	handler.POST("/", JSONHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}

	// Check the response body is what we expect.
	var valEror map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &valEror)
	if err != nil {
		t.Fatalf("unable to Unmarshal ValidationErrors: %v", err)
	}

	if len(valEror) <= 0 {
		t.Fatal("Expected validation errors, got none!")
	}
}

func TestNotFoundHandlerIsCalled(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	handler := Default(io.Discard)
	notFoundHandlerCalled := false
	handler.NotFound(func(ctx *Context) {
		notFoundHandlerCalled = true
	})

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	if !notFoundHandlerCalled {
		t.Errorf("NotFound Handler not called!")
	}

}

func TestPathPrefixToRegex(t *testing.T) {
	t.Parallel()

	tt := []struct {
		prefix   string
		expected string
	}{
		{prefix: "/", expected: "^/$"},
		{prefix: "/users/{id:int}", expected: `^/users/(?P<id>\d+)$`},
		{prefix: "/posts/{slug:str}", expected: `^/posts/(?P<slug>\w+)$`},
		{prefix: "/prices/{above:float}", expected: `^/prices/(?P<above>\d+\.\d+)$`},
		{
			prefix:   "/articles/{from:date}/{to:date}",
			expected: `^/articles/(?P<from>\d{4}-\d{2}-\d{2})/(?P<to>\d{4}-\d{2}-\d{2})$`,
		},
		{prefix: "/articles/{published:bool}", expected: `^/articles/(?P<published>true|false)$`},
		{
			prefix:   "/articles/{published_at:datetime}",
			expected: `^/articles/(?P<published_at>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})$`,
		},
	}

	for _, test := range tt {
		regex, err := pathPrefixToRegex(test.prefix)
		if err != nil {
			t.Fatal(err)
		}

		if regex != test.expected {
			t.Errorf("expected regex to be: %s, got: %v", test.expected, regex)
		}
	}
}

func TestRender(t *testing.T) {
	t.Parallel()

	text := `
	<h1>Username: {{ .Username }}</h1>
	<h1>Password: {{ .Password }}</h1>
	`
	f, err := os.CreateTemp("", "index.html")
	if err != nil {
		t.Fatal(err)
	}

	f.WriteString(text)
	f.Close()
	defer os.Remove(f.Name())

	type User struct {
		Username string
		Password string
	}

	data := User{Username: "johndoe", Password: "johndoe-password"}

	handlerFunc := func(ctx *Context) {
		ctx.Render(http.StatusOK, data, f.Name())
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	r := Default(io.Discard)

	r.GET("/", handlerFunc)
	r.ServeHTTP(w, req)

	expected := `
	<h1>Username: johndoe</h1>
	<h1>Password: johndoe-password</h1>
	`

	if w.Body.String() != expected {
		t.Errorf("render returns incorrect html variables: %v\n", w.Body.String())
	}
}

func TestSendFile(t *testing.T) {
	t.Parallel()

	text := "Hello World!"
	f, err := os.CreateTemp("", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	f.WriteString(text)
	f.Close()
	defer os.Remove(f.Name())

	handlerFunc := func(ctx *Context) {
		ctx.File(f.Name())
	}

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()

	r := Default(io.Discard)

	r.GET("/", handlerFunc)
	r.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Errorf("expected status code of %d, got %d", http.StatusOK, w.Result().StatusCode)
	}

	if w.Body.String() != text {
		t.Errorf("file response incorrect: %q, expected: %q", w.Body.String(), text)
	}
}

func TestIsValidEmail(t *testing.T) {
	if !IsValidEmail("user@gmail.com") {
		t.Errorf("email should be valid")
	}
}
