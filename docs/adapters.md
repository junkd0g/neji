# Using neji with chi, echo and gin

neji itself depends only on the standard library, so router adapters are
shipped as recipes rather than as imports: copy the few lines you need.
Each recipe below was compiled and run against the real router (chi
v5.3, echo v4.15, gin v1.12) when written, so the code is known to
compile and behave.

Every recipe uses `nerror.WriteFor` (or `nerror.Handler`, which calls it),
so you get the full behaviour: an incoming `X-Request-ID` becomes the
response's correlation ID, `OnWriteRequest` sees the request, and panics
become the same JSON 500 as everywhere else.

## chi

chi routes plain `http.Handler`s, so `nerror.Handler` needs no adapter:

```go
r := chi.NewRouter()
r.Method("GET", "/users/{id}", nerror.Handler(func(w http.ResponseWriter, r *http.Request) error {
	user, err := findUser(chi.URLParam(r, "id"))
	if err != nil {
		return Errors.Wrap("user_not_found", err)
	}
	return json.NewEncoder(w).Encode(user)
}))
```

## echo

echo handlers already return errors. Point echo's error handler at neji
and every error, including echo's own (unknown route, method not allowed,
body too large), is rendered in the same shape:

```go
e := echo.New()
e.HTTPErrorHandler = func(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		err = nerror.New(he.Code, "http_error", fmt.Sprint(he.Message)).Wrap(he.Internal)
	}
	nerror.WriteFor(c.Response(), c.Request(), err)
}

e.GET("/users/:id", func(c echo.Context) error {
	user, err := findUser(c.Param("id"))
	if err != nil {
		return Errors.Wrap("user_not_found", err)
	}
	return c.JSON(http.StatusOK, user)
})
```

Keep echo's `Recover()` middleware; it turns panics into an
`*echo.HTTPError`, which the handler above renders as a neji 500.

## gin

gin handlers do not return errors, so wrap them:

```go
// Handler adapts an error-returning handler into a gin.HandlerFunc.
func Handler(h func(*gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c); err != nil {
			nerror.WriteFor(c.Writer, c.Request, err)
			c.Abort()
		}
	}
}

// Recovery replaces gin.Recovery so panics produce the same JSON 500.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, rec any) {
		nerror.WriteFor(c.Writer, c.Request,
			nerror.Internal("internal server error").Wrap(fmt.Errorf("panic: %v", rec)))
		c.Abort()
	})
}
```

```go
r := gin.New()
r.Use(Recovery())
r.GET("/users/:id", Handler(func(c *gin.Context) error {
	user, err := findUser(c.Param("id"))
	if err != nil {
		return Errors.Wrap("user_not_found", err)
	}
	c.JSON(http.StatusOK, user)
	return nil
}))
```

## Anything else

If your router hands you an `http.ResponseWriter` and an `*http.Request`,
call `nerror.WriteFor(w, r, err)`. If it hands you only the writer, call
`nerror.Write(w, err)`; you lose request-ID pickup and `OnWriteRequest`,
nothing else.
