# API

Documentation consists list of HTTP resources used for communication and SSE
types and data schemas.

## HTTP

All communication with a server is based on the HTTP protocol. Below there is a
list of HTTP resources (endpoints) with corresponding methods and other required
data, which can be used with any modern HTTP client, like web browser.

### POST `/login`

Login to the chat with given nickname. Client will receive cookie
`SzmaterlokSession` with valid session token for one week.

**Body** (required)

```
nickname=value
```

**Response**

One of the following.

- [303](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/303) -
  Successful login attempt. See `Location` header for next resource, which
  client is being redirected (it will happen automatically on browser).
- [500](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/500) - Internal
  server error. Something wen wrong, so try again later.

### POST `/logout`

Logout from the chat. This resource deletes `SzmaterlokSession` cookie.

**Response**

- [303](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/303) -
  Successful logout attempt. See `Location` header for next resource, which
  client is being redirected (it will happen automatically on browser).

### POST `/message`

Sent message to all chat clients.

**Body** (required)

```json
{
  "content": "string"
}
```

**Response**

- [202](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/202) -
  Accepted. Message will be sent to clients.

```json
{
  "data": {
    "id": "string"
  }
}
```

- [400](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/400) - Bad
  Request. Invalid body.
- [403](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/403) -
  Forbidden. Resource require authentication. See `/login` resource.

### GET `/stream`

HTTP Stream with
[SSE events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events).
Using `/stream` resource requires client to has valid `SzmaterlokSession` cookie
set.

See `SSE Events` section for more information about particular events.

## SSE Events

Every `SSE` event sent consists of `data` field. All of `data` fields of every
_szmaterlok_ event are encoded as [json](https://www.json.org/json-en.html)
object. Below you can find schemas for every event sent by `/stream` endpoint.

### message-sent

`message-sent` is fired every time when some user is sending message through
`/message` endpoint. Every user receives every message sent event.

```json
{
  "id": "string",
  "from": {
    "id": "string",
    "nickname": "string"
  },
  "content": "string",
  "sentAt": "string (datetime)"
}
```

### user-join

`user-join` event is fired by server when new user joins chat.

```json
{
  "id": "string",
  "user": {
    "id": "string",
    "nickname": "string"
  },
  "joinedAt": "string (datetime)"
}
```

### user-left

`user-left` event is fired by server when some user lefts chat.

```json
{
  "id": "string",
  "user": {
    "id": "string",
    "nickname": "string"
  },
  "leftAt": "string (datetime)"
}
```
