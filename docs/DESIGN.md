# Design

Following document was established before we're started to work on the system.
It contains specification of system from the user and software engineering
perspective.

## User story

User story paragraph describes features of Szmaterlok chat and business
requirements to call it version `1.0.0`.

- User can choose its nickname in order to login into the system.
- There can be multiple users using the chat at the same time.
- Every user will get notification that someone else joined the chat.
- Every user can lookup list of user currently using the chat.
- Every user will be notify when someone leave the chat.
- Message cannot be longer than established amount of characters.
- Chat history will be stored locally in the browser.
- User is able to logout from the chat and delete the chat history at the same
  time.

In the future, there may be more features if all of the above list are
implemented.

## Architectural decisions

Below decisions describe how the system should be implemented in order to
achieve all the user story points.

- Szmaterlok is client/server web application.
- User interface will be written with usage of modern web technologies like:
  [html](https://developer.mozilla.org/en-US/docs/Web/HTML),
  [css](https://developer.mozilla.org/en-US/docs/Web/CSS) and
  [js](https://developer.mozilla.org/en-US/docs/Web/JavaScript).
- User interface will be implemented with reactive design pattern in mind.
- Users will be communicating with server through the HTTP API (data encoded in
  [JSON](https://www.json.org/) format).
- User will be receiving messages from the server in the form of events with the
  help of
  [SSE](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events).
- Application server will be written in the [Go](https://go.dev/) programming
  language, with the usage of [net/http](https://pkg.go.dev/net/http) package.
- Static files will be handled by server written in Go.
- Static files will be [embedded](https://pkg.go.dev/embed) into the server
  binary file.
- Server will be single executable (binary) file.
- Application data will be stored in sqlite3 database file.
- Target platform is linux/amd64, but it will be possible to run server on any
  os/arch supported by Go and
  [sqlite3 driver](https://pkg.go.dev/modernc.org/sqlite).
- Software architecture will be based on the fact, that server is run as a
  single OS process.
- Model of storing data will be based on
  [Event Sourcing](https://docs.microsoft.com/en-us/azure/architecture/patterns/event-sourcing)
  pattern.
- Every user interaction with the server, will have corresponding event stored
  in the event store.
- Application will be able to restore its state based on events in the event
  store.
- State of application will be stored in memory, and will be rebuild every time
  application will start.
- Software configuration will be based on single json config file and single
  environmental variable, which will point to the config file.
- Application is designed to work behind the reverse proxy.

## Interesting technologies

Below is the list of interesting technologies and services that could be used in
order to ease the development process.

- [Alpine.js](https://alpinejs.dev/) as a reactive web framework for UI.
- [chi](https://github.com/go-chi/chi), golang http router.
- [fly.io](https://fly.io/) as potential deployment platform.
- [picnicss](https://picnicss.com/) to boost native html elements.
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), native golang
  sqlite3 driver.
