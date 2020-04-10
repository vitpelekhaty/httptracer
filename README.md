Another yet simple HTTP tracer

**Warning**: not threadsafe!

## Install

``
go get https://github.com/vitpelekhaty/httptracer
``

## Usage

Just wrap your http client with a _Trace_ function from package:
```go
...
client := &http.Client{Timeout: time.Second * 5}
client = httptracer.Trace(client)
...    
```

If you want to use your own entry handler, specify it as an option:
```go
...
callback := func(entry *Entry) {
    // Do something
}

client = httptracer.Trace(client, WithCallback(callback))
```

Specify the appropriate option to keep a request/response body in the dump:
```go
...
client = httptracer.Trace(client, WithBodies(true))
...
```

Use option _WithWriter(...)_ to write a dump, in a file, for example:
```go
...
f, err := os.Create(path)
...
client = httptracer.Trace(client, WithWriter(f))
```

**Example**:
```go
empty := true

f, err := os.Create(path)

if err != nil {
	return err
}

defer func() {
	f.WriteString("]")
	f.Close()
}()

f.WriteString("[")

callback := func(entry *Entry) {
	if !empty {
		f.WriteString(",")
    }

	empty = false
}

client = Trace(client, WithWriter(f), WithCallback(callback), WithBodies(true))

form := url.Values{}
form.Set("username", "John Doe")

req, err := http.NewRequest("POST", uri, strings.NewReader(form.Encode()))

req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

if err != nil {
	return err
}

_, err = client.Do(req)

if err != nil {
	return err
}

return nil
```

**Sample of tracing**:
```json
[
  {
    "time":"2020-04-10T14:58:31.518891597+07:00",
    "request":[
      "POST /echo HTTP/1.1",
      "Host: 127.0.0.1:38251",
      "Content-Type: application/x-www-form-urlencoded;charset=utf-8",
      "",
      "username=John+Doe"
    ],
    "response":[
      "HTTP/1.1 200 OK",
      "Content-Length: 15",
      "Content-Type: text/plain; charset=utf-8",
      "Date: Fri, 10 Apr 2020 07:58:31 GMT",
      "",
      "Hello, John Doe"
    ],
    "stat":{
      "dns-lookup":0,
      "tcp-connection":579193,
      "tls-handshake":0,
      "server-processing":1411986,
      "content-transfer":262879,
      "name-lookup":0,
      "connect":579193,
      "pre-transfer":0,
      "start-transfer":1991179,
      "total":2254058
    },
    "error":null
  }
]
```