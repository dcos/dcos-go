# dcos/http/transport

`dcos/http/transport` is an `http.RoundTripper` implementation that adds
`Authorization` and `User-Agent` headers to each request.

The `Authorization` header is a signed Javascript web token (JWT).

The `User-Agent` defaults to `dcos-go`, and may be customized.

If request returns 401 response code, the library will generate a new token,
sign it with a bouncer and retry the current request.

#### Warning!

This package breaks the `RoundTripper` interface spec defined in
`https://golang.org/pkg/net/http/#RoundTripper` by mutating request instance.
