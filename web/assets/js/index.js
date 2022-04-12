const eventStreamResource = "/stream";
const apiMessageResource = "/message";
const apiOnlineUsers = "/users";

const ssePrefix = "sse:";
const sseTypes = ["message-sent", "user-join", "user-left"];

document.addEventListener("alpine:init", () => {
  window.s8k = {};

  window.s8k.sse = {
    setup() {
      let evtSource = new EventSource(eventStreamResource, {
        withCredentials: true,
      });

      const handleEvent = (event) => {
        document.dispatchEvent(
          new CustomEvent(ssePrefix + event.type, {
            bubbles: true,
            detail: {
              data: JSON.parse(event.data),
            },
          }),
        );
      };

      sseTypes.forEach((eventType) => {
        evtSource.addEventListener(eventType, handleEvent);
      });

      return evtSource;
    },
  };

  window.s8k.api = {
    async sendMessage(msg) {
      return await fetch(apiMessageResource, {
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ content: msg }),
      });
    },

    async fetchUsers() {
      let res = await fetch(apiOnlineUsers, {
        method: "GET",
        credentials: "include",
      });
      return await res.json();
    },
  };

  s8k.sse.setup();
});
